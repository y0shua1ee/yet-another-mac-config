---
phase: 01-safety-privacy-and-state-foundation
plan: "05"
subsystem: safety-sentinel-evidence
tags: [go, protected-surfaces, read-only-adapters, hmac, four-state-verdict, fail-closed, symlink-safety]

requires:
  - phase: 01-safety-privacy-and-state-foundation
    plan: "04"
    provides: marker-owned external fixtures, frozen-primary teardown, closed offline tiers, and proof-gated live policy
provides:
  - exact six-surface/five-domain protected manifest with bounded per-run HMAC snapshots
  - strict four-state evidence, exit, and scoped-claim contract with a structural synthetic ceiling
  - source-bound exact read-only adapter registry and monotonic real-before/real-after envelope mechanism
  - fail-closed manual-required behavior while modern launchctl read-only semantics remain unproved
affects: [01-06-diagnostics, 01-07-phase-e2e, future-read-only-inspection, current-host-evidence]

tech-stack:
  added: []
  patterns: [frozen manifest scope, per-run opaque HMAC, source-bound adapter capability, rooted read-only observation, monotonic outer envelope]

key-files:
  created:
    - safety/internal/sentinel/manifest.go
    - safety/internal/sentinel/snapshot.go
    - safety/internal/sentinel/sentinel_test.go
    - safety/internal/sentinel/verdict.go
    - safety/internal/sentinel/verdict_test.go
    - safety/internal/sentinel/real.go
    - safety/internal/sentinel/real_test.go
    - safety/internal/e2e/sentinel_cli_test.go
    - safety/internal/e2e/real_sentinel_cli_test.go
    - safety/manifests/protected-surfaces.v1.json
    - safety/manifests/real-adapters.v1.json
  modified:
    - safety/scripts/test.sh
    - safety/cmd/yamc-safety/main.go
    - safety/internal/workflow/synthetic.go

key-decisions:
  - "Allow only the exact six logical refs across five closed domains; freeze manifest policy before observation and render only logical refs plus per-run opaque tokens."
  - "Reserve covered-surfaces-unchanged-for-run for complete real evidence; synthetic evidence is structurally limited to synthetic-sentinel-passed."
  - "Treat proof metadata as necessary but insufficient: exact adapters require a private registry-issued source-bound capability, current implementation and negative-suite digests, and one shared bounded observation window."
  - "Keep the tracked launchctl proof explicitly missing and return indeterminate/manual-required with exit 32 before any adapter or workload call until current official no-side-effect semantics are established."
  - "Expose the controlled real-envelope mechanism in Plan 01-05, while leaving phase-runner outer-envelope wiring to Plan 01-07."

patterns-established:
  - "Opaque surface evidence: a fresh process-local HMAC key binds type, existence, mode, symlink identity, and bounded content/tree state without persisting physical roots, names, contents, or stable fingerprints."
  - "Four-state monotonicity: only complete equal required evidence passes; change, incomplete observation, invariant failure, workload failure, or finalization failure cannot be restored, retried, attributed, ignored, or downgraded to pass."
  - "Proof-gated host observation: exact command/path semantics, limits, official sources, review dates, implementation source, and isolated negative tests are bound before a private capability can reach an adapter."

requirements-completed: [SAFE-06, SAFE-07]

duration: 2h 3m
completed: 2026-07-10
---

# Phase 01 Plan 05: Protected-Surface Sentinel Contract Summary

**An exact five-domain manifest now produces privacy-safe bounded evidence and a fail-closed four-state verdict, with source-bound read-only adapters available only through a current proof-gated real observation envelope.**

## Performance

- **Duration:** 2h 3m
- **Started:** 2026-07-10T16:39:58Z
- **Completed:** 2026-07-10T18:43:42Z
- **Tasks:** 3
- **Files modified:** 14

## Accomplishments

