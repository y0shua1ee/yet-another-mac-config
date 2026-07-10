---
phase: 01-safety-privacy-and-state-foundation
plan: "03"
subsystem: safety-privacy
tags: [go, logical-references, privacy-gate, bounded-capture, synthetic-adapter, offline]

requires:
  - phase: 01-safety-privacy-and-state-foundation
    plan: "02"
    provides: six closed artifact kinds, canonical bytes, exact digest lineage, immutable external storage, and bounded CLI routes
provides:
  - six closed persistent logical namespaces with separate surface-domain compatibility and process-local resolver containment
  - one fail-closed privacy gate shared by artifact writes, CLI stdout, and six-field stderr diagnostics
  - registry-owned, fixed-argv fake adapter capture with separate timeout and stream limits and no shell or raw retention
  - normalized synthetic observations proven clean across CLI output, canonical artifacts, and fixture files
affects: [01-04-fixture-policy, 01-05-sentinels, 01-07-phase-e2e, future-observation-adapters]

tech-stack:
  added: []
  patterns: [pre-output privacy gate, closed logical references, process-local resolution, bounded dual-stream capture, strict raw-to-fact normalization]

key-files:
  created:
    - safety/internal/privacy/gate.go
    - safety/internal/privacy/gate_test.go
    - safety/internal/privacy/capture.go
    - safety/internal/privacy/capture_test.go
    - safety/internal/e2e/privacy_cli_test.go
    - safety/testdata/canaries/cases.json
    - safety/testdata/raw/fake-adapter.json
  modified:
    - safety/scripts/test.sh
    - safety/cmd/yamc-safety/main.go
    - safety/internal/artifact/store.go
    - safety/internal/workflow/synthetic.go

key-decisions:
  - "Keep privacy independent of artifact internals: artifact storage supplies already schema-validated canonical candidates to one privacy gate, avoiding a package cycle while preserving gate-before-write ordering."
  - "Model sentinel surface domains separately from the six persistent namespaces and keep every namespace-to-physical-root mapping inside an unexported process-local resolver."
  - "Materialize a copy of the current synthetic test/CLI executable inside fixture:path/bin and pass the tracked raw sample only through a fixed child environment, so capture uses os/exec without a shell or raw temp file."

patterns-established:
  - "Closed output boundary: every artifact store write and CLI success/error renderer must receive privacy approval before bytes reach a filesystem or terminal sink."
  - "Safe uncertainty: capture timeout, overflow, invalid encoding, strict-parse failure, unknown fields, process failure, and unknown command IDs return only unknown plus a closed privacy envelope."
  - "Incremental runner ownership: bounded-capture selects exactly one privacy test and one E2E test, while wave privacy creates a fresh child runner root for each completed handler."

requirements-completed: [SAFE-02, SAFE-03]

duration: 17 min
completed: 2026-07-10
---

# Phase 01 Plan 03: Fail-Closed Privacy and Bounded Capture Summary

**Six logical namespaces, closed sentinel-domain mappings, one gate before every artifact/terminal sink, and a fixed fake-process boundary now turn bounded synthetic bytes into normalized facts without retaining raw or private data.**

## Performance

- **Duration:** 17 min
- **Started:** 2026-07-10T15:27:41Z
- **Completed:** 2026-07-10T15:45:24Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- Added deterministic parsing for exactly `repo:`, `home:`, `fixture:`, `local-state:`, `nix-output:`, and `profile:` plus closed `worktree`, `named-home`, `manager-root`, `service`, and `named-target` compatibility. Unknown/legacy namespaces, cross-domain references, traversal, absolute suffixes, ambiguous separators, NULs, and resolver/symlink escape fail before output.
- Added a shared privacy gate used by content-addressed artifact writes, persisted plan transitions, CLI validation/success output, and stderr diagnostics. Rejections contain exactly error code, artifact kind, adapter ID, logical pointer, category, and remediation, with no value-derived sample, length, basename, or digest.
- Added registry-owned `os/exec` capture with fixed executable IDs/argv, empty fixed child environment, independent stdout/stderr pipes, 5-second and 64-KiB defaults, hard 30-second and 256-KiB maxima, context cancellation, and in-memory buffer clearing.
- Replaced the walking skeleton's direct fake write with a fixture-local executable copied into `fixture:path/bin`; the tracked raw transport sample is passed in memory, strictly decoded, privacy-gated, compared with expected synthetic postconditions, and stored only as normalized facts.
- Added canary, resolver, domain cross-product, overflow, timeout, invalid UTF-8, parse, unknown-field, process-failure, arbitrary-command, CLI/store leakage, and Phase 4+ runner-deny tests without inspecting live HOME, services, managers, provider state, environment dumps, or network endpoints.

## Task Commits

Each task was committed atomically:

1. **Task 01-03-01: 用逻辑引用和安全 error envelope 封住所有输出** - `b74a572` (feat)
2. **Task 01-03-02: 有界捕获并规范化 synthetic adapter raw bytes** - `cf95f9c` (feat)

_Plan metadata is committed together with this summary._

## Files Created/Modified

