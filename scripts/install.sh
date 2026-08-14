#!/usr/bin/env bash
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "run as root: sudo ./scripts/install.sh /path/to/agent.json" >&2
  exit 1
fi
if [[ $# -ne 1 ]]; then
  echo "usage: $0 /path/to/agent.json" >&2
  exit 1
fi
CONFIG_SOURCE="$(readlink -f "$1")"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
for cmd in go git python3 systemctl install; do
  command -v "$cmd" >/dev/null || { echo "missing dependency: $cmd" >&2; exit 1; }
done

python3 - "$CONFIG_SOURCE" <<'PY'
import json,sys
p=sys.argv[1]
d=json.load(open(p))
for k in ("node_id","panel_url","panel_token"):
    if not d.get(k): raise SystemExit(f"missing required config key: {k}")
if d.get("panel_token") in {"CHANGE_ME","REPLACE_ME"}: raise SystemExit("replace panel_token before install")
PY

mkdir -p /etc/xnode /var/lib/xnode/traffic-spool /var/log/xnode
chmod 700 /etc/xnode /var/lib/xnode /var/lib/xnode/traffic-spool /var/log/xnode

cd "$ROOT"
go test ./...
go build -trimpath -ldflags='-s -w' -o /tmp/xnode-agent ./cmd/xnode-agent
install -m 0755 /tmp/xnode-agent /usr/local/bin/xnode-agent

# Build the pinned Xray core with the maintained xnode dispatcher limiter.
XRAY_BUILD_DIR="${XRAY_BUILD_DIR:-/tmp/xnode-xray-build}"
"$ROOT/patches/xray-v26.7.28/build-xray.sh" "$XRAY_BUILD_DIR" >/tmp/xnode-xray-build-path
install -m 0755 "$XRAY_BUILD_DIR/xray" /usr/local/bin/xray

install -m 0600 "$CONFIG_SOURCE" /etc/xnode/agent.json
install -m 0644 "$ROOT/systemd/xnode-agent.service" /etc/systemd/system/xnode-agent.service
install -m 0644 "$ROOT/packaging/logrotate/xnode" /etc/logrotate.d/xnode
systemctl daemon-reload
systemctl enable --now xnode-agent
systemctl --no-pager --full status xnode-agent || true

echo "installed xnode-agent v$(cat "$ROOT/VERSION")"
echo "local status: curl http://127.0.0.1:19090/status"
