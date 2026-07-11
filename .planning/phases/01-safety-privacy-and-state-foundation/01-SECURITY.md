---
phase: "01"
slug: safety-privacy-and-state-foundation
status: verified
threats_open: 0
asvs_level: 1
block_on: high
register_authored_at_plan_time: true
created: 2026-07-11
last_audited: 2026-07-11
---

# Phase 01 — Security

> Per-phase security contract for the completed Safety, Privacy, and State Foundation phase.

This audit verifies the plan-authored STRIDE register against the final implementation after code-review fixes and exceptional safety stabilization. Threat IDs are namespaced by plan because every PLAN intentionally uses local IDs `T-01` through `T-06`.

---

## Audit Configuration and Input State

| Field | Value |
|-------|-------|
| Input state | B — no prior SECURITY.md; PLAN and SUMMARY artifacts exist |
| Security enforcement hook | enabled at `verify:post` |
| ASVS depth | L1 |
| Blocking threshold | high |
| PLAN threat models | 7/7 parseable |
| Registered threats | 42 HIGH |
| SUMMARY threat flags | none |
| Preliminary classification | 42 closed, 0 open |
| Auditor dispatch | skipped by the ASVS L1 clean-register short-circuit |
| Blocking result | `threats_open: 0` |

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| CLI → workflow | Public arguments enter a closed command and operation router. | Untrusted subcommands, logical refs, tier and fixture inputs |
| Frozen Git source → workflow | One HEAD OID and one bounded stage-0 index snapshot authorize repository inputs. | Tracked bytes, modes, manifests and blueprints |
| Workflow → artifact store | Only a fresh marker-owned single-writer capability can mutate; existing stores reopen read-only. | Canonical typed artifact bytes and digest lineage |
| Artifact/candidate → output sink | One privacy gate precedes filesystem, stdout and stderr output. | Logical identifiers, diagnostics and normalized facts |
| Fixed child process → capture adapter | Registry-owned executable/argv are captured with time and byte limits. | Synthetic stdout/stderr bytes before strict normalization |
| Public caller → fixture lifecycle | Fresh external roots isolate HOME, XDG, cache, trust, manager and runtime state. | Operator base path, retention request and marker metadata |
| Tier/network manifest → capability | Metadata can authorize validation only; Phase 1 has no HTTP/DNS/socket/live executor. | Exact test IDs, public URL metadata and limits |
| Protected-surface manifest → sentinel | A closed manifest and proof-gated registry select bounded adapters. | Logical surface IDs, proof state and observation bounds |
| Surface snapshot → verdict/report | Exact before/after evidence feeds a strict four-state evaluator and claim allowlist. | Opaque per-run tokens, manifest/window digests and verdicts |
| Ownership contract → policy | Typed owner facts and observed extra state feed a report-only policy. | Scope, executable, owner role, status and empty operations |
| Suite manifest → runner | One supervisor and one process group enforce fixed offline routes and nested deadlines. | Task/wave/phase IDs, private deadline frames and bounded output |
| Documentation → operator | Executable docs state commands, prohibitions and claim ceilings. | Human-visible operational guidance |

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Evidence | Status |
|-----------|----------|-----------|----------|-------------|---------------------|--------|
| P01-T-01 | Spoofing / Tampering | artifact kind and lineage | high | mitigate | E-A1, E-W1 | closed |
| P01-T-02 | Information Disclosure | CLI/store output | high | mitigate | E-P1 | closed |
| P01-T-03 | Tampering / Elevation of Privilege | operation routing | high | mitigate | E-C1 | closed |
| P01-T-04 | Tampering / Elevation of Privilege | fixture filesystem and environment | high | mitigate | E-F1, E-R1 | closed |
| P01-T-05 | Elevation of Privilege / Information Disclosure | toolchain/network escalation | high | mitigate | E-N1, E-R1 | closed |
| P01-T-06 | Repudiation / Denial of Service | sentinel evidence and claim ceiling | high | mitigate | E-S1, E-V1 | closed |
| P02-T-01 | Spoofing / Tampering | kind/schema/lineage/lifecycle | high | mitigate | E-A1, E-A2 | closed |
| P02-T-02 | Information Disclosure | validation errors | high | mitigate | E-P1 | closed |
| P02-T-03 | Tampering / Elevation of Privilege | generated-plan payload | high | mitigate | E-A1, E-C1 | closed |
| P02-T-04 | Tampering / Elevation of Privilege | artifact store filesystem | high | mitigate | E-A2 | closed |
| P02-T-05 | Elevation of Privilege / Information Disclosure | validation runner | high | mitigate | E-N1, E-R1 | closed |
| P02-T-06 | Repudiation / Denial of Service | evidence/report graph and retention | high | mitigate | E-A1, E-A2, E-W1 | closed |
| P03-T-01 | Spoofing / Tampering | artifact input | high | mitigate | E-A1, E-P1 | closed |
| P03-T-02 | Information Disclosure | store/stdout/stderr/raw capture | high | mitigate | E-P1, E-P2 | closed |
| P03-T-03 | Tampering / Elevation of Privilege | subprocess execution | high | mitigate | E-P2 | closed |
| P03-T-04 | Tampering / Elevation of Privilege | resolver/domain/fixture paths | high | mitigate | E-P1, E-F1 | closed |
| P03-T-05 | Elevation of Privilege / Information Disclosure | process/network environment | high | mitigate | E-P2, E-N1, E-R1 | closed |
| P03-T-06 | Repudiation / Denial of Service | capture failure/evidence | high | mitigate | E-P2 | closed |
| P04-T-01 | Spoofing / Tampering | fixture marker/network manifest | high | mitigate | E-F1, E-N1 | closed |
| P04-T-02 | Information Disclosure | environment/retained fixture | high | mitigate | E-F1, E-P1 | closed |
| P04-T-03 | Tampering / Elevation of Privilege | fixture teardown and tier routing | high | mitigate | E-F1, E-N1, E-C1 | closed |
| P04-T-04 | Tampering / Elevation of Privilege | fixture filesystem | high | mitigate | E-F1 | closed |
| P04-T-05 | Elevation of Privilege / Information Disclosure | network/live capability | high | mitigate | E-N1 | closed |
| P04-T-06 | Repudiation / Denial of Service | lifecycle outcomes | high | mitigate | E-F1, E-R1 | closed |
| P05-T-01 | Spoofing / Tampering | manifest/evidence identity | high | mitigate | E-S1, E-S2 | closed |
| P05-T-02 | Information Disclosure | surface snapshots | high | mitigate | E-S1, E-P1 | closed |
| P05-T-03 | Tampering / Elevation of Privilege | sentinel adapters | high | mitigate | E-S2 | closed |
| P05-T-04 | Tampering / Elevation of Privilege | symlink/tree observation | high | mitigate | E-S2 | closed |
| P05-T-05 | Elevation of Privilege / Information Disclosure | real observation boundary | high | mitigate | E-S2 | closed |
| P05-T-06 | Repudiation / Denial of Service | verdict/claim | high | mitigate | E-V1, E-R1 | closed |
| P06-T-01 | Spoofing / Tampering | owner claims | high | mitigate | E-C1 | closed |
| P06-T-02 | Information Disclosure | control-plane rendering | high | mitigate | E-C1, E-P1 | closed |
| P06-T-03 | Tampering / Elevation of Privilege | mutable/destructive routes | high | mitigate | E-C1 | closed |
| P06-T-04 | Tampering / Elevation of Privilege | synthetic receipt target | high | mitigate | E-C1, E-F1 | closed |
| P06-T-05 | Elevation of Privilege / Information Disclosure | module/manager invocation | high | mitigate | E-C1, E-N1 | closed |
| P06-T-06 | Repudiation / Denial of Service | extra-state disposition | high | mitigate | E-C1 | closed |
| P07-T-01 | Spoofing / Tampering | suite and artifact graph | high | mitigate | E-W1, E-A1, E-S1 | closed |
| P07-T-02 | Information Disclosure | all sinks/docs/testdata | high | mitigate | E-P1, E-D1 | closed |
| P07-T-03 | Tampering / Elevation of Privilege | CLI/dependency graph | high | mitigate | E-C1, E-P2 | closed |
| P07-T-04 | Tampering / Elevation of Privilege | full phase filesystem | high | mitigate | E-F1, E-R1 | closed |
| P07-T-05 | Elevation of Privilege / Information Disclosure | network/live/toolchain/outer sentinels | high | mitigate | E-N1, E-S2, E-R1 | closed |
| P07-T-06 | Repudiation / Denial of Service | phase verdict/claim | high | mitigate | E-V1, E-R1, E-W1 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` count toward `threats_open`.*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party).*

## Mitigation Evidence Index

| Evidence | Verified control and implementation/test anchors |
|----------|--------------------------------------------------|
| E-A1 | Six closed artifact kinds, duplicate-key-safe canonicalization, recomputed SHA-256, exact lineage and semantic fresh-state binding: `safety/internal/artifact/{kinds,canonical,envelope,lineage}.go`, `validate_test.go`, `artifact_cli_test.go`. |
| E-A2 | Fresh capability-owned single-writer store, existing-store read-only reopen, append-only staging/object/transition publication, no filename rollback/delete, bounded no-follow reads: `store.go`, `store_fs.go`, `store_stabilization_test.go`. |
| E-P1 | Closed logical namespaces, surface compatibility, forbidden-field/value rejection, safe six-field diagnostics and one pre-output gate across sinks: `privacy/gate.go`, `gate_test.go`, `privacy_cli_test.go`. |
| E-P2 | Fixed registry-owned subprocesses, no shell/arbitrary argv, bounded dual-stream capture, strict normalization and raw-buffer disposal: `privacy/capture.go`, `capture_test.go`. |
| E-F1 | Fresh external marker-owned fixture, canonical overlap/symlink/traversal rejection, blank allowlisted HOME/XDG/manager/cache roots and capability-scoped teardown: `fixture/{root,environment,retention}.go`, `fixture_test.go`. |
| E-N1 | Closed offline/integration/live tiers, exact validation-only network contract, forbidden ambient credentials/proxies, no request executor and `manual-required` fallback: `fixture/network.go`, `network_test.go`, `network-tests.v1.json`. |
| E-S1 | Closed protected-surface manifest, bounded privacy-safe snapshots, frozen manifest/window identity and per-run opaque tokens: `sentinel/{manifest,snapshot,synthetic}.go`, `sentinel_test.go`. |
| E-S2 | Source-bound proof registry, current proof/negative-suite digest checks, symlink/tree escape rejection, missing/stale proof fail-closed before adapter/workload calls: `sentinel/real.go`, `real_test.go`, `real-adapters.v1.json`. |
| E-V1 | Strict `passed` / `violation` / `indeterminate` / `harness-error` evaluator, exact evidence bindings, non-pass exits and scoped-claim rejection: `sentinel/verdict.go`, `verdict_test.go`, `sentinel_cli_test.go`. |
| E-C1 | Typed one-primary ownership contract, data-only operations, fixture-only synthetic receipt and report-only `extra` / `unmanaged-present` with `operations: []`: `contract/{controlplane,policy}.go`, their tests and `no_cleanup_cli_test.go`. |
| E-R1 | Empty offline environment, local-toolchain-only behavior, one supervisor/PGID, private fixed deadline protocol, hard 15/47/305 ceilings, exit 124 and no-orphan cleanup: `safety/scripts/test.sh`, `phase_e2e_test.go`, `real_sentinel_cli_test.go`. |
| E-W1 | Run-wide frozen Git view, rooted component-by-component no-follow input reads, complete seven-instance graph reload and exact manifest/report binding: `workflow/synthetic.go`, `tracked_snapshot_test.go`, `phase_e2e_test.go`, `offline-suite.v1.json`. |
| E-D1 | Operator and maintainer docs are structural test inputs and state no-live/no-cleanup/no-overclaim boundaries: `README.md`, `safety/README.md`, `safety/CLAUDE.md`, `safety/AGENTS.md`. |

## Accepted Risks Log

No registered threat uses an `accept` or `transfer` disposition. No accepted risks were required to reach `threats_open: 0`.

## Security Boundaries and Non-Claims

- Phase 1 uses a fresh directory capability and append-only persistence; it does not claim that macOS/Go can prevent an uncooperative same-UID process from renaming that directory object. Namespace drift freezes further mutation and never authorizes filename-based cleanup or deletion of a replacement.
- The tracked controlled-service adapter proof remains intentionally `missing`. Current-host execution returns `manual-required`, verdict `indeterminate`, exit `32`, and zero adapter/workload calls before any real observation.
- A completely valid network manifest remains validation metadata only. Phase 1 contains no HTTP, DNS, socket, cache-miss or live-command executor.
- Synthetic success proves the isolated mechanism only. It cannot establish whole-Mac integrity, current-host readiness, multi-host consistency or `fresh-install-verified`.
- No real apply, Nix/Home Manager activation, Homebrew/mise/uv/rustup mutation, service/defaults/link change, trust mutation or unmanaged-state cleanup exists in the Phase 1 execution graph.

## Security Verification Run

| Gate | Result |
|------|--------|
| 42-row register extraction and plan-time authorship | passed |
| SUMMARY Threat Flags scan | none found |
| ASVS L1 source/test mitigation anchors | 42/42 closed |
| Production artifact filename-delete guard | passed |
| Production shell/network executor absence guard | passed |
| Missing-proof current-host gate | present and fail-closed |
| Full offline `./safety/scripts/test.sh phase` | `synthetic-sentinel-passed` |
| Final `wave phase-integration` | `synthetic-sentinel-passed` |
| Current-host/live operations | not run |

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-11 | 42 | 42 | 0 | Codex `gsd-secure-phase` ASVS L1 |

## Sign-Off

- [x] All threats have a disposition.
- [x] All 42 HIGH threats have implementation/test evidence.
- [x] Accepted risks log reviewed; no registered accepted risks.
- [x] `threats_open: 0` confirmed.
- [x] `status: verified` set in frontmatter.

**Approval:** verified 2026-07-11