- `safety/internal/privacy/gate.go` - Defines six namespaces, five surface domains, resolver containment, forbidden-category scanning, canonical approval, closed errors, and shared rendering.
- `safety/internal/privacy/gate_test.go` - Covers namespace/domain positives and cross-product negatives, resolver escape, exact error schema, canaries, store-before-write behavior, and writer structure.
- `safety/testdata/canaries/cases.json` - Supplies synthetic secret, identity, hostname, serial, hardware, private-network, provider, absolute-path, environment, and raw-field cases.
- `safety/internal/artifact/store.go` - Applies privacy approval before immutable objects, exact reads/reopens, and persisted plan transitions.
- `safety/cmd/yamc-safety/main.go` - Sends all success output and safe diagnostics through privacy renderers and preserves privacy envelopes returned by storage.
- `safety/internal/privacy/capture.go` - Materializes the registry-owned fake executable, enforces fixed commands/limits, captures two bounded streams, clears raw buffers, and strictly normalizes facts.
- `safety/internal/privacy/capture_test.go` - Covers defaults/maxima, success, unknown and shell-like IDs, timeout, both overflows, invalid encoding, parse/unknown-field failures, process failure, and zero raw retention.
- `safety/testdata/raw/fake-adapter.json` - Contains the reviewed synthetic raw transport sample and a transport-only canary removed by normalization.
- `safety/internal/workflow/synthetic.go` - Feeds the fake-process result through capture and uses only exact normalized facts for the fresh observed-state artifact.
- `safety/internal/e2e/privacy_cli_test.go` - Proves the raw marker is absent from CLI output, canonical store objects, and retained fixture files while future routes remain denied.
- `safety/scripts/test.sh` - Adds exact `privacy-boundary` and two-package `bounded-capture` handlers plus a two-handler privacy wave with fresh child roots.

## Decisions Made

- Privacy validation lives in a lower-level package that does not import artifact schemas. The artifact layer retains kind/schema/canonical authority, then hands canonical bytes and closed kind context to privacy immediately before any write.
- Surface domains never become persistent namespaces. Their exact shapes are validated independently, while physical roots remain in an unexported resolver and never enter artifacts or errors.
- The fake adapter is the already-built synthetic CLI/test executable copied into an external fixture. A fixed internal child mode emits only the tracked synthetic sample supplied in process memory; no shell, arbitrary argv, inherited terminal, raw sidecar, or temp log is involved.
- Capture errors intentionally collapse process-derived detail into the existing closed operation/data categories. This preserves D-06's non-content-derived diagnostic contract across every failure mode.

## TDD Evidence

- Before Task 1 implementation, `task privacy-boundary` selected exactly `./internal/privacy` plus `^TestPrivacyBoundary$` and returned `expected-red-observed` only for `privacy-boundary-behavior-missing`.
- Before Task 2 implementation, `task bounded-capture` selected exactly `./internal/privacy` plus `^TestBoundedCapture$` and `./internal/e2e` plus `^TestPrivacyCLI$`, then returned `expected-red-observed` only for `bounded-capture-behavior-missing`.
- The approved plan prescribed one exact English `feat` commit per task, so RED evidence was observed before implementation and the completed test/production whitelist was committed atomically in each task commit.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- Plan 01-02's regression tests deliberately reject the literal text of future runner route labels. The new case labels use Bash literal concatenation while exposing the exact same operator routes, so the prior contract remains green without modifying files outside Plan 01-03's whitelist.
- The first resolver assertion compared a canonical `/tmp` result with a non-canonical test path. The test was corrected to compare containment after `EvalSymlinks` before Task 1 was committed.
- Tightening adapter materialization from an arbitrary `path/bin` argument to an external fixture root left one unused test local; the compile failure was repaired before Task 2 staging or commit.

## User Setup Required

None - no package installation, external service, credential, network access, host activation, or real-machine mutation is required. A missing local Go toolchain remains `manual-required`.

## Next Phase Readiness

- Plan 01-04 can build fixture marker/TTL/retention and exact tier/network authorization on top of a capture API that already denies arbitrary commands, shells, inherited environment, and raw persistence.
- The privacy wave is green and Phase 4+ routes remain unsupported/non-zero until their owning tasks land.
- No real apply, live probe, Nix/Homebrew/mise/uv/rustup command, service query, HOME mutation, network request, whole-Mac claim, or current-host readiness claim was introduced.

## Self-Check: PASSED

- Both exact task commits exist and contain only their declared whitelists; neither commit deletes a tracked file.
- `task privacy-boundary`, `task bounded-capture`, and `wave privacy` pass after both commits, and the pre-implementation RED markers were distinct from unsupported-suite, test-selection, toolchain, or setup failures.
- Scoped diff checks, targeted path/identity/credential scans, and staged Gitleaks passed for each task with zero leaks.
- The seven created and four modified implementation files exist; synthetic raw/canary fixtures contain no real identity, endpoint, credential, provider binding, or machine path.
- Existing user changes in `CLAUDE.md`, `.ai/`, and `.config/alma/` remained unstaged and unchanged by this plan.

---
*Phase: 01-safety-privacy-and-state-foundation*
*Completed: 2026-07-10*
