---
phase: 01-safety-privacy-and-state-foundation
verified: 2026-07-11T03:12:37Z
status: passed
score: 14/14 unique must-haves verified
overrides_applied: 0
gaps: []
human_verification: []
---

# Phase 1 Verification Report

**Phase goal:** 操作者拥有一套可验证的安全基础，可以区分恢复 artifacts、隔离测试并证明真实 Mac 未被改变。

**Result:** PASSED

Phase 1 achieves its approved Vertical MVP outcome. The implementation distinguishes and validates the six recovery artifact kinds, enforces privacy before persistence or rendering, confines all default workloads to fresh external fixture roots, rejects mutable/destructive routes, and implements a bounded sentinel envelope whose strongest possible positive claim is `covered-surfaces-unchanged-for-run` for the exact covered run only.

The current-host route is deliberately fail-closed. Because the repository does not contain current official-semantics proof for the required real adapters, that route returns `manual-required`, verdict `indeterminate`, exit `32`, and no claim **before any adapter or workload call**. No live current-host probe, `launchctl`, package-manager operation, service operation, network access, installation, activation, switch, update, repair, or cleanup was run during this verification.

## Vertical MVP Interpretation

`ROADMAP.md` declares `Mode: mvp`, but its Goal is an outcome statement rather than the canonical English `As a ... I want ... so that ...` user-story syntax. The user had already explicitly approved a Vertical MVP, so verification used the following equivalent interpretation without modifying the roadmap:

> As the operator maintaining this canonical Mac-config repository, I want a verifiable safety foundation that distinguishes recovery artifacts, runs tests in isolation, and proves protected real-Mac surfaces unchanged for the exact covered run, so that later recovery/environment work cannot silently damage the current Mac or overclaim readiness.

This is an interpretation of the approved goal, not an override. `overrides_applied` therefore remains `0`.

## User Flow Coverage

| Step | Expected operator-visible behavior | Verification |
|---|---|---|
| 1. Start the default safety phase | `safety/scripts/test.sh phase` accepts only its fixed public route and immediately enters one supervisor with a bounded phase deadline. | VERIFIED — fixed route table and supervisor are implemented in `safety/scripts/test.sh`; the full phase completed successfully. |
| 2. Run the safety workload in isolation | The runner sanitizes its environment, creates fresh repository-external HOME/XDG/tool roots, denies ambient network/install/repair/trust behavior, and owns cleanup only through its fixture marker. | VERIFIED — runner, fixture-root, child-environment, retention, and tier/network tests passed. |
| 3. Produce and inspect recovery evidence | The walking skeleton stores the exact desired → observed → plan → receipt → fresh observed → evidence → report graph, with six distinct public kinds and exact digest lineage. | VERIFIED — graph, kind, canonicalization, lineage, store, privacy, and phase E2E assertions passed. |
| 4. Protect the actual host boundary | A real-host claim is available only inside the controlled before/workload/after envelope and only with current proof-valid adapters. Missing proof stops at `manual-required`/`32`, before adapters and workload. | VERIFIED — zero-call proof-gate tests and phase report assertions passed. No live host route was invoked. |
| 5. Report without overclaiming | Synthetic execution reports `synthetic-sentinel-passed`; one-shot isolated proof-double evidence may bind only `covered-surfaces-unchanged-for-run`; standalone or current-host-ineligible reports emit no scoped claim. Whole-Mac, recovery-ready, current-host-ready, multi-host, and fresh-install claims remain impossible. | VERIFIED — claim-ceiling, provenance, binding, and CLI tests passed. |

**End-to-end outcome:** later phases now have one executable safety boundary for artifacts, fixtures, privacy, mutation policy, and exact-run sentinel evidence. The flow does not claim that this Mac has completed a recovery drill; that belongs to Phase 13.

## Goal Achievement

The seven plans declare 29 truth clauses. Closely overlapping clauses were normalized into 14 non-duplicative observable outcomes for scoring; every original clause remains traceable below.

### Observable Truths

