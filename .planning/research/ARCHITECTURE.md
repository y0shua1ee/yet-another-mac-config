# Architecture Research

**Domain:** 公开、隐私安全、非破坏性的多 Mac 配置与恢复系统（brownfield）
**Researched:** 2026-07-10
**Confidence:** HIGH（现有边界与官方工具语义）；MEDIUM（本仓库建议的本地 host binder 与 plan artifact 细节尚待实现验证）

## Standard Architecture

### System Overview

推荐保留现有两个 activation plane，并在它们上方增加一个**只编排、不拥有配置内容**的恢复控制面。控制面必须把 desired、observed、plan、applied receipt 与 verification evidence 作为五种不同状态；任何一步都不能用“上一条命令退出码为 0”代替下一种状态。

```text
┌──────────────────────────── Public Git: desired state ────────────────────────────┐
│                                                                                    │
│  Shared baseline       Logical host profiles        Public contracts              │
│  Nix/HM modules        `primary-mac`, roles         toolchains / links / secrets  │
│        │                       │                              │                    │
│        └───────────────────────┴──────────────┬───────────────┘                    │
│                                               │ desired projection                 │
└───────────────────────────────────────────────┼────────────────────────────────────┘
                                                │
┌──────────────────────── Local privacy boundary┼────────────────────────────────────┐
│  identity-only host binder                    │      secret bindings / local files │
│  logical id → real user/host/home             │      never passed into Nix         │
│                 │                             │                    │               │
└─────────────────┼─────────────────────────────┼────────────────────┼───────────────┘
                  │                             │                    │ presence only
                  ▼                             ▼                    ▼
┌──────────────────────────── Read-only observation plane ───────────────────────────┐
│ Nix/HM │ Homebrew │ toolchains │ symlinks │ defaults/services │ secret obligations │
│ adapters emit normalized facts only; raw paths, identities and values are dropped  │
└───────────────────────────────────────────────┬────────────────────────────────────┘
                                                │ observed snapshot
                                                ▼
┌──────────────────────────── Recovery control plane ────────────────────────────────┐
│  Normalizer → Policy/ownership validator → Diff engine → Immutable plan compiler  │
│                                                        │                           │
│                                  typed operations + digest + rollback metadata      │
│                                                        ▼                           │
│                                               explicit confirmation                 │
└────────────────────────────────────────────────────────┬───────────────────────────┘
                                                         │ exact reviewed plan
                           ┌─────────────────────────────┴────────────────────────┐
                           ▼                                                      ▼
┌────────────────── Declarative activation plane ─────────────┐   ┌──────── Symlink plane ────────┐
│ non-activation build → activation checkpoint → switch       │   │ backup → explicit links only  │
│ nix-darwin → Home Manager → generated Homebrew Bundle       │   │ no discovery fallback / rm -rf│
└──────────────────────────────┬───────────────────────────────┘   └──────────────┬────────────────┘
                               └──────────────────┬───────────────────────────────┘
                                                  ▼
┌──────────────────────── Verification and readiness plane ──────────────────────────┐
│ fresh read-only inventory → expected-vs-observed comparison → sanitized report     │
│ verified / declared-unapplied / drift / private-missing / manual / excluded        │
└────────────────────────────────────────────────────────────────────────────────────┘
```

The key architectural choice is **not** to turn `setup_mac.sh` into a second package manager and **not** to hide `darwin-rebuild switch` inside a generic “sync” command. The control plane knows that these are different writers, orders them, and verifies them independently.

### State Model

| State | Meaning | Source / lifetime | May contain secret values? | May contain real machine identity? |
|-------|---------|-------------------|----------------------------|------------------------------------|
| Desired state | What the selected logical profile and tracked contracts declare | Git-tracked; durable | Never | Never |
| Local binding | Maps a logical profile to the current account/hostname/home and maps public secret IDs to local providers | Outside Git; mode-restricted | Host binding: no; secret binding may contain only provider references, never values | Yes, locally only |
| Observed state | Normalized facts from read-only probes | Local state directory or memory; short-lived | Never | Never after normalization |
| Generated plan | Typed operations needed to reconcile one observed snapshot with one desired digest | Local, immutable/content-addressed | Never | Never |
| Applied receipt | Which exact operations ran, their outcome, rollback anchors, and post-apply checkpoint | Local, append-only for a run | Never | Never |
| Verification evidence | Fresh facts proving or disproving expected outcomes | Local; reportable | Never | Never |

This separation matters because a successful Nix build is only build evidence, a successful switch is only declarative-plane apply evidence, and neither proves that app symlinks, permissions, private overlays, services, or project tool contracts are ready.

### Component Responsibilities

