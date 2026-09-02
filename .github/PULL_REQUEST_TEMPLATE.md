---
name: Pull request
about: Propose a change to router-core
title: ""
labels: []
assignees: []
---

## Summary

One or two sentences describing the change.

## Why now?

A second sentence explaining the trigger. What bug did this
fix? What need did this serve? If it is tied to an issue, say
"Closes #<n>".

## What did I miss?

The third question a reviewer asks. Did you read the relevant
ADR? Did you re-run `gofmt`, `go vet`, `go test -race`? Did you
check the live router if your change touches the adapter?

## Type of change

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds capability)
- [ ] Breaking change (fix or feature that would cause existing
      functionality to change)
- [ ] Documentation only
- [ ] Refactor (no behavior change)
- [ ] Test-only change

## Safety impact

This project is read-only by design. If your change touches a
safety invariant, link the ADR in `docs/adr/` that authorizes it.

- [ ] No safety invariant affected
- [ ] Affects a safety invariant (explain below)
- [ ] Adds or modifies a vendor adapter

If you checked any of the last two, link the ADR. If there is no
ADR yet, this PR will not be merged.

## Verification

- [ ] `gofmt -l .` is empty
- [ ] `go vet ./...` is silent
- [ ] `go build ./...` succeeds
- [ ] `go test ./... -race` is green on all eight packages
- [ ] Live physical verification when the change touches the
      adapter: `router-core probe --host 192.168.1.1` returns
      the expected fingerprint, and `serve` answers the
      relevant `/v0/*` endpoints
- [ ] Added or updated tests for the new behavior
- [ ] Updated `docs/STATUS.md` if a capability flipped Verified
- [ ] Updated `CHANGELOG.md` under `## [Unreleased]`

## Rollback plan

One sentence on how to revert this change. If it is a
non-trivial revert, link to the issue or commit that will
revert it.

## Checklist

- [ ] Commit messages follow Conventional Commits
- [ ] No secrets, tokens, or operator-specific paths in the diff
- [ ] New exported types and functions have doc comments
- [ ] License headers preserved (MIT)
- [ ] `humanizer` skill run on changed prose (`.md` files)
- [ ] Branch rebased onto the latest `main`

## Related

Closes #<issue-number> (if any). Related ADRs, issues, or
discussions.
