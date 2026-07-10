# Testing Patterns

**Analysis Date:** 2026-07-10

## Test Framework

**Runner:**
- No unit-test runner or repository-wide test harness is configured. Validation is command-driven through Nix, application-native CLIs, and documented manual post-checks in `nix/CLAUDE.md`, `.config/nvim/CLAUDE.md`, `.config/ghostty/CLAUDE.md`, and `.hammerspoon/CLAUDE.md`.
- Nix CLI / nix-darwin provide the broadest evaluation and integration path through `flake.nix`; the Nix input graph is locked by `flake.lock`, while the local Nix executable version is not pinned in a test-runner config.
- Config: `flake.nix` and `flake.lock` for Nix evaluation/build; `.config/nvim/stylua.toml` for Neovim Lua formatting; `.gitleaks.toml` for optional/manual secret scans.

**Assertion Library:**
- Not detected. Documented checks use process exit status plus targeted post-check instructions in `nix/language-stack-plan.md`, `nix/phase-3-plan.md`, `.config/nvim/CLAUDE.md`, and `.hammerspoon/CLAUDE.md`.

**Run Commands:**
- There is no single repository-level validation sequence. Root `CLAUDE.md` requires documentation synchronization and a manual pre-commit diff review for privacy leaks; Nix and application-specific commands live in their owning guidance.
- The current configured flake target is `.#AresdeMacBook-Air`. `nix/CLAUDE.md` documents these reusable evaluation and non-activating build commands:

```bash
nix flake check
darwin-rebuild build --flake .#AresdeMacBook-Air
```

- For the language-stack workflow specifically, `nix/language-stack-plan.md` adds the following example checks. `.gitleaks.toml` supplies the scan policy, but neither the file nor this phase plan makes Gitleaks mandatory for every repository change:

```bash
git diff --check
/opt/homebrew/bin/gitleaks detect --no-git --redact --source .
```

- There is no watch mode or coverage command in `nix/CLAUDE.md`, `.config/nvim/CLAUDE.md`, or the root scripts `setup_mac.sh` and `install_yazi_plugins.sh`.
- Activation is intentionally separate from validation and requires review plus elevated privileges, as documented in `nix/CLAUDE.md`:

```bash
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

## Test File Organization

**Location:**
- No tracked `tests/`, `__tests__/`, `*.test.*`, or `*.spec.*` suite exists. Neovim runtime scratch tests are explicitly ignored as `.tests` in `.config/nvim/.gitignore`, not committed as a repository test suite.
- Validation instructions are colocated with the affected configuration: `nix/CLAUDE.md`, `.config/nvim/CLAUDE.md`, `.config/ghostty/CLAUDE.md`, `.config/aerospace/CLAUDE.md`, `.hammerspoon/CLAUDE.md`, and `.config/tmux/CLAUDE.md`.
- Nix migration and language-stack gates plus post-switch commands live in planning/operations documents such as `nix/language-stack-plan.md` and completed acceptance checklists in `nix/phase-3-plan.md`.
- User-facing smoke-test and activation instructions live in `README.md`, `.config/nvim/README.md`, and `nix/README.md`.

**Naming:**
- No test-file naming convention is established because no formal test files exist; use the existing `## Verification`, validation command, and post-check sections in `.config/nvim/CLAUDE.md`, `.config/ghostty/CLAUDE.md`, and `nix/language-stack-plan.md` when documenting a new check.
- Name durable validation documentation for its owned subsystem rather than creating ad hoc root notes, following `nix/CLAUDE.md`, `.config/nvim/CLAUDE.md`, and `.hammerspoon/CLAUDE.md`.

**Structure:**
- The effective verification layout is documentation-driven and mirrors the configuration ownership documented in `CLAUDE.md`:

```text
README.md                         # user-facing setup and smoke checks
nix/CLAUDE.md                    # Nix check -> build -> switch contract
nix/language-stack-plan.md       # language-stack gates and post-checks
nix/phase-3-plan.md              # system-phase acceptance checklists
.config/nvim/CLAUDE.md           # Neovim-native validation
.config/ghostty/CLAUDE.md        # Ghostty-native validation
.hammerspoon/CLAUDE.md           # IPC reload and live-state checks
```

## Test Structure

