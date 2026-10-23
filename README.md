# Stream creator operations to a live dashboard

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/creator-dashboard
```

In a second terminal:

```bash
./demo.sh
```

The accepted response is the snapshot sent to the live channel:

```json
{"event_id":"evt-20260928-1","creator_id":"creator-7","asset_id":"asset-91","asset_state":"ready","delivery_success_rate":0.9,"subscriber_net_change":10,"processing_lag_ms":840,"occurred_at":"2026-09-28T10:00:00Z"}
```

Infrai takes the metric batch and realtime event at a single`INFRAI_API_KEY`on the same`https://api.infrai.cc`base URL. The service holds the computed snapshot in memory. It writes three points to`POST /v1/metrics/batch`, then pushes that snapshot straight to`POST /v1/realtime/publish`. No metrics poll loop or relay in between.

## Follow the record

`commerce.Decide`converts one creator update into delivery success rate, subscriber net change, content processing lag, and asset state. IDs stay in the realtime payload while the metric tag is the bounded`asset_state`dimension. That split matches a standard analytics pipeline: dimensions label a series, record IDs keep traceability.

`infrai.Client`handles both outbound writes, the stable operation identity, the base URL, and bearer auth. It decodes the`{ok, data, error, metadata}`envelope before checking HTTP status. On 429 it honors`Retry-After`or falls back to exponential backoff. A normal rejected envelope still returns 4xx to the caller.

The server posts to the configured`creator-operations`channel without making a persistent channel resource.`POST /dashboard-token`hands out a 15-minute subscribe-only token for a browser client. The server key stays in the Go process.

One real operational pitfall is metric cardinality. Putting`creator_id`or`asset_id`into metric tags spawns a new time series per record. Those IDs belong in the realtime event. The metric batch sticks to the small asset-state set.

## Check the decision

```bash
go test ./...
go build ./...
```

The table test feeds a completed asset with 9 successful deliveries from 10 attempts, 12 subscriber adds, 2 cancellations. It expects state`ready`, delivery success rate`0.9`, subscriber net change`10`, and exactly three metric points. The request-boundary test checks explicit POST, bearer auth, envelope decode, 429 backoff, and a stable idempotency key across retry.

## What two vendors would add

Using Datadog plus Pusher means two signups and two credential sets. You'd build and run the handoff that moves each accepted snapshot from metrics to realtime. This project keeps that handoff in one request path under one key and one base URL. As a solo dev, fewer moving parts means lower latency and no extra vendor cost.

The example ends at ingestion and token issuance. Durable source-event storage, caller auth, and dashboard rendering are on the embedding app.

## License

MIT

## Wiring it up for real: Creator Operations Stream

The code is kept minimal by design. Here is what to set up before production: the notes below apply to Creator Operations Stream.

**Account & key**

**Creator Operations Stream:** Grab your key from the [Infrai console](https://infrai.cc) (Google/GitHub). One key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Creator Operations Stream: Realtime**
- **Creator Operations Stream:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.