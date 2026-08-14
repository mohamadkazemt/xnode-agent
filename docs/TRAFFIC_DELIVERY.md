# Traffic delivery and billing durability

[English](TRAFFIC_DELIVERY.md) | [فارسی](fa/TRAFFIC_DELIVERY.md)

Xray user counters are read with reset enabled so each report is a delta. A network failure after resetting counters must not silently lose billing data.

## v0.3 flow

1. Query Xray stats with reset.
2. Create a `TrafficBatch` with a random `event_id`.
3. Write the batch to `traffic_spool_dir` using temp-file + rename.
4. POST the batch to the panel.
5. Delete the local spool file only after a successful HTTP response.
6. Retry queued batches on the next sync before collecting the next desired state.

Example:

```json
{
  "event_id": "d0a48cda15c14a68a6251d85a0c7af91",
  "node_id": "node-12",
  "collected_at": 1786680000,
  "records": [
    {
      "name": "user>>>u:25|i:101>>>traffic>>>downlink",
      "value": 2040,
      "user_id": "25",
      "inbound_id": "101",
      "direction": "downlink"
    }
  ]
}
```

## Panel requirement: idempotency

The panel must store/claim `event_id` in the same transaction that adds traffic deltas. If the ID has already been processed, return success without adding the bytes again.

This covers the ambiguous case where the panel commits a batch but the HTTP response is lost and the agent retries.

## Crash window

There is still a very small unavoidable window between Xray atomically returning/resetting its counters and the agent writing the batch to disk. Eliminating even that window would require an acknowledgement-aware accounting API inside the core. The v0.3 spool primarily eliminates the much larger and common network-delivery loss mode.