**Suite Organization:**
- Select checks by change scope. Nix changes follow evaluation, a non-activating configured-host build, human review, authorized switch, and targeted post-checks. The language-stack plan additionally demonstrates diff hygiene and a redacted Gitleaks scan for that workflow.

```text
Nix: evaluation -> configured host build -> review -> authorized switch -> targeted post-check
Application config: native validator/reload -> behavior smoke check -> manual diff/privacy review
```

- Representative language-stack post-check commands from `nix/language-stack-plan.md` are:

```bash
command -v mise uv rustup direnv
mise --version
uv --version
rustup --version
direnv version
```

**Patterns:**
- Setup pattern: run Nix commands from the repository root and explicitly use the configured host target documented above, matching `nix/CLAUDE.md` and `nix/README.md`.
- Isolation pattern: use `darwin-rebuild build` without `sudo` before any mutating `switch`; `nix/CLAUDE.md` explicitly separates store-only construction from system activation.
- Teardown/rollback pattern: use `sudo darwin-rebuild switch --rollback` for activated Nix generations, as documented in `nix/CLAUDE.md`, `nix/README.md`, and `nix/phase-3-plan.md`.
- Artifact pattern: allow Nix build result links to remain untracked through `/result` and `/result-*` entries in `.gitignore`.
- Assertion pattern: combine successful exit codes with concrete postconditions such as executable resolution/version, generated Homebrew inventory, app startup, or UI behavior, as recorded in `nix/language-stack-plan.md` and `nix/phase-3-plan.md`.
- Change-scope pattern: run the native validator/reload for the application being changed in addition to cross-cutting diff and privacy checks, following `.config/ghostty/CLAUDE.md`, `.config/nvim/CLAUDE.md`, `.config/aerospace/CLAUDE.md`, `.hammerspoon/CLAUDE.md`, and `.config/tmux/CLAUDE.md`.

### Validation Matrix

| Scope | Validation documented for that scope | Evidence |
|---|---|---|
| Nix evaluation | `nix flake check` | `nix/CLAUDE.md`, `nix/README.md` |
| Nix host integration | `darwin-rebuild build` against the configured host target | `nix/CLAUDE.md`, `nix/phase-3-plan.md` |
| Nix activation/E2E | reviewed, authorized `darwin-rebuild switch` against the configured host target, then targeted post-check | `nix/CLAUDE.md`, `nix/language-stack-plan.md` |
| Diff hygiene | `git diff --check` | `.config/nvim/CLAUDE.md`, `.config/ghostty/CLAUDE.md`, `nix/language-stack-plan.md` |
| Pre-commit privacy review | manually inspect the diff for credentials and personal data | `CLAUDE.md` |
| Optional/scoped secret scan | use `.gitleaks.toml`; the redacted command is demonstrated by the language-stack plan | `.gitleaks.toml`, `nix/language-stack-plan.md` |
| Ghostty | `ghostty +validate-config` against the live config, then restart/smoke test shader changes | `.config/ghostty/CLAUDE.md`, `.config/ghostty/shaders/README.md` |
| Neovim | headless plugin sync, Lazy/Treesitter health checks, and clean headless startup | `.config/nvim/CLAUDE.md`, `.config/nvim/README.md` |
| AeroSpace | inspect app/window identifiers with AeroSpace, then `aerospace reload-config` | `.config/aerospace/CLAUDE.md` |
| Hammerspoon | query `hs.configdir`, reload through the Aqua session, then query again | `.hammerspoon/CLAUDE.md` |
| tmux | source the live `tmux.conf` or use prefix+r and inspect behavior | `.config/tmux/CLAUDE.md` |
| Yazi | run the package sync against `package.toml`, restart Yazi, and exercise affected opener/plugin behavior | `install_yazi_plugins.sh`, `README.md`, `.config/yazi/CLAUDE.md` |
| Bash setup scripts | strict-mode execution with precondition guards; no separate automated syntax-test command is documented | `setup_mac.sh`, `install_yazi_plugins.sh` |
| Shared Zsh | Nix build plus login-shell/runtime behavior; no separate automated Zsh suite is documented | `zsh/CLAUDE.md`, `nix/modules/zsh.nix`, `nix/CLAUDE.md` |

## Mocking

**Framework:** Not detected. No mocking library, fake service layer, or test-double directory exists alongside `flake.nix`, `setup_mac.sh`, `.hammerspoon/init.lua`, or `.config/nvim/lua/`.