| Component | Responsibility | Inputs | Output | Mutation authority |
|-----------|----------------|--------|--------|--------------------|
| Shared baseline | Holds common Nix, nix-darwin, Home Manager, shell, package, defaults, and app-config policy | Tracked modules | Reusable modules | None by itself |
| Logical host profile | Selects architecture, roles, optional components, and expected differences without real identity | Public profile ID | Profile projection | None |
| Local host binder | Supplies real username, hostname, and home directory to final Nix composition | Ignored/local `host.json` plus public profile | Local wrapper flake / concrete config | None; evaluation input only |
| Ownership manifest | Declares one primary owner and allowed supporting roles for every runtime/tool | Public enum-based records | Ownership graph and validation rules | None |
| Link manifest | Allow-lists repo-relative sources and home-relative targets; records writer and conflict policy | Public records | Desired link graph | None |
| Secret-requirements manifest | Declares stable IDs, purpose, required/optional status, and accepted provider kinds | Public records | Secret obligations | None; must not resolve values |
| Desired projector | Evaluates JSON-safe Nix options and reads public manifests without parsing implementation source text | Selected profile + tracked sources | Normalized desired model | Evaluation/build only; never activate |
| Inventory adapters | Probe only named targets and emit typed, normalized facts | Live machine + local binding | Observed model | Read-only |
| Policy validator | Detects duplicate owners, conflicting link writers, forbidden secret fields, unsupported actions, and missing rollback metadata | Desired model | Errors/warnings | None |
| Diff engine | Compares desired and observed facts without deciding how to execute shell commands | Desired + observed | Semantic changes | None |
| Plan compiler | Turns semantic changes into an allow-listed operation graph with preconditions, risk and rollback | Diff + adapter registry | Content-addressed plan | None |
| Confirmation gate | Binds user approval to one plan digest and selected high-risk operations | Plan digest + fresh precondition check | Confirmation receipt | None |
| Apply coordinator | Executes only operations already present in the confirmed plan; never discovers new work while applying | Exact plan + confirmation | Applied receipt | Yes, after confirmation only |
| Declarative adapter | Builds, previews and activates nix-darwin/Home Manager/Homebrew as one coarse plane | Concrete local flake target | Generation/build/apply evidence | Nix store during build; system during switch |
| Symlink adapter | Backs up and creates only manifest-listed targets; detects Home Manager ownership conflicts | Link operations | Link receipts | Named targets only |
| Toolchain adapters | Handle ecosystem-specific, staged actions after ownership validation | Per-ecosystem plan | Runtime/project evidence | Only the selected ecosystem, after confirmation |
| Secret presence adapters | Check whether a required local file/provider/keychain item is configured without retrieving its value | Public obligation + local binding | `present`, `missing`, `manual`, `unknown` | Read-only, value-blind |
| Verifier | Re-runs inventory after apply and compares it with expected postconditions | Plan + fresh observed model | Verification evidence | Read-only |
| Report renderer | Produces machine-readable JSON and human Markdown from normalized evidence | Desired/observed/receipt/evidence | Sanitized readiness report | Local report files only |
| Fixture harness | Runs adapters against fake homes, fake command output, and minimal project contracts | Tracked fixtures | Deterministic test results | Temporary directories only |

## Recommended Project Structure

The repository already requires setup/installation entry scripts at the root, so keep a root facade and place non-entry implementation below a focused library tree.

```text
yet-another-mac-config/
├── mac_config.sh                    # check/plan/apply/verify root entry; Chinese comments
├── setup_mac.sh                     # compatibility entry for link plane; later delegates safely
├── flake.nix                        # public outputs, shared modules, mkDarwinHost helper
├── profiles/
│   ├── common.nix                   # shared baseline composition
│   ├── roles/
│   │   └── development.nix          # role overlay, no identity
│   └── hosts/
│       └── primary-mac.nix          # logical profile only
├── manifests/
│   ├── components.toml              # adapter IDs, planes, risk class, dependencies
│   ├── toolchains.toml              # ownership and project-contract policy
│   ├── links.toml                   # explicit repo-relative → home-relative allow-list
│   └── secret-requirements.toml     # public obligation IDs; no values/provider URI
├── schemas/
│   ├── desired.schema.json
│   ├── observed.schema.json
│   ├── plan.schema.json
│   └── report.schema.json
├── lib/recovery/
│   ├── inventory/                   # read-only adapters
│   ├── planning/                    # normalization, diff, plan compiler
│   ├── apply/                       # typed operation executors
│   ├── verify/                      # postcondition checks
│   └── report/                      # JSON/Markdown renderer and redaction
├── nix/
│   ├── lib/mk-darwin-host.nix       # concrete composition from public profile + local identity
│   ├── darwin/                      # existing system/Homebrew/defaults modules
│   ├── home/                        # existing user/tool-manager modules
│   └── modules/                     # reusable Home Manager modules
├── templates/local-host/
│   ├── flake.nix                    # ignored/local wrapper template
│   └── host.example.json            # fake placeholders only; never a real identity
├── tests/
│   ├── fixtures/
│   │   ├── hosts/                   # fake logical host bindings
│   │   ├── inventory/               # synthetic adapter output
│   │   ├── plans/                   # secret-free golden plans
│   │   └── projects/                # minimal six-ecosystem contracts
│   └── integration/                 # opt-in isolated tests; no real HOME
├── .config/                         # existing app-native desired state
├── .hammerspoon/                    # existing automation desired state
└── zsh/                             # existing shared and fallback shell sources
```

The exact implementation language belongs in stack research, but the boundaries should remain valid regardless of language. The root facade should be a thin dispatcher; it must not embed all inventory, diff, rendering, and mutation logic in one long shell file.

### Structure Rationale

