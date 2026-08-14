# Architecture — v1.0

```text
                         YOUR PANEL / CONTROL PLANE
                         users, nodes, assignment,
                         groups, global failover/LB
                                   |
                             HTTPS + token
                                   v
+------------------------------------------------------------------+
|                         xnode-agent                               |
|                                                                  |
| desired-state reconciler  policy evaluator  traffic spool        |
| runtime HandlerService    session/online     health/recovery      |
| node drain/threshold      core policy file   local health API     |
+-------------------------------+----------------------------------+
                                |
                 localhost API  |  atomic config/policy
                                v
+------------------------------------------------------------------+
| Xray v26.7.28 + narrow xnode dispatcher overlay                  |
| protocols/transports | stats/online map | routing | strict limits|
+------------------------------------------------------------------+
                                |
                             Internet
```

## Separation of responsibilities

The panel owns global truth: users, plans, credentials, cumulative billing, node assignment, groups, and cross-node scheduling. The agent owns convergence of one server. Xray owns the packet/data path.

This keeps the agent horizontally replaceable and avoids giving one node a global database role.

## Convergence loop

```text
retry durable traffic events
       -> query/reset Xray traffic counters
       -> durably spool + submit new traffic event
       -> query native online IPs (access-log fallback)
       -> GET desired state
       -> node threshold/drain policy
       -> quota/expiration/IP/device evaluation
       -> atomically write strict core policy
       -> build complete Xray config
       -> xray run -test candidate
       -> hot HandlerService diff OR validated restart
       -> Xray API health check / recovery / rollback
       -> POST sessions + heartbeat
       -> persist effective applied state
```

The persisted state is the effective state actually applied, including temporary policy blocks. When the panel state makes a user eligible again, the next reconciliation re-adds it.

## Runtime mutations

Hot operations are used for safe inbound and VLESS/VMess/Trojan/Shadowsocks user changes. WireGuard peer changes replace that inbound. Global outbound/routing/DNS changes and new Xray policy levels use a validated restart.

Any hot-operation failure falls back to the complete validated desired config. Process startup failure attempts the saved last-good config.

## Strict policy data path

The agent writes an atomic JSON map keyed by the same synthetic Xray email used for stats. The patched dispatcher:

1. sees the authenticated `MemoryUser`;
2. counts every logical dispatcher connection;
3. checks block/connection admission;
4. wraps authenticated up/down data paths;
5. reloads policy without restart;
6. applies one shared rate bucket per user/direction;
7. terminates sessions when blocked, over limit, deleted (tombstone), or when `session_generation` changes.

This is intentionally a narrow overlay rather than a fork of unrelated Xray subsystems.

## Node failover and load balancing

A single node cannot safely decide global failover because it has no authoritative view of the other nodes or client endpoint distribution. The heartbeat therefore reports the fields the panel scheduler needs: health, region, group, tags, weight, mode, drain state, network rate and traffic threshold state.

Within one Xray process, native routing/balancer JSON remains available through desired-state passthrough.
