# Phase 1：Safety, Privacy, and State Foundation — Pattern Map

**Mapped:** 2026-07-10
**Inputs:** `01-CONTEXT.md`, `01-RESEARCH.md`, `01-VALIDATION.md`, tracked repository files, and the committed codebase map
**Safety posture:** 只读取已跟踪文件；未运行 Nix、Homebrew、manager、app validator 或 live probe。

## Mapping Summary

当前仓库能提供四类可复用惯例：Bash strict mode 与前置条件检查、Nix evaluation/build/activation 的职责分层、Homebrew 的 conservative convergence、以及 README/CLAUDE/AGENTS/Gitleaks 的维护流程。Phase 1 所需的 typed artifact store、kind-specific schemas、privacy pre-output gate、external fixture factory、protected-surface sentinels、四态 verdict 和统一 test runner 都没有现成 analog，必须在新的 repository-owned `safety/` boundary 中建立。

最重要的复用原则不是复制旧脚本，而是复用其小型安全模式并明确拒绝其 live mutation 分支。`setup_mac.sh` 和 `install_yazi_plugins.sh` 都是 operator-run mutation tools；它们可以提供 strict shell、quoted expansion、explicit prerequisite 和 stderr convention，却不能成为 Phase 1 默认 test dependency。

## Proposed Data Flow and Target Groups

```text
tracked synthetic input
  → `safety/cmd/yamc-safety` CLI interaction
  → strict kind validator + privacy/logical-reference gate
  → canonical digest + external content-addressed artifact store
  → external fixture fake adapter
  → fresh synthetic observation + sentinel comparison
  → verification evidence + bounded readiness report

real tracked repository files ── read-only source input only
real HOME / manager state / services / external targets ── never writable in Phase 1 tests
```

| Target file/group | Role | Closest existing analog | Classification | Planner instruction |
|---|---|---|---|---|
| `safety/go.mod` | stdlib-only module boundary | No Go module exists | **No analog** | Keep dependency list empty; local toolchain absence is `manual-required`, never bootstrap authorization. |
| `safety/cmd/yamc-safety/main.go` | One operator CLI for validate/store/fixture/sentinel/report | Root scripts are the only current CLI entrypoints | **Adapt** | Reuse explicit subcommand/usage/error discipline; do not reuse interactive prompts, raw path echoing, shell dispatch, or live mutation. |
| `safety/internal/artifact/*` | Common envelope, six payload validators, canonical digest, lineage, external store | No typed artifact model | **No analog** | Use closed registries and exact digest references; no `latest`, mtime, filename or directory-co-location selection. |
| `safety/internal/privacy/*` | Logical refs, pre-output gate, safe diagnostics, bounded capture | `.gitleaks.toml` is current leak scan boundary | **Adapt, not substitute** | Gitleaks remains defense in depth; the runtime gate must reject unsafe data before stdout/stderr/store. |
| `safety/internal/fixture/*` | Fresh external roots, allowlisted env, retention marker, exact network authorization | `install_yazi_plugins.sh` supports configurable target and guards | **Adapt narrowly** | Reuse explicit target/precondition style; invert defaults so repository/HOME are never writable fallback targets. |
| `safety/internal/sentinel/*` | Protected manifest, snapshots, four-state verdict, scoped claim | No repository sentinel system | **No analog** | Build per-domain synthetic adapters; no whole-HOME scan, attribution, auto-restore, retry-to-pass, or post-hoc exclusion. |
| `safety/internal/contract/controlplane.go` | Typed ownership/control-plane facts | `nix/darwin/default.nix`, `nix/home/dev-toolchains.nix`, `nix/darwin/homebrew.nix` | **Reuse semantics** | Encode declaration/manager/payload/executable/activation separately; do not import or invoke Nix modules from the safety binary. |
| `safety/manifests/*.json` | Protected surfaces and network-test contracts | Existing package/flake locks show tracked declarations, but not these schemas | **Adapt concept** | Track only synthetic/public logical entries; manifests are allowlists, never inventories of the real Mac. |
| `safety/testdata/**` | Synthetic blueprints, golden artifacts, raw samples, canaries | App-native fixtures do not exist | **No analog** | Every value must be synthetic; no copied real config, HOME path, username, hostname, provider, credential or private network detail. |
| `safety/scripts/test.sh` | Stable task/wave/phase sampling entrypoint | `setup_mac.sh`, `install_yazi_plugins.sh` | **Adapt** | Reuse strict mode, quoted expansions and explicit failure; forbid live fallback, install/download, inherited ambient env and arbitrary commands. |
| `safety/README.md` | User-facing CLI/tier/claim/safety contract | Root `README.md`; `nix/README.md` | **Reuse** | Document default offline tier, exact opt-ins, limits, claims and non-goals. Root README must add the new config/tool row and entrypoint. |
| `safety/CLAUDE.md` + `safety/AGENTS.md` | Local maintenance rules for a complex multi-file boundary | `nix/CLAUDE.md` + symlink; `zsh/CLAUDE.md` + symlink | **Reuse exactly** | Create local guidance and an actual `AGENTS.md -> CLAUDE.md` symlink in the same task; record prohibited live commands and doc/test checklist. |
| `.gitignore` | Ignore real run/local-state outputs | Existing local-state and Nix result rules | **Adapt** | Add only the chosen local-state/retained-fixture paths; ignore rules do not make content safe and do not replace runtime validation. |
| `.gitleaks.toml` | Scan tracked synthetic fixtures and planning/implementation diffs | Existing default-rules extension | **Reuse, extend only if proven** | Prefer fixtures that pass existing rules; any allowlist must be path+rule+exact synthetic regex scoped, never a broad directory exemption. |

