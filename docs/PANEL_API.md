# Panel API contract — v1.0

Every agent request uses:

```http
Authorization: Bearer <unique node token>
```

The panel is the source of truth for desired configuration and cumulative committed traffic.

## GET desired state

`GET /api/v1/nodes/{node_id}/desired-state`

Top-level example:

```json
{
  "version":"100",
  "enabled":true,
  "mode":"active",
  "node":{
    "region":"DE",
    "group":"premium",
    "tags":["reality","10g"],
    "weight":100,
    "traffic_threshold_bytes":21990232555520,
    "traffic_used_bytes":12000000000000,
    "drain_target_online":0
  },
  "inbounds":[],
  "outbounds":[],
  "routing":{},
  "dns":{}
}
```

Modes:

- `active`: normal convergence.
- `draining`: keep current membership/configuration, reject new authenticated dispatcher sessions in the patched core, allow already-admitted sessions to finish, and report `drain_ready` when online users are at/below `drain_target_online`.
- `maintenance`: collect/report state and stop Xray.
- `disabled`: stop Xray and remain disabled.

If node `traffic_used_bytes >= traffic_threshold_bytes`, the agent locally enters draining even if the requested mode was active.

### Inbound

```json
{
  "id":"101",
  "tag":"vless-reality-443",
  "listen":"0.0.0.0",
  "port":443,
  "protocol":"vless",
  "ip_limit_mode":"source",
  "settings":{"decryption":"none"},
  "stream_settings":{},
  "sniffing":{},
  "users":[]
}
```

Protocol-specific Xray JSON is passed through. Managed user injection is automatic for VLESS, VMess, Trojan and Shadowsocks; `wireguard` users are mapped to peers. Other Xray protocols can be created from the panel by providing their native settings.

### User / device credential

```json
{
  "id":"device-25-a",
  "account_id":"25",
  "enabled":true,
  "session_generation":3,
  "outbound_tag":"direct",
  "level":0,
  "credential":{"id":"550e8400-e29b-41d4-a716-446655440000","flow":"xtls-rprx-vision"},
  "limits":{
    "traffic_bytes":536870912000,
    "traffic_used_bytes":1234567890,
    "upload_bps":2500000,
    "download_bps":12500000,
    "ip_limit":2,
    "device_limit":3,
    "connection_limit":20,
    "expires_at":0
  }
}
```

`outbound_tag` creates an automatic routing rule for this synthetic user identity. Increment `session_generation` to disconnect existing sessions without disabling the credential.

## POST heartbeat

`POST /api/v1/nodes/{node_id}/heartbeat`

Important v1 fields:

```json
{
  "node_id":"node-12",
  "agent_version":"1.0.0",
  "healthy":true,
  "xray_running":true,
  "xray_api_healthy":true,
  "cpu_percent":13.2,
  "memory_bytes":734003200,
  "load1":0.21,
  "network_rx_bytes":123,
  "network_tx_bytes":456,
  "network_rx_bps":800000000,
  "network_tx_bps":120000000,
  "online_users":14,
  "tracked_ips":19,
  "mode":"draining",
  "drain_ready":false,
  "region":"DE",
  "group":"premium",
  "tags":["reality","10g"],
  "weight":100,
  "strict_limits_ready":true,
  "state_version":"100",
  "message":"ok"
}
```

`network_*_bps` in heartbeat is **bits/sec**. `strict_limits_ready` confirms the maintained patched core marker is present whenever current policies require strict data-path enforcement.

Your global scheduler can use `healthy`, `mode`, `drain_ready`, `weight`, region/group/tags and its own capacity policy for cross-node failover/load balancing.

## POST traffic

`POST /api/v1/nodes/{node_id}/traffic`

```json
{
  "event_id":"d0a48cda15c14a68a6251d85a0c7af91",
  "node_id":"node-12",
  "collected_at":1786680000,
  "records":[
    {"name":"user>>>u:device-25-a|i:101>>>traffic>>>downlink","value":2040,"user_id":"device-25-a","inbound_id":"101","direction":"downlink"}
  ]
}
```

The panel **must transactionally deduplicate `event_id`** and then add each value as a delta. The same event can be retried after an ambiguous network failure.

## POST sessions

When `report_sessions:true`:

`POST /api/v1/nodes/{node_id}/sessions`

```json
{
  "node_id":"node-12",
  "generated_at":1786680000,
  "window_sec":120,
  "records":[
    {"user_id":"device-25-a","inbound_id":"101","ips":["203.0.113.10"],"last_seen":1786679996,"recent_connections":6,"source":"xray-online"}
  ],
  "violations":[
    {"user_id":"device-25-a","inbound_id":"101","reason":"ip_limit","observed":3,"limit":2}
  ]
}
```

Policy reasons are `expired`, `traffic_quota`, `ip_limit`, and `device_limit`.

The native `xray-online` IP list represents currently referenced IPs. `recent_connections` is supplemental access-log activity and is not the exact active dispatcher connection count; exact connection enforcement remains inside the patched core.

## Ordering / retries

- Desired state must be safe to read repeatedly.
- Use a monotonically changing `version` for observability.
- Heartbeats/session reports should tolerate retries.
- Traffic processing is keyed by `event_id`, not arrival time.
