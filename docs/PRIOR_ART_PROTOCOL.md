# Prior-art protocol evidence — TP-Link TL-WR841N family

Status: research note. Treat as **candidate protocol evidence**, not as
verified behavior. Verification requires the sanitized physical capture
described in `BLOCKED_CAPTURE.md` against firmware
`3.13.33 Build 130506 Rel.48660n` on the physical lab unit at
`192.168.1.1`.

## 1. Scope of this document

This document records independent observations from two public
implementations that target the TP-Link TL-WR841N / TL-WR841ND family.

It does NOT:

- copy source code, functions, regular expressions, or implementation
  structure from either repository;
- promote any observation to `Verified: true` in the adapter;
- replace the physical capture as the source of truth.

It exists to:

- reduce uncertainty about Phase 2 by listing protocol hypotheses
  before we run anything against the lab unit;
- record the license boundary so future contributors do not import
  GPL code into this MIT-licensed project;
- justify the capability-based authority decision by referencing an
  independent mutation that travels over GET.

## 2. Sources

| ID | Repository | URL | License | Status |
| --- | --- | --- | --- | --- |
| PA-1 | `mkubicek/tpylink` | https://github.com/mkubicek/tpylink | **None declared** | Single-file Python script. Targets `TL-WR841N/TL-WR841ND (FW 3.16.9)`. Source commit inspected: `master`, single file `tpylink.py`. |
| PA-2 | `maesoser/tplink_exporter` | https://github.com/maesoser/tplink_exporter | **GPL-3.0** | Prometheus exporter written in Go. Targets the same family. Source commit inspected: `master`, `main.go` + `tplink/tplink.go` + `macdb/`. |

Both implementations are **read-only protocol literature for this
project**. The licensing situation rules out direct import:

- PA-2 is GPL-3.0. Importing its code into router-core would force the
  combined work under GPL, which contradicts the project's MIT target.
- PA-1 has no declared license. Default copyright applies; reuse would
  require explicit author permission.

Behavior may be reimplemented from the observations below, from our own
captured traffic, and from manufacturer documentation. No source code
is translated.

## 3. Protocol observations

Each row links to the source(s) that independently describe the
behavior. "Single source" rows are weaker hypotheses and flagged as
such.

### 3.1 Authentication exchange

| # | Observation | Source | Hypothesis strength |
| --- | --- | --- | --- |
| O-1 | The client sends an HTTP **GET** to `/userRpm/LoginRpm.htm?Save=Save` to perform the login. | PA-1 `LOGIN_URL` constant; PA-2 `LOGIN_URL` constant. | Strong (two independent sources). |
| O-2 | The login request carries an `Authorization` header whose value is `Basic <base64(user + ":" + md5hex(password))>`. | PA-1 `__init__` builds the cookie by `urllib.quote("Basic " + base64(user + ":" + md5hex(password)))`. PA-2 `NewRouter` builds `http.Cookie{Name: "Authorization", Value: base64(user + ":" + hashpass)}` with the same MD5-hex construction. | Strong (two independent sources, same construction). |
| O-3 | The username is `admin` by default. | PA-1 `__init__` default `username="admin"`. PA-2 `main.go` default `User="admin"` from flag/env. | Strong (factory default). |
| O-4 | Password hashing is plain MD5, hex-encoded, no salt, no HMAC, no iteration. | PA-1 `hashlib.md5().hexdigest()`. PA-2 `crypto/md5` + `hex.EncodeToString`. | Strong. **Security caveat**: this is the firmware's design, not our choice; we cannot upgrade it. |
| O-5 | The `Authorization` material is sent as an HTTP cookie named `Authorization` rather than as an `Authorization:` request header. | PA-1 sets `cookies={'Authorization': auth}`. PA-2 stores it as `http.Cookie{Name: "Authorization"}` and attaches via `req.AddCookie`. PA-1 also `urllib.quote`s the value (percent-encoding `Basic` and the base64). | Strong (two sources, one detail variant). |
| O-6 | A successful login response body contains a script that redirects to `http://<host>/<TOKEN>/userRpm/Index.htm`, where `<TOKEN>` matches `^[A-Z]{16}$`. | PA-1 `AUTH_KEY_RE = r"http\://[0-9A-Za-z.]+/([A-Z]{16})/userRpm/Index.htm"` and the docstring example shows a 16-uppercase-letter key. PA-2 `AUTH_KEY_RE = "[0-9A-Za-z.]+/([A-Z]{16})/userRpm/Index.htm"`. | Strong. |
| O-7 | The pre-login GET to `http://<host>/` is performed before the login GET, presumably to establish a session cookie context. | PA-1 performs `requests.get("http://{0}".format(self.host))` before the login GET. PA-2 does NOT show an equivalent step in `Login()` (it only does the login GET). | Single source (PA-1 only). Mark as **uncertain**. |

