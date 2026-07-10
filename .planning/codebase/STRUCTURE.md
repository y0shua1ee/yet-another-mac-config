# Codebase Structure

**Analysis Date:** 2026-07-10

## Directory Layout

```text
yet-another-mac-config/
├── .config/                     # XDG-style, app-native configuration sources
│   ├── aerospace/                 # Window-manager rules and bindings
│   ├── borders/                   # JankyBorders options
│   ├── btop/                      # btop configuration
│   ├── gh/                        # Non-secret GitHub CLI preferences
│   ├── ghostty/                   # Terminal config plus vendored shaders
│   ├── mise/                      # Global runtime fallbacks
│   ├── mpv/                       # Media-player configuration
│   ├── nvim/                      # LazyVim bootstrap, modules, and lockfiles
│   ├── tmux/                      # Tracked oh-my-tmux override
│   ├── typora/                    # Typora theme assets
│   └── yazi/                      # Yazi config, keymaps, theme, and plugins
├── .hammerspoon/                 # macOS hotkeys and event automation
├── .planning/codebase/           # Generated GSD codebase maps
├── nix/                          # Declarative macOS and Home Manager configuration
│   ├── darwin/                    # System, Homebrew, service, and defaults modules
│   ├── home/                      # User packages, environment, and tool managers
│   └── modules/                   # Reusable Home Manager modules
├── zsh/                          # Shared shell core and fallback entry
├── flake.nix                    # Nix composition root and host output
├── flake.lock                   # Pinned Nix input graph
├── setup_mac.sh                 # Interactive symlink bootstrap
├── install_yazi_plugins.sh      # Yazi package synchronization
├── README.md                    # User-facing inventory and setup guide
├── CLAUDE.md                    # Repository-wide agent instructions
├── AGENTS.md                    # Symlink to `CLAUDE.md`
└── .gitignore                   # Local-state and generated-output boundary
```

The versioned application set is the set of top-level app directories represented by tracked files beneath `.config/`. `setup_mac.sh` computes candidates from that tracked set only when Git is available, the repository is a Git worktree, and the normalized `git ls-files` result yields at least one tracked top-level app directory. If any condition fails, including an empty tracked result in a valid worktree, it falls back to physical top-level `.config` subdirectories, which can include ignored or untracked local state and therefore require review at the interactive prompt.

## Directory Purposes

**Repository Root (`.`):**
- Purpose: Holds the human-facing documentation, Nix composition root, bootstrap scripts, and repository policy at `README.md`, `flake.nix`, `setup_mac.sh`, and `CLAUDE.md`.
- Contains: Shell entry scripts, Nix lock data, ignore/security policy, documentation, and links to subordinate configuration trees in `README.md` and `.gitignore`.
- Key files: `flake.nix`, `flake.lock`, `setup_mac.sh`, `install_yazi_plugins.sh`, `README.md`, `CLAUDE.md`, `.gitignore`, `.gitleaks.toml`

**Nix Tree (`nix/`):**
- Purpose: Contains all declarative system and user configuration imported by `flake.nix`.
- Contains: Runtime modules in `nix/darwin/`, `nix/home/`, and `nix/modules/`, plus user and maintainer documentation in `nix/README.md` and `nix/CLAUDE.md`.
- Key files: `nix/darwin/default.nix`, `nix/home/default.nix`, `nix/modules/zsh.nix`, `nix/README.md`, `nix/CLAUDE.md`

**Darwin Modules (`nix/darwin/`):**
- Purpose: Owns host-level settings and packages evaluated by nix-darwin from `nix/darwin/default.nix`.
- Contains: Module aggregation, Homebrew taps/formulae/casks/services, and conservative macOS defaults in `nix/darwin/default.nix`, `nix/darwin/homebrew.nix`, and `nix/darwin/defaults.nix`.
- Key files: `nix/darwin/default.nix`, `nix/darwin/homebrew.nix`, `nix/darwin/defaults.nix`

**Home Manager Modules (`nix/home/`):**
- Purpose: Owns user packages, session variables, and developer-tool managers composed by `nix/home/default.nix`.
- Contains: Small purpose-specific modules `nix/home/packages.nix`, `nix/home/shell-env.nix`, and `nix/home/dev-toolchains.nix`.
- Key files: `nix/home/default.nix`, `nix/home/packages.nix`, `nix/home/shell-env.nix`, `nix/home/dev-toolchains.nix`

