<!-- refreshed: 2026-07-10 -->
# Architecture

**Analysis Date:** 2026-07-10

## System Overview

```text
┌────────────────────────────────┐   ┌────────────────────────────────┐
│ Declarative Nix plane          │   │ Interactive symlink plane      │
│ `flake.nix`                     │   │ `setup_mac.sh`                 │
│ → `nix/darwin/`, `nix/home/`   │   │ → tracked `.config/` dirs*     │
│ → `nix/modules/zsh.nix`        │   │ → zsh/Hammerspoon/Codex links │
└───────────────┬────────────────┘   └───────────────┬────────────────┘
                │                                    │
                ▼                                    ▼
┌────────────────────────────────┐   ┌────────────────────────────────┐
│ Nix-owned output               │   │ Bootstrap-owned output         │
│ macOS defaults, Homebrew apps, │   │ `~/.config`/`~/.hammerspoon`  │
│ services, Home Manager zsh     │   │ links and fallback `~/.zshrc` │
└───────────────┬────────────────┘   └───────────────┬────────────────┘
                │                                    │
     `nix/modules/zsh.nix` ──────┴────── `zsh/.zshrc` fallback
                             │
                             ▼
                    shared `zsh/shared.zsh`

 Homebrew-installed applications ─┐
 bootstrap-linked app configs ─────┴─→ final application consumption
 (`darwin-rebuild` does not deploy `.config/` or `.hammerspoon/` links)

 * Tracked-only discovery requires Git, a Git worktree, and at least one
   tracked top-level `.config` directory from the normalized `git ls-files`
   result. If any condition fails, discovery falls back to physical top-level
   subdirectories, including ignored/untracked candidates.
```

The repository is a macOS configuration system with two parallel activation planes and different ownership boundaries. The declarative Nix plane rooted at `flake.nix` owns system settings, Homebrew inventory/services, Home Manager packages, and the primary generated zsh configuration; it installs applications but does not deploy `.config/` or `.hammerspoon/` symlinks. The interactive plane rooted at `setup_mac.sh` projects app-native configuration, Hammerspoon, and the fallback zsh entry into a selected home directory. The planes intersect only where both zsh routes consume `zsh/shared.zsh` and operationally where installed applications consume bootstrap-linked configuration.

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Nix composition root | Pins external inputs, selects the single host, installs the mise overlay, and assembles nix-darwin with Home Manager | `flake.nix` |
| Darwin system layer | Sets Apple Silicon/macOS constraints, primary user, and imports system modules | `nix/darwin/default.nix` |
| Homebrew inventory | Declares taps, formulae, casks, and the limited `borders`/`nginx` service activation policy | `nix/darwin/homebrew.nix` |
| macOS defaults | Declares the conservative Finder, Dock, and keyboard preference subset | `nix/darwin/defaults.nix` |
| Home Manager composition | Imports packages, environment, developer-tool entry points, and the zsh module for the configured user | `nix/home/default.nix` |
| Developer-tool entry points | Installs mise, uv, rustup, direnv, and nix-direnv without owning project-specific runtime versions | `nix/home/dev-toolchains.nix` |
| Shell composition | Generates Home Manager zsh startup and embeds the shared public shell fragment | `nix/modules/zsh.nix` |
| Shared shell behavior | Holds Homebrew setup, user tool paths, Starship setup, the Yazi wrapper, Bun setup, and shared aliases | `zsh/shared.zsh` |
| Fallback shell entry | Provides a non-Home-Manager `.zshrc` that consumes the same shared shell fragment | `zsh/.zshrc` |
| Interactive bootstrap | Uses tracked `.config` discovery only when Git is available, the repository is a worktree, and the normalized tracked-file query yields at least one top-level app directory; otherwise falls back to physical subdirectory discovery; creates user-home symlinks and optionally provisions fallback zsh, Hammerspoon, Codex, and oh-my-tmux links | `setup_mac.sh` |
| Application configuration layer | Stores app-native TOML, Lua, CSS, shader, and plain-text configuration consumed directly by each app | `.config/` |
| Yazi package synchronizer | Resolves a Yazi config root and asks `ya` to materialize the revisions locked by `package.toml` | `install_yazi_plugins.sh` |
| macOS automation runtime | Registers Hammerspoon IPC, hotkeys, event taps, timers, and Ghostty activation behavior | `.hammerspoon/init.lua` |
| Version-control boundary | Separates versioned configuration from credentials, login state, caches, local machine state, and generated outputs | `.gitignore` |

