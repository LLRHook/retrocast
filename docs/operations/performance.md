# B5c capacity, latency and monitoring ? 2026-11-24

Host: Windows 11 with Docker Desktop's Linux engine (Docker 29.9.3), Intel
Core i7-14700, 20 physical cores / 28 logical processors, 34,110,279,680 bytes
physical RAM. Docker reported 28 CPUs and 16,640,962,560 bytes RAM. The client
ran in `golang:1.26.11` on the same Docker network as the deployed stack. These
are local results on this host, not a cross-platform capacity guarantee.

API image: `l30-api`, image ID
`sha256:f8bdbb3648c0f9324f617d9bd63d4f21a67ca72a7df56696c2bf729607ad4613`,
built from fixed head `fdd8fa0f1d014d049c3138ee960cf1235fd680ed`.
The driver expanded B5c to fix gateway liveness issue #58. Harness source is commit `09b03a9`;
subsequent commits add only script tests/line-ending attributes and documentation.

From the repository root, save this override as `l30-compose.yml`. Check these
ports are free first; use another project/ports if occupied. It opens registration
only on this disposable fixture. LiveKit voice/UDP is outside this measurement.
Caddy's HTTP test site accepts the internal hostname used by the harness.

```yaml
services:
  api:
    ports: ["127.0.0.1:18308:8080"]
    environment:
      REGISTRATION_MODE: open
  livekit:
    ports: !override ["127.0.0.1:18380:7880", "127.0.0.1:18381:7881"]
  caddy:
    ports: !override ["127.0.0.1:18300:80"]
    environment:
      DOMAIN: ":80"
    volumes:
      - l30-caddy-config:/config
  grafana:
    ports: !override ["127.0.0.1:3000:3000"]
  prometheus:
    ports: ["127.0.0.1:18390:9090"]
volumes:
  l30-caddy-config:
```

```powershell
docker compose -p l30 -f retrocast/docker-compose.yml -f l30-compose.yml up -d --build --wait
docker run --rm --name l30-perf-acceptance --network l30_default -v "${PWD}/retrocast:/app" -w /app golang:1.26.11 go run ./cmd/perf -base http://caddy -duration 10m
# After inspecting the results, remove ONLY this project's fixture:
docker compose -p l30 -f retrocast/docker-compose.yml -f l30-compose.yml down -v
```

The recorded run also mounted the owned `l30-mod-cache` and `l30-build-cache`
volumes at `/go/pkg/mod` and `/root/.cache/go-build`; those affect compilation,
not the measured traffic. The run uses a new guild/channel. Provisioning retries
credential 429s using Retry-After and refreshes credentials before measurement;
production limits are unchanged. Seed writes are distributed over 20 users.
After seeding, all 20 gateway clients have received READY. History measures 100
random cursors (seed 1), each returning exactly 50 rows, against 1,000 messages.
Measured requests do not retry. The ten-minute phase attempts 60 messages per
member at ten-second intervals. Each message requires one receipt at every
other member (19 peers); the sender echo is excluded. Send-to-receive timing
includes HTTP submission through Caddy and the gateway event. A five-second
final drain includes late deliveries in the percentile. The separate single
sender phase sends 100 messages with 1.3s spacing and the same 20 live clients.

## Monitoring

On the owned fresh stack Prometheus reported `health: up`, no last error,
scrape URL `http://api:8080/metrics`, and interval `15s`. Caddy still refuses
public `/metrics`. Grafana loaded at `http://127.0.0.1:3000`, signed in with the
compose development admin account, and opened the provisioned Retrocast API
dashboard with its provisioned datasource UID `PBFA97CFB590B2093`.

The original handler panel grouped by a nonexistent `handler` label and yielded
one aggregate series. The exported label is `url`; the fixed panel's query
returned eight distinct handler series at capture time, including messages,
registration, health and metrics. Status code rates and p50/p95/p99 queries
returned live samples. The error panel returned the expected setup 429 rate;
5xx had no samples. No datasource or dashboard setup was performed manually.

The shared browser exposed the dashboard DOM and successful datasource queries,
but screenshot capture twice failed with "The browser is not producing
screenshot frames". No screenshot or pixel-level chart rendering claim is made.
The private handoff retains the browser/query result and Prometheus target JSON.
Recheck by opening the same dashboard and the Prometheus target on a new stack.

## Reproduce the deployed path

Use an unused owned compose project and check that ports 18300, 18308, 18380,
18381, 18390 and 3000 are free. The recorded run used project `l30`; do not stop
an existing project or reuse someone else's resources. Save this override as
`l30-compose.yml` at the repository root. The original record used identical
contents in the private L30 handoff directory. Voice/media is outside this run;
the override removes the default UDP publication to avoid host port conflicts.

```yaml
services:
  api:
    ports: ["127.0.0.1:18308:8080"]
    environment:
      REGISTRATION_MODE: open
  livekit:
    ports: !override ["127.0.0.1:18380:7880", "127.0.0.1:18381:7881"]
  caddy:
    ports: !override ["127.0.0.1:18300:80"]
    environment:
      DOMAIN: ":80"
    volumes:
      - l30-caddy-config:/config
  grafana:
    ports: !override ["127.0.0.1:3000:3000"]
  prometheus:
    ports: ["127.0.0.1:18390:9090"]
volumes:
  l30-caddy-config:
```

