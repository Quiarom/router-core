# router-core

> Local-first, read-only observation layer for legacy consumer routers.
> Built for the **GMI Cloud × MiniMax Week** hackathon (track: Reasoning).
> See [HACKATHON.md](./HACKATHON.md) for the submission brief, model usage,
> and demo steps.

`router-core` turns questions about your home router into structured
read-only requests against the device, parses the firmware's response
deterministically, and hands the typed result to a MiniMax model that
phrases the answer in plain language. It cannot change the router's
settings, reboot it, or do anything destructive.

The first target is the **TP-Link TL-WR841N/ND v8.4** stock dashboard
(firmware `3.13.33 Build 130506 Rel.48660n`). Other vendor families
will grow from the same domain types.

Unknown is not false: absent fields remain `unknown` (or an invalid
optional integer), and unsupported fields carry a reason.
Network-originated text is untrusted data, sanitized for safe display,
and never treated as instructions.

## Build and run

Requires Go 1.25 or newer:

```sh
go build -o router-core ./cmd/router-core
go build -o router-core-learn ./cmd/router-core-learn

# Probe a synthetic fixture (no hardware required)
./router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8

# Probe the physical lab unit
./router-core probe --host 192.168.0.1

# Serve a typed HTTP API on loopback for the MiniMax-driven frontend
./router-core serve --host 192.168.0.1 --addr 127.0.0.1:8484
# (then prompts for the admin password)
curl http://127.0.0.1:8484/v0/status
```

## Security

Only GET requests are available. Targets must be literal loopback or
RFC1918 addresses (or `localhost`), redirects must remain on the same
host, and responses have a bounded size of 2 MiB. No credentials,
tokens, API keys, or Wi-Fi secrets belong in this repository. Sanitized
real captures use opaque placeholders such as `<SESSION_TOKEN>` and
`<ROUTER_ADMIN_PASSWORD>`. The `serve` binary reads the admin password
from `/dev/tty` with echo disabled, holds it only in process memory,
and zeroes it on exit.

## What's verified

The full per-capability matrix is in
[docs/STATUS.md](./docs/STATUS.md). Authentication and six read-only
capabilities were verified against the physical lab unit on 2026-08-30
and 2026-08-31. Sanitized evidence is in `fixtures/captured/`. The
verified Basic Auth recipe is documented in
[docs/adr/0005-verified-wr841n-auth-recipe.md](./docs/adr/0005-verified-wr841n-auth-recipe.md).

## Environment variables

- `ROUTER_ALLOW_UNVERIFIED=1` — explicitly permits requests to
  unverified local endpoint recipes. Off by default.
- `ROUTER_LIVE_TESTS=1` — opts into the conservative local integration
  test.

## Layout

- `cmd/router-core/` — runtime CLI: `probe`, `inspect`, `serve`.
- `cmd/router-core-learn/` — experimental probe and observation
  capture.
- `internal/domain/` — vendor-neutral types and the
  `RouterAdapter` contract.
- `internal/transport/` — guarded HTTP client.
- `internal/adapters/tplinkwr841v8/` — vendor-specific code for the
  WR841N v8.4.
- `internal/adapters/fixture/` — fixture-backed adapter for testing
  without hardware.
- `fixtures/synthetic/` — synthetic dashboard pages for replay.
- `fixtures/captured/` — sanitized evidence from the physical lab unit.
- `docs/adr/` — architecture decision records.
- `HACKATHON.md` — MiniMax-Week submission brief.

## License

MIT. See [LICENSE](./LICENSE).
