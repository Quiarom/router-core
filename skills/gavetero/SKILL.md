---
name: gavetero
description: Use gavetero to investigate home network state. Trigger when the user asks about network exposure, slow Internet, router configuration, connected devices, or any other home-network diagnostic question. Do not hardcode a command sequence: choose the smallest useful observation, then update your hypothesis and choose the next.
---

# Gavetero

Gavetero is a local-first, read-only observation layer for
consumer routers. It does not change the router. It returns
typed observations, each with one of four knowledge states:

- `verified` — the runtime actually read the value
- `absent` — the firmware does not implement the capability
- `unsupported_or_unverified` — the runtime has no parser
- `unavailable` — transport failure

Unknown is never silently interpreted as `false` or
`disabled`. State that remains unverified stays
unverified.

## When to use Gavetero

Use Gavetero when the user asks about:

- home network health
- slow Internet or connectivity
- router configuration or exposure
- connected devices
- DNS, routing, or VPN-related connectivity

If the question is unrelated to the local network, do not
invoke Gavetero. The skill is for one specific domain.

## How to use it

Gavetero is a CLI. The binary is `gavetero` (or `gvt`).
After `make install-user`, both live in `~/.local/bin/`.

The non-AI commands are:

- `gvt version`   — print the build version
- `gvt doctor`    — check that the install is healthy
- `gvt inspect`   — show the current router observations

The AI command is:

- `gvt ask "<question>"` — investigate a question with
  MiniMax M3 (or M2.7 fallback). The CLI spawns its own
  router-core and router-core-agent sidecars, captures the
  answer, and prints it. You do not need to start any
  sidecar manually.

## Procedure

Do not precompute a fixed sequence of commands. Gavetero
hands the choice of evidence back to the model.

1. If the user has not set up Gavetero yet (e.g. `gvt
   doctor` reports a missing API key), tell them to run
   `gvt setup` once. Do not pretend the answer.

2. For simple questions, one observation may be enough.
   Use the smallest useful command.

3. For complex questions, choose the first observation,
   read the result, update your hypothesis, then choose
   the next. Stop when evidence is sufficient.

4. State what remains unverified in the final answer.
   Do not infer that a capability is disabled just because
   you cannot verify it.

5. For non-AI inspection (always safe), use:

       gvt inspect --output json

   This gives you the device identity, status, capabilities
   matrix, and DHCP clients without contacting GMI.

6. For AI reasoning, prefer:

       gvt ask "<question>" --output jsonl

   The JSONL stream emits one event per line: `tool_call`,
   `observation`, `completed`. The `completed` event holds
   the final answer.

## What Gavetero does NOT do

- It does not change the router. Read-only.
- It does not invent metrics. Unknown is unknown.
- It does not put secrets in argv. The GMI key is read
  from the environment or the OS credential store.
- It does not require a TTY. `--api-key-stdin` and the
  JSON output formats are designed for non-interactive use.

## Boundaries

- One verified adapter today: TP-Link TL-WR841N/ND v8.4
  on firmware 3.15.9 (Build 140724 Rel.63227n). Other
  routers may be detected but their observations are
  reported as `unsupported_or_unverified` until a
  specific adapter is verified.
- The MiniMax M3 path requires the user to have run
  `gvt setup` once. The setup is local to the user, not
  to Gavetero, not to the model.
