# Pitfalls Research

**Domain:** Public, privacy-safe macOS configuration, six-ecosystem toolchain governance, and workstation recovery readiness
**Researched:** 2026-07-10
**Confidence:** HIGH for documented tool behavior; MEDIUM for future multi-host design choices that still require implementation spikes

## Roadmap Phase Vocabulary

The final roadmapper may rename or renumber these phases. This document uses the following topic labels so every prevention measure has an explicit owner:

| Label | Roadmap topic |
|---|---|
| **Safety Foundation** | Non-mutating test harness, privacy rules, plan schema, state vocabulary, and current-state inventory |
| **Ownership Inspector** | One-owner matrix and effective binary/version/config-source inspection across all six ecosystems |
| **Node Governance** | Node, npm, pnpm, and Corepack contract and migration |
| **Go Governance** | Go toolchain contract and migration |
| **Python Governance** | Python and uv contract and migration |
| **Rust Governance** | Rust and rustup contract and migration |
| **Deno/Bun Governance** | Deno and Bun contract and migration |
| **JVM Governance** | Java/JDK, Maven, and Gradle contract and migration |
| **Multi-host Composition** | Shared baseline, privacy-safe logical host overlays, architecture/prefix handling |
| **Recovery Engine** | `check -> plan -> confirm -> apply -> verify`, backups, resume, and component rollback |
| **Readiness Drill** | Read-only/current-Mac drill and machine-readiness report |
| **Clean-host Validation** | Future macOS VM or second-Mac end-to-end evidence; not part of the current milestone |

## Critical Pitfalls

### PF-01: Treating a Nix build or switch as one atomic workstation transaction

**Severity / confidence:** CRITICAL / HIGH

**What goes wrong:**
`nix flake check` or `darwin-rebuild build` succeeds, so the workstation is reported as restored. Alternatively, `darwin-rebuild switch` fails late and the operator assumes a Nix generation rollback reverses every effect. Neither conclusion is safe: the repository has separate Nix, Home Manager, Homebrew Bundle, service, and symlink planes, and the symlink plane is not run by `darwin-rebuild` at all.

**Why it happens:**
Nix's build model is strongly reproducible, which encourages an incorrect extension of the transaction boundary to mutable external tools. The nix-darwin Homebrew module explicitly generates a Brewfile and invokes `brew bundle` during activation. Home Manager also has a check/write boundary for activation scripts. These are related stages, not proof that all macOS state changes are one reversible closure.

**Consequences:**
- A Homebrew tap, cask installer, service start, or Home Manager file collision can fail after earlier work has completed.
- A Nix rollback can restore the selected Nix generation while leaving Homebrew packages, cask files, service/login-item state, or hand-created symlinks changed.
- Reports can show a false green status while app configuration, TCC permissions, secrets, and login state remain missing.

**Prevention:**
- Model every mutable subsystem as a separate operation with its own preconditions, apply result, postcondition, and rollback notes.
- Keep `check` and non-activating build evidence separate from `apply` evidence.
- Never describe a build as activation, and never describe activation as full workstation readiness.
- Isolate Homebrew inventory, Homebrew services, Home Manager files, macOS defaults, and repository symlinks into distinct plan sections and confirmation groups.
- Record before-state only for allowlisted, non-secret facts required for rollback.

**Warning signs:**
- A single `success=true` field represents the entire recovery.
- A plan contains only `darwin-rebuild switch` and no post-switch component checks.
- Rollback documentation contains only `darwin-rebuild --rollback`.
- The readiness report does not mention the interactive symlink plane.

**Phase to address:** Safety Foundation establishes state vocabulary; Recovery Engine enforces component boundaries; Readiness Drill verifies every plane.

**Recovery steps:**
1. Stop at the failed component; do not rerun the full switch blindly.
2. Preserve the redacted plan ID, repo revision, and component results.
3. Compare each component with its captured before-state.
4. Roll back the Nix generation only for Nix-owned state.
5. Revert Homebrew, service, file-link, or app-specific effects with their component procedure; do not assume the Nix rollback handled them.
6. Re-run `check` and regenerate a new plan before any retry.

**Verification:** A test plan must enumerate every activation plane and refuse to mark the machine ready when any plane is `unknown`, `manual`, `not-applied`, or `failed`. Automated tests use synthetic state only; an authorized real switch is verified separately and is never run by the test suite.