- **`profiles/`:** separates public logical differences from real machine identity. A profile can say “Apple Silicon development role” without storing an account name or hostname.
- **`manifests/`:** contains policy that is not naturally owned by Nix modules. It references sources and adapter IDs rather than duplicating versions or embedding shell commands.
- **`schemas/`:** makes state kinds and redaction boundaries mechanically reviewable. A plan schema can reject fields such as `command`, `environment`, `secret_value`, absolute home paths, or arbitrary provider URIs.
- **`lib/recovery/`:** keeps read adapters, pure planning, write adapters, and reporting physically separate. Code review can then prove that `check` cannot import the apply layer.
- **`nix/lib/`:** lets a local wrapper produce a concrete `darwinConfiguration` without hard-coding identity in the public repository.
- **`templates/local-host/`:** documents the local binder shape. The instantiated copy must live outside the public Git tree and contain identity only; secret bindings live elsewhere and are never passed into Nix.
- **`tests/fixtures/`:** keeps synthetic state explicit and prevents tests from treating the current Mac as a disposable fixture.

## Architectural Patterns

### Pattern 1: Shared Baseline + Logical Profile + Local Identity Binder

**What:** Public Git exports reusable modules and logical profiles. A small local wrapper selects a profile and supplies concrete username, hostname and home directory. The wrapper calls an exported `mkDarwinHost` function or imports exported modules; public leaf modules never read real identity directly.

**Why here:** Home Manager's official guidance for multiple machines is a common module plus one top-level composition per machine/user, while nix-darwin flakes require a concrete configuration. The local binder preserves that composition pattern without publishing the binding.

**Important Nix boundary:** The public repository should be consumed as a Git flake (`git+file` for local development or a pinned GitHub revision), because Git flakes include files added to Git. A plain `path:` flake can copy an entire non-Git directory to the world-readable Nix store. Keep the local wrapper directory tiny and identity-only, and never place secret bindings beside its `flake.nix`.

**Example public API shape:**

```nix
{
  lib.mkDarwinHost = {
    logicalHost,
    username,
    hostname,
    homeDirectory,
    system,
  }: /* compose shared modules + profiles.${logicalHost} */;
}
```

**Trade-offs:** Adds a one-time local wrapper bootstrap and means the public repository alone cannot evaluate a fully concrete personal host. That is desirable: “public reproducible policy” and “private machine binding” are different artifacts. Provide a fake binder for CI and a local template for users.

### Pattern 2: One Canonical Value, Multiple Projections

**What:** Do not copy package versions or inventory into recovery manifests. Nix remains canonical for Nix/Homebrew/Home Manager declarations; `.config/mise/config.toml` and project contracts remain canonical for runtime versions; `links.toml` is canonical only for link topology; `toolchains.toml` is canonical only for ownership policy.

**How:** Desired adapters evaluate JSON-safe Nix option values (`nix eval --json`) and parse native manifests. The recovery model is a generated projection, not another editable source of truth.

**Trade-offs:** Adapters must normalize several native formats, but this avoids the larger failure mode where the report and the actual activation configuration disagree.

### Pattern 3: CQRS-Style Read/Write Separation for Machine State

**What:** Inventory adapters form a read model; apply adapters form a write model. `check`, `plan`, and `verify` may import read interfaces but cannot load or dispatch mutation functions. `apply` accepts a previously compiled typed plan rather than re-running discovery.

**Fail-closed rules:**

1. Unknown adapter, action, schema version, target or profile → stop.
2. Missing local binding → stop before evaluation/apply; report only the logical profile.
3. Probe cannot guarantee read-only behavior → status `manual` or `unknown`, never execute it.
4. Physical directory enumeration is never a fallback for tracked desired state.
5. Any precondition drift after plan creation → invalidate the plan and return to `check`.

**Trade-offs:** Slightly more code than a command script, but it makes “plan cannot mutate” and “apply cannot discover” testable properties.

### Pattern 4: Content-Addressed Typed Plan

**What:** A plan contains semantic operation records, not executable shell strings. Its digest covers schema version, logical profile, relevant desired-source digests, observed snapshot digest, adapter versions and ordered operations.

```json
{
  "schema_version": 1,
  "logical_host": "primary-mac",
  "desired_digest": "sha256:<digest>",
  "observed_digest": "sha256:<digest>",
  "operations": [
    {
      "id": "links.ensure.terminal-config",
      "component": "links",
      "plane": "symlink",
      "action": "ensure_link",
      "target_ref": "home:.config/terminal",
      "risk": "replace-with-backup",
      "preconditions": ["target-kind:directory"],
      "rollback": "restore-backup",
      "requires_confirmation": true
    }
  ]
}
```

Forbidden plan fields include raw commands, arbitrary environment variables, resolved secret/provider values, real host/user names and absolute home paths. `target_ref` uses logical namespaces (`home:`, `repo:`, `nix-profile:`) that the local binder resolves only inside the executor.

**Confirmation:** `apply --plan <file>` revalidates the digest and preconditions, displays the exact operation IDs and risks, and requires explicit approval bound to that digest. There is no initial `--yes` path for destructive or privileged operations.

**Trade-offs:** A stale plan must be regenerated, even if the difference looks minor. This is the correct bias for the only working Mac.

### Pattern 5: Dual-Plane Saga, Not a Fake Transaction

**What:** Treat Nix/HM/Homebrew activation and filesystem-link activation as a sequence of independently verifiable steps with compensating rollback. Do not claim ACID semantics across them.

**Recommended apply sequence:**