**Reusable Home Modules (`nix/modules/`):**
- Purpose: Holds user-facing modules that are broader than a package list and are explicitly imported by `nix/home/default.nix`.
- Contains: The generated-shell implementation in `nix/modules/zsh.nix`.
- Key files: `nix/modules/zsh.nix`

**Shell Sources (`zsh/`):**
- Purpose: Provides one shared public shell core and a separate fallback entry outside Home Manager in `zsh/shared.zsh` and `zsh/.zshrc`.
- Contains: Route-neutral behavior in `zsh/shared.zsh`, direct-link startup ordering in `zsh/.zshrc`, and maintenance rules in `zsh/CLAUDE.md`.
- Key files: `zsh/shared.zsh`, `zsh/.zshrc`, `zsh/CLAUDE.md`, `zsh/AGENTS.md`

**XDG Configuration Root (`.config/`):**
- Purpose: Mirrors the app directory names expected beneath a user's `.config` path and serves as the discovery root for `setup_mac.sh`; discovery is tracked-only when Git is available, the repository is a worktree, and the normalized tracked-file query yields at least one top-level app directory, and is physical-directory based when any condition fails.
- Contains: Independent app-native modules and the shared guidance link `.config/AGENTS.md` backed by `.config/CLAUDE.md`.
- Key files: `.config/CLAUDE.md`, `.config/AGENTS.md`, `.config/.gitignore`

**AeroSpace Configuration (`.config/aerospace/`):**
- Purpose: Declares workspaces, floating-window rules, monitor assignment, gaps, and keybinding modes in `.config/aerospace/aerospace.toml`.
- Contains: One app config plus local guidance at `.config/aerospace/CLAUDE.md` and its `.config/aerospace/AGENTS.md` symlink.
- Key files: `.config/aerospace/aerospace.toml`, `.config/aerospace/CLAUDE.md`

**Ghostty Configuration (`.config/ghostty/`):**
- Purpose: Declares terminal behavior and packages a local shader collection referenced by `.config/ghostty/config`.
- Contains: The main config, 33 tracked GLSL shader files, shader provenance notes, and local guidance beneath `.config/ghostty/`.
- Key files: `.config/ghostty/config`, `.config/ghostty/shaders/README.md`, `.config/ghostty/CLAUDE.md`

**Neovim Configuration (`.config/nvim/`):**
- Purpose: Implements a LazyVim starter with explicit extras and room for local override specs in `.config/nvim/lua/`.
- Contains: Entry `init.lua`, bootstrap and option modules under `.config/nvim/lua/config/`, local specs under `.config/nvim/lua/plugins/`, plugin locks, and formatting metadata.
- Key files: `.config/nvim/init.lua`, `.config/nvim/lua/config/lazy.lua`, `.config/nvim/lua/config/options.lua`, `.config/nvim/lua/config/keymaps.lua`, `.config/nvim/lua/config/autocmds.lua`, `.config/nvim/lazy-lock.json`, `.config/nvim/stylua.toml`

**Tmux Configuration (`.config/tmux/`):**
- Purpose: Stores the tracked user override for a locally installed oh-my-tmux base configured by `setup_mac.sh`.
- Contains: `.config/tmux/tmux.conf.local`, guidance, and an ignored machine-specific `.config/tmux/tmux.conf` symlink when provisioned.
- Key files: `.config/tmux/tmux.conf.local`, `.config/tmux/CLAUDE.md`, `.config/tmux/AGENTS.md`

**Yazi Configuration (`.config/yazi/`):**
- Purpose: Defines Yazi manager behavior, openers, keymaps, theme, plugin loading, and locked dependencies.
- Contains: Core TOML files, Lua initialization, vendored/materialized plugin directories ending in `.yazi`, and local guidance in `.config/yazi/CLAUDE.md`.
- Key files: `.config/yazi/yazi.toml`, `.config/yazi/keymap.toml`, `.config/yazi/theme.toml`, `.config/yazi/package.toml`, `.config/yazi/init.lua`, `.config/yazi/CLAUDE.md`

