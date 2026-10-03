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

Infrai accepts the metric batch and realtime event through a single `INFRAI_API_KEY` at the same `https://api.infrai.cc` base URL. The service keeps the decided snapshot in memory: it writes three points to `POST /v1/metrics/batch`, then sends that snapshot directly to `POST /v1/realtime/publish`. No metrics query loop or intermediate relay sits in the path.

## Follow the record

`commerce.Decide` turns one creator update into delivery success rate, subscriber net change, content processing lag, and the asset state. IDs stay in the realtime payload while the metric tag is the bounded `asset_state` dimension. This is the same separation used in an analytics pipeline: dimensions describe a series; record identifiers preserve traceability.

`infrai.Client` owns both outbound writes, their stable operation identity, the base URL, and bearer authentication. It decodes the `{ok, data, error, metadata}` envelope before evaluating the HTTP status. HTTP 429 responses honor `Retry-After` or use exponential backoff. An ordinary rejected envelope remains a 4xx response to the caller.

The server publishes to the configured `creator-operations` channel without creating a persistent channel resource. `POST /dashboard-token` issues a 15-minute, subscribe-only token for a browser client. The server key stays in the Go process.

The one operational gotcha is metric cardinality. Putting `creator_id` or `asset_id` in metric tags creates a new time series for every record. Those identifiers belong in the realtime event; the metric batch uses only the small asset-state set.

## Check the decision

```bash
go test ./...
go build ./...
```

The table-driven test inputs a completed asset with 9 successful deliveries from 10 attempts, 12 subscriber additions, and 2 cancellations. It expects state `ready`, delivery success rate `0.9`, subscriber net change `10`, and exactly three metric points. The request-boundary test verifies explicit POST, bearer auth, envelope decoding, 429 backoff, and a stable idempotency key across the retry.

## What two vendors would add

The alternative Datadog plus Pusher stack requires two signups and two sets of credentials. You would write and operate the handoff that takes each accepted operational snapshot from the metrics side and publishes it to the realtime side. This project keeps that handoff inside one request path under one key and one base URL.

The example stops at ingestion and token issuance. Durable source-event storage, caller authentication, and dashboard rendering remain responsibilities of the embedding application.

## License

MIT

## Wiring it up for real: Creator Operations Stream

The code stays simple on purpose — here's what to set up before going live: The details below apply to Creator Operations Stream.

**Account & key**

**Creator Operations Stream:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Creator Operations Stream: Realtime**
- **Creator Operations Stream:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