- Added a versioned protected-surface manifest containing exactly six refs across `worktree`, `named-home`, `manager-root`, `service`, and `named-target`. Namespace/domain compatibility, required/optional/excluded policy, adapter identity, and finite file/byte/time bounds are validated and frozen before observation.
- Added bounded synthetic snapshots that compare type, existence, mode, symlink identity, and content/tree state with a fresh per-run HMAC key. Traversal, resolver escape, unreadability, races, caps, FIFO/special files, and window exhaustion become typed incomplete evidence rather than unchanged tokens.
- Added the exact `passed`, `violation`, `indeterminate`, and `harness-error` verdict contract with deterministic non-zero exits for every non-pass state. Required changes expose only `change-detected-during-window`; optional status cannot be rewritten after observation starts.
- Bound successful evidence to the exact suite ID/digest, tier, protected-manifest digest, closed window, required before/after tokens, and predeclared optional/excluded lists. Synthetic-only evidence cannot construct or render the real-surface claim.
- Added a versioned real-adapter proof manifest and exact registry for tracked worktree/index, named HOME entry, manager root, controlled service, and named target. Git observation disables lazy fetch, optional locks, fsmonitor, untracked discovery, and submodule traversal; rooted file/tree observation uses bounded nonblocking reads and double-pass race checks.
- Added a controlled real envelope that performs real-before, isolated workload, frozen primary verdict, fixture finalization, real-after, and monotonic combination in that order. A required after-window change overrides an earlier workload or teardown failure with the exact D-15 violation code.
- Kept the tracked controlled-service proof explicitly `missing`: current public Apple material did not establish modern `launchctl print` no-side-effect semantics. The default real command therefore exits `32` as `indeterminate/manual-required` before any adapter or workload call instead of manufacturing a host-safety claim.
- Added exact task routes for `sentinel-manifest`, `sentinel-verdicts`, and `real-sentinel-envelope`, plus fixed `wave:sentinels` aggregation. All routes retain literal package/pattern ownership, exactly-one-test selection, shared wall deadlines, and bounded failure output.

## Task Commits

Each task was committed atomically:

1. **Task 01-05-01: 校验 protected manifest 并取得 bounded opaque snapshots** - `044c0c8` (feat)
2. **Task 01-05-02: 强制四态 verdict、non-pass exits 与 scoped claim** - `6b99f4d` (feat)
3. **Task 01-05-03: 用 exact real adapters 包裹 isolated workload 并证明 failure monotonicity** - `32a8e29` (feat)

_Plan metadata is committed together with this summary._

## Files Created/Modified

- `safety/internal/sentinel/manifest.go` - Parses the closed manifest, validates exact namespace/domain compatibility and bounds, and freezes policy before snapshots.
- `safety/internal/sentinel/snapshot.go` - Produces bounded privacy-safe synthetic file/tree/service state using a fresh per-run HMAC and typed incomplete outcomes.
- `safety/internal/sentinel/sentinel_test.go` - Covers exact scope, compatibility-table rejection, mutation freeze, replacement detection, races, caps, symlinks, unreadability, and private output.
- `safety/internal/sentinel/verdict.go` - Defines four-state evaluation, exit classes, evidence binding, required/optional semantics, and exact scoped claims.
- `safety/internal/sentinel/verdict_test.go` - Proves all verdicts, exits, evidence consistency, warning limits, D-15 change handling, and overclaim rejection.
- `safety/internal/sentinel/real.go` - Implements proof validation, private adapter capabilities, rooted read-only observers, aggregate limits, and the monotonic controlled real envelope.
- `safety/internal/sentinel/real_test.go` - Binds proof to exact source/test digests and exercises proof failure, zero-call canaries, FIFO/race/symlink safety, deadlines, caps, and complete envelope ordering.
- `safety/internal/e2e/sentinel_cli_test.go` - Verifies CLI verdict/exit/claim behavior and the structural synthetic claim ceiling.
- `safety/internal/e2e/real_sentinel_cli_test.go` - Verifies default manual-required behavior, exact exits, bounded private logs, runner routes, and proof-gated envelope behavior.
- `safety/manifests/protected-surfaces.v1.json` - Tracks the exact six logical protected refs and required/optional/excluded policy without physical identity or path data.
- `safety/manifests/real-adapters.v1.json` - Tracks exact adapter invocations, limits, official-source metadata, source/test digests, and an explicit missing controlled-service proof.
- `safety/scripts/test.sh` - Adds three exact task routes and the fixed sentinels wave with process-group wall deadlines.
- `safety/cmd/yamc-safety/main.go` - Adds bounded manifest/verdict/real sentinel commands and nonblocking race-safe artifact reads.
- `safety/internal/workflow/synthetic.go` - Preserves the synthetic-only status ceiling while accepting only complete outer real evidence for the scoped claim.

## Decisions Made