**Patterns:**
- Prefer direct live-path and native-command inspection rather than mocks. `.config/ghostty/CLAUDE.md` verifies that the live Ghostty file and repository source resolve to the same file before editing:

```python
print(os.path.samefile(live, repo))
```

- Prefer native application identity inspection instead of fabricated identifiers; `.config/aerospace/CLAUDE.md` requires `aerospace list-windows --all` and `aerospace list-apps` before adding matching rules.

**What to Mock:**
- Not applicable under the established practice. Configuration integration is validated against Nix evaluation, the real application parser, or the live user session in `nix/CLAUDE.md`, `.config/ghostty/CLAUDE.md`, and `.hammerspoon/CLAUDE.md`.

**What NOT to Mock:**
- Do not substitute guessed AeroSpace app IDs or window titles for inspection output; use the commands in `.config/aerospace/CLAUDE.md`.
- Do not simulate Ghostty parsing when the native validator in `.config/ghostty/CLAUDE.md` can load the actual symlinked configuration.
- Do not treat a Hammerspoon Lua parse alone as proof of behavior; event taps, IPC, Accessibility permission, and Ghostty integration require the live checks described in `.hammerspoon/CLAUDE.md` and `README.md`.
- Do not treat `nix flake check` alone as activation proof; `nix/CLAUDE.md` and `nix/phase-3-plan.md` require build, switch when authorized, and a targeted post-check.

## Fixtures and Factories

**Test Data:**
- No test fixture/factory system exists. Reproducibility inputs are committed lock/config files rather than generated test objects: `flake.lock`, `.config/nvim/lazy-lock.json`, and `.config/yazi/package.toml`.
- Host identity values in `flake.nix` and global fallback versions in `.config/mise/config.toml` are production configuration inputs, not fixtures; validate their real effect through `nix/CLAUDE.md` and `nix/language-stack-plan.md`.

```text
flake.lock                       # locked Nix dependency graph
.config/nvim/lazy-lock.json     # locked Neovim plugin revisions
.config/yazi/package.toml       # locked Yazi plugin revisions/hashes
```

**Location:**
- Keep reproducibility inputs next to their owning entry points: `flake.lock` beside `flake.nix`, `.config/nvim/lazy-lock.json` within `.config/nvim/`, and `.config/yazi/package.toml` within `.config/yazi/`.
- Keep machine state, caches, and runtime downloads outside the repository as required by `.config/nvim/CLAUDE.md`, `.gitignore`, and `README.md`; they are not valid fixtures for repeatable verification.

## Coverage

**Requirements:** None enforced. No line/branch coverage target or coverage instrumentation exists for the shell, Lua, Nix, TOML, or application configuration in `setup_mac.sh`, `.hammerspoon/init.lua`, `flake.nix`, and `.config/`.

**View Coverage:**

```bash
# Not applicable: no coverage-producing test runner is configured.
```

- Treat the validation matrix in `nix/CLAUDE.md` and local app guidance such as `.config/nvim/CLAUDE.md` and `.config/ghostty/CLAUDE.md` as behavioral scope coverage, not numeric code coverage.
- For a cross-cutting change, cover every affected activation route: shared shell changes touch both `zsh/.zshrc` and `nix/modules/zsh.nix`, while app dependency changes also touch `nix/darwin/homebrew.nix` and the relevant instructions in `README.md` / `nix/README.md`.

## Test Types

**Unit Tests:**
- Not used for repository-owned code. No isolated tests exercise helpers in `setup_mac.sh`, `install_yazi_plugins.sh`, `zsh/shared.zsh`, or `.hammerspoon/init.lua`.
- The `mise` package's upstream Darwin check is explicitly disabled with `doCheck = false` in `flake.nix`; the repository compensates with runtime version/resolution post-checks documented in `nix/CLAUDE.md` and `nix/language-stack-plan.md`, not with a local unit-test replacement.
- `.config/nvim/lua/plugins/example.lua` references development tools and typed examples but returns `{}` immediately, so it is not a unit-test or active quality configuration.

