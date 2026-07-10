---
phase: 01-safety-privacy-and-state-foundation
plan: "05"
subsystem: safety-sentinel-evidence
tags: [go, protected-surfaces, read-only-adapters, hmac, four-state-verdict, fail-closed, symlink-safety, runner-deadline, process-group]

requires:
  - phase: 01-safety-privacy-and-state-foundation
    plan: "04"
    provides: marker-owned external fixtures, frozen-primary teardown, closed offline tiers, and proof-gated live policy
provides:
  - exact six-surface/five-domain protected manifest with bounded per-run HMAC snapshots
  - strict four-state evidence, exit, and scoped-claim contract with a structural synthetic ceiling
  - source-bound exact read-only adapter registry and monotonic real-before/real-after envelope mechanism
  - fail-closed manual-required behavior while modern launchctl read-only semantics remain unproved
  - stable 15/47/305-second isolated runner contract with exact 124 propagation and no orphan descendants
affects: [01-06-diagnostics, 01-07-phase-e2e, future-read-only-inspection, current-host-evidence]

tech-stack:
  added: []
  patterns: [frozen manifest scope, per-run opaque HMAC, source-bound adapter capability, rooted read-only observation, monotonic outer envelope, shared outer context, fixed-child serial waves]

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
    - safety/internal/e2e/artifact_cli_test.go
    - safety/internal/e2e/tier_cli_test.go

key-decisions:
  - "Allow only the exact six logical refs across five closed domains; freeze manifest policy before observation and render only logical refs plus per-run opaque tokens."
  - "Reserve covered-surfaces-unchanged-for-run for complete real evidence; synthetic evidence is structurally limited to synthetic-sentinel-passed."
  - "Treat proof metadata as necessary but insufficient: exact adapters require a private registry-issued source-bound capability, current implementation and negative-suite digests, and one shared bounded observation window."
  - "Keep the tracked launchctl proof explicitly missing and return indeterminate/manual-required with exit 32 before any adapter or workload call until current official no-side-effect semantics are established."
  - "Expose the controlled real-envelope mechanism in Plan 01-05, while leaving phase-runner outer-envelope wiring to Plan 01-07."
  - "Give every real-envelope stage one caller-supplied outer context/deadline, freeze the authorized adapter set before workload execution, and preserve exact deadline exit 124 through every runner layer."

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
- Remediated the intermittent fresh-cache runner failure with exact `15/47/305`-second budgets, one compilation per exact suite, serial fixed children without nested process-group wrappers, exact exit-`124` propagation, and cleanup of both timed-out and normally orphaned descendants.
- Hardened the real envelope after review: before/workload/finalize/after now share one deadline, the authorized adapter map and resolver are frozen before workload execution, Git version capability is probed once per adapter set, and proof calendar dates are evaluated in UTC.

## Task Commits

Each task was committed atomically:

1. **Task 01-05-01: 校验 protected manifest 并取得 bounded opaque snapshots** - `044c0c8` (feat)
2. **Task 01-05-02: 强制四态 verdict、non-pass exits 与 scoped claim** - `6b99f4d` (feat)
3. **Task 01-05-03: 用 exact real adapters 包裹 isolated workload 并证明 failure monotonicity** - `32a8e29` (feat)

Corrective reliability work was also committed atomically:

4. **Align Phase 1 isolated-runner deadline contract** - `2842686` (fix, planning contract)
5. **Stabilize isolated sentinel deadlines and harden the real envelope** - `436ad78` (fix)

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
- `safety/internal/e2e/real_sentinel_cli_test.go` - Verifies default manual-required behavior, exact exits, bounded private logs, runner deadlines/process groups, UTC proof dates, fixed-child waves, and proof-gated envelope behavior.
- `safety/internal/e2e/artifact_cli_test.go` - Keeps the earlier artifact wave structural contract aligned with fixed `run_wave_child` aggregation and rejects nested deadline wrappers.
- `safety/internal/e2e/tier_cli_test.go` - Keeps fixture-policy aggregation aligned with fixed child handlers and rejects nested deadline wrappers.
- `safety/manifests/protected-surfaces.v1.json` - Tracks the exact six logical protected refs and required/optional/excluded policy without physical identity or path data.
- `safety/manifests/real-adapters.v1.json` - Tracks exact adapter invocations, limits, official-source metadata, source/test digests, and an explicit missing controlled-service proof.
- `safety/scripts/test.sh` - Runs exact suites from one fresh-cache compiled test binary, enforces 15/47/305-second task/wave/phase ceilings, propagates deadline exit 124 unchanged, and cleans complete process groups without nested wave wrappers.
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

