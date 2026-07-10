# Codebase Concerns

**Analysis Date:** 2026-07-10

## Tech Debt

**Two competing zsh activation paths:**
- Issue: Home Manager owns the active `~/.zshrc`, while the legacy setup script can still replace it with a repository symlink. The two paths share `zsh/shared.zsh` but do not have identical behavior: the Home Manager path activates mise after loading the local fragment, while the fallback path does not activate mise and loads its plugins/local fragment in a different order.
- Files: `setup_mac.sh`, `zsh/.zshrc`, `zsh/shared.zsh`, `nix/modules/zsh.nix`, `nix/home/default.nix`
- Impact: Re-running bootstrap on a Nix-managed machine can silently change which component owns shell startup. PATH order, runtime selection, completions, and rollback behavior then depend on how the machine was initialized rather than on one declared state.
- Fix approach: Make the Nix/Home Manager and non-Nix modes explicitly mutually exclusive. Have `setup_mac.sh` detect a Home Manager-managed target and refuse to replace it unless an explicit migration flag is supplied; keep common initialization in one generated/shared unit and add a smoke test for both modes.

**Partially declarative Homebrew state:**
- Issue: The Nix configuration declares a large inventory but intentionally disables update, upgrade, and cleanup, while several services and account-heavy applications remain manually managed.
- Files: `nix/darwin/homebrew.nix`, `nix/README.md`, `README.md`
- Impact: Two machines can evaluate the same flake yet retain different Homebrew versions, extra packages, service states, and application state. A successful Nix evaluation therefore does not prove equivalent machines.
- Fix approach: Preserve the conservative activation policy, but add a read-only drift report that compares declared formulae/casks/services with live Homebrew state. Document intentional exceptions in one machine inventory instead of duplicating them across prose and comments.

**Duplicated Yazi plugin inventory:**
- Issue: The same seven plugins are represented in the shell script array, the package manifest, and vendored plugin directories. The shell array is used only for messaging; installation actually follows the manifest.
- Files: `install_yazi_plugins.sh`, `.config/yazi/package.toml`, `.config/yazi/plugins/`
- Impact: Adding or removing a plugin in only one place produces misleading counts, stale vendored code, or an install result that differs from the documented dependency list.
- Fix approach: Treat `.config/yazi/package.toml` as the only inventory. Derive display/check output from `ya pkg list` or the manifest and add a check that every declared plugin has the expected installed directory and revision.

**Disabled Neovim example module retained as production configuration:**
- Issue: A 197-line starter example is imported by the plugin directory but immediately returns an empty specification.
- Files: `.config/nvim/lua/plugins/example.lua`, `.config/nvim/lua/config/lazy.lua`
- Impact: The dead examples obscure the actual plugin surface, contain APIs that can age independently of the active configuration, and make reviews/search results noisier.
- Fix approach: Delete the disabled module or move the examples into documentation outside the auto-imported plugin directory.

**Phase history mixed with current operating guidance:**
- Issue: Current-state documentation is interleaved with historical phase-by-phase instructions, and the flake description still labels the repository as a Phase 1 skeleton although later phases are active.
- Files: `flake.nix`, `nix/README.md`, `nix/CLAUDE.md`, `nix/language-stack-plan.md`, `nix/phase-3-plan.md`
- Impact: Maintainers must reconstruct which statements describe the live configuration and which describe an earlier migration checkpoint before changing packages or activation order.
- Fix approach: Keep a short current-state document generated or checked against the live modules, and move completed phase narratives to an archive/changelog.

## Known Bugs

**Tracked-only setup guarantee depends on three discovery conditions:**
- Symptoms: Documentation says local ignored configuration directories are never offered for linking, but the fallback branch enumerates every first-level directory under `.config` unless Git is available, the repository is a Git worktree, and the normalized tracked-file query yields at least one top-level app directory. An empty tracked result in a valid worktree also triggers fallback.
- Files: `setup_mac.sh`, `README.md`, `.gitignore`
- Trigger: Run `setup_mac.sh` from a copied/exported tree without usable `.git` metadata, in an environment where `git` is unavailable, or in a valid worktree where the normalized tracked query yields no top-level `.config` app directory.
- Workaround: Run the script only with Git available in a real worktree, first confirm that `git ls-files -- .config` normalizes to at least one tracked top-level app directory, and do not use fallback discovery until ignored/local-state directories have been removed or otherwise made safe.