## Pattern Overview

**Overall:** Layered configuration repository with dual activation planes and app-native leaf modules (`flake.nix`, `setup_mac.sh`, `.config/`).

**Key Characteristics:**
- Composition happens at explicit roots: Nix imports originate in `flake.nix`, system imports in `nix/darwin/default.nix`, and user imports in `nix/home/default.nix`.
- Cross-route shell behavior is centralized once in `zsh/shared.zsh`, then consumed by both `nix/modules/zsh.nix` and `zsh/.zshrc`.
- Application modules retain their native file formats and directory names under `.config/`; `setup_mac.sh` maps tracked top-level app directories only when Git is available, the repository is a worktree, and the normalized tracked-file query yields at least one app directory. If any condition fails, its fallback enumerates physical top-level directories and can surface ignored or untracked local state.
- Declarative ownership is intentionally partial: `nix/darwin/homebrew.nix` owns selected packages and two Homebrew services, while `.gitignore` and `README.md` define what local state stays out of version control. Those ignore rules do not filter the bootstrap fallback.
- Third-party dependency handling is mixed: Nix inputs are pinned in `flake.lock`, Neovim plugins in `.config/nvim/lazy-lock.json`, and Yazi plugins in `.config/yazi/package.toml`; Ghostty shaders are a vendored snapshot documented in `.config/ghostty/shaders/README.md`; Homebrew formulae/casks and the oh-my-tmux clone in `setup_mac.sh` remain floating sources.

## Layers

**Host Composition Layer:**
- Purpose: Defines one `aarch64-darwin` output, injects `username`/`hostname`, and combines nix-darwin with Home Manager in `flake.nix`.
- Location: `flake.nix`
- Contains: Flake inputs, host identity, the mise package overlay, module wiring, and the default configuration alias in `flake.nix`.
- Depends on: `nixpkgs`, `nix-darwin`, and `home-manager` inputs declared in `flake.nix` and pinned by `flake.lock`.
- Used by: `nix flake check`, `darwin-rebuild build`, and `darwin-rebuild switch` as documented in `nix/README.md`.

**System Configuration Layer:**
- Purpose: Owns host-level macOS policy, selected Homebrew inventory, and a narrow system-defaults set in `nix/darwin/`.
- Location: `nix/darwin/`
- Contains: The system entry `nix/darwin/default.nix`, package/service inventory `nix/darwin/homebrew.nix`, and macOS preferences `nix/darwin/defaults.nix`.
- Depends on: Host arguments supplied by `flake.nix` and nix-darwin modules imported from `nix/darwin/default.nix`.
- Used by: The `darwinConfigurations` assembly in `flake.nix`.

**User Configuration Layer:**
- Purpose: Owns low-risk user packages, public session variables, developer tool managers, and generated zsh configuration in `nix/home/` and `nix/modules/`.
- Location: `nix/home/`, `nix/modules/`
- Contains: Home Manager entry `nix/home/default.nix`, package/environment/tool modules in `nix/home/`, and shell module `nix/modules/zsh.nix`.
- Depends on: `pkgs`, `lib`, and `username` passed through `flake.nix`, plus shared shell text read from `zsh/shared.zsh`.
- Used by: `home-manager.users.${username}` in `flake.nix`.

