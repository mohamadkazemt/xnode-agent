# xnode-agent v0.2

A lightweight control-plane bridge between your panel and the official Xray binary.

`xnode-agent` keeps business logic in your panel, compiles declarative node state into Xray config, hot-reloads supported changes through Xray HandlerService, and falls back to a validated restart when a runtime mutation is not safe.

## Implemented

### Core / inbound management

- Multiple Xray inbounds on one server.
- Create/remove/replace inbounds from panel desired state.
- Runtime `AddInbound` / `RemoveInbound` using Xray's official `api adi/rmi` commands.
- Runtime add/remove users using Xray's official `api adu/rmu` commands for supported client-based protocols.
- VLESS, VMess, Trojan and Shadowsocks managed clients.
- WireGuard peers generated from panel-managed users.
- Generic raw `settings`, `streamSettings`, sniffing, routing, DNS and outbound passthrough.

### Safe convergence

- Desired/current diff engine.
- Full config validation with `xray run -test` before any live mutation.
- Hot reload for compatible inbound/user changes.
- Atomic config persistence after successful hot reload.
- Automatic full restart fallback if a HandlerService operation fails.
- Structural inbound changes use remove + short delay + add to avoid rapid re-add races.
- Outbound/routing/DNS changes intentionally use a validated restart in v0.2.

### Accounting

- Per-user/per-inbound traffic identity using `u:<user>|i:<inbound>` as Xray email.
- Xray statistics query with reset and panel traffic reporting.
- Runtime user changes preserve the same accounting identity.
- A new Xray policy `level` triggers a controlled restart so user traffic stats stay enabled for that level.

### Node operations

- Xray process start/stop/restart.
- Heartbeat with Xray version/running state.
- Basic memory/load health.
- Node modes: active / disabled / maintenance.
- Limit policy model and pluggable limiter backend interface.

## Xray compatibility

v0.2 uses the official Xray CLI as a thin HandlerService client:

```text
xray api adi   # AddInbound
xray api rmi   # RemoveInbound
xray api adu   # AlterInbound(AddUserOperation)
xray api rmu   # AlterInbound(RemoveUserOperation)
```

These commands are present in Xray-core stable `v26.3.27`. If an older/custom Xray binary does not support a runtime command, the agent falls back to a validated full restart instead of leaving the node partially applied.

## Panel API contract

### GET `/api/v1/nodes/{node_id}/desired-state`

Returns the shape shown in `examples/desired-state.json`.

### POST `/api/v1/nodes/{node_id}/heartbeat`

Agent sends health, Xray version/running state and desired-state version.

### POST `/api/v1/nodes/{node_id}/traffic`

Agent sends Xray stats records. Managed user records also include parsed `user_id`, `inbound_id`, and `direction`.

WireGuard inbound totals are available, but Xray does not expose the same email-based per-peer counters as client-based protocols, so strict per-peer WireGuard accounting needs a separate provider/patch.

## Accounting strategy

For client-based protocols the agent generates a unique Xray email per user/inbound:

```text
u:25|i:101
```

Xray exposes counters such as:

```text
user>>>u:25|i:101>>>traffic>>>uplink
user>>>u:25|i:101>>>traffic>>>downlink
```

This gives per-user traffic split by inbound without modifying Xray-core.

## Limits

Speed limit, IP limit, device limit and connection limit are represented in the API model, but currently use an observe-only limiter backend. Strict enforcement should be implemented as either:

1. a Linux/eBPF/tc/nftables backend where identity can be preserved reliably, or
2. a small maintained Xray-core patch at the authenticated-user dispatch layer.

Do not advertise these limits as enforced until such a backend is installed.

## Build

```bash
go test ./...
go vet ./...
go build -o bin/xnode-agent ./cmd/xnode-agent
```

## Run

Install the official Xray binary, copy `examples/agent.json` to `/etc/xnode/agent.json`, then:

```bash
./bin/xnode-agent -config /etc/xnode/agent.json
```

## Security notes

- Keep Xray API on loopback only (`127.0.0.1`).
- Use HTTPS and a unique node token for panel communication.
- Store config files with restrictive permissions because REALITY/WireGuard private keys may be present.
- Prefer token rotation and per-node credentials in the panel.

## Next milestones

- Strict limiter backend: speed/IP/connection enforcement.
- Active connection/session registry.
- Drain mode semantics.
- Config rollback history.
- Local diagnostics endpoint.
- CPU/network sampling and Prometheus metrics.
- Node groups/weights/failover orchestration in the control plane.