- Public protected scope is a closed logical contract, not a discovery mechanism. No whole-HOME, whole-service, arbitrary path, new namespace, or runtime-expanded surface can enter the observation window.
- Equal required before/after evidence is necessary but not sufficient for a real claim: every binding field must agree, proof must be current and source-bound, and the envelope must complete without workload/finalization/harness ambiguity.
- Adapter provenance is an unexported pointer capability issued only by the validated registry. An injected adapter cannot forge permission with a copied provenance string or proof JSON.
- File and tree observations use rooted access, nonblocking opens, name/descriptor identity checks, aggregate byte/item/time limits, and two passes. Named symlinks are identity evidence only unless their targets remain within the declared named root or explicit repository root, in which case target state is also fingerprinted.
- Git adapters use fixed `/usr/bin/git` invocations with `--no-lazy-fetch`, `GIT_NO_LAZY_FETCH=1`, `GIT_OPTIONAL_LOCKS=0`, disabled fsmonitor, no untracked enumeration, and ignored submodules so observation cannot trigger promisor fetches or optional index writes.
- The current repository must not pretend modern `launchctl print` safety is proven from old or incomplete public documentation. The missing service proof is intentionally claim-blocking and causes the default production path to stop before touching any real surface.
- Plan 01-05 delivers the sentinel mechanism and proof-gated API. Plan 01-07 remains responsible for wrapping the complete phase runner in the outer real envelope; the present CLI proof gate alone is not described as that final wiring.

## TDD Evidence

- Before Task 1 implementation, `task sentinel-manifest` selected exactly `./internal/sentinel` plus `^TestSentinelManifest$` and returned its expected missing-behavior RED rather than unsupported-suite, setup, or selection failure.
- Before Task 2 implementation, `task sentinel-verdicts` selected exactly `./internal/sentinel` plus `^TestSentinelVerdicts$` and `./internal/e2e` plus `^TestSentinelCLI$`; RED was confined to missing verdict behavior.
- Before Task 3 implementation, `task real-sentinel-envelope` selected exactly `./internal/sentinel` plus `^TestRealSentinelEnvelope$` and `./internal/e2e` plus `^TestRealSentinelCLI$`; RED was confined to the missing real-envelope behavior.
- The task-parent runner diffs add exactly `task:sentinel-manifest)`, then `task:sentinel-verdicts)`, then `task:real-sentinel-envelope)` plus `wave:sentinels)` as complete literal labels. Permanent regressions use only reserved never-registered and malformed probes.
- Each plan-prescribed English feature commit contains only its exact task whitelist; cached diff checks, targeted privacy scans, JSON validation, and staged Gitleaks passed before commit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Security] Hardened real observations against lazy fetches, descriptor races, FIFO blocking, and symlink target escape**

- **Found during:** Task 01-05-03 implementation and adversarial review.
- **Issue:** A merely read-looking Git or filesystem operation can still fetch promised objects, update optional state, block on a FIFO, race a renamed entry, or follow a named symlink outside its declared root.
- **Fix:** Added Git no-lazy-fetch and no-optional-lock controls, disabled fsmonitor/untracked/submodule expansion, used rooted nonblocking opens with descriptor/name rechecks and shared deadlines, rejected special files, and required named symlink targets to remain inside the named or explicit repository root before fingerprinting target state.
- **Files modified:** `safety/internal/sentinel/real.go`, `safety/internal/sentinel/real_test.go`, `safety/cmd/yamc-safety/main.go`, `safety/manifests/real-adapters.v1.json`.
- **Verification:** Exact invocation/source digest tests, FIFO no-block canaries, symlink-escape cases, mutation races, shared-deadline cases, aggregate cap cases, and the complete sentinel wave pass.
- **Committed in:** `32a8e29`.

**2. [Rule 1 - Evidence Integrity] Bound proof to production source, negative tests, and an unforgeable registry capability**

- **Found during:** Task 01-05-03 adversarial review.
- **Issue:** Version/freshness metadata and a textual provenance field alone could drift from implementation or be forged by an injected adapter while still appearing approved.
- **Fix:** Bound the manifest to normalized production implementation and negative-suite source digests, validated exact invocations and limits, and required an unexported registry-issued pointer capability before any real adapter call.
- **Files modified:** `safety/internal/sentinel/real.go`, `safety/internal/sentinel/real_test.go`, `safety/manifests/real-adapters.v1.json`.
- **Verification:** Source-digest checks, forged-provenance rejection, proof-missing zero-call canaries, and proof-valid test-only registry ordering all pass.
- **Committed in:** `32a8e29`.

**3. [Rule 1 - Correctness] Preserved D-15 change evidence across workload and teardown failures**

