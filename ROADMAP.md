# Roadmap / delivery status

## v1.0.0 — feature-complete node agent

Delivered on the v1 development line:

- declarative panel -> node convergence;
- multiple panel-created inbounds;
- generic Xray config passthrough;
- VLESS / VMess / Trojan / Shadowsocks user management;
- WireGuard peer management and per-peer accounting identity;
- per-user/per-inbound traffic accounting;
- durable idempotent traffic delivery;
- runtime inbound/user changes with validated restart fallback;
- quota, expiration and device-credential policies;
- exact patched-core speed, connection and source-IP limits;
- suspend/resume and active-session generation invalidation;
- online-IP/session reporting;
- per-user routing;
- active/draining/maintenance/disabled node lifecycle;
- traffic threshold, region, group, tags and weight;
- CPU/RAM/load/network monitoring;
- health checks, restart, recovery and last-good rollback;
- systemd installer, logrotate, CI, patched-Xray CI and release artifacts.

## Control-plane responsibilities

The following are intentionally not implemented as node-local decisions because the panel already owns global state:

- cross-node user assignment;
- node-group scheduling;
- failover between VPS nodes;
- weighted load balancing between VPS nodes;
- global traffic/capacity policy.

`xnode-agent` reports the fields required for those decisions. Your panel scheduler should select nodes and publish the resulting desired state.

## Post-v1 optional work

- signed desired-state payloads / mTLS;
- Prometheus/OpenTelemetry exporter;
- certificate lifecycle helpers;
- multi-version Xray patch matrix;
- richer diagnostics/history API;
- automated canary rollout across node groups.
