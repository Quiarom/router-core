# Evidence trace: runtime adapter vs. physical capture

Date: 2026-08-31. Author: the post-Phase-0 work. Goal: make the
normal runtime adapter reproduce the physically verified
learning result before the M3 reasoning loop is wired on top.

## Sources read

- `fixtures/captured/tplink-wr841n-v8/auth-evidence.json`
- `fixtures/captured/tplink-wr841n-v8/captured-index.json`
- `fixtures/captured/tplink-wr841n-v8/status-request.json`
- `fixtures/captured/tplink-wr841n-v8/capability-evidence.json`
- `docs/adr/0005-verified-wr841n-auth-recipe.md`
- `cmd/router-core-learn/learn.go` (candidates, tryCandidate, fetchStatus, fetchStatusBasic)
- `internal/adapters/tplinkwr841v8/adapter.go` (Login, Identify, Status, authedFetchWithFallback)
- `internal/transport/http.go` (GetWithBasicAuth, IsAllowedHost)

## Four-column table

| Aspect | Physical evidence | Learning implementation | Runtime adapter | Mismatch |
| --- | --- | --- | --- | --- |
| Login path | `/userRpm/LoginRpm.htm?Save=Save` (`auth-evidence.json:path`) | tries 5 candidates including LoginRpm.htm and `/` | `GET /` only (`Adapter.Login`, `adapter.go:50-67`) | runtime skips the path that returns the session token |
| Auth material | `Authorization: Basic <base64(admin:plaintext)>` HTTP header, plaintext, NOT md5hex (`capability-evidence.json:authentication`) | tries cookie and header variants; the captured one is header with plaintext | `Authorization` HTTP header with plaintext via `transport.GetWithBasicAuth` (`adapter.go:51`, `transport/http.go:131`) | matches |
| Session token | response contains redirect shape `/<TOKEN>/userRpm/Index.htm` (`auth-evidence.json:redirect_shape`); token used in subsequent paths | `tryCandidate` parses the redirect body with `tokenPattern` and stores the token | `Login` only stores user+password; `sessionToken` field stays `""` | runtime never extracts the token because it never hits LoginRpm.htm |
| Status path | `/<TOKEN>/userRpm/StatusRpm.htm` (`status-request.json:path`, 25 760 bytes of real dashboard) | after Basic Auth candidate match, `fetchStatusBasic` against bare `/userRpm/StatusRpm.htm` | `authedFetchWithFallback` against bare `/userRpm/StatusRpm.htm`; fallback only retries if `sessionToken != ""` | runtime hits bare path → 68 bytes "no authority" → no token to retry with |
| `Referer` | `http://192.168.1.1/<TOKEN>/userRpm/StatusRpm.htm` (`status-request.json:headers.Referer`) | `fetchStatus` sets `Referer: u.String()` (the token-prefixed URL) | `transport.GetWithBasicAuth` does not set a `Referer` | runtime misses the Referer; not the primary cause, but worth matching |
| Identify | not in the capture; expected via Status body parse | not applicable; learn/observe use the Status path | `transport.Get(/)` (no auth) → parses the login page body | runtime reads the unauthenticated login page; firmware/hardware are empty |

## Settled questions

1. **Header vs cookie.** Header. The physical capture uses the
   `Authorization: Basic <base64(admin:plaintext)>` HTTP header
   (see `capability-evidence.json:authentication.header_not_cookie: true`
   and `auth-evidence.json` field rename history). The runtime
   adapter is already on the header path.
2. **Exact login path.** `/userRpm/LoginRpm.htm?Save=Save`. The
   runtime adapter currently does `GET /` and skips the
   redirect. The fix is to do the LoginRpm path with Basic Auth
   header and extract the redirect token.
3. **Does the login response contain the session token on this
   firmware?** Yes. The redirect shape `/<TOKEN>/userRpm/Index.htm`
   is observed in the captured response body. The `auth-evidence.json`
   field is the persistence of that observation.
4. **Does Status require the token URL prefix?** Yes for the
   v8.4 firmware. The 25 760-byte capture was fetched with the
   prefix; the bare-path response is 68 bytes of "no authority".
5. **Which request shape physically succeeded?** The
   `status-request.json` shape: `GET /<TOKEN>/userRpm/StatusRpm.htm`
   with `Authorization: Basic <base64(admin:plaintext)>` header and
   `Referer: http://<host>/<TOKEN>/userRpm/StatusRpm.htm`.
6. **Why does `probe` lose hardware/firmware?** `Identify` calls
   the transport's `Get` (no auth) against `/`. The body is the
   login page, not the dashboard. The `ParseIdentity` parser
   finds no fingerprint and returns empty strings. Fix: Identify
   must call `Login` (or use the active session) and then read
   the Status body to extract the fingerprint.

## Fix scope (per the operator's directive)

- `authenticate`: do the LoginRpm path with Basic Auth header,
  extract the session token from the redirect shape with a regex,
  store both on the session. Discard the old `GET /` Basic Auth
  path against `/` for normal login; keep it for the dev/test
  fallback only when the LoginRpm path fails to return a token.
- `identify`: after `Login`, fetch the Status body via the
  token-prefixed path, run `ParseIdentity` against it, populate
  the `DeviceInfo` fields.
- `status`: use the token-prefixed path, run `ParseStatus`,
  return the typed result.

Out of scope (deferred):

- WPS, DMZ, UPnP, clients, forwarding.
- M3 reasoning loop.
- Frontend work.
- Additional docs.

## Verification gate

```sh
# unit tests
go test ./... -race

# live physical verification
./bin/router-core probe --host 192.168.1.1
# expected:
#   Vendor: TP-Link
#   Model: TL-WR841N/ND
#   Hardware: WR841N v8 00000000
#   Firmware: 3.13.33 Build 130506 Rel.48660n
#   Authentication: success
#
./bin/router-core serve --host 192.168.1.1
# in another shell:
curl -s http://127.0.0.1:8484/v0/device   # 200 with full fingerprint
curl -s http://127.0.0.1:8484/v0/status   # 200 with normalized status
```

When the gate passes, mark:

- `PHASE 3 — VERIFIED PROTOCOL EVIDENCE` (already true) and
  `PHASE 3B — VERIFIED RUNTIME ADAPTER` complete.
- `PHASE 4 — LOCAL SERVICE` baseline complete.

When M3 reasoning loop and frontend are wired, mark Phase 5
and Phase 7.
