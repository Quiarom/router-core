# router-core

> **Local-first control plane for legacy consumer routers.**

Give an aging router dashboard a typed local API and an
evidence-aware AI agent — without replacing the firmware.

[![CI](https://github.com/Quiarom/router-core/actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)
[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![GMI Cloud × MiniMax Week](https://img.shields.io/badge/GMI_Cloud-MiniMax_Week-2026-08-24_→_2026-09-06-blue)](https://www.gmicloud.ai/minimax-week)

A real run against the lab unit (192.168.1.1, admin/admin):

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

[Demo video] · [Documentation](docs/) · [MiniMax-Week submission](docs/hackathon/minimax-week-2026.md)

---

## Install

The user-facing install is one command. No sudo, no system
packages, no dependencies.

```sh
curl -sSf https://raw.githubusercontent.com/Quiarom/router-core/integration/gavetero/install.sh | sh
```

This downloads the latest prebuilt `gavetero` binary for your
platform, installs it to `~/.local/bin/gavetero`, and creates
a `gvt` symlink. The script never requires sudo and never
modifies the system PATH; if `~/.local/bin` is not on your
PATH, it prints the one line to add to your shell rc.

After install:

```sh
gvt version       # confirm the binary
gvt setup         # store the GMI Cloud API key (one time)
gvt doctor        # confirm the install
gvt inspect       # see router observations (mock mode, no hardware)
gvt ask "..."     # ask a question with MiniMax M3
```

**For developers working in the repo:** use
`make install-user` (requires Go 1.22+). The user-facing
`install.sh` is for everyone else.

---

## Quickstart

Try it without hardware, in under a minute:

```sh
git clone https://github.com/Quiarom/router-core
cd router-core
go build -o ./bin/router-core ./cmd/router-core
./bin/router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8
```

You get the device identity, firmware fingerprint, and
authentication state parsed from a sanitized fixture — no
network, no router, no admin password.

Then against a real TP-Link WR841N v8.4 (firmware 3.15.9):

```sh
export GMI_SERVING_API_KEY="<jwt-key>"   # GMI Cloud Inference Engine
./bin/router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484 &
./bin/router-core-agent \
    --router-core-url http://127.0.0.1:8484 \
    --question "Is my Wi-Fi exposed?" \
    --model MiniMaxAI/MiniMax-M3
```

Or run the whole demo with one command:

```sh
./scripts/dev.sh --mock
```

---

## What you get

- **Typed local API.** 9 endpoints on `127.0.0.1:8484` exposing
  the router as JSON: device, status, clients, capabilities, and
  per-capability security observations. See
  [`docs/FRONTEND_CONTRACT.md`](docs/FRONTEND_CONTRACT.md).
- **MiniMax reasoning layer.** An HTTP server on
  `127.0.0.1:8585` that takes a question, calls the
  appropriate tools, and returns a structured Spanish audit with
  provenience and explicit evidence limits. See
  [`docs/PHASE5_AGENT_RUN.md`](docs/PHASE5_AGENT_RUN.md).
- **Honest capability states.** Each endpoint reports one of
  `verified`, `absent`, `unsupported_or_unverified`, or
  `unavailable`. The matrix is derived from a live probe, not a
  hardcoded map. The frontend never has to guess what is real.
- **Read-only by construction.** There is no `CapMutate` constant
  in the type system. An architecture test
  ([`internal/architecture_test.go`](internal/architecture_test.go))
  fails the build if any `POST`/`PUT`/`DELETE` shows up in the
  runtime. The agent's only POST goes to the LLM provider, never
  to the router.
- **Fixture replay.** Sanitized captures of the real lab unit
  live in [`fixtures/captured/tplink-wr841n-v8/`](fixtures/captured/tplink-wr841n-v8/).
  Every CI run is reproducible without hardware.

---

## Supported hardware

| Router | Firmware | Verified |
| --- | --- | --- |
| TP-Link TL-WR841N/ND v8.4 | 3.15.9 Build 140724 Rel.63227n | ✅ 2026-09-04, GMI Cloud direct |
| TP-Link TL-WR841N/ND v8.4 | 3.13.33 Build 130506 Rel.48660n | ✅ earlier capture, sanitized |

Other WR841N v8.x firmwares should work via the same Basic Auth
recipe. Other vendors (ASUS, MikroTik, Ubiquiti) are not yet
supported; the vendor-neutral `RouterAdapter` contract is ready
for new adapters.

---

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
     TP-Link WR841N v8.4 firmware (192.168.1.1)

  router-core-agent
    |
    | POST /v1/chat/completions (tool calls)
    v
  GMI Cloud Inference Engine
  (api.gmi-serving.com, MiniMaxAI/MiniMax-M3 primary, M2.7 fallback)
```

The runtime is strictly read-only. The agent's only network call
beyond the local router is to the LLM provider, with a real-time
fallback: if M3 returns a transient error (5xx, timeout, network
reset), the agent retries once with M2.7. OpenRouter is supported
as a drop-in fallback via `--openrouter-url`.

---

## Safety

- **GET only.** Every request to the router is a `GET`. The
  agent's only POST is to the LLM provider. Architecture test
  enforces this at the source level.
- **Loopback or RFC1918 only.** Public IPs and DNS hostnames are
  refused at every layer. The serve binds on `127.0.0.1`; the
  agent refuses `--serve 0.0.0.0`.
- **2 MiB response body cap.** Anything larger is truncated at
  the transport.
- **No cross-host redirects.**
- **In-memory session only.** The serve reads the admin
  password from `/dev/tty` with echo disabled, holds it only for
  the process lifetime, and zeros it on exit. The password lives
  in `[]byte`; the runtime overwrites the bytes before releasing
  the reference.
- **Capabilities cannot be invented.** Each capability is
  reported as one of `verified`, `absent`,
  `unsupported_or_unverified`, `unavailable`. The frontend never
  has to interpret; the matrix is honest.

---

## Tested on real hardware

TP-Link TL-WR841N/ND v8.4 — firmware 3.15.9 Build 140724
Rel.63227n.

The full evidence trail (auth recipe, capability matrix,
end-to-end traces for both M3 and M2.7 against the live unit)
is committed in
[`docs/EVIDENCE_TRACE.md`](docs/EVIDENCE_TRACE.md) and
[`fixtures/`](fixtures/).

---

## Documentation

- [Quickstart](docs/EVIDENCE_TRACE.md) — 5-minute setup with mock and
  real router paths.
- [Architecture](docs/STATUS.md) — three-layer design, the
  safety boundary, how the adapter contract works.
- [API reference](docs/FRONTEND_CONTRACT.md) — every endpoint, every state,
  every status code.
- [Agent](docs/PHASE5_AGENT_RUN.md) — how the reasoning layer calls the
  runtime, tool definition, prompt structure.
- [Adapter development](docs/adapters/tplink-wr841n.md) — how to
  add a new vendor adapter.
- [Evidence](docs/EVIDENCE_TRACE.md) — physical capture
  trail, prior-art comparison, security recipe divergence.
- [Demo](scripts/dev.sh) — reproducible end-to-end script (mock router, live agent).
- [Architecture decision records](docs/adr/) — every AD the
  project has committed to.
- [MiniMax-Week submission](docs/hackathon/minimax-week-2026.md)
  — judging context and submitted form.
- [Archive](docs/archive/) — historical context: the original
  Devin AI overnight pass, the Phase 5 design notes.

---

## Development

```sh
# Build
go build -o ./bin/router-core      ./cmd/router-core
go build -o ./bin/router-core-agent ./cmd/router-core-agent
go build -o ./bin/router-core-learn ./cmd/router-core-learn
cd frontend && npm install && npm run build && cd ..

# Test
go test ./... -race                  # 9/9 Go packages
cd frontend && npm test              # 11 frontend tests (5 contract + 6 integration)

# Format
gofmt -l .
```

The integration test runner lives at
`frontend/tests/integration.test.mjs`. The end-to-end demo lives
at `scripts/dev.sh --mock`.

## Contributing

Please read [`CONTRIBUTING.md`](.github/CONTRIBUTING.md) before
opening an issue or pull request. Conventions: Conventional
Commits, branch naming, PR rules, ADR-first for non-trivial
design decisions.

## License & acknowledgements

MIT. See [`LICENSE`](LICENSE). Third-party attribution
(including the original Devin AI overnight pass and prior art) in
[`NOTICE`](NOTICE). The model integration is provided by
[GMI Cloud](https://www.gmicloud.ai).
