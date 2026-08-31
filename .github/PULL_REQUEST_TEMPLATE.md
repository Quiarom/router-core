---
name: Pull request
about: Propose a change to router-core
title: ""
labels: []
assignees: []
---

## Summary

One or two sentences describing the change.

## Type of change

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds capability)
- [ ] Breaking change (fix or feature that would cause existing
      functionality to change)
- [ ] Documentation only
- [ ] Refactor (no behavior change)
- [ ] Test-only change

## Safety impact

- [ ] No safety invariant affected
- [ ] Affects a safety invariant (explain below)
- [ ] Adds or modifies a vendor adapter

If your change affects a safety invariant or adds a new vendor
adapter, link the ADR in `docs/adr/` that authorizes it.

## Verification

- [ ] `gofmt -l .` is empty
- [ ] `go vet ./...` is silent
- [ ] `go build ./...` succeeds
- [ ] `go test ./... -race` is green on all eight packages
- [ ] Added or updated tests for the new behavior
- [ ] Updated `docs/STATUS.md` if a capability flipped Verified

## Checklist

- [ ] Commit messages follow Conventional Commits
- [ ] No secrets, tokens, or operator-specific paths in the
      diff
- [ ] New exported types and functions have doc comments
- [ ] License headers preserved (MIT)

## Related

Closes #<issue-number> (if any). Related ADRs, issues, or
discussions.