**Missing `--config-dir` value produces an internal shell error:**
- Symptoms: Passing `--config-dir` without a following path exits with `install_yazi_plugins.sh: line 24: $2: unbound variable` instead of usage text.
- Files: `install_yazi_plugins.sh`
- Trigger: Run `bash install_yazi_plugins.sh --config-dir`; the behavior is reproducible with the current script.
- Workaround: Always provide a path after `--config-dir`.

## Security Considerations

**Default alias bypasses Claude Code permission checks:**
- Risk: The one-character `c` alias always launches Claude Code with its permission-bypass flag. A typo, copied instruction, malicious repository prompt, or compromised tool can therefore perform filesystem and command actions without the normal interactive authorization boundary.
- Files: `zsh/shared.zsh`, `zsh/.zshrc`, `nix/modules/zsh.nix`
- Current mitigation: The alias runs with the current macOS user rather than root, but both supported shell paths load it by default and there is no project allowlist or confirmation wrapper.
- Recommendations: Make the normal alias invoke the standard permission model. Put the bypass form behind a deliberately named, per-session function that requires an explicit confirmation and is unavailable in untrusted directories.

**Physical-directory fallback can expose local application state to linking:**
- Risk: Unless Git is available, the repository is a Git worktree, and the normalized tracked-file query yields at least one top-level app directory, `setup_mac.sh` falls back to enumerating every first-level directory under `.config`. A local application-state directory that is ignored or simply not tracked can therefore be offered for linking even though the tracked-only path excludes it.
- Files: `setup_mac.sh`, `.gitignore`, `README.md`
- Current mitigation: When all three discovery conditions hold, the tracked-only path derives names from `git ls-files`, and every link or replacement still requires default-no confirmation.
- Recommendations: Fail closed whenever any tracked-discovery condition fails, including an empty normalized query result, or maintain a repository-owned allowlist for the fallback path. Align `README.md` with whichever fallback behavior is retained.

**Destructive link replacement is derived from unvalidated username input:**
- Risk: The setup script constructs `/Users/$username` directly from interactive input and later uses `rm -rf` on existing targets. Values containing path traversal components can resolve outside the intended home, and an ordinary valid username can still cause irreversible deletion of existing configuration directories after a broad confirmation.
- Files: `setup_mac.sh`
- Current mitigation: The constructed user directory must exist, and each replacement requires a default-no interactive answer.
- Recommendations: Resolve the target home through the system account database, reject separators and traversal components, verify the canonical path remains below `/Users`, and rename existing targets to timestamped backups instead of deleting them.

**Remote bootstrap code is executed without a repository-controlled revision:**
- Risk: The documented Nix installer is piped directly to a shell, first-run nix-darwin commands target a moving `master`, and oh-my-tmux is cloned from its default branch without a commit pin. A compromised or incompatible upstream revision is executed before the repository's locked state can protect the machine.
- Files: `nix/README.md`, `flake.nix`, `setup_mac.sh`
- Current mitigation: Downloads use HTTPS and originate from documented upstream projects; Nix dependencies used after bootstrap are recorded in `flake.lock`.
- Recommendations: Download and inspect/verify installers before execution, reuse the nix-darwin revision in `flake.lock` for bootstrap, and pin the oh-my-tmux checkout to a reviewed commit with an explicit update procedure.

**Host identity is embedded in the tracked flake:**
- Risk: A stable local username and device hostname are part of the committed declarative entry point, creating avoidable machine fingerprinting if the repository is public.
- Files: `flake.nix`, `nix/darwin/default.nix`, `nix/home/default.nix`
- Current mitigation: No credentials or tokens are stored in these modules.
- Recommendations: Use a generic host module plus an ignored/local host selection layer, or consciously document that these identifiers are public and acceptable.

## Performance Bottlenecks

No tracked benchmark or profile demonstrates a current performance bottleneck. The entries below are potential risks and measurement candidates inferred from configuration shape, not measured regressions.