## Concrete Repository Analogs

### 1. Strict shell and explicit preconditions — reuse

Both root scripts start with strict Bash mode (`setup_mac.sh:1-6`, `install_yazi_plugins.sh:1-2`):

```bash
#!/usr/bin/env bash
set -euo pipefail
```

`install_yazi_plugins.sh:63-68` also uses an explicit prerequisite guard and sends diagnostics to stderr:

```bash
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}
```

`safety/scripts/test.sh` should keep strict mode, quoted expansions, a usage function, explicit exit codes and small guard functions. Its toolchain guard must differ semantically: missing Go returns a privacy-safe `manual-required` result and must not suggest or execute a manager install.

### 2. Live target fallback and destructive replacement — reject

The current bootstrap has behaviors that are correct only for an interactive operator, not for a safety harness:

- `setup_mac.sh:29-30` falls back from tracked Git paths to physical directory scanning.
- `setup_mac.sh:40-48` derives and creates a live target directory.
- `setup_mac.sh:67-76` can remove an existing target and create a symlink after confirmation.
- `setup_mac.sh:161-175` can clone a remote repository and create another live symlink.
- `install_yazi_plugins.sh:41-50` may choose a live XDG directory, and `install_yazi_plugins.sh:91-117` creates state and asks `ya` to install packages.

None of these branches may be imported, invoked or emulated by default tests. The Phase 1 runner instead requires an explicit external root, proves it is outside both the repository and protected real roots, supplies a blank allowlisted environment, and uses fixture fake binaries only.

### 3. Evaluation/build/activation separation — reuse as a contract pattern

The repository already distinguishes composition from activation:

- `flake.nix:42-69` assembles nix-darwin and Home Manager modules.
- `nix/darwin/default.nix:19-21` keeps the Nix daemon boundary with Determinate Nix by disabling nix-darwin ownership of Nix itself.
- Root and Nix documentation distinguish evaluation/non-activating build from switch; a build result is not an applied receipt or fresh verification.

Phase 1 should mirror this separation in data structures: validate/observe/plan/receipt/verify/report are different kinds and dependency edges. It must not invoke `nix flake check`, `nix build`, `nix run`, `darwin-rebuild`, Home Manager activation or any manager command in task/wave/phase tests.

### 4. Conservative convergence — reuse as a negative-operation policy

`nix/darwin/homebrew.nix:14-18` states the conservative intent, and `nix/darwin/homebrew.nix:31-39` fixes:

```nix
onActivation = {
  autoUpdate = false;
  upgrade = false;
  cleanup = "none";
};
```

This is the closest existing analog for SAFE-05/08: missing or extra state does not authorize update, upgrade, cleanup or removal. The safety contract should represent extra state as `extra`/`unmanaged-present` data and structurally reject destructive operations in Phase 1.

### 5. Layered ownership — reuse the semantic split

`nix/home/dev-toolchains.nix:4-17` explicitly separates stable manager entrypoints from project-local runtime declarations. `nix/home/dev-toolchains.nix:20-34` installs manager binaries and direnv integration, while `nix/home/default.nix:8-24` composes them with user configuration. `nix/darwin/homebrew.nix:55-63` deliberately excludes language runtimes from Homebrew ownership.

The Phase 1 contract should therefore carry separate fields for declaration owner, manager-binary owner, payload owner, selected executable and activation context. A Home Manager module or Nix-built binary must not collapse those fields. One primary owner is enforced per `(scope, executable)`.

### 6. Gitleaks and local-state boundaries — reuse as defense in depth

`.gitleaks.toml:1-4` extends the default rules. Its existing allowlists (`.gitleaks.toml:6-36`) demonstrate the acceptable exception shape: exact rule IDs, exact paths and narrow regex conditions. `.gitignore:10-42` documents categories of credentials, login state, caches, local generated paths and Hermes state that stay outside version control.

Planner consequences:

