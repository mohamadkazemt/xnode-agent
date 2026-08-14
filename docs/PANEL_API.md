# Panel API contract

The agent authenticates with `Authorization: Bearer <node token>`.

## Desired state

`GET /api/v1/nodes/{node_id}/desired-state`

The response is declarative. The panel is the source of truth.

```json
{
  "version": "42",
  "enabled": true,
  "mode": "active",
  "inbounds": [],
  "outbounds": [],
  "routing": {},
  "dns": {}
}
```

Supported node modes in v0.1:

- `active`: apply config and run Xray.
- `maintenance`: stop Xray.
- `disabled`: stop Xray.

`draining` should be implemented at the control plane first: stop assigning new users/connections to the node, then transition to maintenance after the active-session target is reached.

## Inbound model

```json
{
  "id": "101",
  "tag": "vless-reality-443",
  "listen": "0.0.0.0",
  "port": 443,
  "protocol": "vless",
  "settings": {"decryption": "none"},
  "stream_settings": {},
  "sniffing": {},
  "users": []
}
```

The agent passes protocol-specific `settings` and `stream_settings` to Xray. This avoids coupling the agent release cycle to every Xray transport field.

## User model

```json
{
  "id": "25",
  "enabled": true,
  "credential": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "flow": "xtls-rprx-vision"
  },
  "limits": {
    "traffic_bytes": 536870912000,
    "upload_bps": 20000000,
    "download_bps": 100000000,
    "ip_limit": 2,
    "device_limit": 3,
    "connection_limit": 20,
    "expires_at": 0
  }
}
```

For VLESS/VMess/Trojan/Shadowsocks, the agent inserts users into `settings.clients` and generates a deterministic accounting email:

`u:<user_id>|i:<inbound_id>`

For WireGuard, each enabled user credential is treated as a peer object and inserted into `settings.peers`.

## Heartbeat

`POST /api/v1/nodes/{node_id}/heartbeat`

```json
{
  "node_id": "node-12",
  "agent_version": "0.1.0",
  "xray_version": "Xray ...",
  "healthy": true,
  "xray_running": true,
  "memory_bytes": 123456789,
  "load1": 0.21,
  "state_version": "42",
  "message": "ok"
}
```

## Traffic report

`POST /api/v1/nodes/{node_id}/traffic`

```json
{
  "node_id": "node-12",
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

The agent reads counters with reset enabled. The panel should treat reports as deltas and persist/aggregate them transactionally.

## Idempotency

- Desired state is versioned.
- Repeating the same desired state must be safe.
- Heartbeats and traffic posts should accept retries.
- Traffic ingestion should use a request/event id in a later protocol revision to prevent duplicate billing after network retries.
