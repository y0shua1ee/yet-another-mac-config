---
phase: 01-safety-privacy-and-state-foundation
status: fixed
findings_in_scope: 7
fixed: 7
skipped: 0
iteration: 2
---

# Phase 01 Code Review Fix Report

## CR-01 — Fixed

- Files: public fixture CLI, fixture lifecycle E2E, privacy/no-cleanup regressions, and root/safety documentation.
- Commit: `b385442`
- Change: removed the legacy public `--fixture-root` / `--store-root` execution path. Public fixture runs now accept only an external base plus logical fixture ID, create a fresh marker-owned direct child, derive the artifact store internally, and finalize only that owned child. Existing roots, HOME-shaped roots, fixture/store overlap, in-repository bases, traversal, unsupported modes, overclaims, and pre-existing witnesses all fail before mutation.
- Tests: `task walking-skeleton`; `task bounded-capture`; `task fixture-lifecycle`; `task no-destructive-defaults`; `task docs-and-phase-gate`.
- Status: fixed

## CR-02 — Fixed

- Files: artifact run metadata and validation, privacy gate/tests, canary data, affected CLI/E2E fixtures, and root/safety documentation.
- Commit: `75eed85`
- Change: `run_id` is generated only by the trusted metadata builder as an opaque digest-derived ID; suites and operations use fixed registries. Command results use a closed field/type schema with no default arbitrary-string branch. Neutral identities, opaque credentials, stable machine IDs, unknown keys, and numeric identities are rejected through construction, rendering, CLI, and store paths without echo or persistence.
- Tests: `task privacy-boundary`; `task artifact-kinds`; `task artifact-lineage`; `task walking-skeleton`; `task bounded-capture`; `task fixture-lifecycle`; `task tier-network-policy`; `task sentinel-manifest`; `task sentinel-verdicts`; `task controlplane-contract`; `task no-destructive-defaults`; `task phase-e2e`; `task docs-and-phase-gate`.
- Status: fixed

## CR-03 — Fixed

- Files: real sentinel manager-tree observation/tests, real-adapter proof manifest, offline-suite binding, and safety documentation.
- Commits: `d1d3997`, `9a9bbcd`
- Change: manager-tree observation canonicalizes the exact manager root and resolves every internal symlink through lexical, parent, and final-target containment checks. Relative, absolute, and chained escapes return `symlink-escape` / incomplete with no token or claim; changing an escaped external target can no longer preserve a complete token. Source, negative-suite, adapter, and offline manifest digests were refreshed to bind the corrected implementation.
- Tests: `task real-sentinel-envelope`; `task phase-e2e`; `task docs-and-phase-gate`.
- Status: fixed

## WR-01 — Fixed

- Files: fixture root lifecycle, fixture lifecycle tests, and safety documentation.
- Commit: `45077da`
- Change: ownership markers are written to a same-directory temporary file, synced, closed, published with no-replace semantics, and followed by directory sync. Initialization rollback is bound to the fresh direct child's containment, inode, UID, and nonce capability, so a truncated partial marker is removed safely with the child while sibling/base witnesses remain untouched. Retention still requires a complete marker.
- Tests: `task fixture-lifecycle`; `task docs-and-phase-gate`.
- Status: fixed

## WR-02 — Fixed

- Files: offline runner watchdog, phase E2E watchdog canaries, and root/safety documentation.
- Commit: `3956add`
- Change: runner internal re-exec now requires a direct-parent PID, a private inherited pipe FD, and a random 64-hex nonce read from that pipe. Ambient PID-only and stale PID/FD/nonce values cannot bypass the lifecycle watchdog; internal guard variables are cleared after authentication. Forged/stale canaries retain the exact bounded timeout envelope, exit `124`, and leave no orphan or temporary root.
- Tests: `/bin/bash -n safety/scripts/test.sh`; `task phase-e2e`; `task docs-and-phase-gate`; `wave phase-integration`.
- Status: fixed

## WR-03 — Fixed

- Files: real sentinel envelope/tests, privacy reason registry, real-adapter/offline manifest bindings, and root/safety documentation.
- Commit: `2ec3f4b`
- Change: the public real-envelope options no longer accept caller-owned key material. After the proof gate, production creates a fresh internal 32-byte key for each run; only the same run's before/after snapshots share it, and every return path clears the internal buffer. Package-private deterministic factories support tests without exposing a production key input. Tests prove same-run comparability, different tokens across runs, and key clearing after entropy, workload, and claim-consumer failures.
- Tests: `task real-sentinel-envelope`; `task phase-e2e`; `task docs-and-phase-gate`.
- Status: fixed

## WR-04 — Fixed

- Files: synthetic workflow input validation, walking-skeleton tracked-input negatives, no-destructive command-edge contract, and root/safety documentation.
- Commit: `ff306f8`
- Change: repository inputs use a closed five-operation `/usr/bin/git` plumbing path with a blank environment, disabled hooks/fsmonitor/replace objects/protocol/optional locks/lazy fetch/prompts, and bounded context/output. Each input must be a non-symlink regular file in the exact worktree top-level, have one stage-0 index entry, match the frozen HEAD tree mode/blob, and have HEAD blob bytes equal to the bounded bytes actually consumed. Git absence/query failure, non-worktrees, untracked, ignored, symlinked, worktree-substituted, and index-substituted inputs fail before fixture/store creation.
- Tests: `task walking-skeleton`; `task no-destructive-defaults`; `task phase-e2e`; `task docs-and-phase-gate`.
- Status: fixed

## Final Regression

- Tasks: all 14 fixed task routes passed: `walking-skeleton`, `artifact-kinds`, `artifact-lineage`, `privacy-boundary`, `bounded-capture`, `fixture-lifecycle`, `tier-network-policy`, `sentinel-manifest`, `sentinel-verdicts`, `real-sentinel-envelope`, `controlplane-contract`, `no-destructive-defaults`, `phase-e2e`, and `docs-and-phase-gate`.
- Waves: all seven fixed waves passed: `skeleton`, `artifact-contracts`, `privacy`, `fixture-policy`, `sentinels`, `controlplane`, and `phase-integration`.
- Phase: `./safety/scripts/test.sh phase` passed with `synthetic-sentinel-passed`.
- Commit hygiene: each finding or tightly coupled manifest refresh used an atomic English commit; exact-file staging, cached diff checks, targeted privacy review, and staged Gitleaks passed; nothing was pushed.
- Residual boundary: the tracked `launchctl print` isolated negative proof remains intentionally missing, so current-host execution still stops before adapters/workload as `manual-required` / `indeterminate`. This is a preserved safety boundary, not an unfixed review finding.
