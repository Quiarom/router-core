# Security policy

`router-core` runs a local HTTP service on a loopback address
(`127.0.0.1:8484` by default) and talks to a TP-Link-style
consumer router over the local network. It is read-only: the
runtime has no way to mutate the router, the transport layer
rejects non-GET methods, and the mutation surface is
unrepresentable in the type system.

This document covers how to report a vulnerability in this
project. It does not cover the router firmware itself; report
firmware bugs to the vendor.

## Supported versions

| Version | Supported |
| --- | --- |
| `main` branch | yes |
| older tags | no |

We backport fixes to `main`. The release cadence follows
[Semantic Versioning](https://semver.org/).

## Reporting a vulnerability

Please **do not** open a public GitHub issue for a security
problem.

Instead, open a **private security advisory** on GitHub:

1. Go to <https://github.com/Quiarom/router-core/security/advisories/new>.
2. Title it clearly, e.g. `runtime: Authenticated response parser overflow in tplinkwr841v8.ParseIdentity`.
3. Include:
   - The exact commit or tag where you reproduced the issue.
   - A minimal reproduction (a `router-core` invocation, a
     crafted HTTP body, or a test that demonstrates the bug).
   - The observed vs. expected behavior.
   - Your assessment of the impact (e.g. local DoS, RCE on
     the operator's machine, information disclosure).
4. We aim to acknowledge within **72 hours** and to publish a
   fix or a workaround within **14 days** for any issue we
   confirm as in scope.

If you cannot use GitHub's advisory flow, email the maintainer
whose handle is in the latest `git log` author line. Do not
include the exploit payload in the subject line.

## Out of scope

- Vulnerabilities in the TP-Link TL-WR841N v8.4 firmware.
  Report those to TP-Link.
- Vulnerabilities in upstream Go modules. Report those to the
  module author and to [go-vulndb](https://github.com/golang/vulndb).
- The host machine's general security posture (your laptop,
  your network, your ISP). `router-core` only sees the local
  loopback and the local router.

## Hardening guidance for operators

The runtime is safe by default, but the operator's choices
affect the attack surface:

- Always start `serve` with a loopback `--addr` (`127.0.0.1:...`).
  Do not bind to `0.0.0.0`.
- Use a strong admin password on the router. The runtime
  forwards it as Basic Auth and the password is in process
  memory only.
- Do not set `ROUTER_ALLOW_UNVERIFIED=1` in production. It
  permits requests to endpoints that have not been physically
  captured.
- If you operate over a hostile local network (e.g. a coffee
  shop), connect through a VPN before running the live probe.
- The frontend (separate project) MUST talk to the runtime
  over loopback. A frontend that proxies the runtime's HTTP
  surface to a non-loopback address breaks the safety
  invariant.

## Hall of fame

We credit reporters (with their consent) in the release notes
of the fix. If you would prefer to stay anonymous, say so in
the advisory and we will respect it.