**Simple Leaf Configurations (`.config/borders/`, `.config/btop/`, `.config/gh/`, `.config/mise/`, `.config/mpv/`, `.config/typora/`):**
- Purpose: Store single-purpose app-native configuration that inherits repository or `.config/CLAUDE.md` guidance.
- Contains: `.config/borders/bordersrc`, `.config/btop/btop.conf`, `.config/gh/config.yml`, `.config/mise/config.toml`, `.config/mpv/mpv.conf`, and `.config/typora/themes/spring.css`.
- Key files: `.config/borders/bordersrc`, `.config/mise/config.toml`, `.config/mpv/mpv.conf`, `.config/typora/themes/spring.css`

**Hammerspoon Automation (`.hammerspoon/`):**
- Purpose: Owns macOS keyboard automation, Ghostty activation, and Hammerspoon IPC in `.hammerspoon/init.lua`.
- Contains: The single active Lua entry and adjacent agent guidance.
- Key files: `.hammerspoon/init.lua`, `.hammerspoon/CLAUDE.md`, `.hammerspoon/AGENTS.md`

**Generated Planning Maps (`.planning/codebase/`):**
- Purpose: Stores the GSD codebase-reference documents generated from the repository, including `.planning/codebase/ARCHITECTURE.md` and `.planning/codebase/STRUCTURE.md`.
- Contains: Uppercase Markdown reference documents under `.planning/codebase/`.
- Key files: `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/STRUCTURE.md`

## Key File Locations

**Entry Points:**
- `flake.nix`: Nix/nix-darwin/Home Manager composition root and named host output.
- `setup_mac.sh`: Interactive entry for symlinking selected application configuration into a target home; it uses tracked discovery only when Git is available, the repository is a worktree, and the normalized tracked-file query yields at least one top-level app directory, with physical-directory fallback when any condition fails.
- `install_yazi_plugins.sh`: Direct entry for synchronizing `.config/yazi/package.toml` dependencies.
- `nix/darwin/default.nix`: System-layer module entry imported by `flake.nix`.
- `nix/home/default.nix`: User-layer module entry imported by `flake.nix`.
- `nix/modules/zsh.nix`: Primary Home Manager shell entry imported by `nix/home/default.nix`.
- `zsh/.zshrc`: Fallback shell entry offered by `setup_mac.sh`.
- `.config/nvim/init.lua`: Neovim/LazyVim runtime entry.
- `.config/yazi/init.lua`: Yazi plugin initialization entry.
- `.hammerspoon/init.lua`: Hammerspoon runtime and IPC entry.

**Configuration:**
- `flake.lock`: Exact source revisions for the external inputs declared in `flake.nix`.
- `nix/darwin/homebrew.nix`: Homebrew taps, packages, casks, and bounded service ownership.
- `nix/darwin/defaults.nix`: Declared macOS Finder, Dock, and keyboard defaults.
- `nix/home/packages.nix`: Low-risk Home Manager CLI inventory.
- `nix/home/dev-toolchains.nix`: mise, uv, rustup, direnv, and nix-direnv entry points.
- `nix/home/shell-env.nix`: Non-private session variables.
- `.config/mise/config.toml`: Global Node and Go fallback versions consumed by mise.
- `.gitignore`: Canonical boundary for local state, credentials, caches, and generated output.

**Core Logic:**
- `setup_mac.sh`: Three-condition tracked-directory discovery with physical-directory fallback, conflict prompts, target replacement, symlink creation, and oh-my-tmux setup.
- `zsh/shared.zsh`: Public shell behavior shared across both activation routes.
- `nix/modules/zsh.nix`: Shell ordering, private overlay loading, and mise activation.
- `.hammerspoon/init.lua`: Stateful event-tap and hotkey behavior.
- `.config/nvim/lua/config/lazy.lua`: Plugin-manager bootstrap and LazyVim composition.
- `.config/yazi/yazi.toml`: Opener, MIME rule, task, previewer, and plugin routing.
- `.config/aerospace/aerospace.toml`: Window-rule and keybinding behavior.

**Testing:**
- `nix/CLAUDE.md`: Prescribes flake evaluation, Darwin build, switch, rollback, and post-check commands; no separate automated test tree is present.
- `.config/nvim/CLAUDE.md`: Prescribes headless LazyVim synchronization and health checks.
- `.config/ghostty/CLAUDE.md`: Prescribes Ghostty config validation and Git whitespace/status checks.
- `.config/aerospace/CLAUDE.md`: Prescribes app inspection and reload behavior for AeroSpace changes.
- `.hammerspoon/CLAUDE.md`: Prescribes IPC, reload, and runtime verification for Hammerspoon.