**4. [Rule 1 - Reliability] Repaired intermittent fresh-cache deadlines without weakening fail-closed behavior**

- **Found during:** Post-completion replay of the three sentinel tasks and `wave sentinels` from consecutive fresh isolated roots.
- **Issue:** The previous 9-second task/29-second wave budgets could expire while compiling the exact package pairs from a fresh `GOCACHE`. Timeout `124` was then sometimes rewritten as a selection or contract failure; nested process groups could outlive a wave wrapper, and a normally exited direct child could leave descendants running.
- **Fix:** Aligned the planning contract to exact `15/47/305`-second ceilings in `2842686`; each exact suite now compiles once and uses that same binary for list and behavior, every observed `124` is propagated before output-cap checks, waves launch only fixed task children directly after reserving a complete 15-second budget, and the deadline wrapper cleans the original process group after timeout, signal, or normal direct-child exit.
- **Files modified:** `safety/scripts/test.sh`, `safety/internal/e2e/real_sentinel_cli_test.go`, `safety/internal/e2e/artifact_cli_test.go`, `safety/internal/e2e/tier_cli_test.go`.
- **Verification:** Final task timings were 5.23s (`sentinel-manifest`), 7.95s (`sentinel-verdicts`), and 7.81s (`real-sentinel-envelope`). Five consecutive frozen-code sentinel waves passed in 20.48s, 20.66s, 20.63s, 19.69s, and 19.73s. Deadline canaries returned exact `124` with one bounded JSON record even above 64 KiB; both timeout and normal-exit descendant canaries left no marker or orphan.
- **Committed in:** `436ad78`.

**5. [Rule 1 - Evidence Integrity] Froze the post-gate adapter set and shared one outer deadline across the real envelope**

- **Found during:** Adversarial review of cancellation, adapter ownership, proof-calendar rollover, and Git capability probing.
- **Issue:** Before and after observations read a caller-owned adapter map after the proof gate, so workload code could replace or delete an entry. The real stages also lacked one shared caller deadline, proof dates depended on an implicit local calendar, and Git version support could be probed more than once.
- **Fix:** Copied the authorized adapter map and resolver values before the first observation, passed one required outer context through before/workload/finalize-state checks/after, derived bounded child deadlines for every filesystem adapter, cached the Git version capability once per adapter set, and normalized production proof assessment to UTC. Refreshed the source-bound proof material to test digest `sha256:9db0aab399a95926db5874d8c6767d6704b125ed9c5125792bc014411afb4997` and implementation digest `sha256:0f33eba9d72f321aa10a11ee02c3bc7040db4951fb3ff31aae447c8178ca7a69`, together with all five current negative-suite digests.
- **Files modified:** `safety/internal/sentinel/real.go`, `safety/internal/sentinel/real_test.go`, `safety/internal/e2e/real_sentinel_cli_test.go`, `safety/cmd/yamc-safety/main.go`, `safety/manifests/real-adapters.v1.json`.
- **Verification:** Cancellation skips after adapters but still finalizes the marker-owned fixture and returns `indeterminate` without a claim; workload substitution never reaches after observation; resolver mutation cannot redirect a default adapter; Git probes exactly once; hash/registry checks and the complete real task pass.
- **Committed in:** `436ad78`.

### Research-Gated Scope Decision

**6. [Rule 4 - Architectural] Left controlled-service proof missing instead of asserting unsupported launchctl safety**

- **Found during:** Official-documentation research for Task 01-05-03.
- **Issue:** Available public Apple material did not establish current modern `launchctl print` no-side-effect semantics strongly enough to satisfy the plan's current official proof requirement.
- **Decision:** Keep the tracked service proof `missing` with no `valid_until`, fail the production registry as `indeterminate/manual-required`, and assert that neither adapters nor the isolated workload are called. A proof-valid registry exists only in isolated tests to verify the complete mechanism.
- **Files modified:** `safety/manifests/real-adapters.v1.json`, `safety/internal/sentinel/real.go`, `safety/internal/sentinel/real_test.go`, `safety/internal/e2e/real_sentinel_cli_test.go`.
- **Verification:** The default CLI returns exact exit `32`, zero-call canaries pass, output is bounded/private, and the proof-valid test path exercises the required envelope order.
- **Impact:** The repository has the full fail-closed mechanism, but the tracked default cannot yet emit the real scoped claim. Plan 01-07 may wire the phase runner only through this gate and must preserve manual-required until the missing proof is legitimately supplied.
- **Committed in:** `32a8e29`.