1. Revalidate plan digest, selected logical host, source digests and live preconditions.
2. Resolve and validate link-plane backup destinations, permissions and free-space preconditions, but do not copy, move or replace targets yet.
3. Perform the non-activation Nix build inside the apply stage; compute/store closure evidence.
4. Show the concrete build/closure change and require a privileged activation checkpoint before `darwin-rebuild switch`.
5. Run the declarative-plane switch and immediately inventory Nix generation, Home Manager target ownership, Homebrew package/service status and exit evidence.
6. If the declarative plane is not verified, stop; do not continue to links.
7. Apply manifest-listed links one at a time: create the target's local backup immediately before replacement, verify the new link, then record a receipt before continuing.
8. Run fresh cross-plane verification and render readiness.

Nix generation rollback can compensate for Nix/Home Manager state, but Homebrew installs, casks and service actions may need separate compensating operations. Link rollback restores the recorded backup. The applied receipt must say which rollback boundaries exist instead of exposing one misleading “rollback available” flag.

### Pattern 6: Secrets as Obligations, Never Desired Values

**What:** Public Git declares that a capability needs a secret, not what the secret is or exactly where a personal vault stores it.

```toml
[[requirements]]
id = "code-host-auth"
purpose = "Authenticate the code-host CLI"
required_for = ["source-control"]
accepted_providers = ["local-file", "keychain", "password-manager", "manual"]
```

An ignored local binding may map `code-host-auth` to a provider-specific reference. That reference is treated as private metadata and never copied into desired state, plans, logs, fixtures, reports or Nix arguments.

**Presence-only contract:**

- Local file adapter checks existence, type, owner and permissions; it never reads content.
- Keychain adapter checks an item selector with value output disabled/discarded; it never requests the password payload.
- Optional password-manager adapter checks reference resolvability with all output suppressed and emits only a boolean/status code. It never uses a command that writes the secret to stdout, a file or the environment during recovery checks.
- v1 recovery never injects secrets. It reports `private-missing` or `manual` and links to a human step.

This is stricter than relying on masking. Official Nix documentation says the store is readable to all users, and 1Password documents that `op read` and `op inject` can materialize values. Therefore neither Nix evaluation nor readiness probes should resolve a value at all.

### Pattern 7: Explicit Symlink Manifest with Writer Ownership

**What:** Replace discovery-by-directory with an allow-list. Each link record includes source, target, owner, allowed target kinds, backup policy and verification rule.

```toml
[[links]]
id = "terminal-config"
source = "repo:.config/terminal"
target = "home:.config/terminal"
owner = "symlink-plane"
conflict = "backup-and-replace-after-confirmation"
```

**Rules:**

- `source` must resolve under the repository and be Git tracked.
- `target` must resolve under the locally bound home after canonicalization.
- Home Manager-owned targets are rejected unless a migration operation explicitly transfers ownership.
- Existing targets are renamed to a unique backup; never `rm -rf`.
- Correct existing links are idempotent no-ops.
- Missing Git/worktree metadata is an error, not permission to scan physical `.config` directories.

**Trade-offs:** Adding an app requires one manifest entry, but the manifest is reviewable and prevents ignored local application state from entering a plan.

### Pattern 8: Isolated Fixture Pyramid

**What:** Divide validation into three levels with progressively stronger effects:

1. **Static/schema:** shell syntax, TOML/JSON parsing, Nix evaluation, ownership graph, forbidden-field scans, plan determinism.
2. **Adapter contract:** fake binaries and synthetic JSON/status output under a temporary fake home; no network and no real manager state.
3. **Opt-in isolated integration:** actual installed managers against temporary `HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_CACHE_HOME`, `XDG_STATE_HOME`, `MISE_DATA_DIR`, `MISE_CACHE_DIR`, `CARGO_HOME`, `RUSTUP_HOME`, `DENO_DIR`, `BUN_INSTALL`, `UV_CACHE_DIR`, `GRADLE_USER_HOME`, and other manager-specific roots.

For mise checks, set `MISE_AUTO_INSTALL=false`, `MISE_NOT_FOUND_AUTO_INSTALL=false`, `MISE_OFFLINE=true`, and a ceiling path. Official mise docs state that auto-install is enabled by default for `mise exec`/tasks, so merely changing `HOME` is not enough. Default tests must never run `mise use --global`, `rustup default`, `direnv allow`, `brew install`, `darwin-rebuild switch`, `defaults write`, service commands, or real link replacement.

Fixtures contain fake identities (`test-user`, `test-host`) and obviously fake provider references; they never snapshot live machine output.

## Data Flow

### Request Flow

```text
User selects logical profile
        │
        ▼
`check`
        ├── validate tracked manifests and local binder shape
        ├── evaluate desired projections (no switch)
        ├── run read-only named-target probes
        └── normalize → observed snapshot
        │
        ▼
`plan`
        ├── validate ownership and privacy invariants
        ├── diff desired vs observed
        └── compile typed operations + preconditions + rollback + digest
        │
        ▼
Human reviews exact plan and confirms digest
        │
        ▼
`apply --plan ...`
        ├── re-check source and target preconditions
        ├── non-activation build
        ├── privileged activation checkpoint
        ├── declarative plane
        ├── stop-on-failure verification
        └── symlink / selected ecosystem operations
        │
        ▼
`verify --plan ...`
        ├── new read-only observation
        ├── compare expected postconditions
        └── sanitized JSON + Markdown readiness report
```

### Desired-State Flow