| # | Observable truth | Evidence in implementation | Status |
|---:|---|---|---|
| 1 | Exactly six SAFE-01 recovery artifact kinds are closed, independently validated, and unable to impersonate one another. | Closed kind constants and registry in `safety/internal/artifact/kinds.go` and `safety/internal/privacy/gate.go:43`; cross-kind, unknown-field, duplicate-key, version, canonical-number, and forged-digest cases in `safety/internal/artifact/validate_test.go`. | VERIFIED |
| 2 | Artifact integrity is derived from canonical content and exact digest lineage, including an independently stored fresh post-receipt observation. | `safety/internal/artifact/canonical.go`, `safety/internal/artifact/lineage.go`, graph validation in `safety/internal/artifact/store.go`, and exact graph assertions in `safety/internal/e2e/artifact_cli_test.go` and `walking_skeleton_test.go`. | VERIFIED |
| 3 | Runtime artifacts use a fresh external store with closed lifecycle rules, transitive pinning, a single fresh writer, and physically append-only bytes. | Store creation/graph write/read in `safety/internal/artifact/store.go`; deletion is unconditionally denied after policy validation at `store.go:290-321`; reopen, rewind, pin, expiry, and append-only stabilization tests passed. | VERIFIED |
| 4 | Persisted and rendered data uses logical namespaces and privacy-safe identifiers; real identity, absolute HOME, secret canaries, provider references, private network material, and unconstrained raw output are rejected or structurally reduced first. | Namespace and candidate validation in `safety/internal/privacy/gate.go`; bounded renderer/gate paths at `gate.go:380-489`; adversarial corpus and stdout/stderr tests in `safety/internal/privacy/gate_test.go` and privacy CLI tests. | VERIFIED |
| 5 | Command observation is available only through fixed allowlisted adapters with bounded argv, time, bytes, processes, status, and structured diagnostics; no caller shell or raw dump becomes evidence. | Registry and capture path in `safety/internal/privacy/capture.go:166`; fixed fake/negative adapters and capture E2E tests; source-route checks reject arbitrary shell dispatch. | VERIFIED |
| 6 | Every test workload receives a fresh repository-external, marker-owned fixture with isolated HOME, XDG, manager, cache, data, trust, runtime, and temporary roots; retention occurs only after verdict freeze. | `fixture.Create` in `safety/internal/fixture/root.go:82`, child environment in `environment.go`, marker ownership and teardown in `retention.go`, plus success/failure/keep/escape/symlink tests. | VERIFIED |
| 7 | Default tiers are offline and non-mutating; network, download, install, repair, trust, and live execution require fixed contracts and otherwise fail closed as `unknown` or `manual-required`. Missing local Go never bootstraps a manager. | Tier/network policy in `safety/internal/fixture/network.go`; runner pins `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, `GOENV=off`, and `GOWORK=off`; exact network CLI remains non-executing `manual-required`. | VERIFIED |
| 8 | The protected-surface contract is closed over five required domains and six exact logical references: tracked worktree/index, HOME file, manager data, service state, and named target. | `safety/manifests/protected-surfaces.v1.json`, manifest validator in `safety/internal/sentinel/manifest.go`, and compatibility/coverage assertions in sentinel and phase E2E tests. | VERIFIED |
| 9 | Sentinel evaluation has exactly four monotonic verdict families and precise evidence bindings; only complete/equal required observations can authorize the exact-run scoped claim. | `Evaluate` at `safety/internal/sentinel/verdict.go:193`, `RequestClaim` at `verdict.go:317`, before/after token and binding validation, and complete/violation/indeterminate/harness-error tests. | VERIFIED |
| 10 | Real-host execution is proof-gated and ordered `proof-gate → real-before → isolated-workload → freeze-primary → fixture-finalize → real-after → monotonic-combine`; absent tracked proof stops with zero adapter and workload calls. | Registry assessment and controlled envelope in `safety/internal/sentinel/real.go`; explicit missing-proof zero-call assertion at `safety/internal/sentinel/real_test.go:769-782`; CLI and phase E2E require `manual-required`, `indeterminate`, exit `32`, and no claim. | VERIFIED |
| 11 | Determinate Nix, nix-darwin, and Home Manager have separate typed control-plane responsibilities without any invocation or mutation path. | Typed contracts in `safety/internal/contract/controlplane.go`, fixed cases in `safety/testdata/controlplane/cases.json`, and CLI tests that prove zero Nix/Homebrew/manager calls. | VERIFIED |
| 12 | Declaration, manager binary, managed payload, selected executable, and activation context remain distinct; each `(scope, executable)` has one primary executable owner. | `ValidateOwnership` at `safety/internal/contract/controlplane.go:157` plus duplicate/delegated/module-presence cases in control-plane tests. | VERIFIED |
| 13 | Mutable boundaries and destructive convergence are structurally rejected; `extra` and `unmanaged-present` are report-only and generate no operation. | Policy evaluation in `safety/internal/contract/policy.go`, no-cleanup CLI tests, empty-operation assertions in workflow reporting, and absence of apply/cleanup/trust/service/defaults/link/manager routes. | VERIFIED |
| 14 | The public runner exposes only fixed task/wave/phase routes under one supervisor with 15 s task, 47 s wave, and 305 s phase deadlines, sanitized execution, exact selection, process-group termination, and phase documentation. | `safety/scripts/test.sh:5-47` and fixed dispatch near `test.sh:1435-1467`; deadline/process-tree assertions in `safety/internal/e2e/phase_e2e_test.go`; root `README.md` plus local `safety/CLAUDE.md` and `safety/AGENTS.md` symlink. | VERIFIED |

**Score:** 14/14 unique must-haves verified.

## ROADMAP Success Criteria

| # | Criterion | Result | Evidence and boundary |
|---:|---|---|---|
| 1 | Six artifact kinds can be validated separately and spoofing is rejected. | VERIFIED | Closed schemas, canonicalization, exact lineage, storage, and CLI/E2E negative matrices passed. |
| 2 | The default entry provides sentinel evidence that protected HOME, worktree, global-tool, service, and repository-external surfaces were not changed. | VERIFIED — Vertical MVP boundary | The phase entry verifies the full protected-surface envelope with proof-valid isolated doubles and binds only the exact-run scoped claim. The actual current-host branch is intentionally claim-ineligible because tracked proof is absent; it returns `manual-required`/`32` before all adapters/workload. This verifies the safety mechanism without claiming a current-host drill. |
| 3 | Persisted output contains only logical/privacy-safe data and rejects secrets, real identity, absolute HOME, and unconstrained raw output. | VERIFIED | Gate/render/store/capture adversarial tests passed, and `gitleaks` found no leak in the `safety` tree. |
| 4 | Defaults are isolated/offline/non-mutating; unsafe probes fail closed and extra state is report-only. | VERIFIED | Fixture, network, proof-gate, policy, no-cleanup, exact-route, and full phase tests passed. |

**ROADMAP coverage:** 4/4 success criteria verified under the approved Vertical MVP scope.

## Plan Must-Have Traceability

| Plan | Declared truths | Verified implementation outcome | Result |
|---|---:|---|---|
| `01-01` Walking skeleton | 4 | External-root six-kind graph, exact lineage, synthetic sentinel status and claim ceiling, bounded no-Go `manual-required`. | 4/4 |
| `01-02` Artifact contracts | 4 | Closed schemas/canonicalization, tracked-vs-runtime boundary and lifecycle, fail-whole rejection, exact-digest graph with independent fresh observation. | 4/4 |
| `01-03` Privacy boundary | 3 | Logical namespace model, gate-before-store/render, fixed bounded adapter registry with safe diagnostics. | 3/3 |
| `01-04` Fixture and tier isolation | 4 | Fresh marker-owned external root, fully isolated child environment, explicit offline/integration/live tiers, fail-closed network/mutation contract. | 4/4 |
| `01-05` Sentinel envelope | 4 | Closed protected manifest, four verdicts and exact claim, proof-valid real adapter contract, fixed controlled-envelope ordering and claim ceiling. | 4/4 |
| `01-06` Control plane and no destructive defaults | 4 | D-17 layered owners, D-18 single executable owner, D-19 mutable-route rejection, SAFE-08 report-only extras. | 4/4 |
| `01-07` Phase integration and documentation | 6 | Fixed aggregate, one supervisor, fresh-root/store coherence, privacy-clean integrated report, exact deadline/process-tree behavior, synced documentation and local guidance. | 6/6 |

**Plan truth coverage:** 29/29 declared truth clauses verified.

## Required Artifacts and Wiring

- GSD artifact verification was run against all seven PLAN files: **32/32 declared artifacts exist and satisfy their declared structural checks**.
- GSD key-link verification plus manual source tracing covered **33/33 declared links**.
- The helper automatically recognized 21 links. Its 12 reported misses were anchored test names such as `^TestArtifactKinds$`; those are regex selectors passed as literal runner arguments, so treating them as regexes against shell source produces false negatives. Each literal selector was manually found in `safety/scripts/test.sh`, and each corresponding Go test function exists in the declared package.
- No required artifact is a placeholder or orphan. The integrated data flow is:

```text
tracked synthetic blueprint
  -> fixed CLI route
  -> RunSynthetic
  -> six public kinds / seven stored instances
  -> exact digest graph in fresh append-only external store
  -> BuildPhaseReport
  -> one-shot BindPhaseReport inside the controlled sentinel envelope
  -> privacy gate
  -> bounded JSON result
