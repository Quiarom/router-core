---
name: Bug report
about: Something in router-core does not work as documented
title: "[bug] "
labels: ["bug", "needs-triage"]
assignees: []
---

## Summary

One or two sentences describing the bug. If you can answer
"what did you expect?" and "what happened instead?", write
both here.

## Environment

- `router-core` version (output of `router-core --version`):
- Go version (`go version`):
- OS and architecture:
- Target router model and firmware version (if applicable):
- Subcommand that fails (`probe`, `inspect`, `serve`, `learn`,
  `observe`):

## Reproduction

```bash
# Minimal command(s) that reproduces the issue.
# Include the exact flags and any input.
./bin/router-core probe --host 192.168.1.1
```

## Expected behavior

What the documentation says should happen. Cite the
section of the README, the ADR, or the HACKATHON_FAQ if
relevant.

## Actual behavior

What actually happened. Paste the full output, including any
stack traces.

## Relevant log output

```
paste here
```

## Configuration

- Are you running against the live router or a synthetic
  fixture?
- Is `ROUTER_ALLOW_UNVERIFIED` set?
- Is the agent (`router-core-agent`) involved, or just the
  CLI?

## Additional context

Anything else that might help: sanitized captures, network
topology, related issues, what you already tried.
