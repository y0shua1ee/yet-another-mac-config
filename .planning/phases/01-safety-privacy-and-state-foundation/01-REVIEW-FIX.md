---
phase: 01-safety-privacy-and-state-foundation
status: fixed
findings_in_scope: 6
fixed: 6
skipped: 0
iteration: 1
---

# Phase 01 Code Review Fix Report

## WR-01 — Fixed

- Files: `safety/internal/fixture/root.go`, `safety/internal/fixture/fixture_test.go`
- Commit: `2187a1c`
- Change: fixture initialization is transactional; rollback is limited to the same freshly-created direct child after containment, inode, UID, nonce, and marker checks.
- Tests: `./safety/scripts/test.sh task fixture-lifecycle`
- Status: fixed

## WR-02 — Fixed

- Files: `safety/internal/sentinel/snapshot.go`, `safety/internal/sentinel/sentinel_test.go`, `safety/internal/sentinel/real_test.go`
- Commits: `d4ffe0e`, `348d8ab`
- Change: synthetic regular-file snapshots now use non-following file descriptors, size prechecks, `remaining+1` streaming limits, wall-deadline checks, and before/after identity validation.
- Tests: `./safety/scripts/test.sh task sentinel-manifest`; `./safety/scripts/test.sh task real-sentinel-envelope`
- Status: fixed

## WR-03 — Fixed

- Files: `safety/internal/artifact/store.go`, `safety/internal/e2e/artifact_cli_test.go`, `safety/README.md`, `safety/CLAUDE.md`
- Commit: `4bb1c7e`
- Change: external snapshot ingestion now binds lifecycle timestamps to `store.now()` with a documented two-minute positive skew allowance across write, reopen, read, and delete.
- Tests: `./safety/scripts/test.sh task artifact-lineage`
- Status: fixed

## CR-02 — Fixed

- Files: artifact envelope/kind contracts, shared privacy gate/tests, synthetic blueprint/raw/canary fixtures, affected E2E tests, and `safety/` docs.
- Commit: `93c5105`
- Change: string-bearing fields now use field-specific closed validators for public IDs, enums, logical refs, digests, HMAC tokens, and timestamps. Unregistered free strings fail closed. Allowed-field canaries cover `run_id`, `suite_id`, `state`, `reason`, `status`, and `operation_ids` through artifact construction, CLI/rendering, and `Store.Write` without echo or persistence.
- Tests: `task privacy-boundary`, `task artifact-kinds`, `task artifact-lineage`, `task walking-skeleton`; all six component waves (`skeleton`, `artifact-contracts`, `privacy`, `fixture-policy`, `sentinels`, `controlplane`).
- Status: fixed

## CR-01 — Fixed

- Files: sentinel claim/evidence path, workflow report builder, report CLI/E2E, expected report fixture, real-adapter/offline manifest bindings, privacy status contract, and root/safety docs.
- Commit: `1f3e017`
- Change: standalone/replay reports now emit `synthetic-report-claim-ineligible` with no passed verdict, scoped claim, outer sequence, surface tokens, or claim binding. A claimed report can only be built by the `RunRealEnvelope` one-shot consumer from actual evidence/evaluation via `RequestClaim`, binding evidence/suite/manifest/window digests and actual per-surface evidence. The capability is consumed before the envelope returns.
- Tests: `task phase-e2e`, `task sentinel-verdicts`, `task real-sentinel-envelope`, `wave sentinels`; tracked manifest/source digests and current-host manual-required zero-call gate revalidated.
- Status: fixed

## WR-04 — Fixed

- Files: `safety/scripts/test.sh`, `safety/testdata/runner/block-helper.sh`, `safety/internal/e2e/phase_e2e_test.go`, `safety/internal/e2e/real_sentinel_cli_test.go`, `safety/README.md`, `safety/CLAUDE.md`, `README.md`
- Commit: `ac34ec3`
- Change: every task, wave, and phase runner now enters one lifecycle watchdog before argument parsing or setup. The 15/47/305-second budgets cover setup, fixed documentation checks, build/list/test, child dispatch, descendant termination, and marker-owned cleanup. Parent aggregators accept and propagate only the exact bounded timeout envelope from self-limiting children.
- Tests: fixed blocking-helper canaries for setup, docs, and pre-child dispatch assert bounded wall time, exit `124`, one exact `runner-deadline-exceeded` envelope, no live helper PID, no helper marker, and no added `/tmp/yamc-safety.*` root; `/bin/bash -n safety/scripts/test.sh`; `task phase-e2e`; `task real-sentinel-envelope`; `task docs-and-phase-gate`; `wave phase-integration`; full `phase`; staged privacy scan and Gitleaks.
- Status: fixed
