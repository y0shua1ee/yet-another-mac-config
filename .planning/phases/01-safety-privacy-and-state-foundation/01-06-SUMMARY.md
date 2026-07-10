---
phase: 01-safety-privacy-and-state-foundation
plan: "06"
subsystem: safety-control-plane-policy
tags: [go, control-plane, ownership, report-only, fail-closed, ast-guard]

requires:
  - phase: 01-safety-privacy-and-state-foundation
    plan: "05"
    provides: typed artifacts, privacy-safe rendering, isolated fixtures, four-state sentinel evidence, and fixed-deadline runner routes
provides:
  - closed Determinate Nix, nix-darwin, and Home Manager role contract
  - one selected executable owner per logical scope and executable
  - report-only extra and unmanaged state with an empty operation list
  - one synthetic fixture-scoped fake-write exception and no live mutable route
  - parser and AST guards against executor imports, shell dispatch, and mutable CLI routes
affects: [01-07-phase-e2e, phase-02-ownership-inspection, future-recovery-policy]

tech-stack:
  added: []
  patterns: [closed owner-role enums, exact owner-pattern validation, report-only policy, pre-artifact operation gate, AST dependency guard, fixed-child wave]

key-files:
  created:
    - safety/internal/contract/controlplane.go
    - safety/internal/contract/controlplane_test.go
    - safety/testdata/controlplane/cases.json
    - safety/internal/e2e/controlplane_cli_test.go
    - safety/internal/contract/policy.go
    - safety/internal/contract/policy_test.go
    - safety/internal/e2e/no_cleanup_cli_test.go
  modified:
    - safety/scripts/test.sh
    - safety/cmd/yamc-safety/main.go
    - safety/internal/workflow/synthetic.go

key-decisions:
  - "Represent Determinate Nix, nix-darwin, and Home Manager as exact closed role sets while keeping declaration, manager binary, payload, selected executable owner, and activation context independent."
  - "Allow Phase 1 policy to return extra or unmanaged-present only with an explicit empty operation list; owner or module metadata cannot add authority."
  - "Keep the only operation-capable path as one synthetic fixture-fake-write whose target is a fixture logical reference, and reject every other operation kind before artifact construction."

patterns-established:
  - "Layered ownership: a module declaration may provide a manager entrypoint without taking payload or selected-executable ownership."
  - "Report-only convergence: extra state is visible evidence, never implicit cleanup authorization."
  - "Physical incapability: production source has no executor package, mutable top-level CLI route, or shell dispatch edge."

requirements-completed: [SAFE-08]

duration: 17 min
completed: 2026-07-10
---

# Phase 01 Plan 06: Layered Control Plane and Report-Only Policy Summary

**Closed ownership facts now preserve the Determinate Nix/nix-darwin/Home Manager hierarchy and delegated payload owners, while a data-only policy reports extra state without exposing any destructive convergence route.**

## Performance

- **Duration:** 17 min
- **Started:** 2026-07-10T20:42:26Z
- **Completed:** 2026-07-10T20:59:41Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Added exact typed owner and role enums for the Determinate Nix distribution/daemon/support boundary, nix-darwin machine composition/activation boundary, and Home Manager user configuration, Nix-built manager entrypoint, config-file, and shell-integration boundary.
- Added separate declaration, manager-binary, managed-payload, selected-executable-owner, and activation-context fields. Duplicate `(scope, executable)` entries and every module-to-payload or module-to-selected-owner collapse fail closed.
- Added synthetic positive contracts for Homebrew, mise, uv, rustup, project wrappers, and exclusive Nix devShell scopes without importing, evaluating, inspecting, or invoking any real manager.
- Added `validate controlplane` and `validate policy` data-only CLI routes. They read bounded synthetic contracts, render only logical identifiers, and share the existing privacy gate.
- Added a closed Phase 1 policy in which `extra` and `unmanaged-present` round-trip only with `operations: []`. Cleanup, uninstall, zap, runtime deletion, prune, trust, download, upgrade, switch, service/defaults/link mutation, destructive convergence, arbitrary command, and apply operation kinds are rejected.
- Gated the walking-skeleton operation before generated-plan construction. The only accepted operation is one `fixture-fake-write` with a `fixture:` target and `mode: synthetic`; live mode and non-fixture targets cannot produce a receipt.
- Added production-source parser/AST guards proving there is no executor package/import, no shell-dispatch literal, and no mutable top-level CLI case. The only existing `os/exec` imports remain the previously approved bounded fake-capture and proof-gated read-only sentinel adapters.
- Added exact `controlplane-contract` and `no-destructive-defaults` task routes plus the fixed `controlplane` wave. Every route retains one complete literal label, exact package/pattern pairs, shared 15-second child deadlines, 47-second wave reservation, exact exit-124 propagation, and fresh child roots.

## Task Commits

Each task was committed atomically:

1. **Task 01-06-01: 编码 control-plane 分层与 one-owner invariant** - `23e0838` (feat)
2. **Task 01-06-02: 拒绝 mutable boundaries 并让 extra state 永远 report-only** - `ff26457` (feat)

_Plan metadata is committed together with this summary._

## Files Created/Modified