**Official evidence:** [Nix `flake check` checks evaluation and declared checks](https://nix.dev/manual/nix/2.28/command-ref/new-cli/nix3-flake-check.html); [nix-darwin Homebrew options and activation behavior](https://nix-darwin.github.io/nix-darwin/manual/); [Home Manager activation check/write boundary and dry-run contract](https://github.com/nix-community/home-manager/blob/master/modules/home-environment.nix); [Homebrew Bundle behavior](https://docs.brew.sh/Brew-Bundle-and-Brewfile).

---

### PF-02: Enabling Homebrew cleanup or zap to manufacture machine equality

**Severity / confidence:** CRITICAL / HIGH

**What goes wrong:**
The project changes `homebrew.onActivation.cleanup` from the current conservative `"none"` policy to `"uninstall"` or `"zap"`, or calls `brew bundle cleanup --force`, to make two Macs look identical. Extra formulae disappear, casks are removed, and `zap` may delete preferences, caches, or shared resources that are outside the repository's recovery scope.

**Why it happens:**
Exact inventory equality is easier to measure than safe convergence. The nix-darwin option names make destructive cleanup look like a normal declarative setting, while the actual behavior delegates to Homebrew uninstall/zap operations.

**Consequences:**
- Existing tools or applications used by unmodeled projects can be removed.
- Cask `zap` can remove user-library files and shared resources used by other applications.
- Account-heavy apps may lose local preferences or state that the repository intentionally does not restore.
- A failed activation may leave the machine less usable than before.

**Prevention:**
- Preserve `cleanup = "none"` for this milestone.
- Implement extra packages as `drift` or `unmanaged-present`, not as automatic deletion candidates.
- If cleanup is ever proposed, require a separate future decision, explicit per-item allowlist, data-impact review, backup proof, and a dedicated confirmation—not a general recovery confirmation.
- Treat `"check"` as an activation-blocking policy, not as a harmless report mode; prefer an independent read-only drift reporter.
- Never use cask `zap` in automated recovery.

**Warning signs:**
- A plan contains `--cleanup`, `--force-cleanup`, `--zap`, `brew uninstall`, or a change away from `cleanup = "none"`.
- “No extra packages” is a release criterion.
- The plan cannot explain where an uninstalled cask stores user data.

**Phase to address:** Safety Foundation blocks destructive verbs; Recovery Engine renders extras as drift; Readiness Drill confirms no cleanup occurred.

**Recovery steps:**
1. Stop cleanup immediately.
2. Use the captured inventory to identify removed items.
3. Reinstall only items the user confirms are still wanted.
4. Restore application data from the user's backup source if it was removed; the config repository is not that backup.
5. Re-run privacy and service checks because reinstalling a cask does not restore permissions or login state.

**Verification:** Static policy tests reject destructive Homebrew flags in default plans. A fixture with undeclared packages must report extras without producing uninstall operations.

**Official evidence:** [nix-darwin documents `none`, `check`, `uninstall`, and `zap`](https://nix-darwin.github.io/nix-darwin/manual/); [Homebrew Cask documents that zap may remove preferences, caches, and shared resources](https://docs.brew.sh/Cask-Cookbook); [Homebrew command reference for uninstall/zap](https://docs.brew.sh/Manpage).

---

### PF-03: Declared version equals effective toolchain version

**Severity / confidence:** CRITICAL / HIGH

**What goes wrong:**
A project file or global fallback contains the expected version, but the command that actually runs comes from a different owner or a higher-precedence override. All six ecosystems have independent resolution layers:

- mise merges parent, project, environment-specific, local, and global configuration; environment variables can override configured versions.
- Rust uses command-line, `RUSTUP_TOOLCHAIN`, directory overrides, `rust-toolchain.toml`, then the default toolchain.
- Go's default `GOTOOLCHAIN=auto` can find or download a newer toolchain based on `go.mod` or `go.work`.
- uv can select system or managed Python and automatically download Python by default.
- Gradle auto-detects JDKs and may auto-provision one; Maven can use a machine-local `~/.m2/toolchains.xml` path different from `JAVA_HOME`.
- Node package-manager behavior depends on Corepack availability and the project package-manager contract; Corepack is no longer bundled starting with Node 25.

**Why it happens:**
The governance model records desired versions but does not inspect executable provenance, merged configuration, hidden overrides, auto-download settings, or build-tool-specific JVMs.

**Consequences:**
- The same project builds with different compilers on two Macs.
- Removing a “duplicate” installation deletes the binary that was actually active.
- A project appears pinned but downloads an unreviewed runtime on first use.
- IDEs, hooks, login shells, non-interactive shells, and build daemons use different executables.

**Prevention:**
- Define one primary owner per runtime and separate owners for package managers, project dependencies, and system libraries.
- Build an effective-state inspector that records, without mutation: executable path, version, manager/config source, project file, relevant override status, architecture, and whether auto-download is enabled.
- Verify login zsh, non-interactive zsh, direct manager execution, and project-directory execution separately.
- Make each ecosystem migration prove the new route before removing the old route.
- Treat “declared”, “installed”, “selected”, and “executed” as four different states.

**Warning signs:**
- Tests assert only file contents or `tool --version` without executable path/config source.
- Two managers list the same runtime.
- `mise cfg`, `rustup show`, `go env GOTOOLCHAIN`, uv Python selection, or Gradle toolchain evidence is absent.
- IDE or hook failures occur while an interactive shell works.

**Phase to address:** Ownership Inspector first; then every ecosystem-specific governance phase must close its own resolution chain.

**Recovery steps:**
1. Do not uninstall anything.
2. Capture redacted effective resolution in each execution context.
3. Disable or remove only the unintended override, starting with environment and directory-local state.
4. Restore the last known owner path if a removal already occurred.
5. Re-run project-level and non-interactive verification before resuming cleanup.

**Verification:** Each ecosystem has a fixture and a read-only current-machine probe that deliberately introduces a higher-precedence synthetic override in an isolated HOME and proves the inspector reports it. Real HOME override files are never changed.

**Official evidence:** [mise configuration hierarchy](https://mise.jdx.dev/configuration.html); [rustup override precedence](https://rust-lang.github.io/rustup/overrides.html); [Go toolchain selection and downloads](https://go.dev/doc/toolchain); [uv Python selection and automatic downloads](https://docs.astral.sh/uv/concepts/python-versions/); [Gradle JDK detection/provisioning](https://docs.gradle.org/current/userguide/toolchains.html); [Maven toolchains](https://maven.apache.org/guides/mini/guide-using-toolchains); [Node 25 Corepack change](https://nodejs.org/download/release/v25.8.0/docs/api/corepack.html).

---

### PF-04: Shell ownership race and destructive symlink replacement

**Severity / confidence:** CRITICAL / HIGH

**What goes wrong:**
Home Manager owns the active zsh startup file, but the legacy bootstrap can replace it with the fallback repository symlink. The two paths do not currently have equivalent mise activation/order. The bootstrap also uses repeated `rm -rf` replacement blocks for zsh and app configuration targets.

**Why it happens:**
Two historically valid bootstrap paths remained available after Home Manager became primary. Interactive “yes” prompts are treated as sufficient safety even though target ownership, canonical paths, backups, and whole-plan review are missing.

**Consequences:**
- PATH order and selected runtime depend on whichever bootstrap ran last.
- Home Manager activation produces backup churn or collisions.
- A mistaken target or broad confirmation irreversibly removes existing configuration.
- Recovery behavior differs between the primary and fallback routes.

**Prevention:**
- Make Home Manager and fallback shell modes explicit and mutually exclusive.
- Detect Home Manager-owned targets and refuse replacement unless a dedicated migration plan is confirmed.
- Centralize link operations behind one planner that canonicalizes the target, proves it is under the resolved target home, records target type/owner, and defaults to preserve.
- Replace deletion with a same-filesystem backup/rename and create the new link only after backup succeeds.
- Add equivalence tests for shared aliases, PATH ordering, mise availability, local-overlay ordering, and private-value non-disclosure.

**Warning signs:**
- Both a Home Manager generation link and a repository symlink are considered valid for the same live target.
- The fallback shell cannot resolve the declared global fallback runtime.
- A plan includes `rm -rf` or a user-supplied path that was not canonicalized.
- Existing target type/inode/checksum is absent from the plan.

**Phase to address:** Safety Foundation adds safe link primitives; Ownership Inspector covers shell contexts; Recovery Engine chooses one shell mode.

**Recovery steps:**
1. Stop further bootstrap or Home Manager activation.
2. Identify whether the live target is a regular file, repository link, or Nix-store/Home Manager link.
3. Restore the planner-created backup; if none exists, use the user's external backup rather than guessing contents.
4. Select exactly one owner and re-run its non-mutating verification before apply.

**Verification:** Run all overwrite, conflict, interruption, and idempotency tests against a temporary fake home. Tests assert that paths outside that fake home cannot be selected and that no branch invokes `rm -rf` on a live target.

**Official evidence:** [Home Manager activation requires pre-write checks and idempotent post-boundary actions](https://github.com/nix-community/home-manager/blob/master/modules/home-environment.nix); [Home Manager's documented user-environment ownership](https://nix-community.github.io/home-manager/introduction.html).

---

### PF-05: “Isolated” tests still read or write the real Mac

**Severity / confidence:** CRITICAL / HIGH

**What goes wrong:**
A fixture runs in `/tmp` or with a changed `HOME`, but managers still discover parent/global config, macOS-specific cache locations, trust databases, managed runtimes, virtual environments, or build daemons. A harmless-looking version check can trigger auto-install or network access. The current work Mac becomes the integration fixture.

**Why it happens:**
Each manager has several state roots, and not all of them follow `HOME` alone. For example, mise separates config/data/cache/state; uv has cache, managed Python, tool, and project-environment locations; rustup separates `RUSTUP_HOME` and `CARGO_HOME`; Deno and Bun have their own cache/install roots; Gradle has its user home. Parent-directory config discovery also survives a naïve temporary working directory.

**Consequences:**
- Tests change global defaults, trust state, caches, lockfiles, or installed runtimes.
- A test passes only because the real Mac already has a tool or cached artifact.
- A failed test can break active projects or consume large amounts of disk/network.
- Reports may accidentally expose real paths, registries, or credentials inherited from the shell.

**Prevention:**
- Default to static parsing and read-only inspection; never run install/use/update/sync/uninstall commands merely to make a test pass.
- Create a dedicated harness that starts with an allowlisted environment, temporary HOME/XDG roots, and tool-specific roots for mise, uv, rustup/Cargo, Go, Deno, Bun, Gradle, and Maven.
- Disable config/env/hook discovery and automatic runtime downloads where the tool supports it.
- Set a configuration-search ceiling and run fixtures outside all real project ancestors.
- Deny or stub network for tests whose contract is offline; classify any required network/install test as future clean-host work.
- On test exit, assert the real HOME, repository worktree, global config files, service list, and executable selection are unchanged.

**Warning signs:**
- A test exports only `HOME=/tmp/...`.
- Test output contains the real home directory, Homebrew prefix writes, real Keychain prompts, or a manager's normal data directory.
- A test downloads a runtime or creates `.venv`, `mise.lock`, `bun.lock`, Gradle daemon state, or trust records outside its fixture.
- Tests require closing real IDEs or shells.

**Phase to address:** Safety Foundation, before any ecosystem migration.

**Recovery steps:**
1. Stop the test process and record only non-secret paths it touched.
2. Compare real global config, selected binaries, and manager state against the pre-test snapshot.
3. Restore changed config from version control or the user's backup; do not delete caches/toolchains until ownership is proven.
4. Fix the harness and reproduce only in a fresh temporary root.

**Verification:** The harness runs a sentinel test with read-only marker files in every real state root and proves their metadata/content do not change. It also fails if output contains the real home path or if the repository diff changes.

**Official evidence:** [mise directory roots](https://mise.jdx.dev/directories.html); [mise config discovery and isolation flags](https://mise.jdx.dev/cli/); [uv environment variables](https://docs.astral.sh/uv/reference/environment/); [uv storage locations](https://docs.astral.sh/uv/reference/storage/); [rustup install roots](https://rust-lang.github.io/rustup/installation/); [Deno environment roots](https://docs.deno.com/runtime/reference/env_variables/); [Bun global cache](https://bun.sh/docs/pm/global-cache); [Gradle toolchain/user-home behavior](https://docs.gradle.org/current/userguide/toolchains.html).

---

### PF-06: Keeping secrets out of Git but leaking them through Nix, logs, plans, or history

**Severity / confidence:** CRITICAL / HIGH

**What goes wrong:**
A secret value is not intentionally committed, yet it is interpolated into a Nix expression or generated file, printed by a diagnostic command, captured in a plan/readiness report, embedded in an environment dump, or committed and later “deleted.” The public repository or Nix store still exposes it.

**Why it happens:**
`.gitignore` and token-pattern scanners are mistaken for a complete privacy boundary. Nix copies referenced data into its store, which is readable to local users and may reach caches. Secret scanners do not recognize every private value, and Git history retains deleted content.

**Consequences:**
- Credentials require immediate rotation.
- Private account, host, network, or identity data becomes public even when it is not an API token.
- History rewriting disrupts clones, signatures, pull requests, and forks and still cannot erase other users' copies.
- Build logs or readiness artifacts become unsafe to share or commit.

**Prevention:**
- Git stores only abstract secret IDs, purpose, required/optional state, provider type, and a safe validation rule—never values, account addresses, private endpoints, or recovery material.
- Never interpolate plaintext secret files/values into Nix derivations or store paths.
- Read secrets only at runtime from an access-controlled local file, Keychain, or future provider; readiness checks presence/shape without reading or printing values.
- Use structured allowlisted output, home/host redaction, no full environment dumps, and no shell tracing around secret-aware commands.
- Combine staged diff review, full-repository/history-aware scanning where appropriate, generic privacy pattern checks, and GitHub push protection; scanners supplement rather than replace review.

**Warning signs:**
- Nix code contains a credential string, `${secretPath}` interpolation, or generated config with a token.
- A report includes `env`, shell exports, full absolute home paths, hostnames, emails, private IPs, or provider responses.
- A secret file was committed once and then merely added to `.gitignore` or deleted.
- Redaction is performed after raw output has already been written to disk.

**Phase to address:** Safety Foundation defines the schema and redaction; every later phase must pass privacy gates; Recovery Engine checks presence only.

**Recovery steps:**
1. Revoke or rotate the exposed credential first.
2. Stop publishing artifacts and remove unsafe local reports.
3. Follow GitHub's sensitive-data removal process if the value reached history, coordinating affected clones/forks.
4. Remove the Nix reference and garbage-collect only after confirming no required generations depend on it; external caches require their own remediation.
5. Add a regression fixture containing synthetic canary values, never a real secret.

**Verification:** Synthetic secrets placed in ignored/local provider fixtures must never appear in rendered plans, stdout/stderr, Nix store references, Git diffs, or committed artifacts. A staged and repository scan must pass, plus explicit checks for identifiers not covered by token patterns.

**Official evidence:** [Nix warns that its store is readable and secrets should be read at runtime](https://releases.nixos.org/nix/nix-2.33.1/manual/store/secrets.html); [GitHub sensitive-data removal and rotation guidance](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository); [GitHub secret-scanning detection limitations](https://docs.github.com/en/code-security/reference/secret-security/secret-scanning-scope); [GitHub push protection](https://docs.github.com/en/code-security/concepts/secret-security/push-protection).

---

### PF-07: Treating `.gitignore` as a safe bootstrap allowlist

**Severity / confidence:** CRITICAL / HIGH

**What goes wrong:**
When Git is missing, the checkout is not a worktree, or the tracked query is empty, the current bootstrap scans physical `.config` directories. Ignored or untracked app state can then be offered for linking. A user can also provide a target username/path that reaches an unintended directory before a destructive replacement.

**Why it happens:**
The script tries to remain usable outside Git by failing open. Ignore rules protect Git staging, not filesystem enumeration or shell behavior. Interactive prompts do not validate candidate provenance or destination containment.

**Consequences:**
- Private/untracked application state becomes linked, copied, logged, or accidentally staged later.
- A copied/exported repository behaves less safely than the canonical checkout.
- A malformed target can cause deletion outside the intended home.
- “Tracked-only bootstrap” documentation becomes false.

**Prevention:**
- Fail closed if tracked discovery is unavailable or empty.
- Replace directory discovery with an explicit, versioned, privacy-reviewed deployment manifest.
- Resolve the target account through macOS account metadata, canonicalize its home, reject traversal/separators, and verify every destination remains under the resolved home.
- Never enumerate ignored physical directories as candidates.
- Include source Git status/provenance and target containment proof in the plan.

**Warning signs:**
- A fallback uses `find .config -type d` or equivalent.
- Candidate output includes an untracked/ignored directory.
- The plan accepts an arbitrary user-derived home path without account resolution.
- The bootstrap is advertised as safe from a source archive without a manifest.

**Phase to address:** Safety Foundation for fail-closed discovery and test fixtures; Recovery Engine for the explicit deployment manifest and target validation.

**Recovery steps:**
1. Stop bootstrap and inspect the created links without following them into private state.
2. Remove only links proven to have been created by the plan; never recursively delete their source directories.
3. Restore displaced targets from planner backups or the user's backup.
4. Re-run from a valid worktree with the explicit manifest.

**Verification:** Temporary fixtures cover missing Git, non-worktree, empty tracked result, ignored directories, traversal input, pre-existing symlink, interrupted apply, and repeated apply. Every unsafe discovery condition must produce a non-mutating error.

**Official evidence:** [Nix's Git flake behavior also demonstrates that Git-indexed and physical trees are distinct](https://nix.dev/manual/nix/2.34/command-ref/new-cli/nix.html); [Home Manager's collision checks before writes](https://github.com/nix-community/home-manager/blob/master/modules/home-environment.nix).

---

### PF-08: Designing a “private host overlay” that flakes cannot see—or that leaks identity into the store

**Severity / confidence:** HIGH / HIGH for Nix source behavior; MEDIUM for the final repository-specific design

**What goes wrong:**
The multi-host refactor imports an ignored local Nix file from inside the Git checkout. A local `.#host` build cannot see it because Git flakes include only Git-indexed files. A workaround then uses `path:`/impure environment access or a local flake input containing private values, silently weakening reproducibility or copying private material into the Nix store.

**Why it happens:**
“Ignored local overlay” is a common dotfile pattern, but pure Git-flake evaluation has a different source boundary. The project also needs real account/home facts at activation time while promising not to publish stable machine identifiers.

**Consequences:**
- Local builds work only with `path:.` or `--impure`, while remote/clean clones fail.
- A supposedly private host file is copied into the world-readable Nix store.
- Host selection differs between machines and CI with no auditable explanation.
- Architecture and Homebrew prefix assumptions remain hard-coded and drift on Intel/Rosetta or a differently structured Mac.

**Prevention:**
- Give Multi-host Composition its own design spike; do not bolt an ignored import onto the current flake.
- Keep the shared host schema and privacy-safe logical IDs tracked and pure.
- Define exactly which non-secret machine facts may enter evaluation, how the local selector is supplied, and whether that data is acceptable in the Nix store.
- Never pass secrets through a flake input or `builtins.getEnv`.
- Make architecture support explicit; derive Homebrew prefix from the supported platform rather than embedding Apple-Silicon-only paths in shared shell logic.
- Prove both a normal Git-flake build and a remote clean-clone build before claiming the overlay model works.

**Warning signs:**
- An ignored `.nix` file is imported from `flake.nix`.
- Routine commands require `--impure` or `path:.` to find host configuration.
- A generated host module contains a real username, hostname, serial, private endpoint, or secret.
- Shared modules contain unconditional `/opt/homebrew`, `aarch64-darwin`, or one user's home path.

**Phase to address:** Multi-host Composition, after ownership contracts are stable and before the Recovery Engine assumes host profiles.

**Recovery steps:**
1. Remove the impure/private input from the build graph.
2. Inspect resulting store paths and caches for private data; rotate credentials if any secret entered them.
3. Restore the last pure single-host configuration.
4. Rework the selector with synthetic host identities and prove Git/remote evaluation before migrating the real host.

**Verification:** Tests create two synthetic logical hosts and two architecture records with no personal identifiers. Both evaluate from a clean Git clone. A scanner rejects real host/user patterns and store references to secret fixtures.

**Official evidence:** [Nix documents that local Git flakes expose only Git-indexed files](https://nix.dev/manual/nix/2.34/command-ref/new-cli/nix.html); [Nix store secret warning](https://releases.nixos.org/nix/nix-2.33.1/manual/store/secrets.html); [Homebrew's architecture-specific prefixes](https://docs.brew.sh/Installation); [mise platform-specific config behavior](https://mise.jdx.dev/configuration/environments.html).

---

### PF-09: Applying a stale plan to changed machine state

**Severity / confidence:** HIGH / HIGH

**What goes wrong:**
`check` and `plan` are safe, but the operator applies the plan after the repo revision, lockfiles, host profile, package inventory, target files, or service state has changed. The confirmation no longer corresponds to the operations actually needed.

**Why it happens:**
The workflow names the five stages but does not cryptographically bind them. Interactive scripts recompute state while applying instead of applying an immutable, reviewed operation list with revalidated preconditions.

**Consequences:**
- A target that was absent during planning is overwritten during apply.
- A package/service operation is no longer idempotent.
- The rollback record describes the wrong before-state.
- A confirmation for a small change authorizes a larger recomputed change.

**Prevention:**
- Give every plan a versioned schema and ID derived from sanitized desired-state revision, relevant lockfile hashes, logical host profile, architecture, operation list, and non-secret before-state fingerprints.
- Make apply consume the reviewed plan, not silently regenerate it.
- Recheck all preconditions immediately before the write boundary and refuse on mismatch.
- Expire plans after any repository, target, inventory, or host-profile change.
- Write a local, ignored, redacted operation journal with per-component status for safe resume; never include secrets.

**Warning signs:**
- Apply accepts only `--yes` and has no plan ID.
- Plan output omits repo/lock/profile fingerprints.
- Apply discovers new operations after confirmation.
- A resumed run cannot distinguish completed from pending operations.

**Phase to address:** Safety Foundation defines the schema; Recovery Engine implements the write boundary and resume journal.

**Recovery steps:**
1. Stop at the first precondition mismatch or unexpected operation.
2. Use the journal to identify completed operations.
3. Verify/roll back completed components individually.
4. Discard the stale plan, rerun check, and obtain a new confirmation.

**Verification:** Fixture tests mutate each precondition between plan and apply and assert zero writes. Interruption tests resume only operations whose before/after fingerprints still match.

**Official evidence:** [Home Manager's pre-write `writeBoundary` and required dry-run behavior](https://github.com/nix-community/home-manager/blob/master/modules/home-environment.nix); [chezmoi's explicit dry-run guarantee that the destination is not modified](https://www.chezmoi.io/reference/command-line-flags/global/).

---

### PF-10: Calling a current-Mac drill a clean recovery test

**Severity / confidence:** HIGH / HIGH

**What goes wrong:**
The recovery workflow succeeds on the only existing Mac and is labeled “fresh install verified.” The machine already contains runtimes, Homebrew metadata, trust approvals, Nix store paths, caches, permissions, application data, and manual fixes that a clean Mac would not have.

**Why it happens:**
There is no disposable Mac today, and a green report is psychologically easier to present than a graded evidence model.

**Consequences:**
- Missing bootstrap dependencies and first-run prompts remain undiscovered.
- Untracked manual state masks defects.
- A later reinstall fails despite the repository claiming full recovery.
- Roadmap work stops before a clean-host test is possible.

**Prevention:**
- Use explicit evidence levels: static validation, non-activating build, isolated fixture, current-host read-only drill, current-host authorized apply, and clean-host E2E.
- Reserve “fresh-install verified” for a macOS VM or second physical Mac started from documented prerequisites.
- Until then, report “recovery-ready on current host,” plus unknown/unverified clean-host items.
- Keep a first-run checklist covering Nix/Homebrew bootstrap, Rosetta/architecture where applicable, TCC, logins, network, App Store, and secrets.

**Warning signs:**
- A report has one boolean `ready` and no evidence level.
- Tests rely on populated caches or previously trusted mise/direnv files.
- No first-run/manual prerequisites are listed.
- A current-host run is described as disaster-recovery proof.

**Phase to address:** Readiness Drill; final proof is deferred to Clean-host Validation.

**Recovery steps:**
1. Downgrade the claim and publish the actual evidence level.
2. Add every newly discovered prerequisite to the readiness schema.
3. Fix issues on the current host only through the normal plan/apply gate.
4. Schedule a clean VM/second-Mac validation when available.

**Verification:** The report generator cannot emit the clean-host status without an artifact identifying the clean environment, starting state, completed plan, verification results, and exclusions.

**Official evidence:** [Homebrew's installer has architecture-specific prerequisites and asks for confirmation](https://docs.brew.sh/Installation); [Apple documents privacy permissions as separate user-controlled state](https://support.apple.com/guide/mac-help/change-privacy-security-settings-on-mac-mchl211c911f/mac).

---

### PF-11: Disabling mise or direnv trust because prompts are inconvenient

**Severity / confidence:** HIGH / HIGH

**What goes wrong:**
The global config trusts `/`, a broad projects tree, or every `.envrc`, so cloned repositories can change the environment or execute shell code without meaningful review. Trust databases are then synchronized as if they were portable configuration.

**Why it happens:**
Non-interactive shells and recovery automation cannot answer prompts, and trust failures look like reproducibility bugs. Broad allowlists make automation green quickly.

**Consequences:**
- A malicious or compromised project config executes hooks/tasks or sources shell code.
- Secrets inherited from the shell are exposed to untrusted commands.
- One Mac's approval is replayed on another without reviewing the actual file/content.
- `source_env` can load another `.envrc` outside the primary security check.

**Prevention:**
- Keep mise's safe project files limited to plain tool versions where possible; review and explicitly trust configs that contain env directives, templates, hooks, or tool options.
- Never set mise trusted paths to `/` and never sync mise state/trust directories.
- Keep `.envrc` minimal, versioned, and manually allowed per content hash.
- For sourced env files, use direnv's `require_allowed` mechanism where supported and keep real secret values outside Git.
- Non-interactive validation should report “trust required,” not silently approve.

**Warning signs:**
- `MISE_TRUSTED_CONFIG_PATHS=/`, `trusted_config_paths = ["/"]`, or broad direnv whitelist prefixes.
- Automation runs `mise trust -a` or `direnv allow` on arbitrary projects.
- Trust/state directories appear in the sync manifest.
- `.envrc` downloads or executes remote code.

**Phase to address:** Safety Foundation sets trust policy; each ecosystem phase keeps project contracts non-executable where possible; Recovery Engine reports approvals as manual state.

**Recovery steps:**
1. Revoke broad trust (`mise untrust` / remove trusted path; `direnv deny`) without executing the project config.
2. Inspect the relevant file and any sourced files.
3. Rotate credentials if untrusted code may have received them.
4. Approve only the reviewed content on each machine.

**Verification:** Security fixtures prove broad trust settings are rejected and untrusted executable config yields a manual status. Trust records never appear in Git, plans, or cross-Mac state.

**Official evidence:** [mise trust behavior and executable-config boundary](https://mise.jdx.dev/cli/trust.html); [mise warns that trusting `/` disables the mechanism](https://mise.jdx.dev/configuration/settings.html); [direnv allow/deny security model](https://direnv.net/man/direnv.1.html); [direnv warns `source_env` targets are not checked by the security framework](https://direnv.net/man/direnv-stdlib.1.html).

---

### PF-12: Trying to restore TCC permissions, logins, or account state as ordinary dotfiles

**Severity / confidence:** HIGH / HIGH

**What goes wrong:**
Recovery scripts attempt to grant Accessibility, Full Disk Access, Automation, Input Monitoring, or other TCC permissions; copy login/session databases; or declare success without them. On an unmanaged personal Mac, many permissions require explicit user action. Apple's managed PPPC path requires device management and supervision, which is outside this project.

**Why it happens:**
The project aims for a complete workstation, so manual/security boundaries look like automation gaps rather than deliberate exclusions.

**Consequences:**
- Security controls are weakened or bypass attempts fail across macOS releases.
- Private application databases or login sessions enter Git/backups.
- Automation apps appear installed but do not function.
- The readiness result is misleading.

**Prevention:**
- Represent each permission/login as `manual-required`, `present`, `missing`, or `not-checkable` using only non-sensitive evidence.
- Link to the exact System Settings pane and explain why the permission is needed.
- Do not automate approval, copy TCC databases, or treat an MDM PPPC payload as applicable to an unmanaged Mac.
- Verify app behavior after the user grants permission; installation alone is insufficient.

**Warning signs:**
- Scripts modify the TCC database or copy `~/Library` login/session databases.
- A cask installation is treated as proof an automation app works.
- The plan contains Apple ID/App Store sign-in or account-session export.

**Phase to address:** Recovery Engine models manual gates; Readiness Drill verifies behavior and exclusions.

**Recovery steps:**
1. Remove any copied private databases from the plan/worktree and run privacy remediation if committed.
2. Reset only the affected permission through supported user/admin controls if necessary.
3. Have the user grant the minimum permission to the known app.
4. Re-run the app-specific behavior check.

**Verification:** The readiness report stays non-green for dependent behavior until an observable app check passes, but it never attempts to grant the permission itself.

**Official evidence:** [Apple Privacy & Security settings](https://support.apple.com/guide/mac-help/change-privacy-security-settings-on-mac-mchl211c911f/mac); [Apple's Accessibility authorization guidance](https://support.apple.com/en-gb/guide/mac-help/mh43185/mac); [Apple PPPC payload requires device management and supervision](https://support.apple.com/en-euro/guide/deployment/dep38df53c2a/web); [Apple platform-security explanation of user consent](https://support.apple.com/guide/security/controlling-app-access-to-files-secddd1d86a6/web).

## Moderate Pitfalls

### PF-13: Assuming one lockfile pins the whole workstation

**Severity / confidence:** MEDIUM-HIGH / HIGH

**What goes wrong:** A locked Nix input graph is presented as complete reproducibility while Homebrew casks/taps, `@latest` channels, unpinned bootstrap commands, unpinned Git clones, manager auto-downloads, and app self-updaters remain time-dependent.

**Root cause:** Different subsystems have different lock semantics. A Brewfile is an inventory, not an exact immutable snapshot of every installed app/version. mise lockfiles are separate and must be created/committed; Corepack availability changes with Node; some GUI apps update themselves.

**Consequences:** Two Macs restored on different dates can have different binaries, schemas, plugin compatibility, or first-run behavior even with the same Git revision.

**Prevention:** Maintain a provenance matrix per item: desired spec, resolver, lock/checksum source, tested version, update owner, and expected drift. Pin bootstrap revisions and vendored/external assets where feasible. Use exact project versions and lockfiles; explicitly label intentionally floating GUI channels.

**Warning signs:** “flake.lock exists” is the only reproducibility evidence; a moving `master`, `latest`, or default Git branch is executed; no mise lock/provenance decision exists; package versions are not included in readiness output.

**Phase to address:** Safety Foundation for provenance schema; every ecosystem phase for its locks; Recovery Engine for bootstrap pins.

**Recovery:** Revert to the last tested lock/provenance set, restore pinned bootstrap URLs/revisions, and revalidate affected apps/toolchains without updating unrelated dependencies.

**Verification:** A clean metadata-only audit lists every floating source and fails if an item marked “pinned” lacks a concrete revision/checksum/lock entry for the supported platform.

**Official evidence:** [Nix flake/lock model](https://nix.dev/manual/nix/2.26/command-ref/new-cli/nix3-flake.html); [mise lockfile behavior](https://mise.jdx.dev/dev-tools/mise-lock.html); [Homebrew Bundle semantics](https://docs.brew.sh/Brew-Bundle-and-Brewfile); [Corepack distribution boundary](https://github.com/nodejs/corepack); [Homebrew architecture and bottle constraints](https://docs.brew.sh/FAQ).

---

### PF-14: Using install/sync commands as “validation” against a real project

**Severity / confidence:** HIGH / HIGH

**What goes wrong:** A smoke test runs `mise use/install`, `uv sync`, `bun install`, Corepack, Go, or Gradle in a real project. The command writes config, lockfiles, virtual environments, caches, or downloads runtimes. Bun can automatically migrate a pnpm lockfile when `bun.lock` is absent.

**Root cause:** Ecosystem CLIs combine validation and convergence, and a successful install is mistaken for a read-only compatibility check.

**Consequences:** Real projects gain unexplained diffs, package-manager ownership changes, or new runtimes; tests pollute global state and can break active work.

**Prevention:** Separate parsers/inspectors from reconcilers. Test mutating commands only on a copied fixture with isolated state, disabled auto-download/network where possible, and a before/after allowlist. Never use a user's real project without separate authorization.

**Warning signs:** Validation produces `.venv`, `node_modules`, `bun.lock`, `mise.lock`, Gradle daemons/JDKs, or config edits; a command includes `use`, `install`, `sync`, `upgrade`, or `go env -w`.

**Phase to address:** Safety Foundation, then each ecosystem phase supplies explicit read-only and isolated-mutating test sets.

**Recovery:** Stop, preserve the real project's preexisting work, remove only artifacts proven to be created by the test, restore changed lock/config files from their exact pre-test content, and rerun in a fixture.

**Verification:** Test harness records a repository tree hash and global-state sentinels before/after. Only declared fixture paths may change.

**Official evidence:** [mise distinguishes `install` from config-writing `use`](https://mise.jdx.dev/faq.html); [uv creates project environments and can download Python](https://docs.astral.sh/uv/reference/storage/); [Bun's pnpm lock migration](https://bun.sh/docs/pm/cli/install); [Go automatic toolchain download](https://go.dev/doc/toolchain); [Gradle auto-provisioning](https://docs.gradle.org/current/userguide/toolchains.html); [Corepack downloads/caches the selected manager](https://github.com/nodejs/corepack/blob/main/README.md).

---

### PF-15: Equating Homebrew inventory with service and login-item state

**Severity / confidence:** MEDIUM-HIGH / HIGH

**What goes wrong:** A formula is installed and therefore reported ready even though its service is stopped, running with stale config, or registered differently. Conversely, activation starts a service that the user intentionally kept stopped.

**Root cause:** Package presence, service registration, process health, and application behavior are collapsed into one state. nix-darwin can emit service instructions through Brewfile entries, but service effects are external mutable state.

**Consequences:** Background processes start unexpectedly, ports conflict, battery/network use changes, or rollback leaves a service running.

**Prevention:** Treat package install and service intent as separate ownership records. `check` reports declared state, registration, runtime status, and health without starting/stopping. Apply service changes in their own confirmation group and record whether this plan changed the state.

**Warning signs:** “formula installed” is the only service check; adding `start_service` is bundled with unrelated packages; rollback has no service-state step.

**Phase to address:** Ownership Inspector inventories services; Recovery Engine manages explicit service operations; Readiness Drill checks health.

**Recovery:** Restore only service states changed by the plan, then verify port/process health and login registration; do not stop a service merely because it is undeclared.

**Verification:** Synthetic state tests cover installed/stopped, installed/running, stale config, and unmanaged-running states without invoking `brew services start/stop`.

**Official evidence:** [Homebrew service command behavior](https://docs.brew.sh/Manpage); [nix-darwin Homebrew formula/service options](https://nix-darwin.github.io/nix-darwin/manual/).

---

### PF-16: Warm caches, trust records, and daemons hide first-run failures

**Severity / confidence:** MEDIUM-HIGH / HIGH

**What goes wrong:** Checks pass because the current Mac has downloaded runtimes, trusted configs, populated Nix/Homebrew caches, Gradle daemons, Maven metadata, or previously approved TCC state. A clean Mac fails or chooses different artifacts.

**Root cause:** Tests reuse real state to save time, and cache hits are not reported as part of evidence.

**Consequences:** Offline/bootstrap gaps and missing checksums remain hidden; cold recovery is slower or fails; a cached vulnerable/floating artifact is mistaken for a lock guarantee.

**Prevention:** Add isolated empty-state tests that prohibit writes outside the fixture and clearly distinguish cache-hit from cold-resolution evidence. Do not clear real caches. Treat full clean-host execution as deferred until a disposable Mac exists.

**Warning signs:** Tests pass only after a manual trust/install; removing a fixture cache changes selected versions; no output states whether network/cache was used.

**Phase to address:** Safety Foundation and every ecosystem phase; final closure in Clean-host Validation.

**Recovery:** Correct the claim, identify the hidden prerequisite, add it to the readiness report, and reproduce in isolated empty state without clearing the real machine.

**Verification:** An isolated run starts with empty manager roots and disabled downloads; expected missing prerequisites are reported deterministically rather than auto-fixed.

**Official evidence:** [mise separates shareable config from machine-local cache/state/trust](https://mise.jdx.dev/directories.html); [uv automatic downloads](https://docs.astral.sh/uv/guides/install-python/); [Gradle daemon/toolchain cache behavior](https://docs.gradle.org/current/userguide/toolchains.html); [direnv trust storage](https://direnv.net/man/direnv.1.html).

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|---|---|---|---|
| Keep two live writers for `~/.zshrc` | Easy fallback | PATH/ownership drift and unsafe replacement | Never as simultaneous owners; fallback may exist only behind an explicit mode |
| Put all runtimes in mise | One apparent interface | Conflicts with uv/rustup/project-native semantics and system libraries | Never as a blanket rule; only where ownership matrix assigns mise |
| Put every app/package into Homebrew cleanup | Easy equality metric | Removes local exceptions and possibly user state | Never in this milestone |
| Use global fallback as project contract | Fewer project files | Projects silently depend on one Mac's global state | Only for interactive ad-hoc use, never for a reproducible project |
| Import an ignored host `.nix` file | Keeps identity out of Git | Git flake cannot see it; impure/store leak workarounds | Never without a researched, tested source-boundary design |
| Accept broad trust paths | Removes prompts | Turns project config into unreviewed code execution | Never |
| Test against the real HOME | Fast and realistic-looking | Mutates or depends on the work Mac | Never for automated tests |
| Use exact equality as readiness | Simple green/red output | Treats intentional drift and manual state as failure | Never; use categorized state |
| Leave moving bootstrap refs | Less update work | Recovery changes over time without repo diffs | Only for explicitly labeled exploratory commands, not recovery |

## Integration Gotchas

| Integration | Common mistake | Correct approach |
|---|---|---|
| Nix -> nix-darwin | Treat `flake check` as activation | Record evaluation/build evidence separately from switch and postconditions |
| nix-darwin -> Homebrew | Assume Nix rollback reverses Brew operations | Snapshot non-secret inventory and define Homebrew-specific rollback |
| nix-darwin -> Home Manager | Assume backup suffix makes every overwrite safe | Keep collision checks, explicit owner, and planner backups; inspect backup churn |
| Home Manager -> zsh | Activate mise only in one shell route | Test primary/fallback/login/non-interactive routes or explicitly de-support a route |
| mise -> project files | Ignore parent/local/env precedence | Inspect `mise cfg`/effective source and enforce ceiling/ownership |
| mise -> direnv | Trust every config to avoid prompts | Keep approvals content-scoped and machine-local |
| uv -> Python | Assume uv will use only the declared system Python | Record managed/system preference and disable unintended downloads |
| rustup -> Rust | Assume `rust-toolchain.toml` always wins | Inspect env and directory overrides plus executable/proxy path |
| Go -> `go.mod` | Assume the bundled `go` binary is the compiler used | Inspect `GOTOOLCHAIN` and selected toolchain; control auto-download policy |
| Node -> pnpm/Corepack | Assume Node always ships Corepack | Define package-manager provision separately, especially for Node 25+ |
| Bun -> pnpm project | Run `bun install` as a smoke test | Use a copied fixture; Bun may create `bun.lock` from pnpm state |
| Gradle -> JDK | Check only `JAVA_HOME` | Inspect daemon JVM and build toolchain; control auto-detection/download |
| Maven -> JDK | Commit machine paths in `toolchains.xml` | Keep requirements in project and resolve machine paths locally without secrets |
| macOS -> TCC | Treat app install as permission grant | Report manual permission and verify behavior after user approval |
| GitHub -> secrets | Rely only on token scanners | Add human privacy review, generic identifier checks, and rotation/history procedure |

## Performance Traps

| Trap | Symptoms | Prevention | When it breaks |
|---|---|---|---|
| Scan the whole home for drift | Slow runs, permission prompts, private filenames in logs | Scan only versioned manifests and allowlisted targets | Immediately on a mature user account |
| Invoke every manager on every shell prompt | Shell latency and surprise network/cache activity | Keep shell activation minimal; run deep inspector explicitly | With six ecosystems and parent config discovery |
| Download full runtime matrices for tests | Large disk/network use and long tests | Static parsing plus one isolated representative runtime per phase | As soon as multiple versions/architectures are included |
| Build every host for every leaf-doc change | Slow feedback and unnecessary store growth | Scope fast checks, then run all-host evaluation at integration gates | Once multi-host outputs are introduced |
| Persist verbose command logs | Large, privacy-sensitive artifacts | Structured allowlisted results, redaction before persistence, bounded retention | Immediately when paths/env are logged |

## Security Mistakes

| Mistake | Risk | Prevention |
|---|---|---|
| Put plaintext secrets in Nix values/files | Values enter readable store/caches | Runtime-only secret provider; no interpolation into derivations |
| Commit secret then delete/ignore it | Value remains in history/forks/clones | Rotate first, then coordinated history remediation |
| Disable mise/direnv trust | Arbitrary code from cloned projects | Per-content explicit approval; never sync trust state |
| Source private `.envrc` without secondary approval | Included code bypasses primary security check | Use `require_allowed`; minimize executable env code |
| Print full environment for diagnostics | Tokens/private endpoints leak to logs | Allowlist fields and redact before output |
| Use physical directory fallback | Ignored local state enters deployment candidates | Versioned allowlist, fail closed |
| Use `rm -rf` after a broad prompt | Irrecoverable config deletion/path escape | Canonical containment, backup rename, exact operation confirmation |
| Auto-grant TCC or copy login databases | Breaks macOS trust/privacy boundary | Manual status and user-approved settings only |
| Automatically `zap` casks | Removes preferences/shared resources | Never automate zap; explicit future data-impact review |

## UX Pitfalls

| Pitfall | User impact | Better approach |
|---|---|---|
| One giant “Apply all?” prompt | Cannot judge or selectively defer risk | Group operations by owner/risk with exact paths, versions, and rollback |
| Binary ready/not-ready result | Hides manual, excluded, drift, and unknown state | Typed statuses with evidence level and next action |
| Recompute after confirmation | User approves a different plan than executed | Immutable plan ID plus precondition recheck |
| Show raw host paths and command output | Leaks identity and overwhelms review | Logical host ID, `~` redaction, structured summaries with opt-in local detail |
| Treat manual permission as error | Pressures unsafe automation | `manual-required` status with supported System Settings instructions |
| Silently preserve extras | User cannot tell expected drift from missing declaration | Report `unmanaged-present` without deletion |
| Silently auto-install missing runtimes | Surprising downloads and ownership changes | Report missing during check; install only in confirmed apply |

## “Looks Done But Isn't” Checklist

- [ ] **Nix validation:** `nix flake check` passed—but verify the selected host build, reviewed apply, and postconditions separately.
- [ ] **nix-darwin switch:** switch exited successfully—but verify Homebrew, Home Manager, services, defaults, and the separate symlink plane.
- [ ] **Rollback:** a Nix generation is available—but verify component rollback for Homebrew, services, casks, and links.
- [ ] **Homebrew inventory:** declared packages are installed—but report extras, versions, tap health, and service state without cleanup.
- [ ] **Homebrew safety:** cleanup is disabled—but verify no wrapper/extra flag invokes uninstall, cleanup, or zap.
- [ ] **Node:** `node --version` matches—but verify executable provenance, project selection, npm, pnpm, and Corepack availability (especially Node 25+).
- [ ] **Go:** `go version` matches—but verify `GOTOOLCHAIN`, `go.mod`/`go.work`, and whether a newer toolchain can be downloaded.
- [ ] **Python:** uv is installed—but verify the selected interpreter, managed/system policy, project environment path, and automatic-download policy.
- [ ] **Rust:** rustup is installed—but verify rustup proxies win PATH and no environment/directory override selects another toolchain.
- [ ] **Deno/Bun:** versions match—but verify Homebrew, mise, and installer paths do not shadow one another and caches/install roots are intentional.
- [ ] **JVM:** `java -version` matches—but verify Gradle daemon/build toolchains and Maven toolchains do not select different JDKs.
- [ ] **Package locks:** a project has a lockfile—but verify validation did not create/migrate/update it and the correct package manager owns it.
- [ ] **mise config:** a version is declared—but verify merged parent/local/environment config and trust status.
- [ ] **direnv:** the hook is installed—but verify `.envrc` is reviewed/allowed and missing private values are reported without printing them.
- [ ] **Shell:** interactive zsh works—but verify login, non-interactive, fallback, hooks, and IDE contexts or explicitly mark unsupported contexts.
- [ ] **Symlinks:** targets point into the repo—but verify displaced files were backed up and target containment/ownership is recorded.
- [ ] **Multi-host:** logical aliases exist—but verify no real host/user identifiers leak and a clean Git-flake build can resolve the selected host.
- [ ] **Architecture:** Apple Silicon works—but do not claim Intel/Rosetta support until prefix/system paths are parameterized and tested.
- [ ] **Secrets:** secret files are ignored—but verify no values entered Nix store paths, logs, plans, Git history, or generated reports.
- [ ] **TCC:** the application is installed—but verify required permission and observable behavior after manual authorization.
- [ ] **Readiness report:** all automated checks pass—but preserve `manual`, `excluded`, `drift`, and `unknown` categories.
- [ ] **Recovery drill:** the current Mac is green—but label it recovery-ready, not clean-install verified.
- [ ] **Test isolation:** fixtures pass—but verify they did not use real caches, trust records, global config, network, or real project state.

## Recovery Strategies

| Pitfall | Recovery cost | Recovery steps |
|---|---|---|
| PF-01 non-atomic activation | HIGH | Stop; audit each plane; Nix rollback only Nix state; revert external components individually; regenerate plan |
| PF-02 destructive Homebrew cleanup | HIGH | Stop cleanup; reinstall confirmed items; restore user data externally; recheck services/permissions |
| PF-03 wrong effective toolchain | MEDIUM | Preserve all installations; inspect precedence; remove only unintended override; verify every context |
| PF-04 shell/link ownership race | HIGH | Stop writers; classify target; restore planner/user backup; select one owner; verify before apply |
| PF-05 isolation leak | MEDIUM-HIGH | Stop tests; diff sentinels/global state; restore exact changed config; fix harness before rerun |
| PF-06 privacy/secret leak | HIGH | Rotate first; stop artifact publication; remediate Git/store/log copies; add synthetic regression |
| PF-07 fail-open discovery | HIGH | Remove only proven links; restore displaced targets; replace discovery with manifest |
| PF-08 broken/private host overlay | HIGH | Revert to pure config; remove private input; audit store/caches; redesign with synthetic hosts |
| PF-09 stale plan | MEDIUM-HIGH | Stop; inspect journal; roll back completed operations as needed; discard and regenerate plan |
| PF-10 false recovery claim | LOW technically, HIGH trust | Correct status/evidence; add missing prerequisites; defer clean-host claim |
| PF-11 broad config trust | HIGH if code ran | Revoke trust; inspect configs; rotate exposed credentials; approve reviewed content only |
| PF-12 TCC/login overreach | MEDIUM-HIGH | Remove private state from artifacts; use supported user controls; verify app behavior |
| PF-13 incomplete locking | MEDIUM | Restore last tested locks/revisions; label floating items; revalidate only affected components |
| PF-14 mutating validation | MEDIUM-HIGH | Preserve user work; remove only test-created artifacts; restore exact files; rerun in copy |
| PF-15 service-state drift | MEDIUM | Restore only plan-changed services; verify registration, process, port, and health |
| PF-16 warm-state masking | LOW-MEDIUM | Correct evidence level; add prerequisite; reproduce in isolated empty roots |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention phase | Verification required before phase completion |
|---|---|---|
| PF-01 non-atomic activation | Safety Foundation + Recovery Engine | Component graph, independent outcomes/rollbacks, no single global success shortcut |
| PF-02 destructive cleanup | Safety Foundation | Policy rejects cleanup/uninstall/zap; extras fixture reports drift only |
| PF-03 effective-version mismatch | Ownership Inspector + all six ecosystem phases | Path/version/config-source/override evidence in each execution context |
| PF-04 shell/link race | Safety Foundation + Recovery Engine | Mutually exclusive shell owner; temp-home replacement/interruption/idempotency suite |
| PF-05 isolation leak | Safety Foundation | Real-state sentinels unchanged; no real-home paths or network/install side effects |
| PF-06 secrets/privacy leak | Safety Foundation + all phases | Synthetic secret canaries absent from Git, logs, plans, reports, and Nix references |
| PF-07 fail-open discovery | Safety Foundation + Recovery Engine | Missing Git/empty result fails closed; manifest-only candidates; traversal rejected |
| PF-08 host overlay/source boundary | Multi-host Composition | Two synthetic hosts evaluate from clean Git clone; no private facts/store leaks |
| PF-09 stale plan | Recovery Engine | Every plan/apply precondition mutation causes zero writes |
| PF-10 false clean-host claim | Readiness Drill | Evidence-level state machine prevents clean-host claim without disposable-host artifact |
| PF-11 trust bypass | Safety Foundation + ecosystem phases | Broad trust rejected; manual approval status; no trust state synchronized |
| PF-12 TCC/login boundary | Recovery Engine + Readiness Drill | No permission mutation; manual state plus observable app postcheck |
| PF-13 incomplete locks | Ecosystem phases + Recovery Engine | Provenance matrix lists every pinned and intentionally floating input |
| PF-14 mutating validation | Safety Foundation + ecosystem phases | Real project/worktree hashes unchanged; fixture writes are allowlisted |
| PF-15 service drift | Ownership Inspector + Recovery Engine | Package/service/health states separate; no service mutation during check |
| PF-16 warm-state masking | Safety Foundation + Readiness Drill | Empty-root/offline drill reports prerequisites without touching real caches |

## Recommended Roadmap Gates

1. **No ecosystem migration begins before Safety Foundation passes.** Otherwise the only work Mac is exposed to test and cleanup side effects.
2. **Ownership Inspector precedes removals.** Existing duplicates are evidence to classify, not cleanup targets.
3. **Each ecosystem is migrated independently.** The gate is effective execution in isolated fixtures and read-only current-state verification, with a documented rollback.
4. **Multi-host Composition follows stable ownership contracts.** It must solve the Git-flake/private-selector boundary without secrets or routine impurity.
5. **Recovery Engine consumes existing ownership/verification contracts.** It must not invent a second source of truth.
6. **Readiness Drill uses the current Mac non-destructively.** It may identify missing/manual state but cannot auto-fix it during check.
7. **Clean-host Validation remains a future evidence phase.** A VM/second Mac is the only acceptable place for destructive fresh-install tests.

## Sources

### Nix, nix-darwin, and Home Manager

- [Nix `flake check` reference](https://nix.dev/manual/nix/2.28/command-ref/new-cli/nix3-flake-check.html)
- [Nix Git-flake source/index behavior](https://nix.dev/manual/nix/2.34/command-ref/new-cli/nix.html)
- [Nix flake and lock model](https://nix.dev/manual/nix/2.26/command-ref/new-cli/nix3-flake.html)
- [Nix store secrets guidance](https://releases.nixos.org/nix/nix-2.33.1/manual/store/secrets.html)
- [nix-darwin configuration options](https://nix-darwin.github.io/nix-darwin/manual/)
- [Home Manager manual](https://nix-community.github.io/home-manager/introduction.html)
- [Home Manager activation implementation and contract](https://github.com/nix-community/home-manager/blob/master/modules/home-environment.nix)

### Homebrew

- [Homebrew Bundle and Brewfile](https://docs.brew.sh/Brew-Bundle-and-Brewfile)
- [Homebrew command reference](https://docs.brew.sh/Manpage)
- [Homebrew installation/prefixes](https://docs.brew.sh/Installation)
- [Homebrew FAQ and architecture prefixes](https://docs.brew.sh/FAQ)
- [Homebrew Cask uninstall/zap behavior](https://docs.brew.sh/Cask-Cookbook)

### Toolchain managers and ecosystems

- [mise configuration hierarchy](https://mise.jdx.dev/configuration.html)
- [mise directories and machine-local state](https://mise.jdx.dev/directories.html)
- [mise trust](https://mise.jdx.dev/cli/trust.html)
- [mise lockfiles](https://mise.jdx.dev/dev-tools/mise-lock.html)
- [mise CLI isolation flags](https://mise.jdx.dev/cli/)
- [uv Python management](https://docs.astral.sh/uv/concepts/python-versions/)
- [uv environment variables](https://docs.astral.sh/uv/reference/environment/)
- [uv storage](https://docs.astral.sh/uv/reference/storage/)
- [rustup installation and state roots](https://rust-lang.github.io/rustup/installation/)
- [rustup override precedence](https://rust-lang.github.io/rustup/overrides.html)
- [rustup environment variables](https://rust-lang.github.io/rustup/environment-variables.html)
- [Go toolchain selection](https://go.dev/doc/toolchain)
- [Node Corepack status](https://nodejs.org/download/release/v25.8.0/docs/api/corepack.html)
- [Corepack project contract](https://github.com/nodejs/corepack/blob/main/README.md)
- [Deno environment variables and state roots](https://docs.deno.com/runtime/reference/env_variables/)
- [Deno permissions](https://docs.deno.com/runtime/reference/permissions/)
- [Bun installation path](https://bun.sh/docs/installation)
- [Bun global cache](https://bun.sh/docs/pm/global-cache)
- [Bun install and lockfile migration](https://bun.sh/docs/pm/cli/install)
- [Gradle JVM toolchains](https://docs.gradle.org/current/userguide/toolchains.html)
- [Maven toolchains](https://maven.apache.org/guides/mini/guide-using-toolchains)
- [direnv security and shell hooks](https://direnv.net/man/direnv.1.html)
- [direnv stdlib security notes](https://direnv.net/man/direnv-stdlib.1.html)

### macOS, secrets, and recovery design references

- [Apple Privacy & Security settings](https://support.apple.com/guide/mac-help/change-privacy-security-settings-on-mac-mchl211c911f/mac)
- [Apple accessibility authorization](https://support.apple.com/en-gb/guide/mac-help/mh43185/mac)
- [Apple PPPC device-management payload](https://support.apple.com/en-euro/guide/deployment/dep38df53c2a/web)
- [Apple platform security: user-controlled file access](https://support.apple.com/guide/security/controlling-app-access-to-files-secddd1d86a6/web)
- [GitHub sensitive-data removal](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- [GitHub secret-scanning scope](https://docs.github.com/en/code-security/reference/secret-security/secret-scanning-scope)
- [GitHub push protection](https://docs.github.com/en/code-security/concepts/secret-security/push-protection)
- [chezmoi dry-run semantics, used only as a design precedent](https://www.chezmoi.io/reference/command-line-flags/global/)
- [chezmoi machine-specific configuration model, used only as a comparison point](https://www.chezmoi.io/user-guide/manage-machine-to-machine-differences/)

### Repository evidence reviewed

- `.planning/PROJECT.md`
- `.planning/codebase/ARCHITECTURE.md`
- `.planning/codebase/CONCERNS.md`
- `.planning/codebase/TESTING.md`
- `setup_mac.sh`
- `nix/darwin/homebrew.nix`
- `flake.nix`
- `nix/home/dev-toolchains.nix`
- `nix/modules/zsh.nix`
- `zsh/shared.zsh`
- `zsh/.zshrc`
- `.gitignore`

---
*Pitfalls research for: Yet Another Mac Config*
*Researched: 2026-07-10*
