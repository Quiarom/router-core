# Phase 5 — Agent run playbook

This document is the reproducible run path for the MiniMax M3
reasoning layer over router-core. It covers both the offline
deterministic stub and the live OpenRouter call.

## Architecture reminder

```
Operator
  -> router-core-agent
       |
       | GET /v0/device, /v0/status, /v0/capabilities, /v0/clients
       v
     router-core serve (loopback)
       |
       | GET /v0/security/<name>  (tool calls)
       v
     TP-Link WR841N v8.4 firmware (192.168.1.1)

  router-core-agent
    |
    | POST /v1/chat/completions (tool calls)
    v
  OpenRouter (gateway)
    |
    v
  MiniMax M3
```

The agent is the only process that talks to OpenRouter. The
runtime never sees the API key. The frontend (separate process)
talks to the agent over the local HTTP surface (`/v0/chat`).

## Dry-run (no API key)

```sh
# Build
go build -o ./bin/router-core ./cmd/router-core
go build -o ./bin/router-core-agent ./cmd/router-core-agent

# 1. Start router-core serve in one terminal
./bin/router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484 &
# (then type the admin password when prompted)

# 2. In a second terminal, ask the agent
./bin/router-core-agent \
  --router-core-url http://127.0.0.1:8484 \
  --question "¿Es seguro mi Wi-Fi?" \
  --dry-run
```

The dry-run path is the deterministic stub. It picks a tool
sequence based on Spanish keywords in the question and returns
a generic "configure OPENROUTER_API_KEY" answer that does not
summarise the evidence. The trace to stderr shows the tool calls
and the parsed responses.

## Dry-run over HTTP (frontend integration)

```sh
# Start the agent as an HTTP server in addition to (or instead
# of) the CLI mode.
./bin/router-core-agent \
  --router-core-url http://127.0.0.1:8484 \
  --serve 127.0.0.1:8585 &
# (then type the admin password when prompted)

# Health check
curl -sS http://127.0.0.1:8585/healthz
# {"model":"minimax/minimax-m3:free","state":"ok"}

# Chat
curl -sS -X POST http://127.0.0.1:8585/v0/chat \
  -H 'Content-Type: application/json' \
  -d '{"question":"¿Es seguro mi Wi-Fi?"}'
```

In `--dry-run` mode the HTTP handler also runs the stub. Set
the API key to switch to the live path automatically.

## Live run (with API key)

```sh
export OPENROUTER_API_KEY="sk-or-v1-..."
./bin/router-core-agent \
  --router-core-url http://127.0.0.1:8484 \
  --question "Is my Wi-Fi exposed?" \
  --model minimax/minimax-m3:free
# or
./bin/router-core-agent \
  --router-core-url http://127.0.0.1:8484 \
  --serve 127.0.0.1:8585
# (then POST to /v0/chat)
```

The agent detects the key from `OPENROUTER_API_KEY` (override
with `--key-env NAME`) and switches from the stub to the live
path. The model receives:

- the system prompt (device, status, capabilities, the four
  states);
- the user's question;
- a list of tools (`get_security`, `get_clients`).

It then emits one tool call per turn. The agent executes each
call against `router-core serve`, feeds the structured JSON
back, and continues until the model emits a final answer.

## Expected live trace

```text
router-core-agent: connected to http://127.0.0.1:8484
router-core-agent: device  = {"vendor":"TP-Link",...}
router-core-agent: status  = {"reachable":"true",...}
router-core-agent: caps    = {"capabilities":{...}}
router-core-agent: question = "Is my Wi-Fi exposed?"
router-core-agent: calling https://openrouter.ai/api/v1/chat/completions (model=minimax/minimax-m3:free)
router-core-agent: tool call -> get_security("wireless")
router-core-agent: result    -> {"state":"verified",...}
router-core-agent: tool call -> get_security("wps")
router-core-agent: result    -> {"state":"absent",...}
router-core-agent: tool call -> get_security("remote-management")
router-core-agent: result    -> {"state":"absent",...}

Observed facts
  - Reviewed 3 security observations: wireless, wps, remote-management.
  ...
```

The frontend should render each tool call as a step in the UI
("Checking wireless security...", "Checking WPS...") and the
final answer as the bottom of the trace.

## Capturing the trace as a fixture

For the hackathon demo, capture a real run as a fixture so the
team can replay the trace in CI without an API key:

```sh
./bin/router-core-agent --router-core-url http://127.0.0.1:8484 \
  --question "Is my Wi-Fi exposed?" \
  --model minimax/minimax-m3:free \
  --trace-out fixtures/agent-traces/2026-09-04-wifi-exposed.jsonl
```

The JSONL file has one record per turn: timestamp, model,
question, tool calls, tool results, assistant message. The
frontend can `fetch` this file in development mode and replay
the trace step-by-step.

## Safety

- The agent is read-only. It never sends a POST to
  `router-core serve` (the serve has no POST endpoints). The
  architecture test
  (`internal/architecture_test.go`) enforces this at the
  source level.
- The agent's HTTP server only binds on the loopback. `--serve
  0.0.0.0` is refused before any listener is created.
- The agent's CORS middleware only allows loopback origins
  (`127.0.0.1`, `localhost`, `::1`); foreign origins get 403.
- The OpenRouter API key is read from the environment; it is
  never logged or persisted.

## Tests

The agent has a 16-test suite in
`cmd/router-core-agent/main_test.go` that covers:

- `TestRouterCoreClientGet_*` — the HTTP client against a
  mock router-core (happy path, 404, invalid JSON).
- `TestStubSequence` — keyword-driven tool ordering.
- `TestRunStub_HappyPath` — full stub run with a mock
  router-core.
- `TestExecuteQuestion_*` — the top-level entry point.
- `TestRunLive_HappyPath` — full live run with a mock
  OpenRouter that emits one tool call and one final answer;
  the agent executes the tool against a mock router-core and
  feeds the result back.
- `TestRunLive_OpenRouterError` — the agent surfaces a 5xx from
  OpenRouter cleanly.
- `TestHealthzHandler_*` — the `/healthz` route.
- `TestChatHandler_*` — the `/v0/chat` route (bad body,
  wrong method, OPTIONS preflight).
- `TestWithLocalCORS_RejectsForeignOrigin` — the CORS
  middleware blocks non-loopback origins.
- `TestIsAllowedSecurityCapability` — the six-capability
  allowlist.
- `TestValidateLoopbackURL_*` — the URL allowlist.
- `TestRunAgentServer_RejectsNonLoopback` — `--serve 0.0.0.0`
  is refused.
- `TestBuildSystemPrompt_IncludesDevice` — the system prompt
  carries the live device identity.
- `TestIsLoopbackAddr` — the loopback host check.

Run them with `go test ./cmd/router-core-agent/`.

## Next items on `develop`

- [ ] Frontend end-to-end: prove the round trip
  frontend → `/v0/chat` → agent → `/v0/security/*` →
  frontend. Vitest is not configured; the smallest viable test
  is a Vitest smoke test that fetches the mock fixtures and
  asserts the typed response shape.
- [ ] Live OpenRouter trace capture: the operator runs
  `./bin/router-core-agent` with `OPENROUTER_API_KEY` set and
  saves the trace as a fixture in `fixtures/agent-traces/`.
- [ ] Mega-merge `develop` into `main` once the integration
  is verified end-to-end.
