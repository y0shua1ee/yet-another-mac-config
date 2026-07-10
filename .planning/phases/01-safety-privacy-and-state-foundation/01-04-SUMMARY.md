---
phase: 01-safety-privacy-and-state-foundation
plan: "04"
subsystem: safety-fixture-policy
tags: [go, external-fixture, environment-isolation, retention, offline-tiers, network-deny, live-check]

requires:
  - phase: 01-safety-privacy-and-state-foundation
    plan: "03"
    provides: closed logical references, privacy-gated output, bounded fixed-command capture, and synthetic CLI execution
provides:
  - marker-owned fresh external fixture roots with complete HOME/XDG/tool-state isolation and blank child environments
  - frozen-primary lifecycle with default exact teardown, explicit pre-run retention, safe expiry, and monotonic verdict combination
  - three closed non-escalating test tiers plus exact tracked network-contract validation without an HTTP executor
  - bounded live-check unknown/manual behavior and permanent generic/injection-shaped runner denial evidence
affects: [01-05-sentinels, 01-07-phase-e2e, future-isolated-integration, future-live-probes]

tech-stack:
  added: []
  patterns: [type-state verdict freeze, marker-owned one-child teardown, blank allowlisted environment, validation-only network authorization, empty live-probe execution registry]

key-files:
  created:
    - safety/internal/fixture/root.go
    - safety/internal/fixture/environment.go
    - safety/internal/fixture/retention.go
    - safety/internal/fixture/fixture_test.go
    - safety/internal/fixture/network.go
    - safety/internal/fixture/network_test.go
    - safety/manifests/network-tests.v1.json
    - safety/internal/e2e/tier_cli_test.go
  modified:
    - safety/scripts/test.sh
    - safety/cmd/yamc-safety/main.go

key-decisions:
  - "Keep the existing caller-owned synthetic fixture route for inner test composition, and add a managed fixture-base mode whose default lifecycle is exact teardown and whose output contains only a logical reference and expiry category."
  - "Require a frozen primary verdict before retention may preserve or remove one marker-owned child; teardown failure changes passed to harness-error but never rewrites any non-pass to passed."
  - "Treat a valid exact network manifest as authorization metadata only: Phase 1 always returns manual-required because no HTTP, DNS, socket, cache-miss, or live command executor exists."
  - "Represent live-check as a separate tier with proof validation but no registration or execution path; the Phase 1 policy remains empty and returns unknown."

patterns-established:
  - "Fixture containment: one canonical direct child under an external base owns every HOME, XDG, temporary, fake PATH, manager, trust, cache, store, blueprint, and sentinel directory."
  - "Capability monotonicity: missing input, malformed manifests, ambient proxy/credential keys, cache or egress uncertainty, and unproved live probes preserve the requested tier while returning denied manual/unknown results."
  - "Temporal runner ownership: Task 1 introduced only fixture-lifecycle; Task 2 introduced only tier-network-policy and fixture-policy, while lifetime negatives use reserved generic and injection-shaped argv values."

requirements-completed: [SAFE-04, SAFE-05, SAFE-06]

duration: 21 min
completed: 2026-07-10
---

# Phase 01 Plan 04: External Fixture Lifecycle and Offline Policy Summary

**Marker-owned external fixtures now isolate every writable test root, while closed offline/integration/live tiers validate exact synthetic network metadata without ever acquiring network or live execution capability.**

## Performance

- **Duration:** 21 min
- **Started:** 2026-07-10T16:12:52Z
- **Completed:** 2026-07-10T16:34:18Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments

- Added fresh external fixture creation that rejects repository/protected overlap, traversal and symlink bases before child creation, then creates HOME, all XDG roots, TMPDIR, fake-only PATH, manager roots, trust, network cache, artifact store, blueprint worktree, and sentinel scratch under one canonical marker-owned child.
- Added a blank allowlisted child environment with offline Go controls and isolated manager/cache roots; no ambient HOME, proxy, credential, SSH, shell-init, or manager state is inherited.
- Added schema/UID/nonce/TTL/direct-child/non-symlink retention validation. Passed and failed primary workloads are frozen before default teardown; only a pre-run keep choice retains state, and expired retained state can be removed through the same exact ownership boundary.
- Added closed `offline-static`, `isolated-integration`, and `live-check` tiers. Neither missing capability nor any policy failure changes the selected tier or acquires a higher-privilege fallback.
- Added a strict tracked network manifest with exact test/adapter IDs, HTTPS host/port/method/URL, zero redirects, SHA-256 integrity, byte/time bounds, isolated logical cache, forbidden credentials/proxy, exact-URL egress, and exact-ID authorization. The implementation contains no HTTP, DNS, socket, shell, or arbitrary-command executor.
- Added CLI and runner negatives for broad flags, partial authorization, ambient proxy/credential keys, wildcard/generic/injection-shaped IDs, reserved generic task/wave/scope inputs, malformed phase argv, and live-check without approved proof.

## Task Commits

Each task was committed atomically:

1. **Task 01-04-01: 交付 external fixture root、环境隔离与 retention lifecycle** - `3346119` (feat)
2. **Task 01-04-02: 强制 offline tier、exact network ID 与 live-check deny contract** - `ae29031` (feat)
3. **Rule 1 safety correction: route symlink negative setup through owned teardown** - `f5d35f9` (fix)

_Plan metadata is committed together with this summary._

## Files Created/Modified