**Shell Compatibility Layer:**
- Purpose: Keeps one public shell core while supporting a generated Home Manager route and a direct-symlink fallback route in `zsh/`.
- Location: `zsh/`
- Contains: Shared behavior in `zsh/shared.zsh`, fallback bootstrapping in `zsh/.zshrc`, and local maintenance guidance in `zsh/CLAUDE.md`.
- Depends on: Homebrew and optional commands guarded by existence checks in `zsh/shared.zsh`; the primary route additionally depends on mise through `nix/modules/zsh.nix`.
- Used by: `nix/modules/zsh.nix` through `builtins.readFile`, and by `zsh/.zshrc` through `source`.

**Interactive Provisioning Layer:**
- Purpose: Projects repository sources into a selected macOS home directory without taking package-installation ownership in `setup_mac.sh`.
- Location: `setup_mac.sh`, `install_yazi_plugins.sh`
- Contains: Tracked app discovery when Git is available, the repository is a worktree, and the normalized tracked-file query yields at least one app directory; physical-directory fallback when any of those conditions fails; interactive conflict handling, symlink creation, optional oh-my-tmux cloning, and Yazi package synchronization in the two root scripts.
- Depends on: Git and the target filesystem in `setup_mac.sh`; `ya`, Git, and optional plugin helper commands in `install_yazi_plugins.sh`.
- Used by: A user bootstrapping or refreshing a Mac as described in `README.md`.

**App-Native Configuration Layer:**
- Purpose: Keeps each application's source of truth in the exact format and tree that the application expects under `.config/` and `.hammerspoon/`.
- Location: `.config/`, `.hammerspoon/`
- Contains: Standalone leaf configs such as `.config/aerospace/aerospace.toml`, composed runtimes such as `.config/nvim/init.lua`, plugin stacks such as `.config/yazi/`, asset collections such as `.config/ghostty/shaders/`, and automation code in `.hammerspoon/init.lua`.
- Depends on: The corresponding applications and helpers declared partly in `nix/darwin/homebrew.nix` and `nix/home/packages.nix`.
- Used by: Applications through user-home symlinks created by `setup_mac.sh`, except Hammerspoon which uses the separate `.hammerspoon/` target handled by the same script.

**Local-State Boundary:**
- Purpose: Defines which generated output, credentials, login state, caches, machine-specific links, and private shell data Git excludes through `.gitignore`; it protects version-control state but does not filter physical-directory fallback discovery in `setup_mac.sh`.
- Location: `.gitignore`, `.config/.gitignore`
- Contains: Ignore rules for local app directories, authentication state, media history, tmux runtime plugins, backups, Nix `result` links, and agent-local state in `.gitignore`.
- Depends on: Git ignore evaluation for staging and tracked-directory discovery via `git ls-files` when Git is available, `setup_mac.sh` runs inside a Git worktree, and the normalized query yields at least one top-level app directory.
- Used by: Version-control workflows and the tracked bootstrap discovery path; physical-directory fallback bypasses this boundary whenever any tracked-discovery condition fails and requires interactive review of every candidate directory.

## Data Flow

### Primary Request Path

1. A build or switch selects `darwinConfigurations.AresdeMacBook-Air` from the flake output (`flake.nix:24`, `flake.nix:42`).
2. The composition root injects host arguments and evaluates the overlay, `nix/darwin/`, and the Home Manager Darwin module (`flake.nix:42`, `flake.nix:45`).
3. The system entry imports Homebrew and macOS defaults (`nix/darwin/default.nix:10`), while the user entry imports packages, environment, tool managers, and zsh (`nix/home/default.nix:8`).
4. The zsh module embeds the shared fragment, then appends the private local overlay and mise activation in a fixed order (`nix/modules/zsh.nix:24`, `nix/modules/zsh.nix:35`).
5. nix-darwin produces or activates system defaults, Homebrew operations, user packages, and generated Home Manager files; file conflicts use the configured backup suffix (`flake.nix:61`, `flake.nix:64`).

### Interactive Symlink Flow

