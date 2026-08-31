# Frontend contract — router-core HTTP API

This document is the stable interface contract between the
router-core backend and the third-party frontend. It is the
only thing the frontend should depend on.

The contract is **shape-first**. It describes the intended
JSON shapes for each endpoint, not the temporary behavior of
the current runtime. The frontend should consume this document
and the mock JSON fixtures in `fixtures/frontend-mocks/`,
not the current `serve` binary output.

When the runtime adapter disagrees with this document, **the
document wins**. The runtime will be brought into alignment.

## Versioning

Contract version: **v0** (pre-1.0; breaking changes may land
under a new prefix). Each endpoint is `/v0/<name>`. New
endpoints land under `/v1/...` when the contract graduates
from pre-1.0.

## Conventions

- All responses are JSON.
- All status codes are HTTP status codes; the frontend should
  branch on the status code, not on body content.
- 2xx — the body contains the requested data.
- 4xx — the request was rejected (bad path, bad method).
  `application/json` body with `{ "state", "reason" }`.
- 503 — the backend cannot satisfy the request right now
  (router unreachable, session expired, capability unverified
  against captured traffic). The frontend should surface a
  clear "session expired" or "router unreachable" hint.
- 404 — the requested capability is not implemented on this
  firmware build. The frontend should treat it as a
  "this router does not expose this surface" signal, not an
  error.

The runtime distinguishes four "states" for capabilities:

- `verified` — the capability was captured against the physical
  lab unit and the parser produces a real value.
- `absent` — the device has no such surface (e.g. no WPS, no
  UPnP, no remote management on this firmware).
- `unsupported_or_unverified` — the runtime is not yet wired to
  this surface; the endpoint is reserved.
- `unavailable` — the runtime cannot satisfy the request right
  now (transport error, session expired).

The frontend must represent these four states distinctly. Do
not collapse to `true` / `false`.

## Endpoints

### `GET /healthz`

Liveness probe. Always 200 with `{"state":"ok"}` if the
process is alive.

```json
{ "state": "ok", "http_status": 200 }
```

### `GET /v0/device`

The physical identity of the router. Vendor, model, firmware
version, hardware version, management address, and whether
the adapter has an authenticated session.

```json
{
  "vendor": "TP-Link",
  "model": "TL-WR841N/ND",
  "hardwareVersion": {
    "value": "WR841N v8 00000000",
    "trust": "untrusted",
    "source": "router:status"
  },
  "firmwareVersion": {
    "value": "3.15.9 Build 140724 Rel.63227n",
    "trust": "untrusted",
    "source": "router:status"
  },
  "managementAddress": "192.168.1.1",
  "authenticated": "true",
  "provenance": "observed"
}
```

`hardwareVersion.value` and `firmwareVersion.value` may be
empty strings when the device was not yet identified. The
frontend should treat empty as `unknown`, not as `false`.

`authenticated` is `"true"`, `"false"`, or `"unknown"`.

`provenance` is one of `observed`, `fixture`, `absent`.

### `GET /v0/status`

The router's reachability, WAN link state, and uptime.

```json
{
  "reachable": "true",
  "wanStatus": "connected",
  "uptimeSeconds": 20000,
  "uptime": "5h33m20s",
  "provenance": "observed"
}
```

`reachable` is one of `true`, `false`, `unknown`.

`wanStatus` is one of `connected`, `disconnected`,
`connecting`, `disabled`, `unknown`.

`uptimeSeconds` is a non-negative integer or `null` when
absent. `uptime` is a Go duration string (e.g. `5h33m20s`) or
`null`.

### `GET /v0/clients`

The list of devices that have a DHCP lease on the router.

```json
{
  "state": "verified",
  "clients": [
    {
      "name": "omarchy",
      "mac": "00:11:22:33:44:55:66",
      "ip": "192.168.1.100",
      "lease": "01:24:35"
    }
  ]
}
```

`state` is one of `verified` (the parser produced a list) or
`absent` (the device has no such surface on this firmware).

`clients` is always present; an empty array is valid.

`mac` is a canonical 17-character MAC string. `lease` is the
remaining lease time as `HH:MM:SS` or the literal `Permanent`.

### `GET /v0/capabilities`

The full capability matrix for the connected router. One
record per supported capability.

```json
{
  "capabilities": {
    "device": "verified",
    "status": "verified",
    "clients": "verified",
    "wireless_security": "unverified",
    "wps": "absent",
    "dmz": "verified",
    "upnp": "absent",
    "remote_management": "absent",
    "forwarding": "verified"
  }
}
```

`capabilities` is a map from capability name to state. The
state values are exactly the four documented above
(`verified`, `absent`, `unsupported_or_unverified`,
`unavailable`).

### `GET /v0/security/<name>`

Per-capability security observations. Reserved for future
expansion; the runtime today only supports the four states
plus a 503 when the underlying endpoint is not yet wired.

| Name | Path |
| --- | --- |
| Wireless | `/v0/security/wireless` |
| WPS | `/v0/security/wps` |
| DMZ | `/v0/security/dmz` |
| UPnP | `/v0/security/upnp` |
| Remote management | `/v0/security/remote-management` |
| Forwarding | `/v0/security/forwarding` |

For absent capabilities (WPS, UPnP, Remote Management on v8.4),
the runtime returns:

```json
{ "state": "unsupported_or_unverified", "reason": "..." }
```

with HTTP 404.

For unverified capabilities (wireless, DMZ, forwarding on the
current build), the runtime returns:

```json
{ "state": "unavailable", "reason": "..." }
```

with HTTP 503.

## Mock fixtures

Mock JSON for every endpoint lives in
`fixtures/frontend-mocks/`. The frontend should `fetch` the
mock files in development and switch to the live backend by
changing only the base URL. No component should care which
mode it is in.

```text
fixtures/frontend-mocks/
├── healthz.json
├── device.json
├── status.json
├── clients.json
├── capabilities.json
└── security/
    ├── wireless.json
    ├── wps.json
    ├── dmz.json
    ├── upnp.json
    ├── remote-management.json
    └── forwarding.json
```

## What the frontend should not do

- Do not parse CLI output. The CLI is a separate client of
  the same core.
- Do not encode `503` or `404` responses as "errors" without
  checking the state field. They are first-class states.
- Do not collapse `verified` / `absent` /
  `unsupported_or_unverified` / `unavailable` to `true` /
  `false`. They carry different meaning.
- Do not assume the host is reachable. The runtime may return
  503 with `state: "unavailable"` and `reason: "router-core:
  router unreachable"`. The frontend should display a clear
  hint and not retry blindly.

## What the frontend should do

- Show the device identity card with vendor, model, firmware,
  hardware.
- Show a "Verified capabilities" list using the
  `/v0/capabilities` matrix.
- For each capability, render its state with a clear visual
  distinction between `verified`, `absent`,
  `unsupported_or_unverified`, and `unavailable`.
- For the agent/AI view, the frontend should display the
  observation steps in order (wireless → WPS → remote
  management → forwarding → UPnP) so the user can see which
  observation the reasoning layer is currently doing.
- When the runtime returns 503, surface a clear hint that the
  session may have expired. The user can restart
  `router-core serve` to re-authenticate.