Recorded command form (PowerShell, repository root; the actual run redirected
stdout/stderr to the private `perf-fixed.log`):

```powershell
$root = (Get-Location).Path
docker compose -p l30 -f retrocast/docker-compose.yml -f l30-compose.yml up -d --build --wait
docker run --rm --name l30-perf-fixed --network l30_default `
  -v "${root}/retrocast:/app" -v l30-mod-cache:/go/pkg/mod `
  -v l30-build-cache:/root/.cache/go-build -w /app golang:1.26.11 `
  go run ./cmd/perf -base http://caddy -duration 10m
```

Copy the committed `.env.example` to `retrocast/.env` for this disposable fixture.
The command creates 20 accounts, one guild, one channel seeded with 1000 messages,
100 history calls and 1200 measured sends. History cursors are sampled uniformly
with replacement using seed 1; each page must contain 50 rows. It waits for all
20 READY events, measures every expected peer receipt, and then sends 100 messages
from one sender for the separate latency acceptance. Application limits stay at
5 credential requests/minute/IP and 50 protected requests/minute/user. Setup
honors Retry-After and refreshes credentials; measured message/history calls do
not retry. The preflight rejects insufficient remaining access-token lifetime.
The one-sender phase refreshes its token after a cooldown and spaces sends 1.3s
apart. Exit 1 means a missed target, loss, duplicate, disconnect or request error.
Only clean up resources that this run created. See OPERATIONS.md for schedules
and restore commands.

## Unfixed run (diagnostic evidence)

The completed unfixed development run used baseline 140c236:

```json
{
  "channel_id": "250922466740932608",
  "clients": 20,
  "disconnects": 20,
  "duplicates": 0,
  "duration_seconds": 600,
  "expected_deliveries": 22800,
  "gateway_p95_ms": 0,
  "history_calls": 100,
  "history_p95_ms": 1.497,
  "lost_deliveries": 22800,
  "received_deliveries": 0,
  "send_attempts": 1200,
  "send_errors": 10,
  "single_sender_attempts": 100,
  "single_sender_expected_deliveries": 1900,
  "single_sender_p95_ms": 0,
  "single_sender_received_deliveries": 0,
  "unexpected": 0
}
```

The 0ms fields have zero samples and are unavailable, not passing latency
measurements. This development run had an idle cooldown after READY, so
sessions had already closed before measured sends. It also suffered 10 expired
credential requests. A separate real-client probe answered an application
heartbeat and still closed at 60.007s, confirming the gateway defect independently
of these harness problems. The harness now begins immediately after READY/history,
checks that credentials cover the complete load plus one minute, and reports
null for empty latency distributions.

## Gateway fix and fast regression

Issue [#58](https://github.com/LLRHook/retrocast/issues/58) is fixed in fdd8fa0:
every successfully read client frame extends the read deadline. The Pong handler
remains. Production timing remains 41.25s heartbeat, 10s heartbeat grace and a
read wait of at least 60s. Immutable per-connection timing allows short tests
without package-global timing races. Three real-WebSocket tests finish in less
than one second of execution: responsive heartbeats survive three read deadlines,
silent clients close, and non-heartbeat frames do not bypass heartbeat timeout.
Removing deadline renewal reproduces the baseline failure in 0.18s with only
the timing test seam applied; the two negative boundary cases still pass.

## Fixed acceptance run

Command exit: **0**. The full fixed profile ran for 600 seconds with 20 clients;
all 22,800 expected peer deliveries arrived, with no loss, send errors,
disconnects, duplicates or unexpected messages. Load send-to-receive p95 was
**8.706 ms**. The separate single-sender acceptance sent 100 messages and received
all 1,900 peer deliveries, with p95 **3.882 ms**. The 100 random-cursor history
calls on the 1000-message channel returned full 50-row pages with p95 **1.571 ms**.
SYS-NFR-001, SYS-NFR-002 and SYS-NFR-007 passed on this recorded host.

```json
{
  "channel_id": "250931340013342720",
  "clients": 20,
  "disconnects": 0,
  "duplicates": 0,
  "duration_seconds": 600,
  "expected_deliveries": 22800,
  "gateway_p95_ms": 8.706,
  "history_calls": 100,
  "history_p95_ms": 1.571,
  "lost_deliveries": 0,
  "received_deliveries": 22800,
  "send_attempts": 1200,
  "send_errors": 0,
  "single_sender_attempts": 100,
  "single_sender_expected_deliveries": 1900,
  "single_sender_p95_ms": 3.882,
  "single_sender_received_deliveries": 1900,
  "unexpected": 0
}
```

Two initial development runs and the corrected-but-unfixed run were stopped
before completion; they provide no performance evidence. The completed unfixed
diagnostic run is preserved above. Only the fixed acceptance run determines the
final target dispositions. A short six-case mocked operator-script regression also ran once during the
timed phase; the numbers include that host contention. L30 ran no Go builds or
compilation during the timed phase.

Unchanged limitation: permission lookups under the manager lock (#51) are not
tuned by this harness. Stored file recovery and host schedule limits are in
[restore-drills.md](restore-drills.md) and [OPERATIONS.md](../OPERATIONS.md).
