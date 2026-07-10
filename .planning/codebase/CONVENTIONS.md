# Coding Conventions

**Analysis Date:** 2026-07-10

## Naming Patterns

**Files:**
- Use `snake_case.sh` for repository-owned executable scripts at the root, as in `setup_mac.sh` and `install_yazi_plugins.sh`.
- Use the ecosystem entry-point name `default.nix` for compositional Nix modules, and use descriptive kebab-case for sibling modules such as `nix/home/dev-toolchains.nix` and `nix/home/shell-env.nix`.
- Keep application-defined filenames and suffixes unchanged: examples include `.config/ghostty/config`, `.config/tmux/tmux.conf.local`, `.config/yazi/package.toml`, and `.config/yazi/plugins/smart-enter.yazi/`.
- Keep local guidance in `CLAUDE.md` and expose it through an adjacent `AGENTS.md -> CLAUDE.md` symlink, as demonstrated by `.config/nvim/CLAUDE.md` with `.config/nvim/AGENTS.md` and `nix/CLAUDE.md` with `nix/AGENTS.md`.
- Treat vendored names as upstream-owned rather than normalizing them: `.config/ghostty/shaders/README.md` records the shader source, while `.config/yazi/package.toml` pins plugin names and revisions that match directories such as `.config/yazi/plugins/smart-enter.yazi/` and `.config/yazi/plugins/git.yazi/`.

**Functions:**
- Name Bash helpers in `snake_case`, for example `collect_config_names` in `setup_mac.sh` and `require_cmd` / `warn_if_missing` in `install_yazi_plugins.sh`.
- Keep the intentionally short interactive Zsh wrapper `y` as the user-facing command in `zsh/shared.zsh`; use descriptive names for any non-command helper added alongside it.
- Name Hammerspoon local functions and state helpers in lower camel case, such as `stopTimer`, `resetWState`, `injectCmdStroke`, and `toggleGhostty` in `.hammerspoon/init.lua`.
- Prefer framework-native declarative tables over named functions in Neovim configuration; `.config/nvim/lua/config/lazy.lua` passes one options table to `require("lazy").setup`, while `.config/nvim/init.lua` remains a one-line bootstrap.
- Preserve upstream function naming and formatting inside vendored Yazi plugins such as `.config/yazi/plugins/smart-enter.yazi/main.lua`; do not use those files as the style authority for repository-owned Lua in `.hammerspoon/init.lua` or `.config/nvim/lua/`.

**Variables:**
- Use lower `snake_case` for Bash path and state variables (`repo_dir`, `config_source`, `created_any`) in `setup_mac.sh` and `config_dir`, `package_file`, `tracked_config` in `install_yazi_plugins.sh`.
- Use uppercase names for exported environment variables and explicit command-line overrides (`CONFIG_DIR_OVERRIDE`, `LG_CONFIG_FILE`, `YAZI_CONFIG_HOME`) in `install_yazi_plugins.sh`, and for public shell environment variables such as `BUN_INSTALL` in `zsh/shared.zsh`.
- Quote shell expansions and array expansions unless language syntax specifically requires otherwise; representative patterns are `"$target_path"`, `"${plugins[@]}"`, and `"${BASH_SOURCE[0]}"` in `setup_mac.sh` and `install_yazi_plugins.sh`.
- Use lower camel case for mutable Hammerspoon state (`cmdIsDown`, `waitingForSecondW`, `doublePressTap`) and keep it `local` at file scope in `.hammerspoon/init.lua`.
- Use simple lowercase bindings for local Nix composition values (`system`, `username`, `hostname`) in `flake.nix`; retain upstream option spelling, including camel-case options in `nix/home/default.nix` and hyphenated attributes in `nix/home/dev-toolchains.nix`.
- Preserve each application's schema naming rather than translating it: kebab-case keys belong in `.config/ghostty/config` and `.config/aerospace/aerospace.toml`, while aligned snake-case keys belong in `.config/yazi/yazi.toml`.

**Types:**
- No repository-wide static type layer exists; Nix modules express shape through destructured argument sets and returned attribute sets in files such as `nix/darwin/default.nix` and `nix/modules/zsh.nix`.
- Use Lua tables and API-specific values directly in repository-owned Lua, as in the modifier arrays and callback returns in `.hammerspoon/init.lua` and the plugin specification tables in `.config/nvim/lua/config/lazy.lua`.
- Do not infer a project type-annotation requirement from `---@...` comments in `.config/nvim/lua/plugins/example.lua` or `.config/yazi/plugins/git.yazi/types.lua`: the former is an inactive starter example and the latter belongs to a vendored plugin.
- Shell scripts rely on Bash/Zsh runtime semantics rather than declared types; represent lists with arrays (`plugins` in `install_yazi_plugins.sh`) and booleans with explicit string values (`created_any` in `setup_mac.sh`).

