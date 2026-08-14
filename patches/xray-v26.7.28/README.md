# xnode Xray dispatcher overlay — pinned v26.7.28

This overlay is the strict per-authenticated-user data-path backend for xnode-agent v1.0.

It provides:

- aggregate upload/download byte-rate limiting across all connections for a synthetic Xray email;
- exact concurrent dispatcher admission counting;
- exact active source-IP admission with per-IP refcounts;
- live block enforcement for suspend/quota/expiration/device decisions;
- true node drain admission: new authenticated dispatcher sessions are rejected while existing sessions keep running;
- session-generation disconnect;
- deleted-credential tombstone enforcement;
- live atomic policy reload without restarting Xray.

The change surface is intentionally small: `xnode_limiter.go` plus four checked anchor insertions into `app/dispatcher/default.go`. `patch_default.py` fails if the pinned upstream source no longer matches the expected anchors.

Build manually:

```bash
./patches/xray-v26.7.28/build-xray.sh
sudo install -m 0755 /tmp/xnode-xray-build/xray /usr/local/bin/xray
```

Xray v26.7.28 declares Go 1.26, so use Go 1.26+ for this build. CI overlays the files onto the official XTLS/Xray-core tag, runs `go test ./app/dispatcher`, builds `./main`, then the repository end-to-end job starts that Xray binary and exercises its runtime API.

Runtime contract:

```text
XNODE_LIMITS_FILE=/var/lib/xnode/limits.json
```

The patched core writes `/var/lib/xnode/limits.json.core-ready` when the overlay is loaded. The agent can require that marker with `require_patched_core:true`.
