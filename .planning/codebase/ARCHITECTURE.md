# Architecture

**Analysis Date:** 2026-04-01

## Pattern Overview

**Overall:** Configuration Management Repository with Symlink Distribution

This is a centralized Mac configuration repository that uses symbolic links to distribute versioned configurations to system locations. The architecture emphasizes:
- **Single source of truth**: Git-tracked configurations in `.config/` directories
- **Non-invasive linking**: Setup scripts create symlinks rather than copying files
- **Privacy-aware structure**: Sensitive/local data excluded via `.gitignore`
- **Multi-tool integration**: Supports application-specific configurations (aerospace, nvim, yazi, ghostty, etc.) plus scripted automation (Hammerspoon, Zsh)

**Key Characteristics:**
- Declarative configuration format (TOML, Lua, JSON, shell)
- Script-driven initialization and synchronization
- Symlink-based deployment (no file copies)
- Layered configuration with public (tracked) and private (local) separation

## Layers

**Repository Root Layer:**
- Purpose: Coordination and initialization entry points
- Location: `/Users/areslee/Documents/yet-another-mac-config/`
- Contains: Setup scripts, installation instructions, project metadata
- Depends on: Bash, Git (for tracking)
- Used by: End user during initial setup

**Configuration Distribution Layer:**
- Purpose: Store versioned application configurations
- Location: `.config/` subdirectories
- Contains: TOML files (aerospace, btop, yazi), Lua configs (nvim, hammerspoon), JSON configs (op, linearmouse, jgit), shell scripts
- Depends on: Git tracking (`git ls-files` for determining what to link)
- Used by: Symlink bootstrap process in `setup_mac.sh`

**Automation Layer:**
- Purpose: Runtime behavior and system integration
- Location: `.hammerspoon/init.lua` and `zsh/.zshrc`
- Contains: Event taps, keyboard bindings, shell initialization
- Depends on: Hammerspoon runtime, Zsh shell, external tools (Ghostty, AeroSpace, etc.)
- Used by: macOS at login and during shell sessions

**Terminal/Shell Layer:**
- Purpose: Shell environment initialization and prompt configuration
- Location: `zsh/.zshrc`
- Contains: Oh My Zsh setup, theme configuration, plugin sourcing, tool initialization (Starship)
- Depends on: Oh My Zsh framework, Brew completions, local `.zshrc.local` (not tracked)
- Used by: Terminal on each new Zsh session

**Workspace Management Layer:**
- Purpose: Editor and file manager configurations
- Location: `.config/nvim/`, `.config/yazi/`, `.config/ghostty/`
- Contains: Neovim plugins (via LazyVim), Yazi layout/keymaps/plugins, terminal theme
- Depends on: External package managers (lazy.nvim, Yazi plugin system)
- Used by: Development workflow in editors and file navigation

**Window Management Layer:**
- Purpose: macOS window tiling and automation
- Location: `.config/aerospace/` and `.hammerspoon/init.lua`
- Contains: AeroSpace window layout rules, Hammerspoon hotkey bindings and event taps
- Depends on: AeroSpace daemon, Hammerspoon runtime
- Used by: System-wide window behavior

**Development Tools Layer:**
- Purpose: Specialized tool configurations
- Location: `.config/btop/`, `.config/borders/`, `.config/op/`, `.config/raycast/`, etc.
- Contains: System monitoring, tool-specific settings, API client configs
- Depends on: Individual tool CLIs and runtimes
- Used by: Specific development utilities and system apps

## Data Flow

**Initial Setup Flow:**

1. User runs `setup_mac.sh`
2. Script reads target macOS username
3. Script enumerates Git-tracked `.config/*/` directories via `git ls-files`
4. For each directory, prompt user for confirmation
5. Create symlink: `/Users/<username>/.config/<app>` → repository `.config/<app>/`
6. Special handling for `.zshrc`, Hammerspoon, Codex configs
7. User manually runs `install_yazi_plugins.sh` to sync Yazi plugins

**Configuration Load Flow:**

- **Shells**: `~/.zshrc` (symlink) → sources Oh My Zsh → sources Starship → sources `~/.zshrc.local` (local, untracked)
- **Editors**: Neovim reads `~/.config/nvim` (symlink) → loads LazyVim → installs plugins from `lazy-lock.json`
- **File Manager**: Yazi reads `~/.config/yazi` (symlink) → loads `yazi.toml` config → installs plugins from `package.toml`
- **Window Manager**: AeroSpace reads `~/.config/aerospace` (symlink) → parses `aerospace.toml` rules
- **Automation**: Hammerspoon reads `~/.hammerspoon` (symlink) → executes `init.lua` → starts event taps at login

**State Management:**