## Code Style

**Formatting:**
- Format Nix with two-space indentation, terminating option assignments with semicolons and placing non-trivial list items on separate lines, following `flake.nix`, `nix/darwin/homebrew.nix`, and `nix/modules/zsh.nix`.
- Keep small Nix attribute sets inline only when they remain readable, such as `specialArgs = { inherit inputs username hostname; };` in `flake.nix`; use expanded blocks for nested policy in `nix/darwin/homebrew.nix`.
- Format repository-owned Bash with two-space indentation, `[[ ... ]]` conditionals, quoted expansions, and `set -euo pipefail`, following `setup_mac.sh` and `install_yazi_plugins.sh`.
- Format Neovim Lua with StyLua's configured spaces, width 2, and column width 120 from `.config/nvim/stylua.toml`; use `-- stylua: ignore` only for a deliberate compact exception such as the inactive guard in `.config/nvim/lua/plugins/example.lua`.
- Scope `.config/nvim/stylua.toml` to Neovim Lua. `.hammerspoon/init.lua` and vendored files such as `.config/yazi/plugins/smart-enter.yazi/main.lua` and `.config/yazi/plugins/lazygit.yazi/main.lua` have independent surrounding styles and no repository-wide Lua formatter configuration.
- Preserve schema-oriented alignment and indentation in application configs instead of applying one global formatter: `.config/aerospace/aerospace.toml` uses four-space nested tables, while `.config/yazi/yazi.toml` aligns keys and uses tabs in arrays.
- Keep app-native syntax intact in non-general-purpose configuration formats, including `key = value` in `.config/ghostty/config`, `option=value` in `.config/mpv/mpv.conf`, and the oh-my-tmux variable layout in `.config/tmux/tmux.conf.local`.

**Linting:**
- Use `nix flake check` as the declarative Nix evaluation gate documented in `nix/CLAUDE.md`; the current configured flake target is `.#AresdeMacBook-Air`, and `darwin-rebuild build` against that target is the documented non-activating host build gate.
- Use `git diff --check` for whitespace/error checks where the repository documents it, including `.config/nvim/CLAUDE.md`, `.config/ghostty/CLAUDE.md`, and `nix/language-stack-plan.md`.
- Treat `.gitleaks.toml` as the available policy for an explicit manual secret scan. The redacted Gitleaks command in `nix/language-stack-plan.md` is a validation example for that language-stack workflow, not a repository-wide command required before every configuration commit.
- Use application-native validators when available: Ghostty's `+validate-config` command is specified in `.config/ghostty/CLAUDE.md`, and Neovim headless startup/health checks are specified in `.config/nvim/CLAUDE.md`.
- Do not treat the `stylua`, `shellcheck`, and `shfmt` entries in `.config/nvim/lua/plugins/example.lua` as an active repository lint pipeline because that example returns an empty spec before those declarations.
- No single root lint configuration or CI workflow orchestrates every format; validation contracts live in `nix/CLAUDE.md` and application-local guidance such as `.config/nvim/CLAUDE.md` and `.config/ghostty/CLAUDE.md`.

## Import Organization

**Order:**
1. In Nix aggregators, put the destructured module argument set first, then the `imports` block, then owned options; `nix/darwin/default.nix` and `nix/home/default.nix` are the reference layout.
2. In `flake.nix`, declare external `inputs` before `outputs`, define host bindings in the `let`, then compose modules in dependency order inside `darwinConfigurations`.
3. In Neovim, bootstrap local values before external setup; within `.config/nvim/lua/config/lazy.lua`, import LazyVim core first, LazyVim extras second, and `{ import = "plugins" }` last so local specs can override upstream behavior.
4. In Hammerspoon, load required modules before registering callbacks and event taps; `.hammerspoon/init.lua` calls `require("hs.ipc")` before defining hotkeys and automation state.
5. In the fallback shell path, establish path/completion state, source shared public logic, source private local overrides, and load syntax highlighting last; preserve the order in `zsh/.zshrc` and the Home Manager embedding order in `nix/modules/zsh.nix`.

**Path Aliases:**
- No TypeScript-style or compiler path aliases exist; Nix uses explicit relative imports such as `./homebrew.nix` in `nix/darwin/default.nix` and `../modules/zsh.nix` in `nix/home/default.nix`.
- Lua uses runtime module namespaces supplied by the host application, for example `require("config.lazy")` in `.config/nvim/init.lua`, `require("hs.ipc")` in `.hammerspoon/init.lua`, and plugin names in `.config/yazi/init.lua`.
- Resolve shell-owned files relative to the script rather than the caller's working directory, as shown by `repo_dir` in `setup_mac.sh` / `install_yazi_plugins.sh` and `_zshrc_dir` in `zsh/.zshrc`.