## Naming Conventions

**Files:**
- Nix composition entries use `default.nix` inside importable directories, as in `nix/darwin/default.nix` and `nix/home/default.nix`.
- Purpose-specific Nix modules use lowercase kebab-case names, as in `nix/home/dev-toolchains.nix` and `nix/home/shell-env.nix`.
- App configuration preserves upstream-native filenames rather than normalizing them, as in `.config/aerospace/aerospace.toml`, `.config/borders/bordersrc`, `.config/ghostty/config`, and `.config/tmux/tmux.conf.local`.
- Shell scripts at the repository root use snake_case action names, as in `setup_mac.sh` and `install_yazi_plugins.sh`.
- Lua runtime entries use conventional `init.lua`, while subordinate Neovim modules use lowercase purpose names under `.config/nvim/lua/config/`.
- Documentation and agent instruction files use uppercase conventional names such as `README.md`, `CLAUDE.md`, and `AGENTS.md`; each local `AGENTS.md` is a symlink to the adjacent `CLAUDE.md`.
- Generated GSD reference documents use uppercase names under `.planning/codebase/`, including `.planning/codebase/ARCHITECTURE.md` and `.planning/codebase/STRUCTURE.md`.

**Directories:**
- XDG app directories match the application's live config directory name, as in `.config/ghostty/`, `.config/nvim/`, and `.config/yazi/`.
- Nix directories name the ownership layer (`nix/darwin/`, `nix/home/`) or reusable role (`nix/modules/`).
- Yazi plugin directories use the `.yazi` suffix expected by the host app, as in `.config/yazi/plugins/smart-enter.yazi/` and `.config/yazi/plugins/git.yazi/`.
- Asset collections use simple plural nouns beneath their owner, as in `.config/ghostty/shaders/` and `.config/yazi/plugins/`.

## Where to Add New Code

**New System-Level Feature:**
- Primary code: Add a focused module under `nix/darwin/` and import it from `nix/darwin/default.nix`; extend `nix/darwin/homebrew.nix` directly only for bounded Homebrew inventory/service changes.
- Tests: Add the relevant evaluation/build/switch verification instructions to `nix/README.md` or `nix/CLAUDE.md`, following the existing Nix validation location.

**New User-Level Feature:**
- Primary code: Add a focused module under `nix/home/` and import it from `nix/home/default.nix`; use `nix/modules/` when the feature is a reusable behavioral module rather than a list of values.
- Tests: Keep module evaluation and runtime post-check instructions with `nix/README.md` and `nix/CLAUDE.md` because the repository has no separate test harness.

**New Shared Shell Behavior:**
- Primary code: Put public, cross-machine behavior used by both routes in `zsh/shared.zsh`.
- Route-specific code: Put Home Manager ordering/features in `nix/modules/zsh.nix` and fallback-only bootstrap behavior in `zsh/.zshrc`.

**New Application Configuration:**
- Primary code: Create `.config/<app>/` using the live XDG directory name and add its intended public configuration to Git so the tracked-only `setup_mac.sh` path can discover it when all three discovery conditions hold; do not rely on the broader physical-directory fallback.
- Documentation: For a complex, multi-file, or frequently changed app, add `.config/<app>/CLAUDE.md` and an adjacent `.config/<app>/AGENTS.md` symlink; a simple leaf can inherit `.config/CLAUDE.md`.
- Package ownership: Add required runtime packages to the appropriate existing owner in `nix/darwin/homebrew.nix` or `nix/home/packages.nix`, not to `setup_mac.sh`.

**New Neovim Component/Module:**
- Implementation: Put plugin specs or overrides under `.config/nvim/lua/plugins/`; put startup options, keymaps, and autocmds in the matching file under `.config/nvim/lua/config/`.
- Tests: Use the headless checks documented in `.config/nvim/CLAUDE.md`.