```text
Nix modules ──`nix eval --json`──┐
mise/project native files ───────┼──→ normalized desired model
ownership manifest ──────────────┤
link manifest ───────────────────┤
secret obligations ──────────────┘
```

The projector should query evaluated option values rather than scrape Nix source. For example, Homebrew desired inventory comes from the evaluated nix-darwin configuration; the same package list should not be copied to `components.toml`.

### Observation Flow

```text
Live command/filesystem output
        │ adapter parses privately
        ├── drop absolute home/store paths where not required
        ├── map concrete user/host → logical profile
        ├── reduce secret probe → status only
        ├── map executable path → owner class + version + precedence
        └── emit typed evidence code
                    │
                    ▼
          normalized observed snapshot
```

Adapters should consume machine-readable output where official tools provide it (`nix eval --json`, `mise ls --json`, Homebrew service JSON). They must not persist raw subprocess output and must treat parse failure as `unknown`, never “healthy”.

### Nix / Home Manager / Homebrew Flow

1. The local wrapper binds identity to a public logical profile and produces the concrete nix-darwin configuration.
2. `check` evaluates only JSON-safe configuration projections and validates the flake. `nix flake check` is evaluation/test evidence; it is not apply evidence.
3. `plan` may use `nix build --dry-run` for a preview, but must not invoke switch or a command that updates the lockfile.
4. After confirmation, apply first builds without activating. Nix's official CLI separates `nix build` from activation, and `--no-link` avoids a shared root `result` link when the orchestrator records an explicit output path.
5. The plan records that nix-darwin activation also runs Home Manager and a generated Homebrew Bundle. Home Manager's official manual confirms that HM activation is part of nix-darwin activation.
6. Preserve `autoUpdate = false`, `upgrade = false`, and `cleanup = "none"` unless a later explicit decision changes policy. Extra Homebrew packages are reported as drift/intentional extras, not silently removed.
7. Do not use generic `brew bundle check` as the sole read-only inventory primitive: official Homebrew documentation warns that `system` entries in a Brewfile execute even during `check`. Query named package/tap/service state directly or validate a known generated Brewfile shape first.
8. After switch, verify Nix generation, Home Manager-owned targets, declared formulae/casks, bounded services and the selected runtime-manager entry points separately.

### Toolchain Contract Flow

```text
Public ownership policy
   + global fallback source
   + project-native contract(s)
   + executable provenance/PATH observation
                         │
                         ▼
                ownership validator
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
   one owner          conflict         missing contract
   verify only        plan migration   report / fixture guidance
```

The ownership manifest should describe roles such as `runtime-owner`, `package-manager-owner`, `system-library-owner`, and `environment-loader`; it should not assume that “one tool owns the entire ecosystem.” This allows mise, uv, rustup, project lockfiles, Nix devShell and direnv to cooperate without competing for the same binary/version decision.

Mise project configuration is hierarchical, so inventory must record the source/precedence of the selected version rather than only the final version. For scripts and fixtures, prefer explicit `mise exec` with auto-install disabled; shell activation is a separate interactive-shell concern.

Direnv remains an explicit environment loader. Official direnv behavior requires `direnv allow`; the recovery system may report an unapproved `.envrc`, but it must not approve project code automatically. nix-direnv is an implementation of `use_nix`/`use_flake`, not a replacement for that authorization boundary.

### Symlink Flow

1. Projector reads only `links.toml`; no `find` fallback exists.
2. Inventory resolves each target under the bound home and uses `lstat`/`readlink` without traversing source content.
3. Planner classifies each target as `correct`, `missing`, `wrong-link`, `unmanaged-file`, `unmanaged-directory`, `Home-Manager-owned`, or `unsafe`.
4. Only `missing`, `wrong-link`, or explicitly approved unmanaged targets can produce operations.
5. Apply creates/records a backup before replacement, creates the new link, and verifies its canonical source.
6. On failure, stop at the current target and preserve both receipt and backups; never continue to unrelated targets.

### Secret and Private-State Flow

```text
Public requirement ID ───────┐
                             ├── local presence adapter ──→ status only
Ignored provider binding ────┘                              (`present|missing|manual|unknown`)

Secret value ──────────────────────────────────────────────X never enters this flow
```

The report may say “code-host-auth: missing” but must not contain a vault name, item name, account, email, token prefix, keychain label, file contents, or private absolute path. Login sessions and application databases remain `excluded`, not “drift”.

### Readiness Report Model

Use a bounded status vocabulary so reports from different Macs are comparable:

| Status | Meaning |
|--------|---------|
| `verified` | Declared and proven by fresh postcondition evidence |
| `declared-unapplied` | Desired state exists but the machine does not yet have it |
| `drift` | Observed public state conflicts with the selected desired profile |
| `private-missing` | A required local overlay or secret obligation is absent; no value was read |
| `manual-required` | A login, TCC permission, App Store step, or unsafe-to-automate action remains |
| `excluded` | Deliberately outside recovery scope, such as chat history or application databases |
| `unknown` | Probe unsupported, failed to parse, or could not prove read-only behavior |
| `unverified-platform` | Declared structurally but not validated on this architecture/host class |

A top-level readiness result is the conjunction of required component evidence, not a percentage. Optional/excluded components can be summarized separately. Before a clean VM/second Mac run, the strongest allowed label is `recovery-ready-on-current-host`, never `fresh-install-verified`.