**Potential shell-startup measurement candidate:**
- Observation: Each interactive Home Manager shell runs Homebrew shell environment generation, Starship initialization, and mise activation; the fallback path additionally performs completion initialization and probes multiple plugin paths.
- Files: `zsh/shared.zsh`, `zsh/.zshrc`, `nix/modules/zsh.nix`
- Potential risk: These commands run dynamically on shell startup and may contribute latency, but the repository contains no timing data that establishes their cost or identifies any one initializer as a bottleneck.
- Measurement path: Benchmark both shell paths before changing behavior. If results justify it, evaluate Home Manager's Starship/mise integration, cached stable init output, or a cached completion dump while retaining existence guards for rollback environments.

**Potential Ghostty resource measurement candidate:**
- Observation: The shared Ghostty config enables an animated custom shader and sets a 100,000-line scrollback limit for windows using that profile.
- Files: `.config/ghostty/config`, `.config/ghostty/shaders/cursor_blaze.glsl`, `README.md`
- Potential risk: Animation and a large scrollback budget are resource-sensitive settings, but no tracked GPU, battery, or RAM measurement establishes a material impact on this machine.
- Measurement path: Compare the profile with and without animation and with a lower scrollback limit. Add a low-power profile or adjust the defaults only if measurements or an explicit usage requirement justify the change.

## Fragile Areas

**Global Hammerspoon keyboard interception:**
- Files: `.hammerspoon/init.lua`, `README.md`
- Why fragile: Two global event taps depend on Accessibility permission, mutable timer/state flags, exact modifier event behavior, and stop/restart sequencing while synthetic keystrokes are injected. A tap disabled by macOS or an unexpected event sequence can affect Cmd+W, Cmd+Q, or right Cmd across every application.
- Safe modification: Change one gesture at a time, reload through the documented CLI, test key-down/key-up/modifier transitions in multiple applications, and keep a quick disable/reload path available.
- Test coverage: No automated event-sequence tests or runtime health monitor is present; validation is manual and depends on a signed-in GUI session.

**Title- and bundle-ID-based window routing:**
- Files: `.config/aerospace/aerospace.toml`, `.config/aerospace/CLAUDE.md`
- Why fragile: Floating rules depend on application bundle identifiers, localized titles, negative regular expressions, and startup timing. Application updates, renamed windows, or locale changes can silently route windows differently.
- Safe modification: Prefer stable bundle identifiers, keep title regexes narrowly scoped, validate the full config with the installed AeroSpace version, and test both startup and post-startup window creation.
- Test coverage: No fixture set exercises representative application IDs/titles or multi-monitor workspace assignment.

**Mixed Nix, Home Manager, and Homebrew activation transaction:**
- Files: `flake.nix`, `nix/darwin/default.nix`, `nix/darwin/homebrew.nix`, `nix/home/default.nix`, `nix/README.md`
- Why fragile: One switch combines Nix evaluation, Home Manager file ownership/backups, Homebrew package installation, and service starts. Failure in a mutable Homebrew tap or an existing-file conflict can leave some external state changed even when the overall command fails.
- Safe modification: Run evaluation and build first, inspect the activation diff, snapshot service/package state, then switch. Keep package/service additions atomic and document a component-specific rollback rather than assuming a Nix generation rollback reverses Homebrew side effects.
- Test coverage: The repository records successful manual switches but has no automated disposable-host activation test.

**Vendored application extensions:**
- Files: `.config/yazi/plugins/`, `.config/yazi/package.toml`, `.config/ghostty/shaders/`, `README.md`
- Why fragile: Third-party Lua plugins and GLSL shaders execute inside host applications and can break when Yazi/Ghostty APIs change. Yazi records revisions and hashes, and `.config/ghostty/shaders/README.md` records the imported shader source commit, but pinned provenance alone does not guarantee compatibility with future host versions.
- Safe modification: Update one upstream component at a time, preserve source revision/provenance, run host-specific validation, and review executable diffs rather than replacing vendor trees blindly.
- Test coverage: No automated compatibility test loads all vendored plugins or compiles every shader against the installed host version.

## Scaling Limits

**Single host, user, and architecture:**
- Current capacity: One `aarch64-darwin` configuration with one hard-coded username and one hard-coded hostname.
- Files: `flake.nix`, `nix/darwin/default.nix`, `nix/home/default.nix`
- Limit: A second user, Intel Mac, differently named machine, or host-specific package set requires editing the shared entry point and can expose machine-specific changes to every consumer.
- Scaling path: Define host records in a small attrset, parameterize system/user/home, split common and per-host modules, and keep sensitive/local host facts out of the shared module.

