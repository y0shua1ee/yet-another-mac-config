---
phase: 01-safety-privacy-and-state-foundation
status: fixed
findings_in_scope: 4
fixed: 4
skipped: 0
iteration: 3
---

# Phase 01 Code Review Fix Report

## CR-01 — Fixed

- Files: rooted artifact-store filesystem layer, store lifecycle and CLI preflight, artifact unit/E2E negatives, and safety documentation.
- Commit: `b7e5519`
- Change: the store root is now created or validated through an identity-bound rooted parent handle; `sha256` and `transitions` are exact direct children created through that root handle, opened as retained `os.Root` handles, and bound to the initial parent/root/child filesystem identities. Every list/read/create/link/remove path revalidates the named root, named child, opened handle identity, and resolved containment before and after the operation. Immutable objects and transition records use rooted exclusive temporary files plus rooted hard-link publication and rooted rollback; they no longer use path-following `MkdirAll`, `CreateTemp`, `Link`, or `Remove` on store children.
- Read boundary: object and transition reads require a no-follow/nonblocking regular file, precheck size, read at most `limit+1`, and compare named/opened identity, mode, size, and mtime before and after. Symlink, FIFO, directory, device/socket-shaped non-regular entries, oversize data, replacement, or uncertainty fail bounded and non-zero.
- Tests: pre-existing `sha256` symlink, pre-existing `transitions` symlink, symlinked digest object, FIFO digest object, post-open `sha256` replacement, post-open `transitions` replacement, and FIFO transition record. Every case proves no write in the escape target or original moved child; FIFO cases return within the bounded window. Invalid read-only CLI lineage is also rejected before store-root creation.
- Verification: `task artifact-kinds`; `task artifact-lineage`; `task walking-skeleton`; `task phase-e2e`; all seven waves; full phase.
- Status: fixed

## WR-01 — Fixed

- Files: public offline runner, phase/real-sentinel E2E canaries, root/safety documentation.
- Commit: `d4a58f1`
- Change: removed the caller-selectable watchdog re-exec handshake entirely. Every public `test.sh` invocation now unconditionally starts the watchdog in a scrubbed environment; the watchdog reads one fixed, size-bounded embedded body from the same script and executes it in the monitored process group. There is no internal argv mode and no environment/PID/inherited-FD/nonce combination that skips wrapper creation.
- Tests: the deadline matrix now includes a self-consistent direct parent with a live inherited FD and matching 64-hex nonce, in addition to setup/docs/child, PID-only, and stale guard cases. All cases return the single `runner-deadline-exceeded` envelope with exit `124`, terminate the fixed helper/process group, remove its marker and marker-owned runner root, and stay within the wall bound. Test-only 800 ms budgets use a 200 ms termination grace; production 15/47/305-second budgets retain the 500 ms grace.
- Verification: Bash syntax; `task phase-e2e`; `task real-sentinel-envelope`; `task docs-and-phase-gate`; `wave phase-integration`; all tasks/waves/full phase.
- Status: fixed

## WR-02 — Fixed

- Files: tracked-input reader/proof, walking-skeleton Git fixtures, root/safety documentation.
- Commit: `47c84ad`
- Change: bounded worktree reads now use no-follow/nonblocking open plus before/opened/after identity, mode, size, and mtime checks. The mode observed with the exact consumed bytes is mapped to Git `100644` or `100755` from its executable bits and must equal both the unique stage-0 index entry and frozen HEAD tree mode before the HEAD blob is accepted as those consumed bytes.
- Tests: canonical temporary Git roots now prove the existing negatives reach the intended Git gate. Added `100644 -> 100755` and `100755 -> 100644` chmod-only worktree drift cases; both fail before fixture/store creation while byte substitution, index substitution, untracked, ignored, symlink, and non-worktree negatives remain intact.
- Verification: `task walking-skeleton`; `task no-destructive-defaults`; all tasks/waves/full phase.
- Status: fixed

## WR-03 — Fixed

- Files: read-only lineage validator, artifact/store/CLI E2E, safety documentation.
- Commit: `2b8f368`
- Change: read-only freshness now requires `FreshObserved.State` to be present in the typed facts of the exact observed-state object, matching the apply-path semantic check. Digest, content digest, scope, provenance, and lexical logical-ref validity cannot substitute for the observed fact.
- Tests: correct-state control, absent-state, and different-valid-logical-state cases cover `ValidateLineage`; the invalid graph is also rejected by `Store.WriteGraph` with zero objects and by public CLI `store --mode read-only` before store-root creation.
- Verification: `task artifact-lineage`; all tasks/waves/full phase.
- Status: fixed

## Final Regression

- Tasks: all 14 fixed task routes passed: `walking-skeleton`, `artifact-kinds`, `artifact-lineage`, `privacy-boundary`, `bounded-capture`, `fixture-lifecycle`, `tier-network-policy`, `sentinel-manifest`, `sentinel-verdicts`, `real-sentinel-envelope`, `controlplane-contract`, `no-destructive-defaults`, `phase-e2e`, and `docs-and-phase-gate`.
- Waves: all seven fixed waves passed: `skeleton`, `artifact-contracts`, `privacy`, `fixture-policy`, `sentinels`, `controlplane`, and `phase-integration`.
- Phase: `./safety/scripts/test.sh phase` passed with `synthetic-sentinel-passed`.
- Static checks: `/bin/bash -n safety/scripts/test.sh`, `gofmt -d` over every `safety/**/*.go`, and an isolated offline `go vet ./...` all passed.
- Privacy and commit hygiene: each finding used an atomic English commit with exact-file staging, cached diff checks, targeted path/identity/credential review, and staged Gitleaks. Final `gitleaks detect --no-git --source safety --redact --no-banner` scanned the full safety tree with no leak. Nothing was pushed.
- Residual boundary: the tracked `launchctl print` isolated negative proof remains intentionally missing. Current-host execution still returns `manual-required` / `32` / `indeterminate` before every adapter and workload call; no live host probe, service command, network, install, activation, switch, update, or cleanup was run.
