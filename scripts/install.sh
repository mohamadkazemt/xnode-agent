#!/usr/bin/env bash
set -Eeuo pipefail

readonly REPO="mohamadkazemt/xnode-agent"
readonly CONFIG_PATH="/etc/xnode/agent.json"
readonly SERVICE_PATH="/etc/systemd/system/xnode-agent.service"
readonly LOGROTATE_PATH="/etc/logrotate.d/xnode"
readonly MENU_PATH="/usr/local/bin/xnode"
readonly LIB_DIR="/usr/local/lib/xnode"
readonly UNINSTALL_PATH="$LIB_DIR/uninstall.sh"
readonly VERSION_PATH="$LIB_DIR/VERSION"

VERSION=""
CONFIG_SOURCE=""
NODE_ID=""
PANEL_URL=""
PANEL_TOKEN=""
FORCE=0
NO_START=0
TMP_DIR=""
BACKUP_DIR=""
COMMITTED=0

log() { printf '[xnode] %s\n' "$*"; }
die() { printf '[xnode] error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage: install.sh [options]
  --node-id ID          Node identifier (prompted when omitted)
  --panel-url URL       Panel URL (prompted when omitted)
  --panel-token TOKEN   Per-node token (prompted securely when omitted)
  --config FILE         Install an existing JSON config
  --version VERSION     Install a release (for example v1.1.0); default: latest
  --force               Replace an existing config
  --no-start            Do not start the service (CI/images without systemd)
  -h, --help            Show this help

Examples:
  sudo bash -c "$(curl -fsSL https://raw.githubusercontent.com/mohamadkazemt/xnode-agent/main/scripts/install.sh)"
  sudo bash install.sh --node-id node-1 --panel-url https://panel.example.com --panel-token 'TOKEN'
  sudo bash install.sh --config ./agent.json --version v1.1.0
EOF
}