1. The bootstrap resolves its own repository root and asks for a target macOS username (`setup_mac.sh:8`, `setup_mac.sh:34`).
2. It derives app names only from tracked paths under `.config/` when Git is available, the repository is a worktree, and the normalized `git ls-files` result contains at least one top-level app directory. If any condition fails, including an empty tracked result in a valid worktree, it scans physical top-level `.config` subdirectories, so ignored or untracked local state can become a candidate (`setup_mac.sh:17`, `setup_mac.sh:30`).
3. For each selected app, it handles an existing destination interactively and links the repository directory into the target `.config` tree (`setup_mac.sh:51`, `setup_mac.sh:76`).
4. It separately offers local Codex config, the fallback zsh entry, and Hammerspoon because those targets do not follow the normal `.config/<app>` shape (`setup_mac.sh:81`, `setup_mac.sh:109`, `setup_mac.sh:135`).
5. If tmux lacks its upstream base config, it clones oh-my-tmux into local application data and links that base beside the tracked override (`setup_mac.sh:161`, `setup_mac.sh:174`).

### Application Runtime Flow

1. Neovim enters through `.config/nvim/init.lua`, bootstraps `lazy.nvim`, loads LazyVim extras, then imports local specs from `.config/nvim/lua/config/lazy.lua`.
2. Yazi reads app-native TOML from `.config/yazi/yazi.toml`, then `.config/yazi/init.lua` initializes plugin modules locked in `.config/yazi/package.toml`.
3. Ghostty reads `.config/ghostty/config`, which points to a vendored shader in `.config/ghostty/shaders/`.
4. Hammerspoon evaluates `.hammerspoon/init.lua`, creates persistent event taps/timers, and exposes IPC through `hs.ipc` in `.hammerspoon/init.lua`.

### Yazi Plugin Synchronization

1. The helper selects an explicit config directory, an existing XDG target, or the repository source in that order (`install_yazi_plugins.sh:39`, `install_yazi_plugins.sh:43`).
2. It validates `.config/yazi/package.toml` and mandatory commands before exporting `YAZI_CONFIG_HOME` (`install_yazi_plugins.sh:57`, `install_yazi_plugins.sh:63`, `install_yazi_plugins.sh:96`).
3. `ya pkg install` materializes the locked revisions and `ya pkg list` reports the result (`install_yazi_plugins.sh:113`, `install_yazi_plugins.sh:116`).

**State Management:**
- Versioned desired state lives in `flake.nix`, `nix/`, `zsh/`, `.config/`, and `.hammerspoon/`; active state is materialized by Nix generations or filesystem symlinks from `setup_mac.sh`.
- Shared mutable shell state is limited to environment variables and shell functions in `zsh/shared.zsh`; private overrides enter through the unversioned local file sourced by `nix/modules/zsh.nix` and `zsh/.zshrc`.
- Hammerspoon runtime state is module-local Lua variables, timers, and event taps created in `.hammerspoon/init.lua`; it is rebuilt when the configuration reloads.
- Generated and personal application state stays outside the desired-state graph through rules in `.gitignore`; the Nix build result is represented only by the ignored `result` symlink documented there.

## Key Abstractions

**Nix Module Composition:**
- Purpose: Splits system, user, package, environment, and shell concerns into independently evaluable modules.
- Examples: `nix/darwin/default.nix`, `nix/home/default.nix`, `nix/modules/zsh.nix`
- Pattern: Explicit `imports` lists assembled by the host root in `flake.nix`.

**Tracked App Directory:**
- Purpose: Makes a top-level `.config/<app>` directory the versioned source of truth and the unit projected into a user's home.
- Examples: `.config/aerospace/`, `.config/ghostty/`, `.config/nvim/`, `.config/yazi/`
- Pattern: App-native files plus optional local `CLAUDE.md`/`AGENTS.md`; `setup_mac.sh` derives discovery from Git paths rather than a duplicated manifest only when all three tracked-discovery conditions hold, and otherwise uses physical-directory fallback.

**Shared Shell Fragment:**
- Purpose: Prevents behavior drift between the primary generated zsh and fallback symlinked zsh.
- Examples: `zsh/shared.zsh`, `nix/modules/zsh.nix`, `zsh/.zshrc`
- Pattern: One shell fragment embedded by Nix and sourced by the fallback entry, with route-specific ordering kept in each caller.

