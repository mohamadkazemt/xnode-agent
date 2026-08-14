# Architecture

```text
Your Panel / Control Plane
        |
        | HTTPS + per-node token
        v
   xnode-agent
   |   |   |\
   |   |   | +-- limiter backend
   |   |   +---- health / recovery
   |   +-------- stats / accounting
   +------------ desired/current reconciler
        |
        | localhost Xray API + config/process
        v
    official Xray binary
```

## Design principles

1. Keep the official Xray binary replaceable and easy to update.
2. Keep business rules in the panel, not in Xray config templates.
3. Make the node agent declarative: panel sends desired state; agent converges to it.
4. Keep Xray's gRPC API on loopback only.
5. Separate policy description from policy enforcement.
6. Prefer hot reload, but never leave runtime state ahead of an invalid/unpersisted config.

## v0.2 convergence path

```text
GET desired state
        |
        v
load last applied state
        |
        v
build complete Xray JSON
        |
        v
xray run -test -c candidate
        |
        +---------------------------+
        |                           |
        | hot-safe diff             | restart-required diff
        v                           v
HandlerService operations       atomic config write
adi/rmi/adu/rmu                    + restart
        |
        v
persist validated full config
        |
        v
query/reset stats
        |
        v
POST traffic + heartbeat
```

If any hot operation fails, the agent writes the already validated desired config and performs a full restart. This prevents partial runtime convergence from becoming permanent.

## Runtime diff rules

Hot reload is used for:

- add inbound
- remove inbound
- structural inbound replacement (remove + add)
- add/remove/update users on VLESS, VMess, Trojan and Shadowsocks

A validated restart is used for:

- outbound changes
- routing changes
- DNS changes
- introducing a new Xray user policy `level`
- any runtime API failure

WireGuard peer changes currently replace the inbound because WireGuard does not use the same Xray `UserManager` model as VLESS/VMess/Trojan.

## HandlerService adapter

The agent intentionally does not import Xray-core protobuf packages. Instead it invokes Xray's official CLI commands, which build the typed configuration and call HandlerService internally:

```text
adi -> HandlerService.AddInbound
rmi -> HandlerService.RemoveInbound
adu -> HandlerService.AlterInbound(AddUserOperation)
rmu -> HandlerService.AlterInbound(RemoveUserOperation)
```

This keeps the agent's Go module small and reduces coupling to Xray internal protobuf changes.

## Accounting

Client-based protocols use a synthetic Xray `email` containing both panel user id and inbound id. This creates native Xray user counters that can be mapped back to `(user, inbound)`.

WireGuard is different: peers do not use the same Xray `email` user-account model. Per-peer billing therefore needs a dedicated provider or a core/kernel implementation; v0.2 only guarantees aggregate inbound counters for WireGuard.

## Limits

The desired-state model already contains:

- quota
- upload/download rate
- IP limit
- device limit
- connection limit
- expiration

v0.2 does not claim strict enforcement of rate/IP/device/connection limits. The `limits.Backend` interface is the extension point.

Preferred long-term enforcement order:

1. authenticated-user layer in a small maintained Xray patch for exact per-user semantics;
2. kernel/eBPF/tc/nftables backend where identity can be preserved reliably;
3. panel-side policy for quota/expiration/device credential issuance.