## Plan / Apply Safety Contract

The following invariants should be enforced by tests and code structure, not only documentation:

1. `check`, `plan`, and `verify` cannot dispatch any apply adapter.
2. `apply` requires a plan file, matching digest, matching logical profile, matching relevant source digests, and fresh preconditions.
3. Apply operations are enum-based and adapter-owned; no plan-provided shell is evaluated.
4. Newly discovered drift during apply invalidates the plan; it is never appended dynamically.
5. Privileged commands are visible and isolated. `sudo` is never acquired at program start or cached for unrelated steps.
6. Declarative build and declarative switch have separate evidence and an activation checkpoint.
7. Every replace/remove/service operation has an explicit risk class and rollback/compensation text before confirmation.
8. Symlink replacement uses backup-and-restore, never recursive deletion.
9. Secret obligations cannot produce a value-bearing operation in v1.
10. Reports and receipts pass schema validation plus privacy scanning before they are written.

## Scaling Considerations

This is a personal workstation system, not a fleet-control service. Scale means more profiles and heterogeneous Macs, not thousands of remote agents.

| Scale / scenario | Architecture adjustment |
|------------------|-------------------------|
| One current Mac | One logical profile, one local binder, local report store; keep implementation simple and validate non-destructively |
| Two to five Macs | Reuse common/role modules, add public logical profiles, keep one local binder per Mac, compare sanitized reports by schema version |
| Different roles | Compose role overlays (`development`, future optional roles); differences must be explicit profile data rather than conditionals on real hostname |
| Intel + Apple Silicon | Put `system`/capabilities in public profile; adapters return `unverified-platform` until tested; never infer support from successful evaluation alone |
| Clean VM / replacement Mac | Instantiate a new binder, run check/plan first, apply component waves, retain receipts; only then promote fresh-install verification status |
| Remote/fleet orchestration | Out of scope. Do not add a central database, daemon or remote execution plane until there is a real multi-host operational need |

### Scaling Priorities

1. **First bottleneck — conditional sprawl:** host-specific `if hostname == ...` branches become hard to audit. Fix with common → role → logical-host composition and explicit expected differences.
2. **Second bottleneck — report/schema drift:** independent scripts emit incompatible text. Fix with versioned normalized schemas and adapter contract tests.
3. **Third bottleneck — mutable external systems:** Homebrew/app services differ across hosts despite the same Nix config. Keep conservative activation and improve evidence; do not respond with silent cleanup.

## Anti-Patterns

### Anti-Pattern 1: One “Sync Everything” Script

**What people do:** A script discovers packages, installs tools, switches Nix, replaces links and prints “done”.
**Why it's wrong:** It erases activation-plane boundaries, makes partial failure impossible to reason about, and can mutate the Mac before the complete change set is visible.
**Do this instead:** Thin control plane, typed plan, independent adapters, stop-and-verify gates.

### Anti-Pattern 2: Executable Plan Files

**What people do:** Generate a shell script as the plan and execute it after review.
**Why it's wrong:** Quoting/injection problems, accidental edits and environment expansion make the reviewed artifact differ from executed behavior; secrets can leak through command lines.
**Do this instead:** Data-only operations validated against a schema and executed by versioned allow-listed adapters.

### Anti-Pattern 3: Local Identity Hidden Through `--impure`

**What people do:** Read `$USER`, hostname or an ignored repo file with `builtins.getEnv`/impure evaluation.
**Why it's wrong:** Results depend on ambient state, make evaluation harder to reproduce, and tempt storing secret/local files inside a flake source.
**Do this instead:** Explicit local wrapper with a small identity-only binder and a Git-tracked public flake input.

### Anti-Pattern 4: Secret Values or Personal Provider URIs in Nix

**What people do:** Put credentials, decrypted templates, provider-specific item references or local secret files into Nix module arguments.
**Why it's wrong:** Nix store paths are readable to local users and may reach external caches; provider URIs can reveal personal metadata even without the value.
**Do this instead:** Public obligation IDs plus a runtime, ignored binding that is never evaluated by Nix; v1 reports presence only.

### Anti-Pattern 5: Physical `.config` Discovery as Safety Fallback

**What people do:** If Git discovery fails, scan all local application directories and offer them for linking.
**Why it's wrong:** Ignore rules stop Git staging, not filesystem enumeration; private/untracked app state can enter the operation set.
**Do this instead:** Explicit tracked link manifest and fail closed when Git/source validation is unavailable.

### Anti-Pattern 6: Duplicating Runtime Versions in an Ownership Manifest

**What people do:** Copy Node/Go/etc. versions from mise/project files into a central YAML/TOML matrix.
**Why it's wrong:** The matrix becomes another desired-state writer and will drift.
**Do this instead:** Manifest declares who owns the version and which native file is authoritative; adapters project the actual value.

### Anti-Pattern 7: Assuming Nix Rollback Reverts Homebrew and Links

**What people do:** Advertise one generation rollback for the whole workstation.
**Why it's wrong:** nix-darwin activation invokes mutable Homebrew operations, and the separate link script can replace filesystem state outside the generation.
**Do this instead:** Component-specific compensation and explicit rollback coverage in receipts.

### Anti-Pattern 8: Blind `brew bundle check` as a Pure Probe

