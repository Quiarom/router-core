# router-core

`router-core` is a local-first, read-only observation layer for legacy
consumer routers. The first target is the TP-Link TL-WR841N/ND v8.4 stock
dashboard. It emits deterministic facts and does not interpret them or issue
configuration changes.

Unknown is not false: absent fields remain `unknown` (or an invalid optional
integer), and unsupported fields carry a reason. Network-originated text is
untrusted data, sanitized for safe display, and never treated as instructions.

## Build and run

Requires Go 1.24 or newer:

```sh
go build -o router-core ./cmd/router-core
./router-core probe --host 192.168.0.1
./router-core inspect --host 192.168.0.1
./router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8
./router-core inspect --fixtures fixtures/synthetic/tplink-wr841n-v8 --json
```

Live authentication is currently blocked because no sanitized hardware
captures exist. Live endpoint recipes are also unverified and refuse dispatch
by default. See `BLOCKED_CAPTURE.md`.

Environment variables:

* `ROUTER_ALLOW_UNVERIFIED=1` explicitly permits requests to unverified local
  endpoint recipes.
* `ROUTER_LIVE_TESTS=1` opts into the conservative local integration test.

## Security

Only GET requests are available. Targets must be literal loopback or RFC1918
addresses (or `localhost`), redirects must remain on the same host, and
responses have a bounded size. No credentials, tokens, API keys, or Wi-Fi
secrets belong in this repository. Sanitized real captures use opaque
placeholders such as `<SESSION_TOKEN>` and `<ROUTER_ADMIN_PASSWORD>`.

The physical router has not been tested by this repository yet.
