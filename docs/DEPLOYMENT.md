# Deployment

## Requirements

- Linux on `amd64` or `arm64`.
- systemd for a live install (`--no-start` is intended for CI and image builds).
- `curl` or `wget`, `python3`, `tar`, and `sha256sum` (or `shasum`).

Debian and Ubuntu provide these packages out of the box or through `apt`. The
installer otherwise avoids distribution-specific package management.

## One-line install

Interactive installation of the latest GitHub release:

```bash
sudo bash -c "$(curl -fsSL https://raw.githubusercontent.com/mohamadkazemt/xnode-agent/main/scripts/install.sh)"
```

Piped input cannot be used for prompts. Use flags for unattended installation:

```bash
curl -fsSL https://raw.githubusercontent.com/mohamadkazemt/xnode-agent/main/scripts/install.sh | \
  sudo bash -s -- --node-id node-12 --panel-url https://panel.example.com --panel-token 'UNIQUE_NODE_TOKEN'
```

To install a prepared config or pin a release:

```bash
sudo bash scripts/install.sh --config ./agent.json --version v1.0.1
```

Supported options are `--node-id`, `--panel-url`, `--panel-token`, `--config`,
`--version`, `--force`, and `--no-start`. Existing `/etc/xnode/agent.json` is
never replaced unless `--force` is provided. Do not put real tokens in the repo
or shell history; a root-readable config file is preferable for automation.

The installer resolves the latest release unless `--version` is supplied,
detects `amd64`/`arm64`, downloads both release assets, verifies the archive
against `SHA256SUMS`, and installs:

```text
/usr/local/bin/xnode-agent
/usr/local/bin/xray
/etc/xnode/agent.json             (0600)
/etc/systemd/system/xnode-agent.service
/etc/logrotate.d/xnode
/var/lib/xnode/traffic-spool/
/var/log/xnode/
```

Configuration and data directories are mode `0700`. If file installation,
service startup, or the local health check fails, previous managed files are
restored and the failed installation is removed.

## Verify

```bash
systemctl status xnode-agent
curl -fsS http://127.0.0.1:19090/healthz
curl -fsS http://127.0.0.1:19090/readyz
curl -fsS http://127.0.0.1:19090/status
```

## Upgrade

Run the installer again with `--force`. The existing managed files are backed up
for the duration of the transaction and restored if the upgrade fails. Use
`--version vX.Y.Z` to pin or roll back to a specific published release.

## Uninstall

```bash
sudo bash scripts/uninstall.sh
```

This removes binaries and service integration but preserves configuration,
state, and logs. Remove those too only when intentionally decommissioning a node:

```bash
sudo bash scripts/uninstall.sh --purge
```

## Firewall

Expose only Xray inbound ports. Do not expose the Xray API
(`127.0.0.1:10085`) or agent status listener (`127.0.0.1:19090`).