**Interactive-only bootstrap:**
- Current capacity: One operator answers a prompt for every tracked top-level configuration plus separate Codex, zsh, Hammerspoon, overwrite, and oh-my-tmux choices.
- Files: `setup_mac.sh`, `README.md`
- Limit: The script has no dry-run, non-interactive profile, machine-readable plan, or resumable state, so fleet setup and repeatable disaster recovery do not scale beyond manual use.
- Scaling path: Add a safe dry-run and an explicit declarative profile file, make operations idempotent, emit a summary before changes, and require a separate apply flag for replacements.

**Recovery intentionally restores only part of the workstation:**
- Current capacity: The Nix documentation targets roughly 70% to 85% recovery and leaves secrets, login state, several services, and application state manual.
- Files: `nix/README.md`, `README.md`, `nix/darwin/homebrew.nix`
- Limit: A fresh machine cannot reach a fully verified ready state from this repository alone, and the remaining manual steps are not represented as checkable completion criteria.
- Scaling path: Keep sensitive data external, but add a non-secret post-install checklist/validator that reports missing permissions, services, local files, logins, and optional components without trying to store them in Git.

## Dependencies at Risk

**Home Manager and nixpkgs compatibility can drift:**
- Risk: `flake.nix` follows moving `master` / `nixpkgs-unstable` inputs, while `flake.lock` pins whichever revisions were last resolved. Updating only part of that compatibility set can select Home Manager and nixpkgs revisions with different release expectations.
- Files: `flake.nix`, `flake.lock`
- Impact: A module option or activation behavior can fail after an input refresh even if unrelated configuration code is unchanged.
- Migration plan: Refresh and test the related inputs as one compatibility set, or pin matched release branches/revisions; keep the release check enabled so drift remains visible.

**mise package tests disabled globally for the overridden derivation:**
- Risk: The flake sets `doCheck = false` for mise to bypass one Darwin failure, so all upstream package tests—not only the known failing case—are skipped for that build.
- Files: `flake.nix`, `nix/home/dev-toolchains.nix`, `.config/mise/config.toml`
- Impact: Regressions in the runtime manager can reach the activated shell and affect Node/Go resolution without a build-time test signal.
- Migration plan: Narrow the override to the specific failing check when possible, track an upstream fix, remove the override promptly, and automate the documented runtime post-checks until then.

**Moving bootstrap and application channels:**
- Risk: nix-darwin bootstrap uses `master`, Homebrew includes an `@latest` AI CLI, Neovim defaults to latest plugin commits between lock updates, and oh-my-tmux is cloned without a revision.
- Files: `nix/README.md`, `flake.nix`, `nix/darwin/homebrew.nix`, `.config/nvim/lua/config/lazy.lua`, `.config/nvim/lazy-lock.json`, `setup_mac.sh`
- Impact: Reinstalling or refreshing on different dates can introduce incompatible CLIs/config schemas before repository changes are reviewed.
- Migration plan: Pin bootstrap revisions, review lockfile changes as dependency upgrades, record tested host versions, and provide explicit update commands instead of implicit moving channels.

**Third-party Homebrew taps and mutable service state:**
- Risk: Several formulae/casks come from third-party taps, while switch can start two services and does not update or clean the tap/package state first.
- Files: `nix/darwin/homebrew.nix`, `nix/README.md`
- Impact: Tap ownership, formula renames, trust policy changes, or stale local metadata can block the entire activation path; Nix rollback does not necessarily undo Homebrew service/package changes.
- Migration plan: Minimize taps, audit ownership and formula source during upgrades, add a preflight `brew bundle check`/tap health step, and keep service changes isolated from unrelated Nix changes.

## Missing Critical Features

**No automated validation gate:**
- Problem: The repository has configuration for secret scanning and written requirements for manual checks, but no committed CI workflow, pre-commit hook, or single verification command covering shell syntax, Nix evaluation, application config validation, and leak scanning.
- Files: `.gitleaks.toml`, `AGENTS.md`, `nix/README.md`, `setup_mac.sh`, `install_yazi_plugins.sh`
- Blocks: Changes can be committed after only ad hoc local checks; contributors cannot reproduce one authoritative pass/fail gate before activation.