- `safety/internal/contract/controlplane.go` - Defines closed control-plane owners/roles, layered ownership facts, strict synthetic parsing, and one-owner validation.
- `safety/internal/contract/controlplane_test.go` - Covers exact primary layers, six delegated ownership patterns, duplicate keys, module role collapse, unknown fields, and synthetic-only fixtures.
- `safety/testdata/controlplane/cases.json` - Supplies public logical positive cases and named negative mutations without host-derived data.
- `safety/internal/e2e/controlplane_cli_test.go` - Verifies logical CLI output, bounded rejection, fixed runner pairs, and zero Nix/Homebrew/manager invocation with synthetic canaries.
- `safety/internal/contract/policy.go` - Defines the data-only Phase 1 report/fixture policy with no callback, executor, command, or arbitrary argv field.
- `safety/internal/contract/policy_test.go` - Proves report-only status, fixture-only fake writes, forbidden mutable operations, closed fields, and callback absence.
- `safety/internal/e2e/no_cleanup_cli_test.go` - Verifies empty unmanaged operation lists, pre-output rejection, synthetic receipt scope, source dependency guards, CLI route closure, and fixed wave aggregation.
- `safety/internal/workflow/synthetic.go` - Evaluates the fixed fixture operation through the closed policy before constructing plan or receipt artifacts.
- `safety/cmd/yamc-safety/main.go` - Adds bounded synthetic control-plane and policy validation routes without adding a mutable command.
- `safety/scripts/test.sh` - Adds the two exact task handlers and fixed two-child control-plane wave.

## Decisions Made

- Control-plane layers are validated as exact role sets, not inferred from module availability or current machine state.
- A selected executable owner determines one accepted ownership pattern for each logical `(scope, executable)`; supporting declaration and manager-binary roles remain separate fields and cannot silently replace payload ownership.
- Report status and operation authorization are different intents. `extra` and `unmanaged-present` accept only an explicitly empty list, while the sole operation-capable intent is a fixed synthetic fixture write.
- The operation policy is evaluated before generated-plan or receipt artifact construction, so invalid authority cannot be sanitized after persistence.
- No real apply interface, callback, arbitrary command, manager execution route, cleanup generator, or mutable policy branch is present in Phase 1.

## TDD Evidence

- Before Task 1 implementation, `task controlplane-contract` selected exactly `./internal/contract` plus `^TestControlPlaneContract$` and `./internal/e2e` plus `^TestControlPlaneCLI$`. Both failed with `EXPECTED_RED: controlplane-ownership-behavior-missing`; the runner returned `expected-red-observed`, not unsupported-suite, test-selection, build, or harness failure.
- Before Task 2 implementation, `task no-destructive-defaults` selected exactly `./internal/contract` plus `^TestNoDestructiveDefaults$` and `./internal/e2e` plus `^TestNoCleanupCLI$`. Both failed with `EXPECTED_RED: destructive-policy-behavior-missing` and no unsafe fallback.
- The Task 1 parent diff adds exactly `task:controlplane-contract)`. The Task 2 parent diff adds exactly `task:no-destructive-defaults)` and `wave:controlplane)`; all are single unsplit literals.
- Permanent negative dispatch checks continue to use only `never-registered-task`, `never-registered-wave`, `never-registered-scope`, and malformed phase arguments. No planned future route name is blacklisted.

## Verification

- `/bin/bash -n safety/scripts/test.sh` passed.
- `./safety/scripts/test.sh task controlplane-contract` passed with `synthetic-sentinel-passed`.
- `./safety/scripts/test.sh task no-destructive-defaults` passed with `synthetic-sentinel-passed`.
- `./safety/scripts/test.sh wave controlplane` passed with two fresh fixed child runners and `synthetic-sentinel-passed`.
- `walking-skeleton`, `tier-network-policy`, and `sentinel-verdicts` regression tasks passed after the policy and CLI changes.
- Reserved unknown task/wave probes returned bounded `harness-error/unsupported-suite`; reserved scope and malformed phase probes returned bounded usage rejection. None produced expected RED or a success state.
- Both task commits contain exactly their six-file whitelist. Their parent diffs contain exactly the plan-owned runner labels and no tracked-file deletion.
- Cached diff checks, targeted physical-path/identity/credential scans, and staged Gitleaks passed over the exact whitelist for both commits with zero leaks.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. The existing missing controlled-service proof remains an intentional Plan 01-05 manual-required boundary and was neither altered nor bypassed.

## User Setup Required

None - no package installation, credential, network access, Nix evaluation, Homebrew or delegated-manager invocation, service/defaults/link query, host activation, or cleanup was performed. All writes were repository source changes or fresh synthetic test roots created and marker-cleaned by the established runner.

## Next Phase Readiness

- Plan 01-07 can aggregate the completed control-plane wave into the full phase runner while preserving the existing real-sentinel proof gate, exact 15/47/305-second deadlines, and scoped claim ceiling.
- Phase 2 can consume typed ownership facts for a read-only inspector without inheriting mutation authority.
- No Plan 01-06 blocker remains. Complete real-surface evidence still correctly returns manual-required until the tracked controlled-service proof is current and complete.

## Self-Check: PASSED

- All seven created and three modified implementation/test files exist. Task commits `23e0838` and `ff26457` are present and delete no tracked files.
- Both exact task suites, the `controlplane` wave, and the walking-skeleton/tier/sentinel regressions are green under isolated offline roots.
- SAFE-08 is mechanically covered by report-only decisions, forbidden-operation negatives, synthetic fixture receipt checks, closed CLI dispatch, and production AST/import guards.
- No current-host, whole-Mac, multi-host, recovery-ready, or fresh-install claim was introduced.
- Existing user changes in `CLAUDE.md`, `.ai/`, and `.config/alma/` remain unstaged and unchanged.

---
*Phase: 01-safety-privacy-and-state-foundation*
*Completed: 2026-07-10*
