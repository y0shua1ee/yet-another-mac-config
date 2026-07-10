---
phase: 01-safety-privacy-and-state-foundation
plan: "02"
subsystem: safety-artifacts
tags: [go, canonical-json, content-addressed-store, exact-lineage, lifecycle, offline]

requires:
  - phase: 01-safety-privacy-and-state-foundation
    plan: "01"
    provides: isolated offline runner, external synthetic roots, walking-skeleton CLI, and the initial six-kind artifact round trip
provides:
  - closed payload validators and restricted canonical JSON for exactly six artifact kinds
  - apply and read-only exact-digest lineage with an independent post-receipt observed-state artifact
  - immutable content-addressed writes, durable transitive pins, persisted terminal transitions, and graph-aware rollback
  - bounded validate/store CLI routes and synthetic substitution, staleness, lifecycle, reopen, and latest-alias negatives
affects: [01-03-privacy, 01-04-fixtures, 01-05-sentinels, recovery-artifacts]

tech-stack:
  added: []
  patterns: [closed typed payloads, restricted canonical JSON, exact digest graph, seven instances across six kinds, persistent pin reconstruction, immutable hard-link commit]

key-files:
  created:
    - safety/internal/artifact/kinds.go
    - safety/internal/artifact/canonical.go
    - safety/internal/artifact/lineage.go
    - safety/internal/artifact/validate_test.go
    - safety/internal/e2e/artifact_cli_test.go
    - safety/testdata/artifacts/kind-cases.json
    - safety/testdata/artifacts/lineage-cases.json
  modified:
    - safety/scripts/test.sh
    - safety/internal/artifact/envelope.go
    - safety/internal/artifact/store.go
    - safety/internal/workflow/synthetic.go
    - safety/cmd/yamc-safety/main.go
    - safety/internal/e2e/walking_skeleton_test.go

key-decisions:
  - "Supersede Plan 01's embedded fresh-record compromise: keep exactly six closed artifact kinds while storing seven apply-path object instances, with the post-receipt observation as a second full observed-state envelope."
  - "Persist plan transitions outside immutable artifact bytes and rebuild a fully revalidated digest-reference graph on Store reopen so terminal state, expiry, and transitive pins survive process boundaries."
  - "Validate every graph and preflight every exact object key before writing; commit with immutable links and remove only objects created by a failed graph write."

patterns-established:
  - "Typed fail-closed validation: common envelope fields never bypass the kind-specific closed payload decoder or lifecycle table."
  - "Digest-only authority: run IDs, paths, filenames, mtimes, directory co-location, and latest aliases never establish lineage."
  - "Durable lifecycle graph: only validated exact references can pin ancestors, and persisted transitions are revalidated before reuse."

requirements-completed: [SAFE-01]

duration: 40 min
completed: 2026-07-10
---

# Phase 01 Plan 02: Artifact Contracts and Exact Lineage Summary

**Six closed artifact schemas now form durable apply/read-only digest graphs with seven apply-path objects, persistent lifecycle pins, and fail-closed immutable storage.**

## Performance

- **Duration:** 40 min
- **Started:** 2026-07-10T14:42:56Z
- **Completed:** 2026-07-10T15:23:10Z
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments

- Added a closed registry of exactly six typed payload contracts, restricted canonical JSON, duplicate-key and numeric-form rejection, stable error codes, and digest recomputation over every envelope field except `content_digest`.
- Added exact apply and read-only lineage validation. Apply now stores both the pre-apply and post-receipt observations as distinct `observed-state` envelopes, while evidence and reports bind only revalidated exact digests.
- Completed lifecycle enforcement with 24-hour snapshots, append-only plans/evidence, immutable object writes, graph-aware zero-partial-write preflight/rollback, persisted terminal transitions, and transitive pins reconstructed after Store reopen.
- Extended the bounded CLI and fixed runner with `artifact-kinds`, `artifact-lineage`, and `artifact-contracts` routes; all routes run offline below fresh external roots and reject future or unknown suites.
- Migrated the walking skeleton to seven content-addressed instances across exactly six kinds without changing the synthetic-only claim ceiling or touching the live Mac.

## Task Commits

Each task was committed atomically:

1. **Task 01-02-01: 封闭六类 payload 与 canonical envelope** - `0e412fd` (feat)
2. **Task 01-02-02: 强制 exact digest lineage 与 immutable store reads** - `52d2576` (feat)

_Plan metadata is committed together with this summary._

## Files Created/Modified

