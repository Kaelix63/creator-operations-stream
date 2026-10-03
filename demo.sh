#!/bin/sh
set -eu
curl --request POST http://localhost:8080/updates \
  --header 'Content-Type: application/json' \
  --data '{"event_id":"evt-20260928-1","creator_id":"creator-7","asset_id":"asset-91","delivery_attempts":10,"successful_deliveries":9,"subscriber_adds":12,"subscriber_cancels":2,"processing_lag_ms":840,"processing_complete":true,"occurred_at":"2026-09-28T10:00:00Z"}'