- **Tracked state**: Configuration files in `.config/*/`, shell scripts, lock files (`lazy-lock.json`, plugin metadata)
- **Local state** (not tracked, per `.gitignore`): `~/.zshrc.local`, `.config/op/`, `.config/linearmouse/`, `.config/raycast/`, `.config/jgit/`, `.config/opencode/cache/`
- **Generated state**: `~/.config/` symlinks after `setup_mac.sh` runs, Neovim plugin installations

## Key Abstractions

**Configuration Container:**
- Purpose: Self-contained app configuration directory
- Examples: `/.config/nvim/`, `/.config/yazi/`, `/.config/aerospace/`
- Pattern: Each app has its own subdirectory with app-native config format + optional `CLAUDE.md` for agent guidance

**Setup Script Abstraction:**
- Purpose: Automate symlink creation with user consent
- Examples: `setup_mac.sh`, `install_yazi_plugins.sh`
- Pattern: Interactive Bash scripts that honor `.gitignore` and handle existence checks

**CLAUDE.md/AGENTS.md Files:**
- Purpose: Provide AI agent guidance for modifications to each section
- Examples: `/.config/nvim/CLAUDE.md`, `/.config/yazi/CLAUDE.md`, `/.hammerspoon/CLAUDE.md`
- Pattern: Guidance files (with AGENTS.md as symlink to CLAUDE.md per project convention)

**Public/Private Split:**
- Purpose: Separate versioned configurations from machine-local secrets
- Example: `~/.zshrc` (symlink, tracked) sources `~/.zshrc.local` (local, untracked) at end
- Pattern: Applications explicitly source local overrides; `.gitignore` prevents accidental commits

## Entry Points

**Setup Initialization:**
- Location: `/Users/areslee/Documents/yet-another-mac-config/setup_mac.sh`
- Triggers: Manual execution by user during new machine setup
- Responsibilities: Read username, enumerate tracked configs, create symlinks, handle special cases (zshrc, Hammerspoon, Codex)

**Shell Entry:**
- Location: `/.config/zsh/.zshrc` → symlinked to `~/.zshrc`
- Triggers: Every new Zsh shell session
- Responsibilities: Load Oh My Zsh, configure theme, enable plugins, initialize Starship prompt, source local config

**Automation Entry:**
- Location: `/.hammerspoon/init.lua` → symlinked to `~/.hammerspoon/init.lua`
- Triggers: Hammerspoon starts at macOS login
- Responsibilities: Set up event taps for keyboard bindings (Cmd+W/Q double-press, Ctrl+Alt+T), remap right Cmd to F19, inject synthetic keystrokes

**Yazi Plugin Setup:**
- Location: `/Users/areslee/Documents/yet-another-mac-config/install_yazi_plugins.sh`
- Triggers: Manual execution after initial setup
- Responsibilities: Locate Yazi config, install plugins from `package.toml`, verify dependencies, configure environment variables

**Editor Entry:**
- Location: `/.config/nvim/init.lua` → symlinked to `~/.config/nvim/init.lua`
- Triggers: When Neovim is launched
- Responsibilities: Bootstrap LazyVim, load user configuration from `lua/` directory, install plugins per `lazy-lock.json`

## Error Handling

**Strategy:** Explicit failure modes with user guidance

**Patterns:**
- Script exit on error: `set -euo pipefail` in Bash scripts prevents silent failures
- User confirmation before destructive operations: Setup scripts prompt before overwriting existing symlinks
- Conditional sourcing: `.zshrc` uses `[[ -f "$file" ]] && source "$file"` to safely load optional local configs
- Guard clauses in automation: Hammerspoon checks if applications exist before activating them (e.g., `hs.application.get("Ghostty")`)

## Cross-Cutting Concerns

**Logging:**
- Bash scripts: Echo status messages to stdout; `setup_mac.sh` logs symlink creation/skips
- Hammerspoon: Minimal logging; only shows alert on startup ("Hammerspoon: Performance Version Loaded!")
- Approach: Prefer explicit user feedback over silent operation; avoid excessive logging to reduce system noise

**Validation:**
- Directory existence checks: `[[ ! -d "$path" ]] && exit 1` in setup scripts
- Git tracking validation: `git ls-files` ensures only tracked configs are linked
- File existence checks before sourcing: Shell sources use `[[ -f "$file" ]]` guards
- App existence checks in automation: Hammerspoon queries `hs.application.get()` before interacting

**Authentication:**
- Not directly handled in this codebase; delegated to application-level configs (1Password CLI, etc.)
- Local secrets stored in machine-untracked files (`.zshrc.local`, `.config/op/`, etc.)

**Configuration Priority:**
- Tracked version (repository) is foundation
- Local overrides loaded after tracked setup (e.g., `~/.zshrc` sources `~/.zshrc.local`)
- User explicit choices honored during setup (symlink prompts in `setup_mac.sh`)

---

*Architecture analysis: 2026-04-01*