**No safe plan/backup mode for destructive setup:**
- Problem: Bootstrap can delete existing config targets after prompts but cannot preview a complete plan, produce backups, or resume after partial completion.
- Files: `setup_mac.sh`, `README.md`
- Blocks: The setup path cannot be used confidently in unattended recovery, and an incorrect answer can only be recovered from an external backup.

**No machine readiness report:**
- Problem: Required manual permissions, ignored local fragments, logins, services, and optional runtime state are documented but not checkable in one read-only command.
- Files: `README.md`, `nix/README.md`, `.hammerspoon/init.lua`, `nix/darwin/homebrew.nix`
- Blocks: A successful setup/switch cannot distinguish “configuration evaluated” from “workstation is operational,” particularly for Accessibility permission, local shell fragments, and manually managed services.

## Test Coverage Gaps

**Bootstrap scripts:**
- What's not tested: Argument boundary cases, physical-directory fallback selection for each failed discovery condition, username/path validation, overwrite/backup behavior, idempotency, interrupted runs, symlink ownership conflicts, and even a committed automated shell-syntax check. The reproducible missing-value case already exposes an untested failure path that syntax checking alone would not catch.
- Files: `setup_mac.sh`, `install_yazi_plugins.sh`
- Risk: A script that passes syntax checking can still link ignored state, delete the wrong target, or fail midway after changing earlier targets.
- Priority: High

**Nix compatibility and activation:**
- What's not tested: A repeatable CI build against the locked inputs, Home Manager/nixpkgs compatibility, Homebrew preflight behavior, service side effects, and rollback from partial activation.
- Files: `flake.nix`, `flake.lock`, `nix/darwin/homebrew.nix`, `nix/home/default.nix`
- Risk: Input refreshes and mutable Homebrew state can break a real switch after evaluation-only checks pass.
- Priority: High

**Permission-sensitive desktop automation:**
- What's not tested: Hammerspoon event sequences, timer expiry, synthetic keystroke re-entry, right-modifier behavior, missing Accessibility permission, Ghostty launch failure, and AeroSpace routing across locales/monitors.
- Files: `.hammerspoon/init.lua`, `.config/aerospace/aerospace.toml`, `README.md`
- Risk: Regressions affect global keyboard/window behavior and may only appear in particular applications or after macOS/app updates.
- Priority: High

**Shell startup equivalence:**
- What's not tested: The Home Manager-generated shell and fallback symlink shell are not compared for PATH order, mise-selected runtimes, completion availability, aliases, or local-fragment ordering.
- Files: `zsh/.zshrc`, `zsh/shared.zsh`, `nix/modules/zsh.nix`, `nix/home/shell-env.nix`
- Risk: One route can work on the current machine while a fresh non-Nix machine or rollback route silently selects different tools.
- Priority: High

**Application and vendored-extension compatibility:**
- What's not tested: All Yazi plugins, Neovim lockfile/plugin bootstrap, every Ghostty shader, and schema/config changes across installed app versions. The repository documents native validators and smoke checks, but they are not encoded as repeatable repository tests.
- Files: `.config/yazi/package.toml`, `.config/yazi/plugins/`, `.config/nvim/lazy-lock.json`, `.config/nvim/lua/config/lazy.lua`, `.config/ghostty/config`, `.config/ghostty/shaders/`
- Risk: Dependency updates can leave syntactically valid files that fail only when a plugin command, shader, or first-run bootstrap path is exercised.
- Priority: Medium

**Privacy and secret-leak prevention:**
- What's not tested: Ignored/untracked application state exposed by physical-directory fallback, newly added configuration directories, personal identifiers, and non-token private data are not checked by an automatic staging/CI gate.
- Files: `setup_mac.sh`, `.gitignore`, `.gitleaks.toml`, `AGENTS.md`
- Risk: A broad staging operation can capture unignored local data that does not match a Gitleaks token rule, while physical-directory fallback can enumerate local directories that the tracked-only path would exclude.
- Priority: High

---

*Concerns audit: 2026-07-10*