### 3.2 Authenticated requests

| # | Observation | Source | Hypothesis strength |
| --- | --- | --- | --- |
| O-8 | Authenticated requests prefix the URL path with the captured `<TOKEN>/`, e.g. `http://<host>/<TOKEN>/userRpm/StatusRpm.htm`. | PA-1 `STATUS_URL = "http://{0}/{1}/userRpm/StatusRpm.htm"`. PA-2 `r.Get("http://" + r.Address + "/" + r.Token + WAN_TRAFFIC_URL)`. | Strong. |
| O-9 | Authenticated requests re-attach the `Authorization` cookie and set `Referer` to the same URL being requested. | PA-1 `cookies=self.auth_cookie, headers={'referer': self.STATUS_URL.format(self.host, self.key)}`. PA-2 `req.AddCookie(&r.Cookie); req.Header.Set("Referer", url)`. | Strong. |
| O-10 | Endpoint paths (independent of the token prefix):<br>- Status: `/userRpm/StatusRpm.htm`<br>- DHCP client list: `/userRpm/AssignedIpAddrListRpm.htm`<br>- Statistics: `/userRpm/SystemStatisticRpm.htm?itnerval=10&Num_per_page=100`<br>- Logout: `/userRpm/LogoutRpm.htm`<br>- Reboot: `/userRpm/SysRebootRpm.htm?Reboot=Reboot` | PA-2 constants `WAN_TRAFFIC_URL`, `CLIENTS_URL`, `STATS_URL`, `LOGOUT_URL`, `REBOOT_URL`. PA-1 confirms `STATUS_URL`, `LOGOUT_URL`, `REBOOT_URL`. | Strong for the four paths both sources name. Statistics query string and DHCP path are PA-2 only but PA-2's code shows them as runtime constants. |
| O-11 | Status HTML contains a `var statistList = new Array("rx", "tx");` block carrying WAN traffic counters as comma-grouped decimals. | PA-2 `GetWANTraffic` regex `var statistList = new Array\(\n\"([^\"]*)\", \"([^\"]*)`. | Single source for the exact regex; PA-1 confirms a `traffic_re` with similar shape but is less specific. Mark as **candidate**. |

### 3.3 DHCP client list

| # | Observation | Source | Hypothesis strength |
| --- | --- | --- | --- |
| O-12 | The DHCP clients page is `/userRpm/AssignedIpAddrListRpm.htm`. | PA-2 `CLIENTS_URL`. | Single source (PA-2). Mark as **candidate**; our synthetic fixtures use the same name in `endpoints.go`. |
| O-13 | Each client row in the page is a 4-tuple string group: `name`, `MAC`, `IP`, `lease`. | PA-2 `updateWirelessClients` parses `\"...\", \"...\", \"...\", \"...\"` groups and assigns them to `Name`, `MACAddr`, `IPAddr`, `DHCPLease` in that order. | Single source. **Matches** the synthetic `dhcpGroupSize = 4` constant in `internal/adapters/tplinkwr841v8/parse_dhcp.go`. |
| O-14 | DHCP lease strings look like `HH:MM:SS` or `Permanent`. | PA-2 `parseLease` splits on `:` and treats `Permanent` specially. | Single source. Confirms the lease shape. |

### 3.4 Reboot — critical architectural evidence

| # | Observation | Source | Hypothesis strength |
| --- | --- | --- | --- |
| O-15 | The router reboots when the firmware receives a **GET** request to `/<TOKEN>/userRpm/SysRebootRpm.htm?Reboot=Reboot`. | PA-1 `reboot()` does `requests.get(self.REBOOT_URL.format(...) + "?Reboot=Reboot", ...)`. PA-2 `Reboot()` does `r.Get("http://" + r.Address + "/" + r.Token + REBOOT_URL)` where `REBOOT_URL = "/userRpm/SysRebootRpm.htm?Reboot=Reboot"`. | Strong (two independent sources). |

This is decisive for the capability-based authority decision. **HTTP
method is not a safety boundary on this firmware family.** A read-only
proxy that only allows GET would still happily trigger a router reboot
because the mutation is encoded in the URL path. The `architecture_test.go`
cannot be a method-text scan; it must be a capability-type scan, or
better, the type system must make `CapMutate` literally unavailable
to runtime paths.

### 3.5 Connection behavior