## Error Handling

**Patterns:**
- Start executable Bash scripts with `set -euo pipefail`; both `setup_mac.sh` and `install_yazi_plugins.sh` use strict mode so command, unset-variable, and pipeline failures stop execution.
- Validate preconditions with guard clauses and explicit non-zero exits. `install_yazi_plugins.sh` checks the configuration directory, `package.toml`, and required commands; `setup_mac.sh` checks the repository config directory, user input, and target home directory.
- Send actionable errors and warnings to stderr in command-oriented helpers, following `require_cmd`, `warn_if_missing`, and argument parsing in `install_yazi_plugins.sh`.
- Require positive user confirmation before destructive replacement; every `rm -rf` path in `setup_mac.sh` is reached only after a `[y/N]` prompt and defaults to preserving the existing target.
- Prefer capability guards for optional integrations so minimal machines degrade quietly, as demonstrated by `command -v starship`, Homebrew existence checks, and optional Bun completion sourcing in `zsh/shared.zsh`.
- Surface bootstrap failure to the user and exit non-zero in interactive Neovim startup; `.config/nvim/lua/config/lazy.lua` checks `vim.v.shell_error`, displays clone output, waits for acknowledgement, and calls `os.exit(1)`.
- Make event propagation explicit in Hammerspoon callbacks: `.hammerspoon/init.lua` returns `true` only when an event is consumed and temporarily stops `doublePressTap` while injecting synthetic key strokes to prevent recursive capture.
- Let Nix evaluation/build failures stop the change before activation; `nix/CLAUDE.md` prescribes check and build before switch, while the isolated upstream `mise` package exception is declared explicitly with `doCheck = false` in `flake.nix` and compensated by documented runtime post-checks in `nix/CLAUDE.md`.

## Logging

**Framework:** Shell stdout/stderr, host-application UI, and native command output; there is no shared logging framework across `setup_mac.sh`, `.hammerspoon/init.lua`, and `nix/`.

**Patterns:**
- Use concise progress messages for user-visible mutations in `setup_mac.sh`, including skipped, preserved, and created links; include both source and target when a symlink is created.
- Use stderr for missing requirements and degraded optional capabilities in `install_yazi_plugins.sh`, while keeping successful package installation summaries on stdout.
- Use Neovim's own UI for an interactive bootstrap error in `.config/nvim/lua/config/lazy.lua` rather than raw `print` output.
- Keep Hammerspoon automation low-noise as required by `.hammerspoon/CLAUDE.md`; `.hammerspoon/init.lua` has one load confirmation alert and otherwise avoids persistent logging in hot paths.
- Rely on `nix flake check`, `darwin-rebuild`, and application-native validator output instead of adding a wrapper logger around commands documented in `nix/CLAUDE.md`, `.config/nvim/CLAUDE.md`, and `.config/ghostty/CLAUDE.md`.

## Comments

**When to Comment:**
- Write Chinese comments in new or modified scripts, matching the repository rule in `CLAUDE.md` and the implementation style in `setup_mac.sh`, `zsh/shared.zsh`, `.hammerspoon/init.lua`, and `nix/darwin/homebrew.nix`.
- Explain safety boundaries, ordering constraints, fallback behavior, and why an option is intentionally excluded; the detailed rationale blocks in `nix/darwin/defaults.nix`, `nix/darwin/homebrew.nix`, and `nix/modules/zsh.nix` are the reference pattern.
- Keep comments close to the relevant declaration, as in the inline package ownership notes in `nix/home/packages.nix` and the section comments in `.config/ghostty/config`.
- Preserve upstream English comments in starter or vendored material such as `.config/nvim/lua/config/autocmds.lua`, `.config/nvim/lua/plugins/example.lua`, and `.config/yazi/plugins/full-border.yazi/main.lua`; do not translate or restyle vendored code without an upstream refresh reason.
- Avoid long commented-out boilerplate in shared shell code, following the explicit constraint in `zsh/CLAUDE.md`; retain only compatibility or ordering context that protects behavior in `zsh/.zshrc` and `zsh/shared.zsh`.

**JSDoc/TSDoc:**
- Not applicable: the repository contains no JavaScript/TypeScript implementation surface; the only annotation-like comments are inactive Neovim example annotations in `.config/nvim/lua/plugins/example.lua` and vendored EmmyLua types in `.config/yazi/plugins/git.yazi/types.lua`.
- If an application API requires Lua annotations, keep them in that module and match its host convention rather than creating a repository-wide documentation system; `.config/yazi/plugins/git.yazi/types.lua` demonstrates the plugin-local pattern.

