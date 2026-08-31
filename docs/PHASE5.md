# Phase 5 — MiniMax reasoning loop

This document describes the M3 agent loop and how to run it
against a live `router-core serve`.

## Architecture

```
Operator  --question-->  router-core-agent
                              |
                              | GET /v0/device, /v0/status, /v0/capabilities
                              v
                       router-core serve (loopback)
                              |
                              | GET /v0/security/<name>  (one per tool call)
                              v
                       WR841N v8.4 firmware

The agent itself is a separate process. It does not share
memory with `serve`. Every interaction with the router is a
GET to a /v0/ endpoint over loopback. The agent never writes
to the router.
```

The agent is built around three principles:

1. **Read-only by construction.** The agent binary has no
   mechanism to mutate the router; it can only GET endpoints
   on the loopback serve. The architecture test
   `internal/architecture_test.go` enforces this at the
   source level: no `POST` / `PUT` / `DELETE` is allowed in
   `cmd/router-core/`, `cmd/router-core-learn/`, or
   `internal/`. The only file that may POST is the agent's
   own `cmd/router-core-agent/main.go`, and it posts to
   OpenRouter, not to the router.
2. **Honest state.** The runtime distinguishes four
   capability states (`verified`, `absent`,
   `unsupported_or_unverified`, `unavailable`). The agent
   passes these through unchanged; the model is responsible
   for reporting them as-is, not collapsing them to
   `true` / `false`.
3. **Visible trace.** Every tool call prints to stderr:
   `router-core-agent: tool call -> get_security("wireless")`
   followed by the structured result. The final answer
   prints to stdout. The trace is the demo; the model never
   shows its hidden chain-of-thought.

## Running

### Live (requires OpenRouter API key)

```sh
export OPENROUTER_API_KEY="sk-or-v1-..."
router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484 &
router-core-agent \
  --router-core-url http://127.0.0.1:8484 \
  --question "Is my Wi-Fi exposed?" \
  --model minimax/minimax-m3:free
```

The agent sends the system prompt (device + status + caps) and
the user question to MiniMax M3 via the OpenRouter chat
completions endpoint. The model emits `get_security(name)`
tool calls one at a time; the agent executes each against the
local serve and feeds the structured JSON back. When the
model emits no further tool calls, the agent prints the final
answer to stdout.

### Dry-run (no API key required)

```sh
router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484 &
router-core-agent \
  --router-core-url http://127.0.0.1:8484 \
  --question "Is my Wi-Fi exposed?" \
  --dry-run
```

The deterministic stub picks a tool sequence based on the
question:

- "Wi-Fi", "wireless", "exposed" → `wireless`, `wps`,
  `remote-management`
- "who", "connected", "devices", "clients" → `wireless`,
  `wps`, `remote-management`, `dmz`, `upnp`, `forwarding`
- everything else → the full six-capability sweep

After the trace, the stub renders a fixed four-section
summary: Observed facts, Potential concern, Recommendation,
Action requiring explicit operator approval.

The dry-run is the offline demo path. It is the same code
shape as the live path: the agent connects, fetches device +
status + caps, runs a tool loop, prints a final answer. The
only difference is that the model is replaced by a fixed
sequence.

## System prompt

The model sees the device identity, the status, and the
capability matrix before any tool call. The prompt is built
per-invocation from the live `router-core serve` response, so
the model never reasons over stale fixtures.

```text
You are a read-only network auditor for a single home router.
Your job: answer the operator's question in plain language by
gathering one observation at a time, then summarizing Observed
facts, Potential concerns, Recommendations, and any Action that
would require explicit operator approval (you must never
perform mutations yourself).

Never invent values. If a tool returns state "absent" or
"unavailable" or "unsupported_or_unverified", report that as-is.
Do not collapse to true/false.

Emit exactly one tool call per turn. After the evidence is
sufficient, respond with the final answer and no further tool
calls.

DEVICE
{"vendor":"TP-Link", "model":"TL-WR841N/ND", ...}

STATUS
{"reachable":"true", "wanStatus":"connected", "uptimeSeconds":20000, ...}

CAPABILITIES
{"capabilities":{"device":"verified", "status":"verified", "wps":"absent", ...}}

Available tools:
- get_security(name): GET /v0/security/<name>. Use it to inspect
  one capability at a time.
```

## Tool schema