- `safety/internal/fixture/root.go` - Creates one canonical external fixture child, marker, complete isolated directory layout, and process-local paths.
- `safety/internal/fixture/environment.go` - Builds a deterministic child environment from a closed allowlist with offline Go and isolated manager state.
- `safety/internal/fixture/retention.go` - Freezes primary verdicts, validates exact ownership, retains by pre-run choice, expires one child, and combines teardown monotonically.
- `safety/internal/fixture/fixture_test.go` - Covers root containment, environment isolation, success/failure cleanup, explicit retention, expiry, marker/UID/nonce/TTL/symlink/base-escape denials, CLI privacy, and narrow teardown structure.
- `safety/internal/fixture/network.go` - Implements closed tiers, strict duplicate-key-safe manifest parsing, exact authorization metadata, ambient-state denial, proof validation, and non-executing live policy.
- `safety/internal/fixture/network_test.go` - Covers tier closure, every manifest field, exact/unknown/wildcard/ambient denials, proof expiry, and structural absence of network/command executors.
- `safety/manifests/network-tests.v1.json` - Tracks one public synthetic `example.invalid` validation-only network contract.
- `safety/internal/e2e/tier_cli_test.go` - Proves CLI manual/unknown behavior and closed runner dispatch under broad, partial, generic, and injection-shaped inputs.
- `safety/cmd/yamc-safety/main.go` - Adds managed fixture lifecycle options and the bounded `test-policy` status command without rendering physical roots.
- `safety/scripts/test.sh` - Adds exact fixture-lifecycle and two-package tier-network-policy routes plus the fresh-child fixture-policy wave.

## Decisions Made

- Managed fixture mode owns creation and teardown when the caller supplies an external base and logical fixture ID. The existing explicit fixture/store roots remain an inner synthetic composition contract whose enclosing runner already owns the fresh external root and final teardown.
- Retention is a type-state boundary: no deletion or keep result is available until the caller freezes one of the four primary verdicts. A teardown ambiguity can preserve a non-pass or worsen pass to harness-error; it has no restore, convergence, retry-to-pass, or arbitrary-path branch.
- Network authorization validates only tracked public metadata at the exact repository manifest path. Even a completely valid exact ID returns `manual-required` because Phase 1 deliberately has no request executor or cache-miss fallback.
- Live-probe proof requires both current official read-only semantics and current isolated negative evidence, including expiry and digest shape. Proof validation does not register or run a probe; the Phase 1 live policy is intentionally empty.

## TDD Evidence

- Before Task 1 implementation, `task fixture-lifecycle` selected exactly `./internal/fixture` plus `^TestFixtureLifecycle$` and returned `expected-red-observed` only for `fixture-lifecycle-behavior-missing`.
- Before Task 2 implementation, `task tier-network-policy` selected exactly `./internal/fixture` plus `^TestTierNetworkPolicy$` and `./internal/e2e` plus `^TestTierCLI$`; both missing behavior paths produced only `tier-network-policy-behavior-missing`.
- The approved plan prescribed one exact English `feat` commit per task, so each RED was observed before implementation and each completed test/production whitelist was committed atomically.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Security] Removed direct recursive deletion from symlink negative-test setup**

- **Found during:** Final plan-level security scan after Task 2.
- **Issue:** The symlink rejection subtest initially used `os.RemoveAll` directly on its external synthetic fixture to prepare a replacement symlink. It could not touch live state, but it bypassed the plan's invariant that fixture teardown itself must pass frozen-verdict and marker ownership checks.
- **Fix:** Freeze a synthetic violation verdict and invoke the production retention finalizer to remove the exact owned child before creating the symlink negative sample. The second finalizer call then proves the symlink and its target remain untouched.
- **Files modified:** `safety/internal/fixture/fixture_test.go`.
- **Verification:** `fixture-lifecycle`, `tier-network-policy`, and `fixture-policy` all pass; the test file contains no direct `os.RemoveAll` call.
- **Committed in:** `f5d35f9`.

---

**Total deviations:** 1 auto-fixed Rule 1 security issue. **Impact:** Production behavior and plan scope are unchanged; the negative test now obeys the same exact teardown boundary it verifies.

## Issues Encountered

- macOS canonicalizes temporary paths through `/private`; fixture tests compare canonical bases rather than alias spellings.
- `go run` emits its own fixed `exit status 32` line when the tested CLI intentionally returns `manual-required`. E2E tests accept only that exact wrapper diagnostic while requiring the CLI decision itself to remain bounded structured output.

## User Setup Required

None - no package installation, external service, credential, network access, live probe, host activation, or real-machine mutation is required. A missing local Go toolchain remains `manual-required`.

## Next Phase Readiness

- Plan 01-05 can build protected-surface manifests and four-state sentinels on fixture roots whose lifecycle and child environment are now mechanically constrained.
- Test suites can distinguish offline, isolated, and live-check intent without acquiring network or live execution capability through failure, cache miss, crafted argv, or unproved metadata.
- No real apply, Nix/Homebrew/mise/uv/rustup operation, service query, defaults write, link replacement, network request, whole-Mac claim, current-host readiness claim, or destructive convergence path was introduced.

## Self-Check: PASSED

- All eight created and two modified implementation files exist; the three commits are present and delete no tracked files.
- `task fixture-lifecycle`, `task tier-network-policy`, and `wave fixture-policy` pass after the safety correction.
- Commit `3346119` adds only the literal `task:fixture-lifecycle)`; commit `ae29031` adds only the literals `task:tier-network-policy)` and `wave:fixture-policy)`.
- The network policy production files contain zero HTTP client, DNS/socket dial/listen, shell, or arbitrary-command execution paths; all manifest entries use only `example.invalid` public synthetic metadata.
- Exact staged whitelists, cached diff checks, targeted identity/path/credential scans, JSON contract checks, and staged Gitleaks passed for both task commits and the corrective commit.
- Existing user changes in `CLAUDE.md`, `.ai/`, and `.config/alma/` remained unstaged and unchanged by this plan.

---
*Phase: 01-safety-privacy-and-state-foundation*
*Completed: 2026-07-10*
