---
phase: 01-safety-privacy-and-state-foundation
plan: "01"
subsystem: safety-testing
tags: [go, synthetic-fixture, content-addressed-store, sentinel, offline]

requires: []
provides:
  - stdlib-only offline safety runner with external HOME, XDG, cache, and manager roots
  - six-kind content-addressed synthetic artifact round trip with exact digest lineage
  - synthetic protected-surface evidence whose only success state is synthetic-sentinel-passed
affects: [01-02-artifact-contracts, 01-03-privacy, 01-04-fixtures, 01-05-sentinels]

tech-stack:
  added: [Go standard library testing]
  patterns: [closed artifact kinds, recomputed SHA-256 store keys, external synthetic roots, bounded CLI output]

key-files:
  created:
    - safety/go.mod
    - safety/scripts/test.sh
    - safety/internal/e2e/walking_skeleton_test.go
    - safety/testdata/blueprints/walking-skeleton/input.json
    - safety/testdata/blueprints/walking-skeleton/protected-surfaces.json
    - safety/cmd/yamc-safety/main.go
    - safety/internal/artifact/envelope.go
    - safety/internal/artifact/store.go
    - safety/internal/workflow/synthetic.go
    - safety/internal/sentinel/synthetic.go
  modified: []

key-decisions:
  - "Persist the post-receipt fresh observation as its own digested record inside verification evidence, preserving exactly six top-level artifact objects and six distinct kinds."
  - "Reject all unsupported claim requests through the closed CLI argument set; synthetic output contains only synthetic-sentinel-passed."

patterns-established:
  - "External-only execution: every test run builds an allowlisted environment under a new marker-owned system temporary root."
  - "Exact lineage: each downstream artifact names verified upstream content digests rather than run IDs, filenames, mtimes, or latest aliases."
  - "Safe CLI errors: rejected inputs produce fixed error codes without echoing physical roots, arguments, or raw errors."

requirements-completed: [SAFE-01, SAFE-04]

duration: 11 min
completed: 2026-07-10
---

# Phase 01 Plan 01: Isolated Safety Round Trip Summary

**A local `fixture run` now produces six digest-addressed synthetic artifacts with exact lineage and fresh sentinel evidence entirely below an external root.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-07-10T14:01:27Z
- **Completed:** 2026-07-10T14:13:19Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Added a strict Bash task/wave runner that starts from an empty environment, disables Go network/toolchain fallback, and isolates HOME, XDG, temporary, cache, and manager roots.
- Added a closed six-kind envelope plus recomputed SHA-256 content-addressed storage and an exact desired → observed → plan → receipt → evidence → report lineage.
- Added an executable synthetic CLI round trip with before/after protected-surface snapshots, a stored post-receipt fresh observation, containment negatives, and an enforced synthetic-only claim ceiling.

## Task Commits

Each task was committed atomically:

1. **Task 01-01-01: 固定失败的 external walking-skeleton E2E** - `4a75ab5` (test)
2. **Task 01-01-02: 让六类 artifact 的真实 CLI round trip 通过** - `4558f6c` (feat)

_Plan metadata is committed together with this summary._

## Files Created/Modified

- `safety/go.mod` - Declares the dependency-free Go 1.26 module boundary.
- `safety/scripts/test.sh` - Runs fixed task/wave suites with a fresh external allowlisted environment and bounded status output.
- `safety/internal/e2e/walking_skeleton_test.go` - Locks RED→GREEN, lineage, containment, privacy, and overclaim behavior.
- `safety/testdata/blueprints/walking-skeleton/input.json` - Supplies public logical-only desired, observed, postcondition, and operation input.
- `safety/testdata/blueprints/walking-skeleton/protected-surfaces.json` - Names one synthetic fixture-scoped protected surface.
- `safety/cmd/yamc-safety/main.go` - Exposes only the fixed `fixture run` interaction and safe error envelopes.
- `safety/internal/artifact/envelope.go` - Defines the closed kind registry, common envelope, canonical encoding, and digest verification.
- `safety/internal/artifact/store.go` - Rejects repository overlap/traversal and atomically writes verified canonical bytes by digest.
- `safety/internal/workflow/synthetic.go` - Orchestrates the six-artifact synthetic run, fake adapter, fresh observation, and report.
- `safety/internal/sentinel/synthetic.go` - Parses logical-only manifests and compares named synthetic surface snapshots.

## Decisions Made

- The exactly-six-object requirement and the fresh-observation requirement are both satisfied by persisting a separately digested post-receipt observation record inside verification evidence; it is not substituted by the initial observed-state artifact or the receipt.
- Synthetic claim requests are deny-by-default at CLI parsing. There is no real-surface claim selector or fallback path in production code.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Completed the strict E2E envelope shape**
- **Found during:** Task 01-01-02 read-first review
- **Issue:** The RED test used unknown-field rejection but initially omitted the required `run`, `producer`, and `provenance` envelope fields, which would have rejected a conforming implementation.
- **Fix:** Added all three common-envelope fields to the strict test decoder without weakening any payload, digest, privacy, or claim assertion.
- **Files modified:** `safety/internal/e2e/walking_skeleton_test.go`, `safety/scripts/test.sh`
- **Verification:** The exported RED commit still returns only `expected-red-observed`; the current GREEN task and wave both pass.
- **Committed in:** `4a75ab5`

**2. [Rule 1 - Bug] Preserved traversal syntax in the negative CLI case**
- **Found during:** Task 01-01-02 GREEN verification
- **Issue:** `filepath.Join` normalized `store/../escape` before invoking the CLI, so the test did not exercise the raw traversal guard.
- **Fix:** Constructed the negative argument without pre-cleaning, then re-exported the RED commit and reran its expected-failure wrapper.
- **Files modified:** `safety/internal/e2e/walking_skeleton_test.go`
- **Verification:** The traversal route now exits non-zero before creating its target, while the full walking-skeleton suite passes.
- **Committed in:** `4a75ab5`

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs).
**Impact on plan:** Both fixes strengthened the committed RED contract and kept the implementation within the original five-file-per-task scope; no live capability or broader architecture was added.

## Issues Encountered

- The normal runner intentionally hides raw Go test output. One failing GREEN iteration was diagnosed with a one-off verbose test in an equivalent empty offline environment under a marker-owned external temporary root; no real HOME, cache, manager state, or network path was used.

## User Setup Required

None - no external service configuration or dependency installation is required. If local Go is unavailable, the runner returns `manual-required` without bootstrapping it.

## Next Phase Readiness

- Ready for Plan 01-02 to refine the walking-skeleton envelope into closed per-kind schemas, canonicalization negatives, and complete lifecycle/lineage validation.
- Root README, local `safety/CLAUDE.md`, the `safety/AGENTS.md` symlink, and final phase documentation remain explicitly scheduled for Plan 01-07.
- No real apply, live probe, network, service, manager, HOME, or current-host claim path exists.

## Self-Check: PASSED

- All ten key files exist and both task commits are present.
- The exported RED commit reports the intended missing-capability failure only.
- Current `task walking-skeleton` and `wave skeleton` runs pass from fresh external roots.
- Exact lineage, containment, synthetic claim ceiling, scoped whitespace, and summary privacy checks pass.

---
*Phase: 01-safety-privacy-and-state-foundation*
*Completed: 2026-07-10*
