# MiniMax-Week (GMI Cloud × MiniMax) judging FAQ

This file is a short reference for the MiniMax-Week hackathon
judges and the router-core author. For full submission rules,
see <https://www.gmicloud.ai/minimax-week>.

## Eligibility

- Solo or team of up to four.
- Worldwide, online.
- One project per submission.
- Choose one track: **Reasoning** (this project), Synthesis, or
  Multimodality.

## Build window

- **24 August 2026 → 6 September 2026** (deadline 6 September).
- The work must be original and created during the campaign.

## What "MiniMax via GMI Cloud" means here

The reasoning layer (the MiniMax model that interprets the
structured router observations) must use MiniMax models served
by GMI Cloud. `router-core` itself is the supporting
infrastructure; the supporting layer may use other tools and
AI assistants, as the MiniMax-Week rules allow.

In this project:

- **Reasoning layer:** MiniMax M3 (primary) and MiniMax M2.7
  (fallback), both served by GMI Cloud, accessed through
  OpenRouter as a routing gateway.
- **Supporting infrastructure:** the Go codebase, the parser
  layer, the test suite, the documentation, and the
  integration with the physical lab unit. The initial Go
  baseline (Phase 0) was produced by Devin AI; subsequent
  edits by the author and other AI coding assistants are
  allowed under the rules.

## Track choice: Reasoning

`router-core` qualifies as a **Reasoning** project: it is an
agent-shaped system that plans its own read-only observations
against a real legacy device and verifies its own recipe
capture. The agent's plan is encoded in the typed HTTP API
exposed by `router-core serve`; the reasoning layer calls those
endpoints, attaches provenance, and phrases the answer in
natural language.

## Models used (per the form)

- **MiniMax M3** (coding, 1M ctx) — primary reasoning model.
- **MiniMax M2.7** (high-speed) — fallback / low-latency
  model.

## Submission materials

- Full name and GMI Cloud account email.
- Country.
- X / Twitter handle.
- Solo or team designation.
- Selected track: **Reasoning**.
- Project name: `router-core`.
- Public repository link: <https://github.com/Quiarom/router-core>.
- Demo video (≤ 3 minutes).
- X post tagging **@MiniMaxAI** and **@GMIcloudHQ**.

## Judging criteria (per the rules)

- **Model usage:** how meaningfully the project uses MiniMax
  models through GMI Cloud. This project uses M3 as the
  reasoning layer over the typed observation surface and M2.7
  as a low-latency fallback for short status queries.
- **Usability:** a new user can install the binary, point it
  at their WR841N, and ask a question in plain language. The
  frontend (third-party, out of scope for this submission)
  is the conversational UI; `router-core` is the deterministic
  observation backend.
- **Originality:** the "Unknown is first-class" + "mutations
  are unrepresentable" pair, applied to legacy consumer
  routers, is not a common pattern in the prior art.

## Prize (per track, per the rules)

- $500 cash.
- $200 in GMI Cloud credits.
- Three-month MiniMax Token Plan Max.

## Timeline

- **Build window:** 24 August 2026 → 6 September 2026.
- **Winners announced:** 11 September 2026.

## Contact

For judging questions, contact GMI Cloud and MiniMax via the
hackathon's official channels. For bugs in this submission,
open an issue on the public repository.
