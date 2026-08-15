# XNode server management

[English](MANAGEMENT.md) | [فارسی](fa/MANAGEMENT.md)

Release `v1.1.0` and later installs the bilingual `xnode` management command at
`/usr/local/bin/xnode`. Run it without arguments for an interactive menu:

```bash
sudo xnode
```

The menu manages the `xnode-agent.service`. Xray is a child process owned by the
agent, so restarting the service safely restarts both the agent and managed
Xray. There is intentionally no separate unmanaged Xray restart operation.

## Commands

```text
xnode status                  Runtime and systemd summary
xnode start                   Enable and start the service
xnode stop                    Stop the service
xnode restart                 Restart the agent and managed Xray
xnode logs [--follow]         Agent journal
xnode xray-logs [--follow]    Xray access log
xnode settings                Connection settings with token redacted
xnode configure               Atomic interactive connection editor
xnode test-panel              Authenticated desired-state request
xnode doctor                  Local, systemd and panel diagnostics
xnode update [vX.Y.Z]         Latest upgrade or version pin/rollback
xnode backup                  Configuration and state backup
xnode backups                 List backups
xnode uninstall [--purge]     Remove while preserving data, or purge all data
xnode version                 Installed agent/Xray/menu versions
```

Mutating and sensitive commands automatically re-run through `sudo` when it is
available. Configuration remains mode `0600`, and the token is never printed by
`settings`. `configure` validates the new node ID, HTTPS panel URL, token and
sync interval before installing the file. If the restarted service fails its
local health check, the previous configuration is restored.

`test-panel` performs the same authenticated desired-state GET used by the
agent. It reports the HTTP status without printing the token. `doctor` combines
that request with binary, JSON, permission, systemd, health and readiness
checks.

## Upgrade and rollback

Install the latest published release while preserving the current config:

```bash
sudo xnode update
```

Pin or return to an older published release:

```bash
sudo xnode update v1.0.2
```

The normal installer still verifies the release archive against
`SHA256SUMS`, takes transaction-scoped backups and rolls back a failed install.

## Backup and removal

`sudo xnode backup` creates a root-only archive under `/var/backups/xnode`.
Uninstall preserves `/etc/xnode`, `/var/lib/xnode` and `/var/log/xnode` by
default. `--purge` permanently removes those directories and requires an
explicit `PURGE` confirmation.

## Security notes

- Do not paste a real node token into shell arguments or GitHub issues.
- Keep the status listener and Xray API on loopback.
- The menu does not open firewall ports or edit generated Xray JSON.
- Use panel desired state for inbounds, outbounds, DNS and routing.
