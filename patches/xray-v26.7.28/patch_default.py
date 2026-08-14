#!/usr/bin/env python3
from pathlib import Path
import sys

if len(sys.argv) != 2:
    raise SystemExit("usage: patch_default.py <app/dispatcher/default.go>")
p = Path(sys.argv[1])
s = p.read_text()

repls = [
    (
        "\treturn inboundLink, outboundLink\n}\n\nfunc WrapLink",
        "\txnodeWrapGeneratedLinks(ctx, inboundLink, outboundLink)\n\treturn inboundLink, outboundLink\n}\n\nfunc WrapLink",
    ),
    (
        "func (d *DefaultDispatcher) Dispatch(ctx context.Context, destination net.Destination) (*transport.Link, error) {\n\tif !destination.IsValid() {\n\t\tpanic(\"Dispatcher: Invalid destination.\")\n\t}\n",
        "func (d *DefaultDispatcher) Dispatch(ctx context.Context, destination net.Destination) (*transport.Link, error) {\n\tif !destination.IsValid() {\n\t\tpanic(\"Dispatcher: Invalid destination.\")\n\t}\n\tif err := xnodeAdmit(ctx); err != nil {\n\t\treturn nil, err\n\t}\n",
    ),
    (
        "func (d *DefaultDispatcher) DispatchLink(ctx context.Context, destination net.Destination, outbound *transport.Link) error {\n\tif !destination.IsValid() {\n\t\treturn errors.New(\"Dispatcher: Invalid destination.\")\n\t}\n",
        "func (d *DefaultDispatcher) DispatchLink(ctx context.Context, destination net.Destination, outbound *transport.Link) error {\n\tif !destination.IsValid() {\n\t\treturn errors.New(\"Dispatcher: Invalid destination.\")\n\t}\n\tif err := xnodeAdmit(ctx); err != nil {\n\t\treturn err\n\t}\n",
    ),
    (
        "\toutbound = WrapLink(ctx, d.policy, d.stats, outbound)\n",
        "\toutbound = xnodeWrapProvidedLink(ctx, outbound)\n\toutbound = WrapLink(ctx, d.policy, d.stats, outbound)\n",
    ),
]

for old, new in repls:
    count = s.count(old)
    if count != 1:
        raise SystemExit(f"anchor mismatch ({count} matches): {old[:80]!r}")
    s = s.replace(old, new, 1)
p.write_text(s)