while (($#)); do
  case "$1" in
    --node-id) [[ $# -ge 2 ]] || die "--node-id needs a value"; NODE_ID=$2; shift 2 ;;
    --panel-url) [[ $# -ge 2 ]] || die "--panel-url needs a value"; PANEL_URL=$2; shift 2 ;;
    --panel-token) [[ $# -ge 2 ]] || die "--panel-token needs a value"; PANEL_TOKEN=$2; shift 2 ;;
    --config) [[ $# -ge 2 ]] || die "--config needs a value"; CONFIG_SOURCE=$2; shift 2 ;;
    --version) [[ $# -ge 2 ]] || die "--version needs a value"; VERSION=$2; shift 2 ;;
    --force) FORCE=1; shift ;;
    --no-start) NO_START=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown option: $1" ;;
  esac
done

[[ ${EUID:-$(id -u)} -eq 0 ]] || die "run as root (for example: curl ... | sudo bash)"
[[ -z $CONFIG_SOURCE || (-z $NODE_ID && -z $PANEL_URL && -z $PANEL_TOKEN) ]] || \
  die "--config cannot be combined with node/panel options"
[[ ! -e $CONFIG_PATH || $FORCE -eq 1 ]] || \
  die "$CONFIG_PATH already exists; use --force to replace it"

for cmd in tar mktemp install python3; do command -v "$cmd" >/dev/null || die "missing dependency: $cmd"; done
if command -v curl >/dev/null; then
  download() { curl -fL --retry 3 --connect-timeout 15 -o "$2" "$1"; }
elif command -v wget >/dev/null; then
  download() { wget -q --tries=3 -O "$2" "$1"; }
else
  die "curl or wget is required"
fi

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported architecture: $arch (supported: amd64, arm64)" ;;
esac

TMP_DIR=$(mktemp -d)
BACKUP_DIR="$TMP_DIR/backup"
mkdir -p "$BACKUP_DIR"

backup_path() {
  local path=$1 name
  name=${path#/}; name=${name//\//__}
  if [[ -e $path ]]; then cp -a "$path" "$BACKUP_DIR/$name"; fi
}

rollback() {
  local status=$?
  [[ $COMMITTED -eq 1 || $status -eq 0 ]] && return
  printf '[xnode] installation failed; rolling back\n' >&2
  if command -v systemctl >/dev/null && [[ $NO_START -eq 0 ]]; then
    systemctl stop xnode-agent.service >/dev/null 2>&1 || true
  fi
  for path in /usr/local/bin/xnode-agent /usr/local/bin/xray "$MENU_PATH" "$UNINSTALL_PATH" "$VERSION_PATH" "$CONFIG_PATH" "$SERVICE_PATH" "$LOGROTATE_PATH"; do
    local name=${path#/}; name=${name//\//__}
    if [[ -e "$BACKUP_DIR/$name" ]]; then cp -a "$BACKUP_DIR/$name" "$path"; else rm -f "$path"; fi
  done
  rmdir "$LIB_DIR" >/dev/null 2>&1 || true
  if command -v systemctl >/dev/null; then systemctl daemon-reload >/dev/null 2>&1 || true; fi
  exit "$status"
}
cleanup() { rm -rf "$TMP_DIR"; }
trap rollback ERR INT TERM
trap cleanup EXIT

if [[ -z $VERSION ]]; then
  latest_url="https://github.com/$REPO/releases/latest"
  if command -v curl >/dev/null; then
    VERSION=$(curl -fsSIL -o /dev/null -w '%{url_effective}' "$latest_url")
  else
    VERSION=$(wget -q --max-redirect=10 --server-response -O /dev/null "$latest_url" 2>&1 | awk '/^  Location:/ {v=$2} END {print v}')
  fi
  VERSION=${VERSION%/}; VERSION=${VERSION##*/}
fi
[[ $VERSION == v* ]] || VERSION="v$VERSION"
[[ $VERSION =~ ^v[0-9][0-9A-Za-z._-]*$ ]] || die "invalid version: $VERSION"

asset="xnode-${VERSION}-linux-${ARCH}.tar.gz"
base="https://github.com/$REPO/releases/download/$VERSION"
log "downloading $asset"
download "$base/$asset" "$TMP_DIR/$asset"
download "$base/SHA256SUMS" "$TMP_DIR/SHA256SUMS"

expected=$(awk -v f="$asset" '$2 == f || $2 == "*" f {print $1}' "$TMP_DIR/SHA256SUMS")
[[ $expected =~ ^[0-9a-fA-F]{64}$ ]] || die "checksum for $asset not found"
if command -v sha256sum >/dev/null; then
  actual=$(sha256sum "$TMP_DIR/$asset" | awk '{print $1}')
elif command -v shasum >/dev/null; then
  actual=$(shasum -a 256 "$TMP_DIR/$asset" | awk '{print $1}')
else
  die "sha256sum or shasum is required"
fi
[[ ${actual,,} == "${expected,,}" ]] || die "SHA-256 verification failed"
log "checksum verified"

mkdir "$TMP_DIR/extract"
tar -xzf "$TMP_DIR/$asset" -C "$TMP_DIR/extract"
for binary in xnode-agent xray; do [[ -f "$TMP_DIR/extract/$binary" ]] || die "release is missing $binary"; done

if [[ -n $CONFIG_SOURCE ]]; then
  [[ -f $CONFIG_SOURCE ]] || die "config file not found: $CONFIG_SOURCE"
  cp "$CONFIG_SOURCE" "$TMP_DIR/agent.json"
else
  if [[ -z $NODE_ID && -t 0 ]]; then read -r -p 'Node ID: ' NODE_ID; fi
  if [[ -z $PANEL_URL && -t 0 ]]; then read -r -p 'Panel URL: ' PANEL_URL; fi
  if [[ -z $PANEL_TOKEN && -t 0 ]]; then read -r -s -p 'Panel token: ' PANEL_TOKEN; printf '\n'; fi
  [[ -n $NODE_ID && -n $PANEL_URL && -n $PANEL_TOKEN ]] || \
    die "node_id, panel_url and panel_token are required (use flags when piping)"
  XNODE_NODE_ID=$NODE_ID XNODE_PANEL_URL=$PANEL_URL XNODE_PANEL_TOKEN=$PANEL_TOKEN \
    python3 - "$TMP_DIR/agent.json" <<'PY'
import json, os, sys
data = {
  "node_id": os.environ["XNODE_NODE_ID"], "panel_url": os.environ["XNODE_PANEL_URL"],
  "panel_token": os.environ["XNODE_PANEL_TOKEN"], "sync_seconds": 15,
  "xray_binary": "/usr/local/bin/xray", "xray_config": "/etc/xnode/xray.json",
  "xray_api": "127.0.0.1:10085", "xray_access_log": "/var/log/xnode/xray-access.log",
  "xray_limits_file": "/var/lib/xnode/limits.json", "state_file": "/var/lib/xnode/state.json",
  "traffic_spool_dir": "/var/lib/xnode/traffic-spool", "ip_window_seconds": 120,
  "report_sessions": True, "require_patched_core": True, "listen": "127.0.0.1:19090",
  "weight": 100
}
with open(sys.argv[1], "w", encoding="utf-8") as f: json.dump(data, f, indent=2); f.write("\n")
PY
fi

python3 - "$TMP_DIR/agent.json" <<'PY'
import json, sys, urllib.parse
with open(sys.argv[1], encoding="utf-8") as f: d=json.load(f)
for key in ("node_id", "panel_url", "panel_token"):
    if not isinstance(d.get(key), str) or not d[key].strip(): raise SystemExit("missing required config key: " + key)
if d["panel_token"] in {"CHANGE_ME", "REPLACE_ME"}: raise SystemExit("replace the placeholder panel_token")
u=urllib.parse.urlparse(d["panel_url"])
if not u.hostname or (u.scheme != "https" and not (u.scheme == "http" and u.hostname in {"127.0.0.1","localhost","::1"})):
    raise SystemExit("panel_url must use HTTPS unless it is loopback")
PY

for path in /usr/local/bin/xnode-agent /usr/local/bin/xray "$MENU_PATH" "$UNINSTALL_PATH" "$VERSION_PATH" "$CONFIG_PATH" "$SERVICE_PATH" "$LOGROTATE_PATH"; do backup_path "$path"; done
install -d -m 0700 /etc/xnode /var/lib/xnode /var/lib/xnode/traffic-spool /var/log/xnode
install -m 0755 "$TMP_DIR/extract/xnode-agent" /usr/local/bin/xnode-agent
install -m 0755 "$TMP_DIR/extract/xray" /usr/local/bin/xray
install -m 0600 "$TMP_DIR/agent.json" "$CONFIG_PATH"
if [[ -f "$TMP_DIR/extract/xnode" && -f "$TMP_DIR/extract/xnode-uninstall" ]]; then
  install -d -m 0755 "$LIB_DIR"
  install -m 0755 "$TMP_DIR/extract/xnode" "$MENU_PATH"
  install -m 0700 "$TMP_DIR/extract/xnode-uninstall" "$UNINSTALL_PATH"
  if [[ -f "$TMP_DIR/extract/VERSION" ]]; then
    install -m 0644 "$TMP_DIR/extract/VERSION" "$VERSION_PATH"
  else
    printf '%s\n' "${VERSION#v}" >"$VERSION_PATH"
    chmod 0644 "$VERSION_PATH"
  fi
else
  log "management menu is not bundled in $VERSION; skipping /usr/local/bin/xnode"
fi

cat >"$TMP_DIR/xnode-agent.service" <<'EOF'
[Unit]
Description=XNode Agent and managed Xray core
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/xnode-agent -config /etc/xnode/agent.json
Restart=always
RestartSec=2
NoNewPrivileges=true
UMask=0077
LimitNOFILE=1048576
PrivateTmp=true
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
LockPersonality=true
CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_NET_ADMIN
AmbientCapabilities=CAP_NET_BIND_SERVICE CAP_NET_ADMIN
ReadWritePaths=/etc/xnode /var/lib/xnode /var/log/xnode

[Install]
WantedBy=multi-user.target
EOF
cat >"$TMP_DIR/xnode.logrotate" <<'EOF'
/var/log/xnode/xray-access.log {
    daily
    rotate 7
    size 100M
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    create 0600 root root
}
EOF
install -m 0644 "$TMP_DIR/xnode-agent.service" "$SERVICE_PATH"
install -m 0644 "$TMP_DIR/xnode.logrotate" "$LOGROTATE_PATH"

if [[ $NO_START -eq 0 ]]; then
  command -v systemctl >/dev/null || die "systemctl not found; use --no-start only for image/CI installs"
  [[ -d /run/systemd/system ]] || die "systemd is not running; use --no-start only for image/CI installs"
  systemctl daemon-reload
  systemctl enable --now xnode-agent.service
  for _ in {1..20}; do
    if command -v curl >/dev/null && curl -fsS --max-time 2 http://127.0.0.1:19090/healthz >/dev/null; then break; fi
    sleep 1
  done
  curl -fsS --max-time 2 http://127.0.0.1:19090/healthz >/dev/null || die "health check failed"
  systemctl is-active --quiet xnode-agent.service || die "service is not active"
fi

COMMITTED=1
trap - ERR INT TERM
log "installed xnode-agent $VERSION ($ARCH)"
[[ $NO_START -eq 1 ]] && log "service files installed but service was not started"
log "status: curl -fsS http://127.0.0.1:19090/status"
[[ -x $MENU_PATH ]] && log "management: sudo xnode"

