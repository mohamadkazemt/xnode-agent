# xnode-agent v1.0.0

`xnode-agent` is a node-side control bridge for a panel you already own. The panel remains the control plane; each server runs `xnode-agent` plus a pinned Xray core. The agent receives desired state, validates and reconciles Xray, reports traffic/health/sessions, and enforces node/user policy.

## What v1.0 implements

### Xray / inbound control

- Multiple independent inbounds on one server.
- Create/update/remove inbounds from panel desired state; no SSH editing is required.
- Generic Xray `settings`, `streamSettings`, `sniffing`, `routing`, `dns`, and outbound passthrough, so current Xray protocols/transports are not hard-coded into the agent.
- Managed runtime users for VLESS, VMess, Trojan and Shadowsocks.
- Managed WireGuard peers; peer-list changes replace that inbound safely.
- Runtime add/remove/replace through Xray HandlerService CLI where safe; full validated restart otherwise.
- Atomic config updates, last-good rollback, process restart, API health-check and recovery.

### Accounting / users

- Deterministic per-user + per-inbound identity: `u:<user_id>|i:<inbound_id>`.
- Per-inbound traffic deltas and durable idempotent delivery to the panel.
- Traffic quota and expiration cutoff.
- Exact online IP list when supported by the pinned Xray core, with access-log fallback.
- IP limit, device credential limit, suspend/resume and session reporting.
- `session_generation`: increment it in the panel to disconnect the user's existing dispatcher sessions while allowing new sessions immediately.
- Per-user routing through `outbound_tag`.
- Per-user protocol selection by placing that credential only in the allowed inbounds.

### Strict data-path limits

The repository includes a narrow maintained overlay for **Xray v26.7.28**. It adds:

- aggregate upload rate per authenticated user;
- aggregate download rate per authenticated user;
- exact concurrent dispatcher connection admission;
- live policy reload without Xray restart;
- active-session cutoff for suspend/quota/expiration/IP/device policy;
- session-generation disconnect;
- removed-user tombstones so an already-established session cannot survive credential deletion indefinitely.

The agent writes `/var/lib/xnode/limits.json` atomically. The patched dispatcher reads that policy using the same synthetic authenticated email used for accounting. Set `require_patched_core: true` in production when you depend on strict limits.

`upload_bps` and `download_bps` are **bytes per second**. Example: `12500000` is approximately 100 Mbit/s.

### Node operations

- modes: `active`, `draining`, `maintenance`, `disabled`;
- drain prevents new inbound/user membership and exposes `drain_ready` when online users reach the configured target;
- traffic threshold can automatically move a node into drain mode;
- region, group, tags and weight telemetry;
- CPU, used RAM, load, network totals and current RX/TX bit-rate;
- auto health-check, process restart, API recovery and last-good rollback;
- local `/healthz`, `/readyz`, `/status` endpoints on loopback.

Cross-server failover/load balancing is intentionally a **panel/control-plane decision**: the agent reports health, mode, weight, group, region and traffic metadata needed for that decision. Xray's own per-node routing/balancer config can still be passed through in desired state.

## Quick start

Requirements for building the bundled strict core are `git`, `python3`, Go **1.26+** for Xray, and systemd on the target Linux server. The agent itself targets Go 1.23+.

```bash
cp examples/agent.json /tmp/agent.json
# edit node_id, panel_url, panel_token and interface
sudo ./scripts/install.sh /tmp/agent.json
```

The installer builds/tests the agent, builds the pinned Xray core with the maintained dispatcher overlay, installs both binaries, installs systemd/logrotate config and starts the service.

Useful checks:

```bash
systemctl status xnode-agent
curl http://127.0.0.1:19090/healthz
curl http://127.0.0.1:19090/readyz
curl http://127.0.0.1:19090/status
```

## Panel contract

The panel implements:

```text
GET  /api/v1/nodes/{node_id}/desired-state
POST /api/v1/nodes/{node_id}/heartbeat
POST /api/v1/nodes/{node_id}/traffic
POST /api/v1/nodes/{node_id}/sessions
```

Traffic posts carry an `event_id`; the panel must deduplicate it transactionally before adding deltas.

See `docs/PANEL_API.md`, `docs/LIMITS.md`, `docs/ARCHITECTURE.md`, `docs/TRAFFIC_DELIVERY.md`, `docs/RUNTIME_API.md`, and `docs/DEPLOYMENT.md`.

## Security boundary

- Xray API and the agent admin HTTP listener are rejected unless configured on loopback.
- Panel URL must be HTTPS except for loopback development.
- State, policy, traffic spool, configs and logs are created with restrictive permissions.
- A unique node token should be issued per server and rotated by the panel.
- Source-IP policy is meaningful only when Xray sees the real client source. Disable `ip_limit_mode` for an inbound if a CDN/proxy hides it.
- The strict Xray overlay is pinned; do not change the upstream tag unless the `xray-patch` CI job passes against the new source.

## Scope boundary

This repository is the **node layer**, not a replacement for your panel. Node assignment, DNS changes for failover, global load balancing, plan/business logic and user credential issuance remain in your control plane. The agent exposes all node state required to implement those decisions safely.
