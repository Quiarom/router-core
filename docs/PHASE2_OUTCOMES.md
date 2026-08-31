# Phase 2 outcomes — decision tree

Status: pre-written response protocol for the three possible outcomes of
`router-core learn` against the physical WR841N. Nothing here is
implemented; this document exists so that when the physical capture
result lands, the next move is unambiguous.

## Outcome A — Candidate A (PA-2 shape) matches

Evidence:

- `auth-evidence.json` shows `"candidate": "legacy-auth-a"`
- `captured-index.json` shows `fingerprint_match: true`
- `status-response.html` contains `3.13.33 Build 130506 Rel.48660n` and
  `WR841N v8 00000000` (or close enough — see Outcome B for divergences)

What this proves:

1. The PA-2 authentication recipe (cookie `Authorization = base64(user +
   ":" + md5hex(password))`, no URL encoding, no `Basic ` prefix) is the
   one this firmware build uses.
2. The structural session-token redirect shape (`/<16-char>/userRpm/Index.htm`)
   is correct.
3. The `statusPara` array block exists at the expected URL.
4. The firmware/hardware strings appear at the expected positions in
   `statusPara` (or close — see Outcome B for divergences).

What this does NOT prove:

- The other endpoint recipes (DHCP, WPS, DMZ, UPnP, etc.) are still
  unverified.
- The login path `/userRpm/LoginRpm.htm?Save=Save` is confirmed; the
  page-split assumption (DMZ vs Virtual Servers) is still open.
- The token alphabet is confirmed for A's specific match, but PA-2
  used lowercase base64; if A had failed and B had matched, the
  alphabet question would be open.

## Outcome A1 — A matches AND fingerprint matches exactly

```
found firmware    == expected firmware   (3.13.33 Build 130506 Rel.48660n)
found hardware    == expected hardware   (WR841N v8 00000000)
```

This is the happy path. The existing synthetic indices in
`internal/adapters/tplinkwr841v8/parse_status.go` are likely correct.
Phase 3 plan:

1. Implement `Adapter.authenticate` using candidate A's exact recipe.
   Use the captured session token in memory only; never persist it.
2. Add a regression test that replays the captured `auth-evidence.json`
   + `status-response.html` against the adapter and asserts the parsed
   `DeviceInfo` matches the expected fingerprint.
3. Mark `Endpoint{Op: OpStatus, …}.Verified = true` ONLY after the
   regression test passes.
4. Keep the recipe + capability split introduced for `learn` available
   for future endpoint captures.

## Outcome A2 — A matches BUT fingerprint diverges

```
found firmware    != expected firmware   (different version)
found hardware    != expected hardware   (different revision or empty)
```

The transport shape is right but the parser indices are wrong. Likely
causes:

- WR841N v8.4 firmware `3.13.33 Build 130506 Rel.48660n` lays out
  `statusPara` differently than the synthetic fixture.
- The firmware string format differs (e.g. `Rel.55978n` vs
  `Rel.48660n` — they have different trailing letters).
- The hardware string is empty or in a different field.

Phase 3 plan:

1. Inspect `status-response.html` (sanitized, fingerprint preserved).
   Find the actual `var statusPara = new Array(...)` block.
2. Adjust `parse_status.go` indices to the real positions.
3. Update the expected firmware/hardware constants in the probe to
   match the real values from the capture.
4. Re-run the regression test against the captured body.

## Outcome B — Neither candidate matches (protocol mismatch)

```
HTTP 401, HTTP 403, or 200 with no structural redirect
```

The PA-1 and PA-2 authentication recipes do not match this firmware
build. Likely causes:

- This build uses a completely different login mechanism (form POST,
  digest auth, etc.).
- The token alphabet is different (uppercase only, mixed case, longer
  than 16 chars).
- The router returns the login page even on the first request and the
  structural redirect never appears.
- The HTTP endpoint requires a Referer that we are not setting.

Phase 3 plan in this case:

1. **Stop the probe work.** Do NOT modify the adapter.
2. Re-read `login-response.html` from the capture. It contains the
   sanitized login page body. The structural data is still useful:
   form field names, action URL, any visible cookie/token shape.
3. Decide whether to:
   (a) build a third candidate recipe manually based on what the
       sanitized body shows, OR
   (b) implement the Playwright fallback in `tools/router-capture/`
       (TypeScript, headed Chromium) to capture the login exchange
       with the operator typing the password directly into the local
       browser, OR
   (c) re-examine the prior-art sources (PA-1, PA-2) for variants we
       missed and propose candidate C.
4. Whichever path is chosen, it is a NEW ADR — not a continuation of
   Phase 2.

The key rule: **do not iterate candidate C, D, E blindly.** The
"no brute force, no scanning" rule from the original prompt still
applies. One explicit decision per phase.

## Outcome C — Transport error (router unreachable, timeout)

```
[1/4] Reading admin password locally → OK
[2/4] Testing legacy authentication candidates
  transport error: …
```

The host did not respond within the timeout. Likely causes:

- Wrong IP (the Archer or something else).
- The workstation is not on the WR841N's subnet.
- The WR841N management port is not 80 (some firmwares move to 8080).
- WireGuard VPN interferes with local network traffic.

Phase 3 plan in this case:

1. Verify the IP via the WR841N's own sticker or DHCP client list.
2. Test with `--host <ip>:8080` if the port might be different.
3. Disable WireGuard temporarily and retry.
4. If still failing, this is not a probe problem; it's a network
   setup problem that needs operator action.

## Decision matrix

| Outcome | Next step | Risk |
| --- | --- | --- |
| A1 | Implement `Adapter.authenticate`, mark Status `Verified: true`, add regression test. | Low — evidence-backed. |
| A2 | Adjust parser indices, update expected constants, re-test. | Medium — still guessing within an evidence frame. |
| B | Decide: manual candidate C, Playwright fallback, or stop. | High — design decision required. |
| C | Operator verifies network setup. | Low — operational, not code. |

## What is NOT in any outcome

- Marking multiple endpoints `Verified: true` at once. Status only.
- Capturing DHCP / WPS / DMZ / UPnP / Forwarding pages. Out of Phase 2
  scope. Phase 3+.
- Changing `internal/transport` for the runtime adapter. Only the
  probe's transport is allowed to evolve in Phase 3.
- Removing the `ErrCaptureMissing` from `Adapter.authenticate`. It
  stays until the adapter's own authenticate path is implemented.
- Touching `internal/architecture_test.go`. Its current GET-only
  invariant stays; the capability-based redesign is Phase 4.