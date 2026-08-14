# Architecture

```text
Your Panel / Control Plane
        |
        | HTTPS + per-node token
        v
   xnode-agent
   |   |   |
   |   |   +-- health / recovery
   |   +------ stats / accounting
   +---------- desired-state compiler
        |
        | localhost API + config/process
        v
    official Xray binary
```

## Design principles

1. Keep the official Xray binary replaceable and easy to update.
2. Keep business rules in the panel, not in Xray config templates.
3. Make the node agent declarative: panel sends desired state; agent converges to it.
4. Keep Xray's gRPC API on loopback only.
5. Separate policy description from policy enforcement.

## v0.1 apply path

```text
GET desired state
   -> build Xray JSON
   -> write candidate
   -> xray run -test -c candidate
   -> atomic rename
   -> restart Xray if config changed
   -> query/reset stats
   -> POST traffic
   -> POST heartbeat
```

## Runtime mutation roadmap

A later version should diff current/desired state and use Xray HandlerService for supported runtime changes:

- add/remove inbound
- add/remove outbound
- add/remove user for VMess/VLESS/Trojan/Shadowsocks

For changes unsupported by HandlerService (for example some WireGuard peer changes), fall back to validated config replacement/restart.

## Accounting

Client-based protocols use a synthetic Xray `email` containing both panel user id and inbound id. This creates native Xray user counters that can be mapped back to `(user, inbound)`.

WireGuard is different: peers do not use the same Xray `email` user-account model. Per-peer billing therefore needs a dedicated provider or a core/kernel implementation; v0.1 only guarantees aggregate inbound counters for WireGuard.

## Limits

The desired-state model already contains:

- quota
- upload/download rate
- IP limit
- device limit
- connection limit
- expiration

v0.1 does not claim strict enforcement of rate/IP/device/connection limits. The `limits.Backend` interface is the extension point.

Preferred long-term enforcement order:

1. authenticated-user layer in a small maintained Xray patch for exact per-user semantics;
2. kernel/eBPF/tc/nftables backend where identity can be preserved reliably;
3. panel-side policy for quota/expiration/device credential issuance.
