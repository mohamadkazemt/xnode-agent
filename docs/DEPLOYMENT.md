# Deployment

## Production prerequisites

- Linux with systemd and `/proc`.
- `git` and `python3`.
- Go 1.23+ for the agent.
- Go 1.26+ for the pinned Xray v26.7.28 build.
- outbound access to the official XTLS/Xray-core repository during the build.

## Install

Create an agent config from `examples/agent.json`, use a unique panel token, then:

```bash
sudo ./scripts/install.sh /path/to/agent.json
```

Installed paths:

```text
/usr/local/bin/xnode-agent
/usr/local/bin/xray
/etc/xnode/agent.json
/etc/xnode/xray.json
/var/lib/xnode/state.json
/var/lib/xnode/limits.json
/var/lib/xnode/traffic-spool/
/var/log/xnode/xray-access.log
```

The installer enables `xnode-agent.service`; the agent is the parent process that starts and monitors Xray.

## Verify

```bash
systemctl status xnode-agent
curl -fsS http://127.0.0.1:19090/healthz
curl -fsS http://127.0.0.1:19090/readyz
curl -fsS http://127.0.0.1:19090/status | jq
```

If strict policies are configured, verify heartbeat `strict_limits_ready:true`. When `require_patched_core:true`, absence of the patch marker makes the node unhealthy instead of silently claiming strict enforcement.

## Rollback

Every accepted complete Xray config is validated before activation. The manager keeps `<xray_config>.last-good`. If a replacement cannot start, it restores and starts the previous known-good config.

## Upgrading Xray

Do not simply change the patch directory/tag. Create a new pinned patch directory, update anchors as needed, and require the `xray-patch` GitHub Actions job to test the dispatcher package and compile the complete Xray binary against the new official tag.

## Firewall

Expose only the actual Xray inbound ports. Do **not** expose the Xray API (`127.0.0.1:10085`) or the agent status listener (`127.0.0.1:19090`).