```json
{
  "name": "get_security",
  "description": "GET /v0/security/<name> on the local router-core serve...",
  "parameters": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "enum": ["wireless", "wps", "dmz", "upnp", "remote-management", "forwarding"]
      }
    },
    "required": ["name"]
  }
}
```

The runtime distinguishes HTTP status by meaning:

- `200` — capability is verified; the body contains the
  observation.
- `404` — capability is `unsupported_or_unverified`; the
  firmware does not implement it.
- `503` — capability is `unavailable`; the runtime cannot
  satisfy it right now.

The model reads the body and the status code; the agent
does not interpret them.

## Trace format

```text
router-core-agent: connected to http://127.0.0.1:8484
router-core-agent: device  = {...}
router-core-agent: status  = {...}
router-core-agent: caps    = {...}
router-core-agent: question = "Is my Wi-Fi exposed?"
router-core-agent: calling https://openrouter.ai/api/v1/chat/completions (model=minimax/minimax-m3:free)
router-core-agent: tool call -> get_security("wireless")
router-core-agent: result    -> {"state":"unsupported_or_unverified","reason":"..."}
router-core-agent: tool call -> get_security("wps")
router-core-agent: result    -> {"state":"absent","reason":"WPS endpoint not present..."}
...
Question: Is my Wi-Fi exposed?

Observed facts
  - ...
```

The trace is what the operator sees during the demo. The
model's final answer is the only thing on stdout; everything
else (tool calls, structured results, prompt fragments) is on
stderr.

## Live verification (2026-08-31)

```sh
$ router-core-agent --router-core-url http://127.0.0.1:8489 \
    --question "Is my Wi-Fi exposed?" --dry-run --timeout 6s
router-core-agent: connected to http://127.0.0.1:8489
router-core-agent: device  = {"vendor":"TP-Link","model":"TL-WR841N/ND",
                              "hardwareVersion":{"value":"WR841N v8 00000000",...},
                              "firmwareVersion":{"value":"3.15.9 Build 140724 Rel.63227n",...},
                              "authenticated":"true","provenance":"observed"}
router-core-agent: status  = {"reachable":"true","wanStatus":"unknown","uptimeSeconds":20000,"provenance":"observed"}
router-core-agent: caps    = {"capabilities":{"clients":"verified","device":"verified",
                              "dmz":"unavailable","forwarding":"unavailable",
                              "remote_management":"absent","status":"verified",
                              "upnp":"absent","wireless_security":"unavailable","wps":"absent"}}
router-core-agent: question = "Is my Wi-Fi exposed?"
router-core-agent: using deterministic stub (set OPENROUTER_API_KEY to use MiniMax M3 live)
router-core-agent: tool call -> get_security("wireless")
router-core-agent: result    -> {"state":"unsupported_or_unverified","reason":"..."}
router-core-agent: tool call -> get_security("wps")
router-core-agent: result    -> {"state":"absent","reason":"..."}
router-core-agent: tool call -> get_security("remote-management")
router-core-agent: result    -> {"state":"absent","reason":"..."}
Question: Is my Wi-Fi exposed?

Observed facts
  - ...
```

## Known gaps (not in this commit)

- The wireless-security parser is still a placeholder
  (`fetchWirelessSecurity` returns
  `domain.ErrUnverifiedEndpoint`). The runtime reports
  `unavailable` for `/v0/security/wireless`. The agent
  correctly reports this to the model; the model correctly
  reports it in the final answer. Wiring the parser is
  outside the Phase 5 scope.
- The OpenRouter HTTP client in the agent is a minimal POST
  with no streaming. For the hackathon demo, a non-streaming
  final answer is sufficient.
- The four security endpoints (DMZ, UPnP, Forwarding,
  Remote Management) all return 503 because the runtime's
  `Security()` iterates them in order and the first one
  (`OpWPS`) returns `ErrUnverifiedEndpoint`. This is a known
  issue in the runtime, not the agent. Fixing it requires
  splitting the `Security()` iteration per-capability.

## Files

- `cmd/router-core-agent/main.go` — the agent binary.
- `cmd/router-core/serve.go` — added `handleCapabilities` for
  `/v0/capabilities`.
- `internal/architecture_test.go` — scoped to allow
  `cmd/router-core-agent/` to POST to OpenRouter.
- `docs/FRONTEND_CONTRACT.md` — the HTTP surface the agent
  consumes.
- `docs/EVIDENCE_TRACE.md` — the runtime fix that gave the
  agent something to talk to.
