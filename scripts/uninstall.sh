#!/usr/bin/env bash
set -Eeuo pipefail

PURGE=0
if [[ ${1:-} == "--purge" ]]; then PURGE=1; shift; fi
[[ $# -eq 0 ]] || { echo "usage: $0 [--purge]" >&2; exit 1; }
[[ ${EUID:-$(id -u)} -eq 0 ]] || { echo "run as root" >&2; exit 1; }

if command -v systemctl >/dev/null; then
  systemctl disable --now xnode-agent.service >/dev/null 2>&1 || true
fi
rm -f /usr/local/bin/xnode-agent /usr/local/bin/xray /usr/local/bin/xnode
rm -rf /usr/local/lib/xnode
rm -f /etc/systemd/system/xnode-agent.service /etc/logrotate.d/xnode
if command -v systemctl >/dev/null; then systemctl daemon-reload >/dev/null 2>&1 || true; fi

if [[ $PURGE -eq 1 ]]; then
  rm -rf /etc/xnode /var/lib/xnode /var/log/xnode
  echo "xnode-agent removed, including configuration and data"
else
  echo "xnode-agent removed; /etc/xnode, /var/lib/xnode and /var/log/xnode were preserved"
  echo "use --purge to remove configuration and data"
fi

