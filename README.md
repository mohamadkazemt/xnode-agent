# xnode-agent v0.1

A lightweight control-plane bridge between your panel and the official Xray binary.

## Implemented in v0.1

- Pull desired state from your panel.
- Multiple Xray inbounds on one server.
- Protocol-agnostic raw settings/streamSettings passthrough.
- Managed clients for VLESS, VMess, Trojan and Shadowsocks.
- WireGuard peers from panel-managed users.
- Per-user/per-inbound traffic identity using `u:<user>|i:<inbound>` as Xray email.
- Xray statistics query with reset and panel traffic reporting.
- Atomic config generation, `xray run -test`, then apply/restart.
- Xray local API bound to `127.0.0.1` by configuration.
- Heartbeat, basic memory/load health, auto restart after a stopped process is detected.
- Node modes: active / disabled / maintenance.
- Limit policy model and pluggable limiter backend interface.

## Not yet strict-enforced

Speed limit, IP limit, device limit and connection limit are represented in the API model, but v0.1 uses an observe-only limiter. Strict enforcement should be implemented as either:

1. a Linux/eBPF/tc/nftables backend where technically reliable, or
2. a small maintained Xray-core patch at the authenticated-user dispatch layer.

Do not advertise these limits as enforced until such a backend is installed.

## Panel API contract

### GET `/api/v1/nodes/{node_id}/desired-state`
Returns the shape shown in `examples/desired-state.json`.

### POST `/api/v1/nodes/{node_id}/heartbeat`
Agent sends health, Xray version/running state and desired-state version.

### POST `/api/v1/nodes/{node_id}/traffic`
Agent sends Xray stats records. Managed user records also include parsed `user_id`, `inbound_id`, and `direction`. WireGuard inbound totals are available, but Xray does not expose the same email-based per-peer counters as client-based protocols, so strict per-peer WireGuard accounting needs a separate provider/patch.

## Accounting strategy

For client-based protocols the agent generates a unique Xray email per user/inbound:

`u:25|i:101`

Xray then exposes counters such as:

`user>>>u:25|i:101>>>traffic>>>uplink`

This gives per-user traffic split by inbound without modifying Xray-core.

## Build

```bash
go test ./...
go build -o xnode-agent ./cmd/xnode-agent
```

## Run

Install the official Xray binary, copy `examples/agent.json` to `/etc/xnode/agent.json`, then:

```bash
./xnode-agent -config /etc/xnode/agent.json
```

## Security notes

- Keep Xray API on loopback only (`127.0.0.1`).
- Use HTTPS and a unique node token for panel communication.
- Store config files with restrictive permissions because REALITY/WireGuard private keys may be present.
- Prefer token rotation and per-node credentials in the panel.

## Next milestones

- Runtime AddUser/RemoveUser/AddInbound/RemoveInbound through Xray HandlerService (avoid full restart for supported protocols).
- Strict limiter backend.
- Active connection/session registry.
- Drain mode semantics.
- Node groups/weights/failover orchestration in the control plane.
- CPU/network sampling and Prometheus metrics.
