#!/usr/bin/env bash
set -euo pipefail
TAG="${XRAY_TAG:-v26.7.28}"
ROOT="${1:-/tmp/xnode-xray-build}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
rm -rf "$ROOT"
git clone --depth 1 --branch "$TAG" https://github.com/XTLS/Xray-core.git "$ROOT"
cp "$SCRIPT_DIR/xnode_limiter.go.txt" "$ROOT/app/dispatcher/xnode_limiter.go"
cp "$SCRIPT_DIR/xnode_limiter_test.go.txt" "$ROOT/app/dispatcher/xnode_limiter_test.go"
python3 "$SCRIPT_DIR/patch_default.py" "$ROOT/app/dispatcher/default.go"
(
  cd "$ROOT"
  go test ./app/dispatcher
  go build -trimpath -o xray ./main
)
echo "$ROOT/xray"
