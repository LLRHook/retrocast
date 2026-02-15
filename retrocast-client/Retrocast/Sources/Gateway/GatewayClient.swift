import Foundation

@Observable @MainActor
final class GatewayClient {
    enum State: Sendable {
        case disconnected
        case connecting
        case connected
        case resuming
    }

    private(set) var state: State = .disconnected
    private var webSocket: URLSessionWebSocketTask?
    private var sessionID: String?
    private var lastSequence: Int64 = 0
    private var heartbeatTask: Task<Void, Never>?
    private var readTask: Task<Void, Never>?
    private var reconnectionStrategy = ReconnectionStrategy()
    private var token: String?
    private var gatewayURL: URL?

    private let decoder = JSONDecoder()
    private let encoder = JSONEncoder()

    /// Called for every DISPATCH (Op 0) event with (eventName, rawJSONData).
    var onEvent: ((GatewayEventType, Data) -> Void)?

    /// Called when connection state changes.
    var onStateChange: ((State) -> Void)?

    // MARK: - Public

    func connect(to url: URL, token: String) {
        self.gatewayURL = url
        self.token = token
        self.reconnectionStrategy.reset()
        doConnect()
    }

    func disconnect() {
        state = .disconnected
        heartbeatTask?.cancel()
        readTask?.cancel()
        webSocket?.cancel(with: .goingAway, reason: nil)
        webSocket = nil
        onStateChange?(.disconnected)
    }

    // MARK: - Connection

    private func doConnect() {
        guard let url = gatewayURL else { return }

        state = .connecting
        onStateChange?(.connecting)

        let session = URLSession(configuration: .default)
        webSocket = session.webSocketTask(with: url)
        webSocket?.resume()

        readTask = Task { [weak self] in
            await self?.readLoop()
        }
    }

    // MARK: - Read loop

    private func readLoop() async {
        guard let ws = webSocket else { return }

        while !Task.isCancelled {
            do {
                let message = try await ws.receive()
                switch message {
                case .data(let data):
                    handleMessage(data)
                case .string(let text):
                    if let data = text.data(using: .utf8) {
                        handleMessage(data)
                    }
                @unknown default:
                    break
                }
            } catch {
                // Connection lost
                if !Task.isCancelled {
                    await handleDisconnect()
                }
                return
            }
        }
    }

    // MARK: - Message handling

    private func handleMessage(_ data: Data) {
        guard let payload = try? decoder.decode(GatewayPayload.self, from: data) else { return }

        // Track sequence number
        if let seq = payload.s {
            lastSequence = seq
        }

        guard let op = GatewayOpCode(rawValue: payload.op) else { return }

        switch op {
        case .hello:
            handleHello(data)
        case .heartbeatAck:
            // Heartbeat acknowledged — connection is healthy
            break
        case .dispatch:
            handleDispatch(payload, rawData: data)
        case .reconnect:
            Task { await attemptReconnect() }
        default:
            break
        }
    }

    private func handleHello(_ data: Data) {
        // Extract heartbeat interval from the nested "d" field
        struct HelloPayload: Codable { let op: Int; let d: HelloData }
        guard let hello = try? decoder.decode(HelloPayload.self, from: data) else { return }

        startHeartbeat(intervalMs: hello.d.heartbeatInterval)

        // Send IDENTIFY or RESUME
        if let sessionID, state == .resuming {
            sendResume(sessionID: sessionID)
        } else {
            sendIdentify()
        }
    }

    private func handleDispatch(_ payload: GatewayPayload, rawData: Data) {
        guard let eventName = payload.t,
              let eventType = GatewayEventType(rawValue: eventName) else { return }

        // Extract the "d" field as raw JSON for the event handler to decode
        struct RawPayload: Codable { let d: AnyCodable? }
        if let dData = extractDField(from: rawData) {
            if eventType == .ready {
                handleReady(dData)
            }
            onEvent?(eventType, dData)
        }
    }

    private func handleReady(_ data: Data) {
        guard let ready = try? decoder.decode(ReadyData.self, from: data) else { return }
        sessionID = ready.sessionID
        state = .connected
        reconnectionStrategy.reset()
        onStateChange?(.connected)
    }

    // MARK: - Send

    private func sendIdentify() {
        guard let token else { return }
        let identify = IdentifyData(token: token)
        sendPayload(op: .identify, data: identify)
    }

    private func sendResume(sessionID: String) {
        guard let token else { return }
        let resume = ResumeData(token: token, sessionID: sessionID, seq: lastSequence)
        sendPayload(op: .resume, data: resume)
    }

    private func sendHeartbeat() {
        sendPayload(op: .heartbeat, data: nil as String?)
    }

    private func sendPayload<T: Encodable>(op: GatewayOpCode, data: T?) {
        let payload = GatewaySendPayload(op: op.rawValue, d: data)
        guard let jsonData = try? encoder.encode(payload),
              let text = String(data: jsonData, encoding: .utf8) else { return }
        webSocket?.send(.string(text)) { _ in }
    }

    // MARK: - Heartbeat

    private func startHeartbeat(intervalMs: Int) {
        heartbeatTask?.cancel()
        let interval = TimeInterval(intervalMs) / 1000.0

        heartbeatTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(interval))
                guard !Task.isCancelled else { return }
                self?.sendHeartbeat()
            }
        }
    }

    // MARK: - Reconnection

    private func handleDisconnect() async {
        heartbeatTask?.cancel()
        webSocket = nil

        guard reconnectionStrategy.canRetry else {
            state = .disconnected
            onStateChange?(.disconnected)
            return
        }

        await attemptReconnect()
    }

    private func attemptReconnect() async {
        guard reconnectionStrategy.canRetry else {
            state = .disconnected
            onStateChange?(.disconnected)
            return
        }

        let delay = reconnectionStrategy.nextDelay()
        state = sessionID != nil ? .resuming : .connecting
        onStateChange?(state)

        try? await Task.sleep(for: .seconds(delay))
        guard !Task.isCancelled else { return }
        doConnect()
    }

    // MARK: - Helpers

    /// Extract the "d" field from a raw gateway JSON payload as raw bytes.
    private func extractDField(from data: Data) -> Data? {
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let d = json["d"] else { return nil }
        return try? JSONSerialization.data(withJSONObject: d)
    }
}
