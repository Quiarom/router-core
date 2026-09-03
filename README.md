# router-core

[![CI](https://github.com/Quiarom/router-core/actions/workflows/ci.yml/badge.svg)](https://github.com/Quiarom/router-core/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Quiarom/router-core.svg)](https://pkg.go.dev/github.com/Quiarom/router-core)
[![Go Report Card](https://goreportcard.com/badge/github.com/Quiarom/router-core)](https://goreportcard.com/report/github.com/Quiarom/router-core)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Ask your old router a question in plain language and get an answer
backed by evidence.** router-core reads a legacy TP-Link admin page,
exposes it as a typed JSON API on localhost, and lets a MiniMax agent
reason about it. It never writes to the router.

```text
router-core-agent: pregunta="Is my Wi-Fi exposed?"
router-core-agent: get_security -> HTTP 200

**Hechos observados (verified)**
- Wi-Fi **activado**.
- SSID visible: TP-LINK_CBEC16.
- Seguridad: WPA2-PSK (Cipher 332 = AES-CCMP en v8.4).
- Clave precompartida: configurada (no se expone el contenido).
- WPS / UPnP / Gestión remota: ausentes en este firmware.
- Firmware: 3.15.9 Build 140724 Rel.63227n (2014; EOL).

Recomendaciones
- Cambiar el SSID a uno neutro.
- Sustituir la PSK por una de al menos 12-16 caracteres aleatorios.
- Mantener WPS desactivado (ya lo está).
```

That is a real run against the lab unit (192.168.1.1). The agent answers
in Spanish by design; the question can be in either language. The full
trace is in [`fixtures/agent-traces/`](fixtures/agent-traces/).

## Try it in 60 seconds

No router, no API key, no network:

```sh
git clone https://github.com/Quiarom/router-core && cd router-core
make build

./bin/router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8
./bin/router-core serve --mock &
./bin/router-core-agent --dry-run --question "Is my Wi-Fi exposed?"
```

`probe` prints the device identity, firmware fingerprint and auth state
parsed from a sanitized fixture. `serve --mock` exposes the same JSON API
the live mode does, backed by fixtures. `--dry-run` runs the
deterministic offline agent against it, so you see the shape of an audit
without a model.

To get the dashboard too, `make dev` starts all three (API on :8484,
agent on :8585, Vite on :5173). You can also install the binaries
without cloning:

```sh
go install github.com/Quiarom/router-core/cmd/router-core@latest
go install github.com/Quiarom/router-core/cmd/router-core-agent@latest
```

## Against a real router

Works today with a TP-Link TL-WR841N/ND v8.4 (firmware 3.15.9).

```sh
export GMI_SERVING_API_KEY="<jwt-key>"        # GMI Cloud Inference Engine
router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484 &
router-core-agent \
    --router-core-url http://127.0.0.1:8484 \
    --question "Is my Wi-Fi exposed?" \
    --model MiniMaxAI/MiniMax-M3
```

`serve` asks for the admin password on the terminal with echo off, keeps
it in memory only, and zeros it on exit. Nothing is written to disk.

## Usage

```sh
router-core probe   --host 192.168.1.1          # identity + firmware + auth state
router-core inspect --host 192.168.1.1 --json   # full observation as JSON
router-core probe   --fixtures <dir>            # same, replayed from a fixture
router-core serve   --host 192.168.1.1          # JSON API on 127.0.0.1:8484
router-core serve   --mock                      # same API, fixture-backed

router-core-agent --question "..."              # one question, answer on stdout
router-core-agent --question -                  # read the question from stdin
router-core-agent --serve 127.0.0.1:8585        # HTTP mode: POST /v0/chat, GET /healthz
router-core-agent --dry-run                     # deterministic offline agent
router-core-agent --model-fallback MiniMaxAI/MiniMax-M2.7   # retry once on 5xx/timeout
router-core-agent --openrouter-url <url> --key-env <VAR>    # any OpenAI-compatible endpoint

make dev        # full stack on fixtures
make dev-live   # full stack against 192.168.1.1
```

The API has 11 endpoints: `/healthz`, `/v0/device`, `/v0/status`,
`/v0/clients`, `/v0/capabilities` and `/v0/security/{wps,dmz,upnp,
remote-management,forwarding,wireless}`. Every field is documented in
[`docs/FRONTEND_CONTRACT.md`](docs/FRONTEND_CONTRACT.md).

## How it works

```
Operator / dashboard (localhost:5173)
  |
  v
router-core-agent (127.0.0.1:8585)  ----POST /v1/chat/completions---->  GMI Cloud
  |                                                                     MiniMax-M3
  | GET /v0/device, /v0/status, /v0/capabilities                        (M2.7 fallback)
  | GET /v0/security/<name>   (one tool call per turn)
  v
router-core serve (127.0.0.1:8484)
  |
  | GET only, Basic Auth, 2 MiB cap, RFC1918 only
  v
TP-Link WR841N v8.4 firmware (192.168.1.1)
```

Every capability the API reports is in one of four states, derived from a
live probe rather than a hardcoded map:

| State | Meaning |
|---|---|
| `verified` | Observed on the lab unit and parsed into a real value. |
| `absent` | The device has no such surface on this firmware. |
| `unsupported_or_unverified` | The runtime is not wired to this surface yet; it refuses to guess. |
| `unavailable` | The runtime cannot satisfy the request right now (transport error, session expired). |

The agent sees the same four states. When it says "WPS is absent", that
is a parsed fact, not a model guess.

## Real runs

Every trace is committed verbatim (keys redacted) so you can compare
models and reproduce the run.

| Date | Question | Model | Provider | Trace |
|---|---|---|---|---|
| 2026-09-04 | Is my Wi-Fi exposed? | MiniMax-M3 | GMI Cloud direct | [md](fixtures/agent-traces/2026-09-04-wifi-exposed.gmi-serving.live.md) · [jsonl](fixtures/agent-traces/2026-09-04-wifi-exposed.gmi-serving.live.jsonl) |
| 2026-09-04 | Is my Wi-Fi exposed? | MiniMax-M2.7 | OpenRouter | [md](fixtures/agent-traces/2026-09-04-wifi-exposed.live.md) · [jsonl](fixtures/agent-traces/2026-09-04-wifi-exposed.live.jsonl) |

The M3 run names the cipher-332 ambiguity and the hardware EOL risk
explicitly; the M2.7 run does not. Both are one tool call, one answer.

## Supported hardware

| Router | Firmware | Verified |
|---|---|---|
| TP-Link TL-WR841N/ND v8.4 | 3.15.9 Build 140724 Rel.63227n | 2026-09-04, live, GMI Cloud direct |
| TP-Link TL-WR841N/ND v8.4 | 3.13.33 Build 130506 Rel.48660n | earlier capture, sanitized |

Other WR841N v8.x firmwares should work through the same Basic Auth
recipe. Other vendors are not supported yet; the vendor-neutral
`RouterAdapter` interface ([`internal/domain/`](internal/domain/)) is
the extension point. The WR841N recipe is documented in
[ADR 0005](docs/adr/0005-verified-wr841n-auth-recipe.md) and
[`docs/EVIDENCE_TRACE.md`](docs/EVIDENCE_TRACE.md).

## Safety

The runtime cannot change the router. This is enforced in the type
system and in CI, not by convention:

- There is no mutating capability constant. `CapMutate` does not exist.
- An architecture test ([`internal/architecture_test.go`](internal/architecture_test.go))
  fails the build if `POST`, `PUT` or `DELETE` appears in the runtime.
- Every request to the router is a `GET`. The agent's only `POST` goes to
  the LLM provider.
- Public IPs and DNS hostnames are refused at every layer. `serve` binds
  to `127.0.0.1`; the agent refuses `--serve 0.0.0.0`.
- Response bodies are capped at 2 MiB. Cross-host redirects are not
  followed.
- The admin password is read from the terminal with echo off, held in a
  `[]byte` for the process lifetime, and overwritten before release.

Report a vulnerability through [SECURITY.md](SECURITY.md).

## Documentation

| Document | What it covers |
|---|---|
| [Architecture](docs/SDD.md) | Three-layer design, adapter contract, safety boundary |
| [Project status](docs/STATUS.md) | What is verified, what is not, where it is going |
| [HTTP API](docs/FRONTEND_CONTRACT.md) | Every endpoint, state and status code |
| [Agent](docs/PHASE5_AGENT_RUN.md) | Reasoning loop, tool definitions, prompt, how to run it |
| [Evidence](docs/EVIDENCE_TRACE.md) | Capture trail, prior-art comparison, recipe divergence |
| [Prior art](docs/PRIOR_ART_PROTOCOL.md) | Protocol evidence from the WR841N family, unverified |
| [ADRs](docs/adr/) | Every architecture decision the project has committed to |
| [Demo](docs/demo/) | Reproducible end-to-end script |
| [Frontend](frontend/README.md) | React dashboard: build, env vars, tests |
| [Archive](docs/archive/) | Historical context and superseded notes |

<details>
<summary><strong>Development</strong></summary>

Requires Go 1.25+ and a recent Node.js for the frontend.

```sh
make build            # bin/router-core, bin/router-core-agent
make test             # go test ./...
go test ./... -race   # what CI runs
make vet && make fmt
cd frontend && npm install && npm test && npm run lint
```

Fixtures live in [`fixtures/`](fixtures/): `synthetic/` for parser
development, `captured/` for sanitized real captures, `frontend-mocks/`
for the dashboard, `agent-traces/` for recorded runs. Every CI run is
reproducible without hardware.

</details>

<details>
<summary><strong>Troubleshooting</strong></summary>

- **`login: authentication rejected`** with the right password: the
  firmware answers with a 68-byte "no authority" body when the request
  does not follow the verified recipe (token-prefixed path plus
  `Referer`). Make sure you are on a supported firmware; see
  [ADR 0005](docs/adr/0005-verified-wr841n-auth-recipe.md) and
  [`docs/EVIDENCE_TRACE.md`](docs/EVIDENCE_TRACE.md).
- **`unsupported_or_unverified` on a capability you know exists**: the
  endpoint has not been physically captured yet. Set
  `ROUTER_ALLOW_UNVERIFIED=1` only for local experiments, never in
  production.
- **Agent answers without calling any tool**: check that `serve` is
  reachable at `--router-core-url` and that `/healthz` returns 200.
- **`--serve 0.0.0.0` refused**: intended. The agent only binds to
  loopback.

</details>

## FAQ

**Can it change my router settings?**
No. There is no code path for it, and CI fails if one is added.

**Do I need an API key to try it?**
No. `--fixtures`, `--mock` and `--dry-run` run everything offline. You
need a GMI Cloud (or any OpenAI-compatible) key only for live model
answers.

**Why does the agent answer in Spanish?**
The prompt targets Spanish-speaking operators of legacy hardware. The
question can be in English or Spanish.

**Does it work with my router?**
Only the TP-Link WR841N v8.4 is verified. If you have another WR841N
firmware, a sanitized capture is the most useful contribution you can
make; see [`fixtures/captured/README.md`](fixtures/captured/README.md).

**Does it store anything?**
`probe`, `serve` and the agent write nothing to disk: the password lives
in process memory only, and there is no config file or cache. Only the
experimental `router-core-learn` writes sanitized captures, to the
directory you give it.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) first. Conventional Commits,
one concern per branch, ADR before any change to a safety invariant.
AI coding agents should also read [AGENTS.md](AGENTS.md).

The contributions that help most right now: sanitized captures of other
WR841N firmwares, and a second vendor adapter.

## License

MIT. See [LICENSE](LICENSE) and [NOTICE](NOTICE) for third-party
attribution. Model inference for the reference runs was provided by
[GMI Cloud](https://www.gmicloud.ai) during MiniMax Week 2026
([submission](docs/hackathon/minimax-week-2026.md)).
