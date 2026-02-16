import type { GatewayPayload } from "@/types";

const OP_DISPATCH = 0;
const OP_HEARTBEAT = 1;
const OP_IDENTIFY = 2;
const OP_PRESENCE_UPDATE = 3;
const OP_RESUME = 6;
const OP_RECONNECT = 7;
const OP_HELLO = 10;
const OP_HEARTBEAT_ACK = 11;

const BACKOFF_BASE = 1000;
const BACKOFF_MAX = 60000;
const MAX_RETRIES = 10;
const JITTER_FACTOR = 0.1;

type EventCallback = (data: unknown) => void;

interface HelloData {
  heartbeat_interval: number;
}

interface ReadyData {
  session_id: string;
  user_id: string;
  guilds: string[];
}

export class GatewayClient {
  private ws: WebSocket | null = null;
  private sessionId: string | null = null;
  private sequence = 0;
  private heartbeatInterval: number | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private heartbeatAcked = true;
  private listeners = new Map<string, Set<EventCallback>>();
  private retryCount = 0;
  private url: string | null = null;
  private token: string | null = null;
  private intentionalClose = false;

  connect(url: string, token: string) {
    this.url = url;
    this.token = token;
    this.intentionalClose = false;
    this.retryCount = 0;
    this.openSocket();
  }

  disconnect() {
    this.intentionalClose = true;
    this.cleanup();
  }

  updatePresence(status: "online" | "idle" | "dnd" | "invisible") {
    this.send({ op: OP_PRESENCE_UPDATE, d: { status } });
  }

  on(event: string, callback: EventCallback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);
  }

  off(event: string, callback: EventCallback) {
    this.listeners.get(event)?.delete(callback);
  }

  private emit(event: string, data: unknown) {
    this.listeners.get(event)?.forEach((cb) => cb(data));
  }

  private openSocket() {
    if (!this.url) return;

    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      this.retryCount = 0;
    };

    this.ws.onmessage = (event) => {
      const payload: GatewayPayload = JSON.parse(event.data);
      this.handlePayload(payload);
    };

    this.ws.onclose = () => {
      this.stopHeartbeat();
      if (!this.intentionalClose) {
        this.attemptReconnect();
      }
    };

    this.ws.onerror = () => {
      // onclose will fire after onerror
    };
  }

  private handlePayload(payload: GatewayPayload) {
    switch (payload.op) {
      case OP_HELLO: {
        const data = payload.d as HelloData;
        this.heartbeatInterval = data.heartbeat_interval;
        this.startHeartbeat();

        if (this.sessionId) {
          this.send({
            op: OP_RESUME,
            d: {
              token: this.token,
              session_id: this.sessionId,
              seq: this.sequence,
            },
          });
        } else {
          this.send({
            op: OP_IDENTIFY,
            d: { token: this.token },
          });
        }
        break;
      }

      case OP_DISPATCH: {
        if (payload.s !== undefined) {
          this.sequence = payload.s;
        }
        if (payload.t === "READY") {
          const data = payload.d as ReadyData;
          this.sessionId = data.session_id;
        }
        if (payload.t) {
          this.emit(payload.t, payload.d);
        }
        break;
      }

      case OP_HEARTBEAT_ACK:
        this.heartbeatAcked = true;
        break;

      case OP_RECONNECT:
        this.ws?.close();
        break;

      default:
        break;
    }
  }

  private send(payload: { op: number; d: unknown }) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(payload));
    }
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    if (!this.heartbeatInterval) return;

    this.heartbeatAcked = true;
    this.heartbeatTimer = setInterval(() => {
      if (!this.heartbeatAcked) {
        this.ws?.close();
        return;
      }
      this.heartbeatAcked = false;
      this.send({ op: OP_HEARTBEAT, d: this.sequence });
    }, this.heartbeatInterval);
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private attemptReconnect() {
    if (this.retryCount >= MAX_RETRIES) {
      this.sessionId = null;
      this.sequence = 0;
      this.emit("GATEWAY_CLOSED", null);
      return;
    }

    const delay =
      Math.min(BACKOFF_BASE * 2 ** this.retryCount, BACKOFF_MAX) *
      (1 + Math.random() * JITTER_FACTOR);
    this.retryCount++;

    setTimeout(() => {
      if (!this.intentionalClose) {
        this.openSocket();
      }
    }, delay);
  }

  private cleanup() {
    this.stopHeartbeat();
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
    this.sessionId = null;
    this.sequence = 0;
  }
}

export const gateway = new GatewayClient();