**What people do:** Run an arbitrary/generated Brewfile with `brew bundle check` and assume no command can execute.
**Why it's wrong:** Homebrew documents that Brewfile `system` commands run during `check` too.
**Do this instead:** Direct read-only inventory commands or prior validation that the generated Brewfile contains only safe declarative entries.

### Anti-Pattern 9: Tests that Inherit Real Manager State

**What people do:** Run fixture projects under the real HOME or invoke `mise exec` with default auto-install behavior.
**Why it's wrong:** Tests may install runtimes, trust project config, write caches, alter global defaults, or accidentally validate against already-working local state.
**Do this instead:** Fake adapters by default; opt-in integration with all state roots isolated, auto-install/offline controls explicit, and no apply commands.

### Anti-Pattern 10: Automatic `direnv allow`

**What people do:** Recovery approves every tracked `.envrc` to make projects appear ready.
**Why it's wrong:** `.envrc` is executable code and direnv's allow step is an intentional trust decision.
**Do this instead:** Report approval as a manual requirement and let the user review/allow each project.

## Integration Points

### External Tools and Services

| Service / tool | Integration pattern | Notes |
|----------------|---------------------|-------|
| Nix | Evaluate JSON-safe projections; dry-run/build without activation; store no secrets | Flakes copy tracked Git sources; secret values must never enter derivations/store |
| nix-darwin | Concrete config produced by local binder + public modules | One switch is a coarse system activation boundary |
| Home Manager | Remains embedded in nix-darwin for the primary route | Collision checks/backups exist, but do not mix ownership with the fallback link route |
| Homebrew | Desired inventory is generated by nix-darwin; observed inventory queried directly | Preserve no auto-update/upgrade/cleanup; mutable side effects need compensation |
| mise | Runtime owner where policy assigns it; inspect source and selected versions with auto-install disabled | Config is hierarchical; global fallback and project contract must be distinguished |
| uv | Python project/package responsibility per ownership manifest | Do not duplicate Python runtime/project responsibility without an explicit policy |
| rustup | Rust toolchain/channel/target responsibility | Project `rust-toolchain.toml` remains native contract |
| direnv | Explicit project environment loader | Never auto-authorize `.envrc`; use status/manual evidence |
| nix-direnv | Cached `use_nix`/`use_flake` integration | It supplements direnv; fallback behavior must be visible in verification |
| macOS Keychain | Optional local provider/presence source | Access can prompt; no value retrieval in readiness probes |
| 1Password CLI | Future optional provider adapter | Public requirements use opaque IDs; provider URI remains local; no `op read` in check/report |
| Git/GitHub | Public desired-state transport and tracked-source boundary | Never auto-push; Git tracking does not protect physical filesystem scans |

### Internal Boundaries

| Boundary | Communication | Required invariant |
|----------|---------------|--------------------|
| Public profile ↔ local binder | Typed identity fields | Public side never contains real identity; binder never contains secrets |
| Desired projector ↔ Nix | Evaluated JSON-safe options | No source scraping, impure environment reads or lockfile writes |
| Inventory ↔ planner | Normalized observed schema | No raw stdout, absolute personal paths or secret material |
| Planner ↔ apply | Immutable plan schema + digest | No command strings; no new operations during apply |
| Apply ↔ declarative plane | Named adapter calls | Build and switch are distinct; switch requires checkpoint |
| Apply ↔ symlink plane | Manifest operation IDs | Named targets only; backup before replace; no recursive deletion |
| Secret adapter ↔ report | Status enum + public requirement ID | Never pass provider reference or value across boundary |
| Verifier ↔ report | Typed evidence codes | Exit 0 alone is insufficient proof |
| Fixtures ↔ adapters | Fake command/path interfaces | Real HOME and global manager roots are forbidden |

## Build Order Implications

Architecture dependencies suggest this implementation order while preserving the user's “toolchain governance before full recovery readiness” priority:

1. **Safety substrate and schemas**
   - Define ownership, desired/observed/plan/report schemas, forbidden fields, logical path namespaces and isolated fixture harness.
   - Add no live apply behavior.

2. **Read-only toolchain inventory and ownership validator**
   - Normalize current executable provenance, manager source and project-contract facts.
   - Validate six ecosystems with synthetic fixtures; disable manager auto-install/network in all live probes.

3. **Ecosystem-by-ecosystem contracts and migrations**
   - Node, Go, Python, Rust, Deno/Bun, JVM in independent changes.
   - For each: desired owner → isolated verification → read-only live check → rollback → only then duplicate removal.

4. **Logical host composition and local binder**
   - Extract shared Nix/Home Manager modules and public `primary-mac` profile.
   - Add a fake CI binder and local wrapper template before removing tracked identity from the current composition.

5. **Readiness inventory and report**
   - Add Nix/Homebrew/defaults/services/links/private/manual/excluded adapters.
   - Produce sanitized reports without an apply path; prove current-host `recovery-ready` semantics first.

6. **Fail-closed symlink plan/apply**
   - Introduce explicit link manifest, complete dry-run plan, backup/restore and idempotency.
   - Make `setup_mac.sh` delegate or refuse unsafe fallback; preserve the plane boundary.

7. **Integrated confirmed apply orchestration**
   - Content-addressed plans, precondition invalidation, build/switch sub-gate, component receipts and compensating rollback.
   - Keep Homebrew conservative; do not silently clean extras.

8. **Clean environment validation later**
   - macOS VM or second Mac, fresh local binder, staged apply and full evidence review.
   - Only this phase can justify `fresh-install-verified`.

