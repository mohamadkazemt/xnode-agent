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

## v0.2

- Desired/current diff engine
- Xray HandlerService runtime add/remove inbound
- Runtime add/remove user on supported protocols
- Connection/session registry
- Proper draining state
- Config rollback history
- Local diagnostics HTTP endpoint

## v0.3

- Strict speed limiter
- IP limit
- connection limit
- quota cutoff
- expiration cutoff
- device credential lifecycle
- active session disconnect

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
