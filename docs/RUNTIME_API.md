# Runtime Xray API behavior

v0.2 uses Xray's own `api` subcommands as the compatibility layer over HandlerService.

## Add inbound

```bash
xray api adi --server=127.0.0.1:10085 /tmp/inbound.json
```

The temporary document has this shape:

```json
{
  "inbounds": [
    { "tag": "vless-443", "protocol": "vless", "port": 443 }
  ]
}
```

## Remove inbound

```bash
xray api rmi --server=127.0.0.1:10085 vless-443
```

## Add user

```bash
xray api adu --server=127.0.0.1:10085 /tmp/one-user-inbound.json
```

The document contains the real inbound configuration but only the user being added. Xray's CLI builds the typed inbound, extracts the user, and calls `AlterInbound(AddUserOperation)`.

## Remove user

```bash
xray api rmu --server=127.0.0.1:10085 -tag=vless-443 'u:25|i:101'
```

## Fallback behavior

Every desired full config is validated before runtime mutations. If any runtime command fails, `xnode-agent` performs a full validated config apply and restarts Xray.

This is important for protocol/version differences. For example, some inbound implementations may not expose Xray's `UserManager` runtime interface even if their static JSON has a user list.