- Run Gitleaks on every tracked synthetic fixture and staged implementation/doc diff.
- Do not add a blanket `safety/testdata/**` allowlist; canaries must be synthetic strings that exercise the runtime gate without resembling real credentials unless a narrowly scoped test rule is unavoidable.
- Add ignored real-run/local-state paths to `.gitignore`, but continue to validate every artifact before write. `.gitignore` protects Git staging only; it does not protect terminal, temporary files, retained fixtures or external stores.
- Never inspect untracked app/runtime state to construct examples or fixtures.

### 7. Documentation and local instruction pattern — reuse exactly

Root `README.md:5-22` keeps a table of managed directories/files; `README.md:24-37` documents prerequisites and setup; `README.md:139-159` documents private/local exclusions. A new `safety/` subsystem therefore requires all three root README updates: table row, non-mutating test/CLI instructions, and ignored local-state notes.

`nix/CLAUDE.md:19-37` records directory structure and maintenance boundaries. Existing links confirm the repository convention:

```text
AGENTS.md → CLAUDE.md
nix/AGENTS.md → CLAUDE.md
zsh/AGENTS.md → CLAUDE.md
```

Because `safety/` is multi-file, security-sensitive and expected to evolve, it cannot inherit only root guidance. Implementation must add `safety/CLAUDE.md` and create `safety/AGENTS.md` as a symlink to it. The local guide should list the artifact kinds, privacy gate ordering, forbidden live commands, fixture containment, sentinel verdicts, validation commands and documentation checklist.

## Existing Test and Validator Surface

The committed codebase map identifies no repository-wide test runner, assertion library, fixture factory or CI gate. Current validation is tool-specific and generally operator-run: Nix evaluation/build commands, app-native validators, shell syntax checks, Gitleaks and post-activation observations. These are useful inventory for later adapters but do not satisfy Phase 1 isolation because many can read caches, download, write state, or load real config.

Phase 1 must establish its own test boundary instead of wrapping all current validators. Only the following repository-native checks are safe as planning/commit checks outside the new runtime suite:

- `git diff --check` on explicitly scoped files;
- structural `rg`/JSON checks on tracked synthetic artifacts;
- staged Gitleaks with `.gitleaks.toml`;
- Git index/file whitelist verification.

Any command that evaluates Nix, invokes an application, reloads config, calls a manager, inspects services, reads defaults, follows live symlinks or downloads data remains outside the Phase 1 default gate.

## Missing Patterns the Planner Must Create

| Missing capability | Minimum new contract |
|---|---|
| Repository-wide runner | Stable `task` / `wave` / `phase` commands, external roots, minimal env, fixed exit semantics, no implicit escalation |
| Closed artifact schemas | Common envelope + six kind-specific validators, unknown-field rejection, schema version gate |
| Canonical digest and lineage | Restricted canonical JSON, recomputed SHA-256, exact upstream digest graph, content-addressed store |
| Runtime privacy boundary | Registered normalization only, logical namespaces, one pre-output gate, safe bounded error envelope |
| Bounded subprocess capture | Fixed allowlisted executable/argv, timeout/size limits, in-memory parse/discard, no inherited terminal |
| Fixture factory | Tracked blueprint → fresh external root, blank allowlisted env, marker/TTL/containment, delete-by-default |
| Network authorization | Exact test ID + purpose/URL/integrity/bytes/timeout/cache; deny by default and no real network in Phase 1 gate |
| Sentinel system | Explicit manifest, domain adapters, privacy-safe before/after snapshots, four-state verdict, scoped claim |
| Threat-model enforcement | Per-plan `<threat_model>` plus negative tests for kind confusion, leakage, escape, escalation, incomplete observation and overclaim |

## Planner Constraints Derived from the Map

1. Keep `safety/` physically and dependency-wise separate from `setup_mac.sh`, Nix activation and future real apply executors.
2. Make the first user-visible slice a real CLI round trip inside an external synthetic fixture, not a set of horizontal schema-only tasks. Contracts may be implemented first within the same plan/wave only when their task ends in an operator-observable CLI behavior.
3. Use repository-relative paths in plans and public docs. Do not copy current host identity, private paths or live version snapshots into schemas/testdata.
4. Every implementation task must add or extend its negative tests and run one task-scoped sentinel-wrapped command; privacy and sentinel work cannot be deferred to a final hardening plan.
5. Preserve the root dirty worktree: plans must name exact files and commits must use explicit path lists; never stage `.config/` broadly.
6. Treat the root README update, `.gitignore` update, `safety/README.md`, `safety/CLAUDE.md`, and `safety/AGENTS.md` symlink as required implementation deliverables rather than optional polish.
7. Do not modify `flake.nix`, `nix/**`, `.config/mise/**`, shell activation, Homebrew inventory or live setup scripts in Phase 1. Their contents are read-only contract sources and later-phase inputs.

## Mapping Readiness

The repository has enough small patterns to guide implementation style, but no existing subsystem can be repurposed wholesale. Planner should create a thin, isolated `safety/` vertical slice, explicitly adapt only the safe conventions above, and include structural guards proving that the new default dependency graph cannot reach existing live mutation entrypoints.
