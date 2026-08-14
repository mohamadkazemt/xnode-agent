# Limit and session enforcement

[English](LIMITS.md) | [فارسی](fa/LIMITS.md)

## Identity

For managed client protocols the agent assigns each credential a deterministic Xray email:

```text
u:<user_id>|i:<inbound_id>
```

That identity is used by traffic stats, online-IP stats, routing and the strict dispatcher policy.

## Traffic quota

The panel sends both the quota and cumulative usage already committed by the panel:

```json
{"traffic_bytes":536870912000,"traffic_used_bytes":420000000000}
```

Before evaluating policy, the agent retries queued traffic events and adds any still-uncommitted local spool deltas. The effective user is blocked when:

```text
traffic_used_bytes + pending_unsent_bytes >= traffic_bytes
```

With the maintained core overlay, an existing data-path session observes the block and terminates on its next read/write. Without the overlay, removal still prevents new authentication but an already-established session may live until it naturally ends.

## Expiration

`expires_at` is a Unix timestamp in seconds. At or after that time the effective user is disabled. The same live-session cutoff behavior as quota applies.

## IP limit

With the maintained patched core, `ip_limit` is enforced during authenticated dispatcher admission. The patch keeps ref-counted active source IPs per synthetic user identity; a new distinct IP is rejected immediately when the cap is full. Multiple concurrent connections from an already-admitted IP do not consume additional IP slots.

The agent also reads Xray's native online-IP map for reporting and policy reconciliation, with accepted access-log records as a compatibility fallback. Set `ip_limit_mode: "off"` for an inbound when Xray sees a CDN/reverse-proxy address instead of the real client.

## Device limit

A physical device cannot be reliably inferred from IP. Use one credential per device and give those credentials the same `account_id`:

```json
{"id":"device-a","account_id":"25","limits":{"device_limit":2}}
```

The agent admits at most the first N enabled credentials for that account inside an inbound. The panel should also enforce the same cap when issuing credentials.

## Speed limit

The maintained Xray overlay enforces aggregate per-authenticated-user rates across concurrent connections. Values are **bytes/second**:

```json
{"upload_bps":2500000,"download_bps":12500000}
```

That is roughly 20 Mbit/s upload and 100 Mbit/s download. A shared per-user/direction leaky bucket prevents opening more connections from multiplying the configured limit.

## Connection limit

`connection_limit` is enforced at Xray dispatcher admission for the patched core. Every authenticated logical dispatcher connection is counted even if no limit is currently configured, so lowering a limit later is visible to existing sessions on subsequent I/O.

## Node drain admission

When node mode is `draining`, the agent keeps the current Xray configuration and policy identities in place, while the patched dispatcher rejects **new authenticated dispatcher admissions**. Connections that were already admitted are not disconnected merely because drain mode started, so they can finish naturally. The agent reports `drain_ready` when online users reach the panel-provided target.

This is intentionally different from `maintenance` or `disabled`, which stop Xray.

## Suspend / resume

Set `enabled:false` to suspend a credential. HandlerService removes it from authentication and the strict policy marks it blocked. Set `enabled:true` again to resume it.

When a user is completely deleted from desired state, the limits file retains a short blocked tombstone. This closes old sessions that might otherwise survive a runtime credential deletion.

## Disconnect active sessions without suspending

Each managed user has `session_generation`. Increment the integer in your panel while keeping the user enabled. New sessions capture the new generation; connections created under the previous generation are terminated on their next data-path operation.

This provides a clean `Disconnect all sessions` operation without rotating the UUID/password.

## Optional external limiter

`strict_limit_backend_url` remains available as an extension hook. When configured, the agent mirrors user limits to:

```text
PUT    /v1/limits/{node}/{inbound}/{user}
DELETE /v1/limits/{node}/{inbound}/{user}
```

It is not required when the bundled Xray dispatcher overlay is used.

## WireGuard peers

For an inbound WireGuard peer the agent injects the same deterministic `email` and `level` fields used by other managed users. Xray's WireGuard server resolves a peer from its allowed source address, attaches that `MemoryUser` to the dispatcher session, and therefore participates in user traffic stats, online-IP tracking, per-user routing and the dispatcher limiter. Runtime peer membership is still applied by replacing the WireGuard inbound because the current Xray `adu` CLI does not extract WireGuard users.