**Integration Tests:**
- Use `nix flake check` for flake/module evaluation and the configured-host `darwin-rebuild build` command documented above for assembled host configuration, following `nix/CLAUDE.md`.
- Use application parsers and headless modes for focused integration: Ghostty validation from `.config/ghostty/CLAUDE.md` and Neovim sync/health/startup from `.config/nvim/CLAUDE.md`.
- Use `ya pkg install` through `install_yazi_plugins.sh` to reconcile the locked `.config/yazi/package.toml` with a real Yazi config directory; restart and exercise the affected binding as instructed by `README.md` and `.config/yazi/CLAUDE.md`.
- Validate shared Zsh through both ownership routes because `zsh/shared.zsh` is sourced by `zsh/.zshrc` and embedded by `nix/modules/zsh.nix`.

**E2E Tests:**
- An authorized `darwin-rebuild switch` against the configured host target is the host-level E2E activation and is deliberately manual/mutating; run it only after the check/build/review gates in `nix/CLAUDE.md`.
- Verify observable postconditions after switch, such as command resolution/version, Homebrew services, tmux behavior, or macOS defaults, using the examples in `nix/language-stack-plan.md` and `nix/phase-3-plan.md`.
- Hammerspoon behavior requires the running app, Accessibility permission, live IPC, and dependent Ghostty app described in `.hammerspoon/CLAUDE.md` and `README.md`.
- AeroSpace rule behavior requires live windows and identifiers, and tmux changes require a live source/reload, according to `.config/aerospace/CLAUDE.md` and `.config/tmux/CLAUDE.md`.

## Common Patterns

**Async Testing:**
- No asynchronous test framework exists. For Neovim's plugin/network-backed initialization, run the documented headless commands sequentially so sync completes before health and clean-start checks in `.config/nvim/CLAUDE.md`:

```bash
export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"
nvim --headless "+Lazy! sync" +qa
nvim --headless "+checkhealth lazy" +qa
nvim --headless "+checkhealth vim.treesitter" +qa
nvim --headless +qa
git diff --check
```

- Treat Hammerspoon reload as a state transition rather than trusting the reload command's transport status: `.hammerspoon/CLAUDE.md` notes that reload can invalidate the message port, so query `hs.configdir` again after reload.

```bash
/Applications/Hammerspoon.app/Contents/Frameworks/hs/hs -q -c 'return hs.configdir'
launchctl asuser "$(id -u)" /Applications/Hammerspoon.app/Contents/Frameworks/hs/hs -q -c 'hs.reload()'
/Applications/Hammerspoon.app/Contents/Frameworks/hs/hs -q -c 'return hs.configdir'
```

**Error Testing:**
- No automated negative-path suite exists. Root scripts encode error behavior as guards with stderr plus exit status, as in `install_yazi_plugins.sh`:

```bash
if [[ ! -d "$config_dir" ]]; then
  echo "Yazi config directory not found: $config_dir" >&2
  exit 1
fi
```

- Neovim bootstrap explicitly tests clone failure and exits after displaying the command output in `.config/nvim/lua/config/lazy.lua`; validate this implementation path through code review because no harness forces the failure.
- `setup_mac.sh` exercises destructive branches only after interactive confirmation and defaults to skip/preserve; do not automate those branches against a real home directory without an isolated target.
- For Nix failures, stop at the first failing check/build command and do not proceed to switch; this gate is the required error-containment pattern in `nix/CLAUDE.md` and `nix/language-stack-plan.md`.

## Adding or Updating Validation

- For a Nix/system change, preserve the `check -> build -> review -> switch -> post-check` sequence and document the concrete postcondition in `nix/CLAUDE.md` or the relevant durable plan such as `nix/language-stack-plan.md`.
- For an application config, add or update the native validation/reload command in its local `CLAUDE.md` and user-facing steps in `README.md`, following `.config/ghostty/CLAUDE.md` and `.config/nvim/CLAUDE.md`.
- For shared shell behavior, verify both the fallback entry `zsh/.zshrc` and Home Manager path `nix/modules/zsh.nix`; keep the shared implementation in `zsh/shared.zsh` as required by `zsh/CLAUDE.md`.
- For scripts without a formal suite, retain strict mode, guard clauses, stderr diagnostics, and an isolated/configurable target like `--config-dir` in `install_yazi_plugins.sh`; document any new repeatable check in `CLAUDE.md` and the relevant local guidance.
- Finish every configuration change with documentation synchronization, diff/privacy review, and one focused commit as required by `CLAUDE.md`; do not treat a passing parser alone as completion.

---

*Testing analysis: 2026-07-10*