---

**Total deviations:** 5 auto-fixed Rule 1 issues and 1 research-gated architectural decision. **Impact:** Scope remains the approved sentinel contract; the runner is deterministic under fresh caches, the real envelope is closed against post-gate substitution, and the implementation still refuses a current-host claim rather than relying on incomplete service semantics.

## Issues Encountered

- macOS temporary and filesystem aliases require canonical rooted comparisons; tests compare descriptor-backed canonical containment rather than user-visible alias spellings.
- `go run` emits its own fixed `exit status 32` wrapper when the tested CLI intentionally returns `manual-required`; E2E tests accept only that exact wrapper diagnostic while requiring structured bounded decision output.
- Official Apple launchd material available during implementation was insufficient for the required modern service-read proof. This remains an explicit manual proof gap, not an implicit pass or fallback.
- Consecutive fresh-cache replay exposed an intermittent first-suite contract failure. Timing samples showed valid manifest/verdict tests crossing the old 9-second wall budget; the underlying deadline exit was being mislabeled. The corrected runner now compiles each exact suite once, preserves `124`, and has more than 26 seconds of measured margin in the final sentinel waves.
- Updating all wave handlers to the shared fixed-child helper invalidated older artifact and tier structural assertions. Those assertions now bind the exact helper calls and explicitly reject reintroduced nested process-group wrappers.

## User Setup Required

None - no package installation, credential, network access, host activation, real HOME/manager/service observation through a sentinel adapter, or real-machine configuration change was performed. Work remained limited to repository implementation, fresh external synthetic roots/caches, fault-injection canaries, and the required atomic Git commits. The tracked default real gate remains `manual-required` until the service proof is complete.

## Next Phase Readiness

- Plan 01-06 can consume the exact four-state verdict and bounded evidence contracts for diagnostics without acquiring host-mutation or overclaim capability.
- Plan 01-07 must wrap the complete phase runner with the controlled real envelope, provide one fresh per-run key and one shared outer context/deadline, and preserve exact `124` through the future phase layer; it must not treat the current standalone proof gate as the final outer wiring.
- The tracked launchctl proof remains deliberately missing. Until current official no-side-effect semantics and matching isolated negative evidence are recorded, any attempted real run must stop at `indeterminate/manual-required` with exit `32` before adapter or workload execution.
- `sentinel-manifest`, `sentinel-verdicts`, `real-sentinel-envelope`, five consecutive `sentinels` waves, `artifact-contracts`, `privacy`, `fixture-policy`, and the existing `walking-skeleton` regression are green without touching actual HOME, manager state, services, or tracked worktree/index state through real adapters.

## Self-Check: PASSED

- All eleven created and five modified implementation/regression files exist; task commits `044c0c8`, `6b99f4d`, and `32a8e29`, planning repair `2842686`, and corrective implementation commit `436ad78` are present and delete no tracked files.
- Bash syntax, `sentinel-manifest`, `sentinel-verdicts`, `real-sentinel-envelope`, five frozen-code `wave sentinels` runs (20.48s/20.66s/20.63s/19.69s/19.73s), `artifact-contracts`, `privacy`, `fixture-policy`, and `walking-skeleton` pass.
- Commit-parent runner diffs introduce exactly the plan-owned literal labels; fixed package/pattern selection and permanent generic/malformed negative behavior remain bounded and non-zero.
- Both JSON manifests parse; exact source/test/negative-suite digests match; proof-missing manual-required/exit-32 zero-call canaries, shared-context ordering, post-gate substitution rejection, and one-probe Git capability checks pass.
- Deadline fault injection proves exact exit `124` and one bounded JSON record even with more than 64 KiB of child output; timeout and normal-exit descendant markers remain absent after the runner returns.
- Exact task staging, cached diff checks, targeted physical-path/identity/credential scans, and staged Gitleaks passed with zero leaks.
- No real Nix, Homebrew, mise, uv, rustup, launchctl, service, HOME, manager, network, or host-state command ran. Existing user changes in `CLAUDE.md`, `.ai/`, and `.config/alma/` remained unstaged and unchanged.

---
*Phase: 01-safety-privacy-and-state-foundation*
*Completed: 2026-07-10*