- **Found during:** Task 01-05-03 monotonic combination review.
- **Issue:** A prior workload or fixture-finalization failure could otherwise mask a required after-window surface change, losing the exact `change-detected-during-window` safety signal.
- **Fix:** Required any successfully observed required difference to produce `violation` with the D-15 code while retaining the earlier non-pass context only as bounded evidence; no path can turn either failure into pass.
- **Files modified:** `safety/internal/sentinel/real.go`, `safety/internal/sentinel/real_test.go`.
- **Verification:** Complete ordering and workload/finalization/change cross-product tests pass.
- **Committed in:** `32a8e29`.

### Research-Gated Scope Decision

**4. [Rule 4 - Architectural] Left controlled-service proof missing instead of asserting unsupported launchctl safety**

- **Found during:** Official-documentation research for Task 01-05-03.
- **Issue:** Available public Apple material did not establish current modern `launchctl print` no-side-effect semantics strongly enough to satisfy the plan's current official proof requirement.
- **Decision:** Keep the tracked service proof `missing` with no `valid_until`, fail the production registry as `indeterminate/manual-required`, and assert that neither adapters nor the isolated workload are called. A proof-valid registry exists only in isolated tests to verify the complete mechanism.
- **Files modified:** `safety/manifests/real-adapters.v1.json`, `safety/internal/sentinel/real.go`, `safety/internal/sentinel/real_test.go`, `safety/internal/e2e/real_sentinel_cli_test.go`.
- **Verification:** The default CLI returns exact exit `32`, zero-call canaries pass, output is bounded/private, and the proof-valid test path exercises the required envelope order.
- **Impact:** The repository has the full fail-closed mechanism, but the tracked default cannot yet emit the real scoped claim. Plan 01-07 may wire the phase runner only through this gate and must preserve manual-required until the missing proof is legitimately supplied.
- **Committed in:** `32a8e29`.

---

**Total deviations:** 3 auto-fixed Rule 1 issues and 1 research-gated architectural decision. **Impact:** Scope remains the approved sentinel contract; the implementation is more conservative and refuses a current-host claim rather than relying on incomplete service semantics.

## Issues Encountered

- macOS temporary and filesystem aliases require canonical rooted comparisons; tests compare descriptor-backed canonical containment rather than user-visible alias spellings.
- `go run` emits its own fixed `exit status 32` wrapper when the tested CLI intentionally returns `manual-required`; E2E tests accept only that exact wrapper diagnostic while requiring structured bounded decision output.
- Official Apple launchd material available during implementation was insufficient for the required modern service-read proof. This remains an explicit manual proof gap, not an implicit pass or fallback.

## User Setup Required

None - no package installation, credential, network access, host activation, real HOME/manager/service observation through a sentinel adapter, or real-machine configuration change was performed. Work remained limited to repository implementation, isolated synthetic tests, and the required atomic Git commits. The tracked default real gate remains `manual-required` until the service proof is complete.

## Next Phase Readiness

- Plan 01-06 can consume the exact four-state verdict and bounded evidence contracts for diagnostics without acquiring host-mutation or overclaim capability.
- Plan 01-07 must wrap the complete phase runner with the controlled real envelope; it must not treat the current standalone proof gate as the final outer wiring.
- The tracked launchctl proof remains deliberately missing. Until current official no-side-effect semantics and matching isolated negative evidence are recorded, any attempted real run must stop at `indeterminate/manual-required` with exit `32` before adapter or workload execution.
- `sentinel-manifest`, `sentinel-verdicts`, `real-sentinel-envelope`, `sentinels`, and the existing `walking-skeleton` regression are green without touching actual HOME, manager state, services, or tracked worktree/index state through real adapters.

## Self-Check: PASSED

- All eleven created and three modified implementation files exist; task commits `044c0c8`, `6b99f4d`, and `32a8e29` are present and delete no tracked files.
- Bash syntax, `sentinel-manifest`, `sentinel-verdicts`, `real-sentinel-envelope`, `wave sentinels`, and `walking-skeleton` pass after the final commit.
- Commit-parent runner diffs introduce exactly the plan-owned literal labels; fixed package/pattern selection and permanent generic/malformed negative behavior remain bounded and non-zero.
- Both JSON manifests parse; exact source/test digests match; proof-missing zero-call canaries and proof-valid isolated envelope ordering pass.
- Exact task staging, cached diff checks, targeted physical-path/identity/credential scans, and staged Gitleaks passed with zero leaks.
- No real Nix, Homebrew, mise, uv, rustup, launchctl, service, HOME, manager, network, or host-state command ran. Existing user changes in `CLAUDE.md`, `.ai/`, and `.config/alma/` remained unstaged and unchanged.

---
*Phase: 01-safety-privacy-and-state-foundation*
*Completed: 2026-07-10*