## Function Design

**Size:** Keep helpers single-purpose and leave orchestration linear. `install_yazi_plugins.sh` separates `usage`, `require_cmd`, and `warn_if_missing`, while `setup_mac.sh` isolates tracked-directory discovery and its physical-directory fallback in `collect_config_names`.

**Parameters:**
- Validate positional shell parameters before use and keep optional CLI state explicit, as in the `--config-dir` parser and `$1`-based helpers in `install_yazi_plugins.sh`.
- Destructure only needed Nix module arguments and include `...` for framework-supplied values, following `nix/darwin/default.nix`, `nix/home/default.nix`, and `nix/modules/zsh.nix`.
- Use callback closures for Hammerspoon stateful automation and keep shared mutable state local to `.hammerspoon/init.lua`; pass host API objects directly instead of wrapping them in generic abstractions.

**Return Values:**
- Use shell exit status for success/failure and stdout only when a helper intentionally produces data; `collect_config_names` in `setup_mac.sh` prints names for process substitution, while failures exit from the main flow.
- Return explicit booleans from Hammerspoon event callbacks to control whether macOS receives the event, following both event taps in `.hammerspoon/init.lua`.
- Return declarative attribute sets from every Nix module, as shown by `nix/home/shell-env.nix` and `nix/darwin/defaults.nix`; return plugin specification tables from Lua modules such as `.config/nvim/lua/config/lazy.lua` and `.config/yazi/plugins/smart-enter.yazi/main.lua`.

## Module Design

**Exports:**
- Keep Nix concerns in focused modules and compose them through explicit imports: `nix/darwin/default.nix` owns system composition, `nix/home/default.nix` owns Home Manager composition, and `flake.nix` owns host assembly.
- Put shell behavior shared by Home Manager and the fallback entry in `zsh/shared.zsh`; embed it from `nix/modules/zsh.nix` and source it from `zsh/.zshrc` instead of duplicating logic.
- Keep application config in its application directory and treat that directory as the source of truth, as documented by `.config/CLAUDE.md`, `.config/nvim/CLAUDE.md`, and `.hammerspoon/CLAUDE.md`.
- Keep Hammerspoon as an executable configuration entry point rather than an exported library; `.hammerspoon/init.lua` registers its hotkeys and event taps directly.
- Keep Neovim plugin overrides under `.config/nvim/lua/plugins/` and core bootstrap/options under `.config/nvim/lua/config/`, following `.config/nvim/CLAUDE.md`; the inactive `.config/nvim/lua/plugins/example.lua` is a reference, not active behavior.
- Treat plugin directories such as `.config/yazi/plugins/git.yazi/` and the shader directory `.config/ghostty/shaders/` as pinned/vendored assets governed by `.config/yazi/package.toml` and `.config/ghostty/shaders/README.md`; refresh them as upstream units rather than casually refactoring them.

**Barrel Files:**
- Nix `default.nix` files are the repository's composition entry points, as seen in `nix/darwin/default.nix` and `nix/home/default.nix`; add explicit imports there rather than introducing implicit discovery.
- `.config/nvim/init.lua` is a minimal bootstrap into `config.lazy`, not a general barrel, and `.config/nvim/lua/config/lazy.lua` deliberately places local plugin import last.
- No general-purpose shell or Lua index/barrel convention exists outside the explicit shared entry points `zsh/shared.zsh`, `.config/nvim/init.lua`, and `.config/yazi/init.lua`.

## Change Workflow

- Consult the official application/plugin documentation before changing an app config, prefer a small targeted edit, and keep the root and local documentation synchronized as required by `CLAUDE.md` and application guidance such as `.config/aerospace/CLAUDE.md`.
- Update `README.md` plus the relevant local `CLAUDE.md`; create or retain the adjacent `AGENTS.md -> CLAUDE.md` symlink for complex configuration directories, following the checklist in `CLAUDE.md` and existing pairs under `.config/nvim/` and `nix/`.
- Keep private or machine-specific shell values out of shared files and load them only from the ignored local entry point documented by `zsh/CLAUDE.md`, `zsh/.zshrc`, and `nix/modules/zsh.nix`.
- Review the diff manually for credentials and personal data before committing, as required by `CLAUDE.md`. When an explicit secret scan is appropriate, use the policy in `.gitleaks.toml`; the command in `nix/language-stack-plan.md` is a scoped example rather than a universal gate.
- Make one focused English-language commit after the implementation, documentation, and validation are complete; do not push without explicit user direction, per `CLAUDE.md`.

---

*Convention analysis: 2026-07-10*
