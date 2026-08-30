# ADR 0001: Record the read-only architecture

## Decision

Keep transport, vendor parsing, domain contracts, fixture replay, and CLI
separate. The transport exposes only guarded GET dispatch. Parsers produce
facts and preserve unknown values. Fixture replay uses the same parsers as
live access.

## Consequences

The model-facing surface cannot mutate a router. Synthetic fixtures make
parser development deterministic without network access, while provenance
keeps replayed observations distinct from hardware observations.