**Pinned or Vendored Extension Collection:**
- Purpose: Keeps application extensions reproducible even though each host application owns loading.
- Examples: `.config/yazi/package.toml`, `.config/yazi/plugins/`, `.config/ghostty/shaders/`
- Pattern: Revision/hash metadata for Yazi and committed asset snapshots for Ghostty.

**Private Local Overlay:**
- Purpose: Lets machine-specific or sensitive behavior override public defaults without entering Git.
- Examples: the source points in `nix/modules/zsh.nix` and `zsh/.zshrc`, with exclusion policy in `.gitignore`
- Pattern: Optional file or ignored directory loaded after public configuration.

## Entry Points

**Nix Host Entry:**
- Location: `flake.nix`
- Triggers: `nix flake check`, `darwin-rebuild build`, or `darwin-rebuild switch` documented in `nix/README.md`.
- Responsibilities: Selects host identity, composes modules, and exposes both named and default Darwin configurations in `flake.nix`.

**Interactive Mac Bootstrap:**
- Location: `setup_mac.sh`
- Triggers: Direct execution by a user following `README.md`.
- Responsibilities: Projects user-selected config sources into a target home, using tracked-only app discovery only when all three discovery conditions hold and preserving conflicts unless the user explicitly approves replacement in `setup_mac.sh`.

**Yazi Extension Bootstrap:**
- Location: `install_yazi_plugins.sh`
- Triggers: Direct execution, optionally with `--config-dir`, as documented in `README.md`.
- Responsibilities: Validates prerequisites and synchronizes revisions declared in `.config/yazi/package.toml`.

**Generated Shell Entry:**
- Location: `nix/modules/zsh.nix`
- Triggers: Home Manager evaluation from `nix/home/default.nix`.
- Responsibilities: Enables native shell features, establishes ordering, embeds `zsh/shared.zsh`, and activates mise.

**Fallback Shell Entry:**
- Location: `zsh/.zshrc`
- Triggers: A user choosing the fallback link in `setup_mac.sh`.
- Responsibilities: Initializes completion and Homebrew-provided shell plugins around the shared core in `zsh/shared.zsh`.

**Application Entries:**
- Location: `.config/nvim/init.lua`, `.config/yazi/init.lua`, `.hammerspoon/init.lua`, `.config/aerospace/aerospace.toml`, `.config/ghostty/config`
- Triggers: Launch or reload of the corresponding application through its standard configuration path.
- Responsibilities: Delegate into local module/plugin trees or directly declare the application's behavior at the listed paths.

## Architectural Constraints

- **Threading:** Provisioning is sequential shell execution in `setup_mac.sh` and `install_yazi_plugins.sh`. Long-lived app-managed behavior is event-driven: AeroSpace starts at login and handles window-detection callbacks from `.config/aerospace/aerospace.toml`, while Hammerspoon maintains event-loop timers and event taps in `.hammerspoon/init.lua`.
- **Global state:** Host identity is defined once per flake evaluation in `flake.nix`; shell globals and PATH mutations live in `zsh/shared.zsh`; Hammerspoon's key-state flags and timer references are module globals in `.hammerspoon/init.lua`.
- **Circular imports:** No circular Nix or Lua composition chain is present across `flake.nix`, `nix/darwin/default.nix`, `nix/home/default.nix`, `.config/nvim/init.lua`, and `.config/yazi/init.lua`.
- **Platform:** The declared Nix host is fixed to Apple Silicon macOS in `flake.nix` and `nix/darwin/default.nix`; several leaf configs also assume macOS paths or APIs in `.hammerspoon/init.lua`, `.config/mpv/mpv.conf`, and `zsh/shared.zsh`.
- **Single-host identity:** `username`, `hostname`, and `aarch64-darwin` are hard-coded at the composition root in `flake.nix`; adding a second host requires a host inventory or another explicit Darwin output rather than editing leaf modules.
- **Dual zsh ownership:** Home Manager generates the primary shell through `nix/modules/zsh.nix`, while `zsh/.zshrc` remains a fallback offered by `setup_mac.sh`; only one route should own the live target at a time.
- **Partial declaration:** Nix intentionally owns only the bounded package/service/default sets in `nix/darwin/homebrew.nix` and `nix/darwin/defaults.nix`; app-config deployment remains the responsibility of `setup_mac.sh`, and Git ignore rules exclude local state from version control without constraining the script's physical-directory fallback.
- **Private-data boundary:** Sensitive and machine-specific values must not enter `nix/home/shell-env.nix`, `zsh/shared.zsh`, or tracked `.config/` files; `.gitignore` defines local-only locations and both shell routes load a private overlay outside version control. Bootstrap candidates from physical-directory fallback must be reviewed because ignore rules are not applied there.

