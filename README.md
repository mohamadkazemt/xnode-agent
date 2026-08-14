# xnode-agent v1.0.0

`xnode-agent` is a node-side control plane for Xray. Your panel stays the source of truth; each server runs one small Go agent that generates/validates Xray configuration, applies safe runtime changes, collects billing/online state, enforces user policy, monitors the node, and recovers Xray when needed.

## What v1.0 implements

### Inbounds and protocols

- Multiple independent inbounds on one server.
- Inbounds created entirely from panel desired-state; no manual Xray JSON editing on the node.
- Native Xray `settings`, `streamSettings`, sniffing, routing, DNS and outbound passthrough so new Xray transports/protocols do not require an agent rewrite.
- Managed VLESS, VMess, Trojan and Shadowsocks users.
- Managed WireGuard inbound peers with deterministic per-peer Xray identity.
- WireGuard traffic/online/limit accounting uses the same `user + inbound` identity as other managed protocols.

### Runtime control

- Hot add/remove/replace inbound through Xray HandlerService CLI adapters.
- Hot add/remove/update user for VLESS/VMess/Trojan/Shadowsocks.
- WireGuard peer changes use safe inbound replacement because the current Xray `adu` CLI does not extract WireGuard users.
- Candidate validation with `xray run -test` before activation.
- Atomic config persistence, last-good backup, rollback and auto-recovery.

### Per-user enforcement

The bundled, narrow dispatcher overlay for the pinned Xray build enforces at the authenticated data path:

- upload speed limit;
- download speed limit;
- active connection limit;
- active source-IP limit;
- suspend/resume and deleted-user tombstones;
- immediate session invalidation via `session_generation`.

The agent additionally enforces:

- traffic quota using panel-committed usage plus locally queued unsent traffic;
- expiration;
- device credential count using `account_id`;
- per-user outbound routing;
- online user/IP reporting and session activity.

### Billing and observability

- Per-user/per-inbound accounting identity: `u:<user_id>|i:<inbound_id>`.
- Durable traffic spool with idempotent `event_id`, so resetting Xray counters does not lose billing data during panel/network failures.
- Native Xray online-IP API when available, with access-log fallback.
- CPU, RAM, load and network counters/rates in heartbeat.
- Local loopback-only `/healthz`, `/readyz` and `/status` endpoints.

### Node management primitives

- `active`, `draining`, `maintenance`, `disabled` modes.
- drain readiness based on online users.
- traffic-threshold-triggered drain.
- region, group, tags and node weight in heartbeat.
- Xray API health check, auto restart, recovery and last-good rollback.

Cross-node failover and load balancing belong to your panel/control plane because one node cannot safely choose among servers it cannot authoritatively observe. The agent reports the health, mode, drain, region/group/tags, weight and network fields your scheduler needs. Native Xray balancer/routing JSON is still available for balancing inside one Xray process.

## Build and test

```bash
go test ./...
go vet ./...
go build -o xnode-agent ./cmd/xnode-agent
```

The `xray-patch` GitHub Actions workflow also checks out the pinned official Xray source, applies the maintained dispatcher overlay, compiles Xray, starts a real Xray process, verifies its localhost API, and exercises runtime user add/remove end-to-end.

## Production install

Copy `examples/agent.json`, set a unique panel token, then run as root:

```bash
sudo ./scripts/install.sh /path/to/agent.json
```

The installer builds the pinned patched Xray core and installs the agent, Xray, systemd unit and logrotate policy. See `docs/DEPLOYMENT.md`.

## Panel integration

Your panel needs three write endpoints plus desired state:

```text
GET  /api/v1/nodes/{node_id}/desired-state
POST /api/v1/nodes/{node_id}/heartbeat
POST /api/v1/nodes/{node_id}/traffic
POST /api/v1/nodes/{node_id}/sessions
```

The exact JSON contract and retry/idempotency rules are in `docs/PANEL_API.md` and `docs/TRAFFIC_DELIVERY.md`.

## Security

- Xray API and agent status listener are required to bind to loopback.
- Non-loopback panel URLs must use HTTPS.
- Use a unique token per node and store `/etc/xnode/agent.json` mode `0600`.
- Generated state/policy/spool files use restrictive permissions.
- The systemd unit uses `NoNewPrivileges`, a restricted capability set and `UMask=0077`.
- Access logs contain client IPs and synthetic identities; the included logrotate policy limits retention.

## Xray pin

v1.0 maintains its dispatcher overlay against **Xray v26.7.28**. Upgrade the pin only with a new patch directory and a green `xray-patch` workflow; do not silently point the existing overlay at another Xray revision.
