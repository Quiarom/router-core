# ADR 0005 — WR841N v8.4 Basic Auth recipe verified against physical lab unit

Status: **active** — applied 2026-08-30.

## Decision

The TP-Link TL-WR841N v8.4 firmware `3.13.33 Build 130506 Rel.48660n`
runs on the physical lab unit at `192.168.1.1`. Authentication uses
**HTTP Basic Authorization against `/` with the plaintext password
(not pre-hashed)**. The header value is:

```
Authorization: Basic <base64("admin:<plaintext-password>")>
```

The response on success is `200 OK` with the dashboard HTML directly
(no redirect, no URL token). On failure, the response is `401
Unauthorized` with body ` ` and header
`WWW-Authenticate: Basic realm="TP-LINK Wireless N Router WR841N"`.

The session lives entirely in the browser's HTTP cache and is
re-transmitted on every request to the same origin until the browser
process restarts.

## Evidence

Captured 2026-08-30 from the physical lab unit. Sanitized evidence
persisted at `fixtures/captured/tplink-wr841n-v8/`:

- `login-page.html` — confirms `WWW-Authenticate: Basic realm="..."`,
  zero forms, zero inputs.
- `login-response.html` — confirms `Login Incorrect` body for the
  rejected md5hex recipe.
- `auth-evidence.json` — shows `"candidate": "basic-auth-plain-header"`,
  the recipe that matched.
- `captured-index.json` — shows `physical_fingerprint.match: true` with
  the expected firmware and hardware strings.

The probe verified 5 candidate recipes in order. Only the fifth
(`basic-auth-plain-header`) succeeded. The earlier recipes (md5hex +
cookie at `/userRpm/LoginRpm.htm`, md5hex + cookie at `/`, plain +
cookie at `/`) all failed with `HTTP 401 + Login Incorrect`.

## Prior art divergence

This recipe **diverges from public prior art**:

- `mkubicek/tpylink` (PA-1, no license) targets firmware `3.16.9` and
  uses md5hex base64 cookie at `/userRpm/LoginRpm.htm`. Does NOT apply
  to our build.
- `maesoser/tplink_exporter` (PA-2, GPL-3.0) uses the same md5hex
  base64 cookie at `/userRpm/LoginRpm.htm`. Does NOT apply to our
  build. (We deliberately did NOT copy any code from this project;
  observation is independent.)

The v8.4 firmware observable on our physical unit accepts Basic Auth
with plaintext password and does NOT require md5hex hashing.
Implementation must use `base64.StdEncoding.EncodeToString([]byte(user +
":" + password))` and NOT `base64(user + ":" + md5hex(password))`.

## Implementation notes

The recipe is implemented in `internal/transport/http.go` as
`Client.GetWithBasicAuth(ctx, rawURL, user, password)`. This method
reuses the existing transport safety invariants:

- GET only (`http.MethodGet` enforced in `dispatchWithHeader`).
- Loopback / RFC1918 host only (`IsAllowedHost` check).
- 2 MiB body cap (`maxBodySize`).
- Per-request timeout (`c.timeout`).
- Cross-host redirect rejection.

The adapter maintains the session state internally via
`Adapter.session`, populated by `Adapter.Login(ctx, user, password)`
and read by `Adapter.authedFetch`. The session lives only in process
memory; no persistence to disk in P3.

`Adapter.Status(ctx)` requires a session (returns
`ErrCaptureMissing`-wrapped "call Login first" error otherwise) and
uses `authedFetch` against `/userRpm/StatusRpm.htm`.

## What is verified

- `Adapter.Login(ctx, "admin", "<password>")` — verified against the
  physical lab unit on 2026-08-30.
- `Adapter.Status(ctx)` — verified end-to-end against the lab unit;
  returned `RouterStatus{Reachable: True, ...}` with the expected
  fingerprint parsed from the response body.

## What is NOT verified

- The exact session lifetime (browser-cache vs server-side). Browser
  reuses the Authorization across requests; the firmware does not appear
  to issue a Set-Cookie for the session.
- Other endpoints (DHCP, WPS, DMZ, UPnP, Remote Management,
  Forwarding). These remain `Verified: false` in
  `endpoints.go` until physical capture confirms each recipe.

## Future work

- ADR 0003 (capability + recipe authority) is still DRAFT and is the
  next architectural step. It does NOT apply to Phase 3 minimum.
- A OS keyring persistence (`Adapter.SessionFile`) is post-hackathon
  scope (ADR 0005 § future).