## Anti-Patterns

### Competing Writers for the Live Zsh Target

**What happens:** `setup_mac.sh` can replace the live `.zshrc` with a symlink to `zsh/.zshrc`, while `flake.nix` configures Home Manager to generate and back up that same target from `nix/modules/zsh.nix`.
**Why it's wrong:** Activating both routes makes ownership depend on the last command run, obscures which initialization order is active, and can create backup/symlink churn across `setup_mac.sh` and `flake.nix`.
**Do this instead:** Treat `nix/modules/zsh.nix` as the primary route and use `zsh/.zshrc` only as the documented rollback/fallback; put behavior shared by both in `zsh/shared.zsh`.

### Repeated Destructive Link-Replacement Blocks

**What happens:** The `.config`, Codex, zsh, and Hammerspoon branches independently repeat existence checks, interactive replacement prompts, `rm -rf`, and `ln -s` operations in `setup_mac.sh`.
**Why it's wrong:** A safety or backup improvement must be reproduced across several branches in `setup_mac.sh`, and small divergence can give different targets different replacement semantics.
**Do this instead:** When changing bootstrap behavior, centralize target replacement in one helper inside `setup_mac.sh`, preserving the existing opt-in prompts and the three-condition tracked-discovery versus physical-fallback boundary.

## Error Handling

**Strategy:** Fail early for invalid prerequisites and evaluation errors, ask before destructive filesystem replacement, and rely on application-specific guards/validation at the leaf boundaries (`setup_mac.sh`, `install_yazi_plugins.sh`, `flake.nix`).

**Patterns:**
- Both provisioning scripts enable strict shell failure behavior with `set -euo pipefail` and perform explicit directory/command checks before mutation in `setup_mac.sh` and `install_yazi_plugins.sh`.
- Existing destinations are skipped by default and removed only after affirmative input in `setup_mac.sh`; Nix-managed collisions receive a backup suffix configured in `flake.nix`.
- Neovim reports lazy.nvim clone failure, waits for acknowledgement, and exits non-zero in `.config/nvim/lua/config/lazy.lua`.
- Optional shell tools are protected with executable or `command -v` guards in `zsh/shared.zsh` and `nix/modules/zsh.nix`.
- Hammerspoon callbacks generally return without mutation when conditions are not met, and its injected keystroke path stops/restarts the source event tap to prevent recursion in `.hammerspoon/init.lua`.

## Cross-Cutting Concerns

**Logging:** Bootstrap status and errors use standard output/error in `setup_mac.sh` and `install_yazi_plugins.sh`; app automation intentionally keeps runtime noise low in `.hammerspoon/init.lua`, and no repository-wide logging service is present.

**Validation:** Nix evaluation/build checks are documented in `nix/README.md` and `nix/CLAUDE.md`; app-specific validation lives beside complex config in `.config/ghostty/CLAUDE.md`, `.config/nvim/CLAUDE.md`, `.config/aerospace/CLAUDE.md`, and `.hammerspoon/CLAUDE.md`.

**Authentication:** The versioned architecture implements no authentication layer; login and credential-bearing state is excluded by `.gitignore`, while tracked shared preferences such as `.config/gh/config.yml` remain separate from ignored authentication state.

---

*Architecture analysis: 2026-07-10*
