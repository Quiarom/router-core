# router-core frontend

React 19 + Vite 8 + Tailwind 4 dashboard for
[router-core](https://github.com/Quiarom/router-core). It consumes
the typed HTTP surface documented in
[`docs/FRONTEND_CONTRACT.md`](../docs/FRONTEND_CONTRACT.md).

## Endpoints the dashboard calls

The dashboard talks to two loopback services:

- `router-core serve` on `--addr 127.0.0.1:8484` (default),
  served by `cmd/router-core` in the parent repo. Exposes
  `/v0/healthz`, `/v0/device`, `/v0/status`, `/v0/capabilities`,
  `/v0/clients`, and `/v0/security/<name>`.
- `router-core-agent --serve 127.0.0.1:8585` (default), served
  by `cmd/router-core-agent` in the parent repo. Exposes
  `/healthz` and `POST /v0/chat` with the structured
  `agentResult`.

The URLs are configurable via Vite env vars at build time:

```sh
VITE_ROUTER_CORE_URL=http://127.0.0.1:8484 \
VITE_AGENT_API_URL=http://127.0.0.1:8585/v0/chat \
  npm run build
```

## Scripts

```sh
npm run dev       # vite dev server with HMR
npm run build     # production build to dist/
npm run preview   # serve the production build locally
npm run lint      # oxlint
npm test          # contract smoke test (node --test, zero deps)
```

## Testing

The contract test (`tests/contract.test.mjs`) reads the live
OpenRouter trace captured at
[`../fixtures/agent-traces/2026-09-04-wifi-exposed.live.jsonl`](../fixtures/agent-traces/2026-09-04-wifi-exposed.live.jsonl)
and asserts the per-turn shape matches what this frontend
expects. It runs with the built-in `node --test` runner, so
there is no extra dependency to install. Run it from the
parent repo root:

```sh
node --test frontend/tests/contract.test.mjs
# or
cd frontend && npm test
```

The test is a contract shape check, not a full React render
suite. Adding Vitest + @testing-library/react for component
tests is a separate, larger task tracked in the parent repo's
`develop` branch roadmap.

## Dev-mode mock trace

For demos without an OpenRouter key, the live trace JSONL
fixture is a faithful representation of what a real MiniMax M2.7
agent produces. Wire the dev-mode mock in the parent repo's
`AiAssistantView` component to fetch this file and replay it
turn-by-turn:

```ts
const TRACE_URL = `${import.meta.env.VITE_FIXTURE_BASE ?? "/fixtures"}/agent-traces/2026-09-04-wifi-exposed.live.jsonl`;
const response = await fetch(TRACE_URL);
const text = await response.text();
const turns = text.split("\n").filter(Boolean).map(JSON.parse);
```

The trace has one record per turn (user question, assistant
tool call, tool result, final answer). Render each tool call
as a step in the UI ("Checking wireless security...", "Checking
WPS...") and the final answer as the bottom of the trace.
