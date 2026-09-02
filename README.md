# router-core

> **Your old router is still alive. This gives it a brain.**

[![CI](https://github.com/Quiarom/router-core/actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Quiarom/router-core)](https://goreportcard.com/report/github.com/Quiarom/router-core)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](go.mod)
[![GMI Cloud × MiniMax Week](https://img.shields.io/badge/GMI_Cloud-MiniMax_Week-2026-08-24_→_2026-09-06-blue)](https://www.gmicloud.ai/minimax-week)

A small Go program that talks to a real consumer router, reads its
dashboard, and answers plain-language questions about it — without
changing a single setting. A MiniMax reasoning layer asks the
right questions and tells you what's safe, what's not, and what
the evidence actually says.

Submission for the **GMI Cloud × MiniMax Week** (track:
**Reasoning**). Runs on MiniMax M3 (primary) and M2.7 (fallback),
served by [GMI Cloud](https://www.gmicloud.ai) via OpenRouter.

---

## What is this, in plain language

You probably have a router that's been working fine for ten
years. The web UI still works, but it was never good. The
firmware doesn't get security updates. There's no API to talk to
it programmatically.

`router-core` changes that. It runs on your computer, talks to
your router over the local network, parses the same pages the
web UI parses, and gives you a clean HTTP API. Then a small AI
agent — running on the same machine — answers questions like
"is my Wi-Fi exposed?" with evidence and limits.

It's read-only. The runtime cannot change a setting, cannot
reboot, cannot factory-reset. The mutation surface is
**unrepresentable in the type system**: there is no
`CapMutate` constant in any package the runtime imports. An
architecture test enforces this at the source level.

If the AI gets confused, the worst it can do is read. If the
network gets hostile, the worst it can do is read. The router
itself is untouched.

## Why it exists

Stock consumer-router dashboards from 2013 are not getting
maintained. The web UI is missing features, looks bad, and
assumes a user willing to click through six pages to answer
"is my network exposed?" `router-core` turns the dashboard into
a typed API and lets a reasoning model answer that question in
plain language — without you ever touching a login form.

The design is informed by the prior art (see
[`docs/PRIOR_ART_PROTOCOL.md`](docs/PRIOR_ART_PROTOCOL.md)) but
the recipe was discovered by physical capture against the lab
unit, not by copying. The full protocol-evidence trail is in
[`docs/EVIDENCE_TRACE.md`](docs/EVIDENCE_TRACE.md).

## Try it in 5 minutes (no router needed)

The fastest way to see it work is with the bundled mock
fixtures. No physical router, no admin password, no real
network. Just the binaries and a one-liner.

```sh
git clone https://github.com/Quiarom/router-core.git
cd router-core
go build -o ./bin/router-core ./cmd/router-core

# Probe a synthetic fixture (no hardware required)
./bin/router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8
```

You'll see the parsed device identity, the firmware fingerprint,
and the authentication state — all from a sanitized HTML fixture
that looks like the real firmware dashboard.

For the agent loop, the live OpenRouter trace is committed as a
JSONL fixture under
[`fixtures/agent-traces/`](fixtures/agent-traces/). The
agent ran in dry-run and live modes against the real lab unit;
both fixtures are checked in, sanitized, and ready to replay.

## What you get

Eleven typed HTTP endpoints on loopback (`127.0.0.1:8484` by
default). All read-only. All return JSON.

| Endpoint | Returns |
| --- | --- |
| `GET /healthz` | `{"state":"ok"}` — liveness probe |
| `GET /v0/device` | vendor, model, hardware, firmware, management address |
| `GET /v0/status` | reachable, WAN state, uptime |
| `GET /v0/capabilities` | the full capability matrix in one call |
| `GET /v0/clients` | DHCP leases (MAC, IP, name, lease time) |
| `GET /v0/security/wireless` | SSID, security type, cipher, key rotation, PSK presence |
| `GET /v0/security/{wps,dmz,upnp,forwarding,remote-management}` | per-capability observation |

Four documented states for security capabilities:
**`verified`**, **`absent`**, **`unsupported_or_unverified`**,
**`unavailable`**. The runtime never collapses these to
`true`/`false`; the agent and the dashboard render each
honestly. Full shape: [`docs/FRONTEND_CONTRACT.md`](docs/FRONTEND_CONTRACT.md).

## How it works

```
Operator
  -> router-core-agent
       |
       | GET /v0/device, /v0/status, /v0/capabilities, /v0/clients
       v
     router-core serve (loopback :8484)
       |
       | GET /v0/security/<name>  (one tool call per turn)
       v
     TP-Link TL-WR841N v8.4 firmware (192.168.1.1)

  router-core-agent
    |
    | POST /v1/chat/completions (tool calls)
    v
  OpenRouter (gateway)
    |
    v
  MiniMax M2.7 / M3  (provider: GMICloud)
```

The agent collects the device identity, status, capabilities, and
clients, builds a system prompt, and runs a tool loop. Each
tool call hits a `/v0/security/<name>` endpoint. When the model
emits a final answer, the agent prints it. See
[`docs/PHASE5_AGENT_RUN.md`](docs/PHASE5_AGENT_RUN.md) for the
reproducible run playbook and
[`fixtures/agent-traces/2026-09-04-wifi-exposed.live.md`](fixtures/agent-traces/2026-09-04-wifi-exposed.live.md)
for a real captured trace.

A live example (MiniMax M2.7, captured 2026-09-04, lab unit
at `192.168.1.1`):

> **Question:** Is my Wi-Fi exposed?
>
> Tu red Wi-Fi **no está abierta**, pero tiene configuraciones que
> conviene revisar. Seguridad: WPA2-PSK, aceptable para uso
> doméstico. Clave pre-compartida: presente, no es abierta. SSID:
> TP-LINK_CBEC16, visible para cualquiera que escanee redes.
>
> Recomendación: cambia la contraseña WPA2-PSK si hace más de un
> año que no lo haces. No puedo verificar WPS (esta consulta no
> está disponible), pero si tu router lo tiene activado,
> desactívalo desde la interfaz web en `192.168.1.1` — es una
> vulnerabilidad conocida.

The model names its uncertainty ("no puedo ver la contraseña")
and asks a follow-up about port/UPnP. The agent reports
unavailable observations as unavailable, not as "ok". The
frontend gets the same structured `agentResult` with `Steps`
so it can render each tool call as a step in the UI.

## Safety

- **GET only.** `internal/transport.Client.Dispatch` rejects every
  other method. The agent's only POST is to OpenRouter, not to
  the router. The architecture test
  [`internal/architecture_test.go`](internal/architecture_test.go)
  enforces this at the source level.
- **Loopback or RFC1918 only.** Public IPs and DNS hostnames are
  refused at every layer. The serve binds on `127.0.0.1`; the
  agent refuses `--serve 0.0.0.0` before any listener is created.
- **2 MiB response body cap.** Anything larger is truncated at
  the transport.
- **No cross-host redirects.**
- **In-memory session only.** `router-core serve` reads the admin
  password from `/dev/tty` with echo disabled, holds it only for
  the process lifetime, and zeroes it on exit.
- **Mutations are unrepresentable.** There is no `CapMutate`
  constant in any package runtime code imports. Any future
  mutation path requires a new ADR, capability constant, and
  explicit operator approval.

The full safety story is in
[`docs/adr/0003-draft-capability-authority.md`](docs/adr/0003-draft-capability-authority.md)
and the auth recipe is documented in
[`docs/adr/0005-verified-wr841n-auth-recipe.md`](docs/adr/0005-verified-wr841n-auth-recipe.md).

## Live evidence

Every claim in this README is backed by a sanitized capture
from the lab unit at `192.168.1.1` (TP-Link TL-WR841N v8.4,
firmware 3.15.9 Build 140724 Rel.63227n):

```sh
$ router-core probe --host 192.168.1.1
TP-Link TL-WR841N/ND
Hardware: WR841N v8 00000000
Firmware: 3.15.9 Build 140724 Rel.63227n
Host: 192.168.1.1
Authentication: success
```

`fixtures/captured/tplink-wr841n-v8/` holds 17 sanitized captures
from that lab unit, including the wireless-security dashboard
the parser decodes. The capture dates and fingerprints are
auditable in `capability-evidence.json` and `captured-index.json`.

## For engineers

If you want to read the technical depth, the docs are linked
below. If you want to contribute, start with
[`CONTRIBUTING.md`](CONTRIBUTING.md). If you are an AI coding
agent, start with [`AGENTS.md`](AGENTS.md).

| Document | What's in it |
| --- | --- |
| [`docs/FRONTEND_CONTRACT.md`](docs/FRONTEND_CONTRACT.md) | the typed HTTP surface, JSON shapes, four-state semantics |
| [`docs/PHASE5_AGENT_RUN.md`](docs/PHASE5_AGENT_RUN.md) | how to run the agent in dry-run and live modes |
| [`docs/EVIDENCE_TRACE.md`](docs/EVIDENCE_TRACE.md) | the runtime-vs-physical-capture trace with the four-column table |
| [`docs/PRIOR_ART_PROTOCOL.md`](docs/PRIOR_ART_PROTOCOL.md) | the prior-art comparison and license boundary |
| [`docs/STATUS.md`](docs/STATUS.md) | non-engineer project status |
| [`docs/adr/`](docs/adr/) | architecture decision records (0001, 0002, 0003, 0005) |
| [`NOTICE`](NOTICE) | third-party attribution (Devin AI baseline, prior art) |
| [`HACKATHON_FAQ.md`](HACKATHON_FAQ.md) | MiniMax-Week judging context |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | branching model, commit conventions, PR rules |
| [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) | Contributor Covenant v2.1 |
| [`SECURITY.md`](SECURITY.md) | private vulnerability disclosure |
| [`AGENTS.md`](AGENTS.md) | conventions and hard rules for AI coding agents |
| [`CHANGELOG.md`](CHANGELOG.md) | Keep-a-Changelog 1.1.0 history |
| [`LICENSE`](LICENSE) | MIT |

## Installation (for engineers)

Requires **Go 1.25+**, a POSIX shell, and (for the live path)
a TP-Link TL-WR841N/ND v8.4 on the local network. Synthetic
fixtures work without hardware.

```sh
go build -o ./bin/router-core      ./cmd/router-core
go build -o ./bin/router-core-learn ./cmd/router-core-learn
go build -o ./bin/router-core-agent ./cmd/router-core-agent

# Probe a synthetic fixture
./bin/router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8

# Probe the live router
./bin/router-core probe --host 192.168.1.1

# Serve the typed HTTP API on loopback
echo 'admin' | ./bin/router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484

# Run the agent in dry-run mode
./bin/router-core-agent \
    --router-core-url http://127.0.0.1:8484 \
    --question 'Is my Wi-Fi exposed?' \
    --dry-run

# Run the agent in live mode (requires OPENROUTER_API_KEY)
export OPENROUTER_API_KEY="sk-or-v1-..."
./bin/router-core-agent \
    --router-core-url http://127.0.0.1:8484 \
    --question 'Is my Wi-Fi exposed?' \
    --model minimax/minimax-m2.7:free
```

## Quick reference

| Command | Purpose |
| --- | --- |
| `router-core probe` | Talk to the live router (or a fixture) and print the device identity. |
| `router-core inspect` | Print the parsed status, security, and clients as a single document. |
| `router-core serve` | Expose the typed HTTP API on the loopback interface. |
| `router-core-learn learn` | Run the 5-recipe auth probe against the physical unit; write sanitized evidence. |
| `router-core-learn observe` | Run the per-capability observation matrix and update `capability-evidence.json`. |
| `router-core-agent` | The MiniMax reasoning layer. Dry-run by default; live with `OPENROUTER_API_KEY`. |
| `router-core-agent --serve :8585` | Same agent, exposed as `POST /v0/chat` for the frontend. |

## Tests

```sh
go test ./... -race                # all 8 Go packages
cd frontend && npm test            # 5 frontend contract tests (node --test)
go test ./internal -run TestSourceContainsNoMutatingHTTPCalls
# 8 Go test packages green
# 5 frontend contract tests pass
# gofmt clean
# sensitive scan clean
# live CLI verified on 192.168.1.1 with admin/admin
```

## Known limitations

- `/v0/security/wireless` returns `503 unavailable` (parser is
  still a placeholder). The endpoint is reachable on v8.4 (verified
  2026-08-31) but the live parser is pending.
- The v8.4 firmware at the lab unit returns HTTP 501 for WPS,
  UPnP, and Remote Management. These are correctly reported
  as `absent` in the capability matrix. The runtime does not
  pretend the surface is there.
- DMZ and Forwarding are also 501 on this firmware. The
  capability matrix currently lists them as `verified`; that
  is a small inconsistency the live trace reveals and is on the
  to-fix list.

## Roadmap

- **Phase 6 — One Safe Write.** Add the first mutation
  (for example, disable UPnP) gated by policy, verification,
  and explicit human approval. Requires ADR 0003 (capability
  authority) to move from DRAFT to active.
- **Per-firmware session-token fetch.** Implement the production
  path that reads the v8.4 session token from
  `/LoginRpm.htm?Save=Save` and forwards it to
  `authedFetchWithFallback`.
- **Wire the wireless-security parser.** The endpoint is
  reachable (verified 2026-08-31) but the runtime currently
  returns 503 with a clear reason.
- **Frontend reference implementation.** A React dashboard
  under `frontend/` (in progress by a third-party contributor)
  that talks to `serve` and to a MiniMax model through
  OpenRouter.
- **More adapters.** The vendor-neutral `RouterAdapter`
  contract is ready for an ASUS, MikroTik, or Ubiquiti
  implementation.

## Attribution

The baseline implementation was produced as an overnight
autonomous pass by **Devin AI** ([Cognition Labs](https://www.cognition.ai)).
The post-Phase-0 work (verified auth recipe, runtime, agent,
frontend, OSS team files) was produced by the router-core author
and collaborators. Two public prior-art implementations were
studied as research only; no code was imported. See
[`NOTICE`](NOTICE) for the full attribution and
[`docs/PRIOR_ART_PROTOCOL.md`](docs/PRIOR_ART_PROTOCOL.md) for
the per-observation comparison.

## License

MIT. See [`LICENSE`](LICENSE).