```

- The real-host branch is independently wired as:

```text
tracked manifest + tracked adapter proof
  -> proof gate
  -> [proof unavailable in Phase 1]
  -> manual-required / indeterminate / exit 32 / no claim
  -> zero adapter calls / zero workload calls
```

## Requirements Coverage

| Requirement | Verification evidence | Status |
|---|---|---|
| SAFE-01 | Six closed artifact schemas, canonical digests, exact graph lineage, anti-impersonation and invalid-store invariants. | SATISFIED |
| SAFE-02 | Logical path/identity schema and recursive gate reject real username, hostname, serial/hardware identity, absolute HOME, and physical resolver roots. | SATISFIED |
| SAFE-03 | Gate-before-render/store/capture rejects secrets, login/provider/private-network material, environment dumps, and raw output; bounded structured diagnostics only. | SATISFIED |
| SAFE-04 | Fresh external marker-owned fixture isolates HOME, XDG, tool config/data/cache/trust/runtime roots and rejects repository/ambient path escape. | SATISFIED |
| SAFE-05 | Default runner is offline and prohibits automatic network, installation, download, repair, or trust mutation; integration contract is exact and still non-executing in Phase 1. | SATISFIED |
| SAFE-06 | Real adapter allowlisting requires both current official-semantics proof and exact negative-suite proof; missing proof stops pre-adapter as `manual-required`. | SATISFIED |
| SAFE-07 | Closed five-domain/six-reference sentinel envelope, fixed ordering, fresh before/after evidence, exact-run scoped claim, and fail-closed actual-host branch. | SATISFIED |
| SAFE-08 | Extra/unmanaged state survives only as report status with an empty operation set; destructive and mutable vocabulary has no executable route. | SATISFIED |

**Requirement coverage:** 8/8 Phase 1 requirements satisfied.

## Verification Execution

All dynamic verification used repository code, synthetic fixtures, or fresh external temporary roots. No current-host adapter or mutable operation was authorized.

| Check | Result |
|---|---|
| `./safety/scripts/test.sh phase` | PASS in approximately 103 s; exact terminal JSON was `{"status":"synthetic-sentinel-passed","suite":"phase"}`. This transitively ran all fixed component waves and the integrated phase gate. |
| `./safety/scripts/test.sh task phase-e2e` | PASS in approximately 7.1 s. |
| `./safety/scripts/test.sh wave phase-integration` repeated five times | PASS 5/5, each in approximately 6–8 s after deadline-matrix stabilization. |
| `/bin/bash -n safety/scripts/test.sh` | PASS. |
| `gofmt` diff over Go sources | PASS; no formatting delta. |
| isolated offline `go vet ./...` with external HOME/cache and `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`, `GOENV=off`, `GOWORK=off` | PASS. |
| `git diff --check -- safety README.md` | PASS. |
| `gitleaks detect --no-git --source safety --redact --no-banner` | PASS; approximately 798 KB scanned with no leak. |
| debt-marker scan (`TODO`, `FIXME`, `HACK`, placeholders, skipped tests) over Phase 1 implementation/docs | PASS; no unresolved marker found. |
| production-route review for shell execution, deletion, service/network/install/apply behavior | PASS; only fixed runner supervision, fixed tracked-Git reads, proof-gated fixed adapters, and marker-owned fixture teardown exist. No arbitrary mutable route was found. |

The verifier initially reproduced a phase-integration deadline flake (2 failures in 5 runs at the 15 s boundary). Verification was paused rather than downgraded. Commit `a49e4a6` parallelized the nine isolated deadline-matrix subtests without weakening their assertions. The final HEAD then passed the phase E2E, the complete phase, and five consecutive phase-integration runs. Each deadline case still checks exit `124`, a unique envelope, PID/PGID termination, `ESRCH`, no late marker, and its wall-clock bound.

Pre-existing temporary test-root baseline entries were not deleted because ownership was not established. The final verifier executions created no additional retained roots, and the baseline remained unchanged.

### Probe Execution

**N/A — no live probes were executed.** This is intentional and required by SAFE-06. The missing tracked current-host adapter proof is itself tested as a fail-closed result, not worked around through an ad-hoc host observation.

## Review, Commit, and Repository State

- All seven PLAN/SUMMARY pairs exist and their implementation commits are present in Git history.
- `01-REVIEW.md` records a clean code review with zero findings for the implementation preceding final deadline stabilization.
- `01-REVIEW-FIX.md` records 4/4 earlier review findings resolved.
- `01-STABILIZATION.md` records 5/5 stabilization findings resolved.
- The final test-only delta in `a49e4a6` was inspected directly; it changes only the deadline matrix's test scheduling and preserves every safety assertion.
- Relevant final history includes `e288599`, `a9c1039`, `4e8a201`, `76a5bfd`, `ff361a1`, `a61c15e`, `ca1d334`, `3b4723d`, and `a49e4a6`.
- The verifier did not commit or push anything.
- Unrelated pre-existing worktree changes (`CLAUDE.md`, `.ai/`, and `.config/alma/`) were not read as implementation evidence, modified, staged, or cleaned.

## Anti-Patterns and Safety Audit

No blocking anti-patterns were found.

- No placeholder implementation, skipped test, broad catch-all route, or user-derived package/test selector exists.
- No code path exposes real apply, install, switch, update, cleanup, uninstall, zap, trust mutation, service/defaults/link mutation, or arbitrary command execution.
- Reviewed `/bin/rm -rf` usage is confined to marker-owned runner fixture cleanup; `RemoveAll` is guarded by fixture ownership checks.
- Reviewed `/bin/sh -c` usage is a fixed process-tree test command, not caller-controlled execution.
- Reviewed `/usr/bin/git` usage is a fixed rooted tracked-input reader over a frozen repository identity.
- Real adapter implementations remain behind exact manifest and proof gates; the current repository state cannot reach them from the current-host route.
- There is no HTTP/DNS/listener path and no manager bootstrap path in the Phase 1 production surface.

## Claim Ceiling and Deferred Evidence

The strongest positive claim implemented by Phase 1 is exactly:

`covered-surfaces-unchanged-for-run`

It is eligible only when produced inside the one-shot controlled envelope with complete, fresh, equal before/after observations for every required protected reference and proof-valid adapter implementations. A standalone phase report cannot mint it. Synthetic-only artifact execution uses `synthetic-sentinel-passed`; the default current-host status remains `manual-required` and claim-ineligible while tracked proof is unavailable.

This report therefore does **not** claim any of the following:

- the whole Mac is unchanged;
- the current Mac is recovery-ready;
- current-host activation or restoration has been verified;
- a clean or fresh installation has been verified;
- multi-host reproducibility has been demonstrated;
- Phase 1 performed a Nix, Home Manager, Homebrew, mise, uv, service, defaults, link, or toolchain mutation.

Actual non-destructive current-host readiness evidence is deliberately assigned to Phase 13. Future clean-VM or second-Mac evidence is required before any fresh-install claim can be promoted.

## Human Verification Required

None for the Phase 1 safety-framework acceptance. The current-host drill is a later roadmap deliverable rather than a manual step needed to make this phase pass.

## Gaps Summary

No implementation gaps remain for the approved Phase 1 Vertical MVP scope.