- `safety/scripts/test.sh` - Adds exact literal task routes and a two-handler artifact contract wave, each with a new isolated external root.
- `safety/internal/artifact/envelope.go` - Integrates storage/lifecycle policy into the canonical envelope and separates tracked-golden validation from writable runtime state.
- `safety/internal/artifact/kinds.go` - Defines exactly six closed payload schemas and their storage/lifecycle contracts.
- `safety/internal/artifact/canonical.go` - Implements restricted canonical JSON, duplicate-key detection, integer-only numbers, UTF-8 checks, and stable contract errors.
- `safety/internal/artifact/lineage.go` - Validates apply/read-only graphs, ordered operations, fresh observation provenance, and report evidence edges.
- `safety/internal/artifact/store.go` - Implements bounded immutable writes, exact reads, graph preflight/rollback, lifecycle deletion, durable transitions, and reopen-time pin reconstruction.
- `safety/internal/workflow/synthetic.go` - Migrates the fixture round trip to two independent observed-state envelopes and `WriteGraph`.
- `safety/cmd/yamc-safety/main.go` - Adds bounded `validate --expect-kind` and explicit-path `store` routes, including required apply-path fresh observation input.
- `safety/internal/artifact/validate_test.go` - Covers kind substitution, canonicalization, storage class, lifecycle, and stable rejection behavior.
- `safety/internal/e2e/artifact_cli_test.go` - Covers exact lineage, stale/substituted edges, lifecycle, persistent pins/transitions, atomic late failure, CLI paths, and runner contracts.
- `safety/internal/e2e/walking_skeleton_test.go` - Verifies seven stored instances, exactly six kinds, two independent observations, and exact plan/evidence edges.
- `safety/testdata/artifacts/kind-cases.json` - Contains reviewed logical-only synthetic kind cases.
- `safety/testdata/artifacts/lineage-cases.json` - Contains reviewed apply/read-only, substitution, staleness, freshness, and latest-selection cases.

## Decisions Made

- The six-kind registry remains closed. Apply runs store seven object instances because the fresh post-receipt observation is a second full `observed-state`; the compact evidence descriptor is reference metadata only.
- Generated plan bytes are always initially nonterminal and immutable. Applied/abandoned transitions are explicit immutable side records revalidated on reopen; caller-preterminal plan bytes are rejected.
- Reopen reconstructs the reference graph from validated content-addressed objects. Directory entries never select an upstream artifact; only schema-defined exact digests have integrity authority.
- A graph is validated and collision-preflighted in full before the first write. The commit path uses hard links and tracks only newly created objects for rollback.

## Deviations from Plan

None from the final amended plan. Two blocking omissions in the plan contract were repaired before their affected commits:

- `abc7e82` added incremental runner ownership and exact task/wave route files to all Phase 1 plan whitelists.
- `6de70c5` added the walking-skeleton schema migration files and made the seven-instance/six-kind/two-observation contract explicit.

The implementation then followed the amended eight-file Task 2 boundary exactly.

## Issues Encountered

- Task 1 initially asserted that every route beyond the first new handler must remain unsupported; that also rejected Task 2's now-authorized handler. The assertion was narrowed to Phase 3+ routes and the Task 1 commit was amended before Task 2 was committed.
- A pre-commit security review identified four incomplete semantics in the first Task 2 draft: embedded-only fresh observation, process-local pins/transitions, caller-preterminal plans, and partial graph writes on late failure. The implementation and regression tests were expanded to satisfy the plan before any Task 2 commit was created. A final independent review returned PASS with no HIGH or blocking findings.
- The runner intentionally bounds and hides raw Go failures. All diagnosis stayed inside the runner-owned empty environment and repository fixture paths; no direct live probe, package manager, network, real HOME, or service command was used.

## User Setup Required

None - no external service, package installation, secret, or host activation is required. Missing local Go remains `manual-required` rather than triggering bootstrap or download.

## Next Phase Readiness

- Plan 01-03 can build the one-way privacy gate on stable typed envelopes, exact digest lineage, bounded CLI errors, and durable external-local-state storage.
- Root README, local `safety/CLAUDE.md`, the `safety/AGENTS.md` symlink, and final phase documentation remain scheduled for Plan 01-07.
- No real apply, Nix/Homebrew/mise/uv/rustup command, live probe, network access, HOME mutation, service mutation, or current-host claim path exists.

## Self-Check: PASSED

- Both task commits are present and contain only their exact plan whitelists.
- `task artifact-kinds`, `task artifact-lineage`, `wave artifact-contracts`, `task walking-skeleton`, and `wave skeleton` pass from fresh external roots.
- Phase 3+ and unknown routes return non-zero `unsupported-suite` and are not accepted as TDD RED.
- Scoped diff checks, targeted path/identity/credential scans, and staged Gitleaks pass; no real path, identity, endpoint, credential, or private state was added.
- Existing user changes in `CLAUDE.md`, `.ai/`, and `.config/alma/` remained unstaged and unchanged by this plan.

---
*Phase: 01-safety-privacy-and-state-foundation*
*Completed: 2026-07-10*