## Confidence Assessment

| Area | Confidence | Reason |
|------|------------|--------|
| Existing dual-plane boundary | HIGH | Directly verified from `flake.nix`, `setup_mac.sh`, and codebase maps |
| Build versus activation separation | HIGH | Nix, Home Manager and nix-darwin official docs distinguish build/evaluation from activation |
| Homebrew conservative/drift model | HIGH | nix-darwin and Homebrew official docs define activation/cleanup/check behavior |
| Mise/direnv ownership and safe probing | HIGH | Official docs define hierarchical config, auto-install defaults, JSON listing and explicit direnv allow |
| Secret-value exclusion | HIGH | Nix store and provider materialization behavior are explicitly documented |
| Logical local binder | MEDIUM | Sound Nix composition pattern, but exact wrapper API and local UX need implementation validation |
| Content-addressed plan schema | MEDIUM | Standard safety design derived for this repository; exact fields need threat-model and fixture iteration |
| Cross-plane compensation | MEDIUM | Boundaries are known, but each Homebrew/service/link rollback must be proven per operation |
| Intel/VM support | LOW until tested | Structure can represent it, but current evidence covers only the present Apple Silicon host class |

## Sources

All external technical claims below use primary/official documentation or official project repositories.

### Nix

- [Nix manual — Secrets](https://nix.dev/manual/nix/2.34/store/secrets) — the Nix store is readable to all users; keep secret values out of derivations/store.
- [Nix concepts — Flakes](https://nix.dev/concepts/flakes.html) — Git flakes build tracked files and default to pure evaluation.
- [Nix manual — Flake references](https://nix.dev/manual/nix/2.26/command-ref/new-cli/nix3-flake.html) — `git+file` and `path:` semantics.
- [Nix documentation — Working with local files](https://nix.dev/tutorials/working-with-local-files.html) — local flake directories and Git-tracked source boundaries.
- [Nix manual — `nix eval`](https://nix.dev/manual/nix/2.34/command-ref/new-cli/nix3-eval.html) — JSON-safe desired projections.
- [Nix manual — `nix build`](https://nix.dev/manual/nix/2.34/command-ref/new-cli/nix3-build.html) — build, dry-run, JSON and no-link behavior.
- [Nix manual — `nix flake check`](https://nix.dev/manual/nix/2.28/command-ref/new-cli/nix3-flake-check.html) — evaluation and check builds are validation, not activation.
- [Nix manual — `nix store diff-closures`](https://nix.dev/manual/nix/2.28/command-ref/new-cli/nix3-store-diff-closures) — closure-level change evidence.

### nix-darwin and Home Manager

- [nix-darwin official repository README](https://github.com/nix-darwin/nix-darwin/blob/master/README.md) — flake composition and system activation entry.
- [nix-darwin configuration options](https://nix-darwin.github.io/nix-darwin/manual/) — Homebrew generation, activation policy, cleanup modes, system checks and revision metadata.
- [Home Manager manual](https://nix-community.github.io/home-manager/) — nix-darwin integration, common/per-machine module composition, build/switch separation, collision behavior and rollback.
- [Home Manager official repository](https://github.com/nix-community/home-manager) — current project source and manual provenance.

### Homebrew

- [Homebrew Bundle and Brewfile documentation](https://docs.brew.sh/Brew-Bundle-and-Brewfile) — Bundle desired-state behavior, `check`, cleanup and the warning that Brewfile `system` commands run during checks.
- [Homebrew manpage](https://docs.brew.sh/Manpage) — machine-readable/service/list/check/cleanup command semantics and `--no-upgrade` behavior.

### Toolchains and project environments

- [mise configuration](https://mise.jdx.dev/configuration.html) — config hierarchy, local files and data/cache locations.
- [mise settings](https://mise.jdx.dev/configuration/settings.html) — auto-install, offline, ceiling-path and lock settings.
- [mise `ls`](https://mise.jdx.dev/cli/ls.html) — current/missing/source and JSON inventory.
- [mise `install`](https://mise.jdx.dev/cli/install.html) — explicit dry-run and install behavior.
- [mise getting started](https://mise.jdx.dev/getting-started) — `mise exec` versus interactive shell activation.
- [direnv manual](https://direnv.net/man/direnv.1.html) — explicit `allow`/`deny` and `exec` semantics.
- [direnv configuration manual](https://direnv.net/man/direnv.toml.1.html) — trust/whitelist risks and configuration scope.
- [nix-direnv official repository](https://github.com/nix-community/nix-direnv) — cached `use_nix`/`use_flake`, watched files, fallback and Home Manager integration.

### Secret providers

- [Apple Keychain Access User Guide](https://support.apple.com/guide/keychain-access/what-is-keychain-access-kyca1083/mac) — local macOS keychain role and access boundary.
- [1Password Developer — Secret references](https://www.1password.dev/cli/secret-references) — references versus runtime materialization through `op read`, `op run`, or `op inject`.

### Repository evidence

- `.planning/PROJECT.md`
- `.planning/codebase/ARCHITECTURE.md`
- `.planning/codebase/STRUCTURE.md`
- `.planning/codebase/CONCERNS.md`
- `flake.nix`
- `setup_mac.sh`

---
*Architecture research for: Yet Another Mac Config*
*Researched: 2026-07-10*
