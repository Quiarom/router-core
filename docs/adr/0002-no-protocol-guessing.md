# ADR 0002: Do not guess the router protocol

## Decision

Authentication and every endpoint recipe lacking a sanitized physical capture
remain blocked. Authentication returns `ErrCaptureMissing`; unverified live
endpoint dispatch returns `ErrUnverifiedEndpoint` unless the operator
explicitly opts in with `ROUTER_ALLOW_UNVERIFIED=1`.

## Rationale

Legacy firmware builds vary in login fields, cookies, and page paths. Guessing
would turn an observation tool into an unreliable or potentially mutating
client. `BLOCKED_CAPTURE.md` records the exact evidence needed to unblock each
operation.