**New Yazi Plugin or Binding:**
- Implementation: Declare/pin the plugin in `.config/yazi/package.toml`, initialize it in `.config/yazi/init.lua` when required, and add app behavior in `.config/yazi/keymap.toml` or `.config/yazi/yazi.toml`.
- Synchronization: Update helper/prerequisite behavior in `install_yazi_plugins.sh` and user instructions in `README.md` when the plugin adds external commands.

**New Hammerspoon Automation:**
- Implementation: Add explicit behavior to `.hammerspoon/init.lua`; if the automation is intentionally factored as a Hammerspoon Spoon, create and track `.hammerspoon/Spoons/` with the implementation at that time.
- Tests: Record and run IPC/reload/runtime checks following `.hammerspoon/CLAUDE.md`.

**Utilities:**
- Shared helpers: Keep repository bootstrap helpers in the relevant root script (`setup_mac.sh` or `install_yazi_plugins.sh`) rather than adding an unreferenced utility directory.
- Shared shell helpers: Place shell functions used by both shell routes in `zsh/shared.zsh`.

## Special Directories

**`.config/ghostty/shaders/`:**
- Purpose: Stores the external Ghostty shader collection referenced by `.config/ghostty/config`.
- Generated: No; it is a curated vendored snapshot whose source is documented in `.config/ghostty/shaders/README.md`.
- Committed: Yes; tracked GLSL assets live under `.config/ghostty/shaders/`.

**`.config/yazi/plugins/`:**
- Purpose: Stores Yazi plugin code corresponding to the locked entries in `.config/yazi/package.toml`.
- Generated: Yes; plugin content is materialized/refreshed through `ya pkg install` invoked by `install_yazi_plugins.sh`.
- Committed: Yes; plugin sources and their license/readme files are tracked beneath `.config/yazi/plugins/`.

**`.config/nvim/lazy-lock.json`:**
- Purpose: Pins the resolved Neovim plugin graph bootstrapped by `.config/nvim/lua/config/lazy.lua`.
- Generated: Yes; lazy.nvim updates `.config/nvim/lazy-lock.json`.
- Committed: Yes; `.config/nvim/lazy-lock.json` is versioned for reproducibility.

**`result`:**
- Purpose: Points at the most recent Nix/Darwin build output produced from `flake.nix`.
- Generated: Yes; `darwin-rebuild build` creates the `result` symlink described by `.gitignore`.
- Committed: No; `/result` and `/result-*` are ignored by `.gitignore`.

**`.claude/`:**
- Purpose: Stores local Claude Code worktrees and machine/session state described by `README.md`.
- Generated: Yes; local tooling owns `.claude/`.
- Committed: No; `.claude/` is ignored by `.gitignore`.

**`.hermes/`:**
- Purpose: Stores local Hermes planning and debugging state identified by `.gitignore`.
- Generated: Yes; local tooling owns `.hermes/`.
- Committed: No; `.hermes/` is ignored by `.gitignore`.

**`.planning/codebase/`:**
- Purpose: Stores generated GSD architecture, stack, quality, testing, integration, and concern maps under `.planning/codebase/`.
- Generated: Yes; the GSD mapper produces `.planning/codebase/*.md`.
- Committed: Intended to be version-controlled as the durable GSD reference set after generation.

**Ignored Local App State Under `.config/`:**
- Purpose: Keeps authentication, device state, caches, playback history, generated helpers, and machine-specific links outside the tracked app sources listed by `README.md` and `.gitignore`.
- Generated: Yes; the corresponding applications or local setup create those paths under `.config/`.
- Committed: No; explicit exclusions in `.gitignore` and `.config/.gitignore` keep these paths out of Git. `setup_mac.sh` honors the tracked-only boundary only when Git is available, the repository is a worktree, and the normalized tracked-file query yields at least one top-level app directory; physical-directory fallback can surface ignored or untracked directories whenever any condition fails and requires review.

**Local `AGENTS.md` Links:**
- Purpose: Makes the nearest `CLAUDE.md` guidance discoverable under the standard agent filename in `AGENTS.md`, `nix/AGENTS.md`, `zsh/AGENTS.md`, and selected app directories under `.config/` and `.hammerspoon/`.
- Generated: No; each listed `AGENTS.md` is a deliberate relative symlink to its sibling `CLAUDE.md`.
- Committed: Yes; the guidance symlinks are tracked with the repository.

---

*Structure analysis: 2026-07-10*
