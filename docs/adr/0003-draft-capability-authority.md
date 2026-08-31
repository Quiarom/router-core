# ADR 0003 — Replace GET-only safety with capability + recipe authority (DRAFT, not applied)

Status: **DRAFT — do not apply until Outcome A1 from PHASE2_OUTCOMES.md
is reached.** Phase 3 trigger: physical capture confirms a candidate
recipe works against the lab unit.

This ADR is prepared ahead of time so that when Phase 2 produces the
right evidence, the runtime architecture refactor can move quickly.

## Decision (when applied)

Replace the GET-only safety invariant in `internal/transport` with a
capability + recipe model. Until then, do nothing.

## Context

The current `internal/transport/http.go` rejects everything except GET.
This is wrong: prior-art evidence (PA-1, PA-2) shows the WR841N
firmware reboot is invoked by a GET with a `?Reboot=Reboot` query
parameter. Method is not authority.

In Phase 2 the probe (`cmd/router-core-learn`) uses GET for every
endpoint. That is fine for the probe — it has no write surface. But
the runtime adapter will eventually need POST, and possibly GET-with-
mutating-query, when captures justify it. The current invariant
would block legitimate reads.

## Approach (to apply when Outcome A1 lands)

### Capability enum

```go
package transport

type Capability string

const (
    CapAuth    Capability = "auth"     // login exchange, any HTTP method
    CapObserve Capability = "observe"  // read-only data fetch, any HTTP method
)
```

`CapMutate` is intentionally NOT defined. Runtime code cannot
construct it because it does not exist in any package the runtime
imports.

### Recipe registry

```go
package tplinkwr841v8

type Recipe struct {
    Op         string
    Path       string
    Method     string
    Capability transport.Capability
    Verified   bool
    CaptureNote string
}

var Endpoints = map[string]Recipe{ /* … */ }
```

The runtime adapter and the probe both reach the transport only
through recipes. There is no `Dispatch(ctx, method, rawURL string)`
entry point for callers above the transport — only
`Dispatch(ctx, recipe Recipe, body []byte, headers http.Header)`.

### dispatchAllowed

```go
func dispatchAllowed(r Recipe) error {
    if !r.Verified {
        return domain.ErrUnverifiedEndpoint
    }
    return nil
}
```

Note: capability check is enforced by the type system (no CapMutate
exists). The runtime guard is purely about verification state.

### architecture_test.go extension

The current test scans `internal/` for forbidden HTTP method calls.
It should be replaced with a stronger scan that fails the build if:

- `CapMutate` is referenced anywhere in `internal/` or `cmd/` (it
  doesn't exist; any reference means someone tried to introduce it).
- The following path literals appear in `internal/` or `cmd/router-core/`
  (NOT in `cmd/router-core-learn/`, where the probe lives):

```
/userRpm/SysRebootRpm.htm
/userRpm/RestoreRpm.htm
/userRpm/UpgradeRpm.htm
/userRpm/FactoryResetRpm.htm
```

These are known WR841N mutation endpoints. Even mentioning them in
runtime paths is suspicious — the runtime has no business knowing
they exist.

- The strings `"reboot"`, `"factory reset"`, `"restore"`, `"upgrade"`
  appearing in path strings inside `internal/adapters/tplinkwr841v8/`.

## When this ADR is APPLIED (Phase 3)

1. Write `internal/transport/capability.go` with `CapAuth`, `CapObserve`.
2. Replace `internal/transport/http.go`'s `dispatch` signature with
   `Dispatch(ctx, recipe Recipe, body []byte, headers http.Header)`.
3. Replace `internal/adapters/tplinkwr841v8/endpoints.go` `Endpoint`
   struct with `Recipe` struct (renamed).
4. Update `Adapter.fetch` and `Adapter.authenticate` to use recipes.
5. Update `internal/architecture_test.go` per the extension above.
6. Update all runtime tests to pass through recipes.
7. The probe (`cmd/router-core-learn/`) keeps its own private
   request construction — it does NOT use the recipe registry. The
   probe is an experimental harness; the runtime path is what this
   ADR refactors.

## Risks (when applied)

- Recipe type lives in `internal/adapters/tplinkwr841v8/` but the
  transport consumes it. This creates a potential import cycle if
  the transport imports the adapter (it currently does not — adapter
  imports transport). Verify the direction stays
  `adapter -> transport`, never the reverse.
- If a future vendor is added, the recipe type may need to move to
  `internal/domain/`. Phase 4 problem.
- The capability enum being unenforced at the value level means a
  test or future contributor could define `CapMutate` in a new
  package. The architecture test catches that.