| # | Observation | Source | Hypothesis strength |
| --- | --- | --- | --- |
| O-16 | Conservative client timeouts in the order of seconds are sufficient. | PA-2 `http.Client{Timeout: time.Second * 2}`. PA-1 uses default `requests` timeouts. | Single source for the exact number. Our transport default of 5s is more conservative and acceptable. |
| O-17 | The login and authenticated requests are plain HTTP. No TLS. | Both sources use `http://` URLs throughout. | Strong. |

## 4. Firmware / device scope

The two implementations target different firmware revisions and
device variants than the physical lab unit:

- PA-1 docstring: `TL-WR841N/TL-WR841ND (FW 3.16.9)`.
- PA-2 README (from source inspection): targets the same family, no
  pinned firmware.
- Physical lab unit (this project): `TP-Link TL-WR841N v8.4`, firmware
  `3.13.33 Build 130506 Rel.48660n`.

Firmware behavior on TP-Link legacy devices frequently changes between
revisions: login form factor, cookie names, token alphabet, and page
paths can all shift. **The hypotheses above become `Verified` only when
they are reproduced against the physical lab unit's actual firmware.**
Nothing in this document is sufficient to flip any `Verified` flag in
`internal/adapters/tplinkwr841v8/endpoints.go`.

## 5. What the prior art does NOT confirm

- The exact **login response shape** (script redirect vs. HTTP redirect,
  status code, headers).
- Whether our specific firmware uses a cookie or an `Authorization:`
  header. PA-1 and PA-2 disagree subtly (PA-1 percent-encodes `Basic
  `, PA-2 does not). Either may apply; the physical capture decides.
- Whether the 16-character token is always uppercase letters, or
  includes digits (PA-2's regex allows both). **Our `AUTH_KEY_RE`
  should accept both until the physical capture pins the alphabet.**
- The order and meaning of fields inside `statusPara`, `DHCPDynList`,
  `wpsPara`, `dmzPara`, `upnpPara`, `remotePara`, `virtualServerPara`.
  Our synthetic fixtures' indices (e.g. `statusFirmwareIndex=6`,
  `statusHardwareIndex=7`) remain **unverified**.
- Whether the WR841N v8.4 stock firmware accepts a single GET to
  `LoginRpm.htm?Save=Save` or requires additional steps
  (CSRF/nonce/Referer). PA-1 shows an extra GET to `/` before the
  login GET (O-7). PA-2 does not.
- Whether the device responds with HTTP 200 for unauthenticated access
  to a userRpm page or with a 302 to `/`. This affects parser error
  detection.

## 6. Capability-based authority: evidence-backed redesign

O-15 is sufficient evidence on its own to retire the GET-only safety
invariant. The replacement:

- `internal/transport.Capability` enumerates `CapAuth`, `CapObserve`.
  `CapMutate` is **not defined yet**.
- `Client.Dispatch(ctx, cap, method, url, body, headers)` admits any
  HTTP method declared by the caller, restricted by the capability
  argument.
- `Adapter.authenticate` dispatches with `CapAuth`. `Adapter.fetch`
  dispatches with `CapObserve`. There is no path in runtime code that
  can construct a `CapMutate` value because the constant does not exist
  in any package runtime code imports.
- Phase 6 introduces `CapMutate` through a new ADR together with
  policy, approval, verification, and rollback. Until then the type
  system makes mutation unrepresentable in runtime.

The host restrictions, redirect restrictions, body cap, and timeouts
from the current transport are preserved.

## 7. License warning

- PA-2 is GPL-3.0. **Do not import, vendor, or translate source code
  from this repository into router-core.** Any code derived from this
  document must be written from the observations above plus the
  physical capture.
- PA-1 has no declared license. Default copyright applies. The
  observations here are facts about a protocol, not copyrightable
  expression, but reproducing the structure of the script verbatim
  would still be risky. Reimplement from observations.
- If future contributors want to cite additional public
  implementations, the same rule applies: read, observe, reimplement.

## 8. Candidate vs verified distinction

| Layer | State |
| --- | --- |
| Hypotheses O-1 through O-17 above | **Candidate** |
| Adapter endpoint entries in `endpoints.go` | `Verified: false` (unchanged) |
| Authentication implementation | **Not implemented**, returns `ErrCaptureMissing` (unchanged) |
| Status parser indices (`statusFirmwareIndex`, etc.) | **Candidate**, marked `UNVERIFIED` in source comments (unchanged) |
| Host restriction, body cap, redirect rules, GET-only default | **Active**, preserved through the capability refactor |
| Boot-time transport-level GET-only safety invariant | **Retired** by ADR 0003 in favor of capability-based authority |

The transition from candidate to verified happens only in Phase 3,
after the physical capture in `fixtures/captured/tplink-wr841n-v8/`
reproduces each hypothesis against the lab unit. Each flip must be
recorded in its own ADR section with the captured evidence attached.