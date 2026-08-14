# Roadmap

## v0.1 — delivered

- Declarative panel -> node sync
- Multi-inbound config
- Generic Xray settings passthrough
- VLESS/VMess/Trojan/Shadowsocks managed users
- WireGuard peer config
- Per-user/per-inbound native accounting key for client protocols
- Stats upload
- Heartbeat
- Config validation and atomic apply
- Basic recovery
- Limit policy model + backend interface

## v0.2 — delivered

- Desired/current diff engine
- HandlerService-backed runtime add/remove inbound via `api adi/rmi`
- Runtime add/remove user via `api adu/rmu`
- Runtime credential replacement as remove + add
- Unsupported/unsafe mutations fall back to validated restart
- Persist desired config after successful hot reload
- New user policy levels force a restart to preserve Xray stats accounting
- Unit tests for runtime planning and fallback decisions

## v0.3

- Strict speed limiter
- IP limit
- connection limit
- quota cutoff
- expiration cutoff
- device credential lifecycle
- active session registry and disconnect
- proper draining state
- local diagnostics HTTP endpoint
- config rollback history

## v0.4

- Node metrics: CPU/RAM/disk/network
- Prometheus/OpenTelemetry
- node weights
- traffic thresholds
- control-plane failover policies
- node groups/tags/regions

## v0.5

- Per-peer WireGuard accounting backend
- advanced routing controls
- certificate lifecycle integration
- signed desired-state payloads / token rotation
