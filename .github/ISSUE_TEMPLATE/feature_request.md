---
name: Feature request
about: A new capability for router-core
title: "[feat] "
labels: ["enhancement", "needs-triage"]
assignees: []
---

## Problem

What problem does this solve? One or two sentences. If the
user-facing pain is a long workflow, describe it concretely.

## Proposed solution

What should router-core do that it does not do today? If you
have a sketch of the new endpoint, CLI flag, or capability
name, include it.

## Alternatives considered

What other approaches were considered and why is this one
better? Be honest if the alternative is simpler.

## Safety impact

This project is read-only by design. If your proposal would
introduce a write path, a new HTTP method, a new outbound
target, or relax any safety invariant, **stop and open an ADR
in `docs/adr/` first**. The ADR must be merged before the pull
request that implements the feature. See `CONTRIBUTING.md` for
the ADR workflow.

- [ ] No safety invariant affected
- [ ] Affects a safety invariant (link the ADR)
- [ ] Adds a new vendor adapter (link the ADR)
- [ ] Adds a new HTTP method to the runtime (link the ADR)

## Scope

- [ ] Adapter change (`internal/adapters/<vendor>/`)
- [ ] Transport change (`internal/transport/`)
- [ ] Domain change (`internal/domain/`)
- [ ] New binary under `cmd/`
- [ ] Documentation only
- [ ] Agent-loop change (`cmd/router-core-agent/`)

## Mock or contract sketch

If you have a mock JSON body or a sample HTTP request/response,
include it. The frontend team can use this directly.

## Additional context

Links, screenshots, prior art, or related issues. Search
existing issues first.
