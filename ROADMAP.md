# Project status

## v1.0.0 — feature-complete node layer

Delivered:

- declarative panel -> node synchronization;
- multi-inbound lifecycle and generic Xray config passthrough;
- runtime HandlerService reconciliation with validated restart fallback;
- managed VLESS/VMess/Trojan/Shadowsocks users and WireGuard peers;
- per-user/per-inbound traffic accounting and retry-safe billing spool;
- traffic quota, expiration, IP limit and credential-per-device limit;
- exact online-IP integration with access-log fallback;
- per-user routing and protocol membership;
- suspend/resume and session-generation disconnect;
- maintained Xray v26.7.28 dispatcher overlay for aggregate upload/download speed limits and exact concurrent connection admission;
- active-session cutoff and removed-user tombstones;
- node active/draining/maintenance/disabled modes;
- traffic-threshold drain, region/group/tags/weight telemetry;
- CPU/RAM/load/network monitoring, health checks, restart/recovery and last-good rollback;
- loopback admin health/readiness/status API;
- systemd, logrotate, installer and CI for both agent and patched Xray.

## Control-plane responsibilities

These are intentionally not implemented inside a single node agent because they require a global view of multiple servers:

- cross-node user assignment;
- global weighted load balancing;
- failover ordering and DNS/endpoint switching;
- group/region capacity policy;
- billing/business logic and credential issuance.

The heartbeat and node policy model expose the health/mode/weight/region/group/traffic data a panel needs to implement those functions.

## Optional future improvements

- mTLS/signed desired-state payloads in addition to bearer tokens;
- first-class Prometheus/OpenTelemetry exporter;
- dedicated WireGuard per-peer byte accounting provider;
- automated release artifacts for the patched Xray binary;
- multi-version Xray patch matrix after upstream upgrades.
