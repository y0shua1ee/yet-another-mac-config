# Codebase Structure

**Analysis Date:** 2026-04-01

## Directory Layout

```
yet-another-mac-config/
├── .config/                      # Application configurations (symlinked to ~/.config)
│   ├── aerospace/                # Window tiling manager
│   ├── borders/                  # JankyBorders window decoration
│   ├── btop/                     # System monitoring tool
│   ├── ghostty/                  # Terminal emulator configuration
│   ├── jgit/                     # JGit (Java Git) config (local only)
│   ├── linearmouse/              # Mouse/trackpad settings (local only)
│   ├── mole/                     # Cleanup tool state (local only)
│   ├── mpv/                      # Media player config
│   ├── nvim/                     # Neovim editor (LazyVim-based)
│   ├── op/                       # 1Password CLI config (local only)
│   ├── opencode/                 # OpenCode & GSD workflow system
│   ├── raycast/                  # Raycast launcher (local only)
│   ├── typora/                   # Typora markdown editor theme
│   ├── yazi/                     # File manager with plugins
│   ├── CLAUDE.md                 # Guidance for config layer modifications
│   └── .gitignore                # Exclude local/cache dirs from tracking
├── .hammerspoon/                 # Hammerspoon automation (symlinked to ~/.hammerspoon)
│   ├── init.lua                  # Main automation: hotkeys, event taps
│   ├── Spoons/                   # Hammerspoon Spoon modules
│   ├── CLAUDE.md                 # Guidance for Hammerspoon changes
│   └── AGENTS.md                 # Symlink to CLAUDE.md
├── .planning/                    # GSD workflow output (not committed)
│   └── codebase/                 # Architecture/structure analysis docs
├── zsh/                          # Shell configuration
│   ├── .zshrc                    # Main Zsh config (symlinked to ~/.zshrc)
│   └── CLAUDE.md                 # Guidance for shell config
├── setup_mac.sh                  # Main initialization script
├── install_yazi_plugins.sh       # Yazi plugin installation helper
├── README.md                     # Project documentation (Chinese)
├── CLAUDE.md                     # Root-level guidance for agents
├── AGENTS.md                     # Symlink to CLAUDE.md
├── .gitignore                    # Git ignore rules
├── .vscode/                      # VS Code project settings
├── .claude/                      # Claude Code project metadata
└── .git/                         # Git repository
```

## Directory Purposes

**`.config/` - Application Configurations**
- Purpose: Central store for app-specific configurations
- Contains: TOML files, Lua configs, JSON settings, theme files
- Key files: Each app subdirectory has its own config format
- Tracked: Only dirs with git-tracked files are symlinked by `setup_mac.sh`
- Local-only: `jgit/`, `linearmouse/`, `mole/`, `op/`, `raycast/` excluded via `.gitignore`

**`.config/aerospace/`**
- Purpose: AeroSpace window manager configuration
- Contains: `aerospace.toml` with layout rules and window detection patterns
- Key files: `aerospace.toml`
- Pattern: Declarative TOML defining floating/tiling layouts based on app ID and window title

**`.config/borders/`**
- Purpose: JankyBorders window border styling
- Contains: `bordersrc` with border styling configuration
- Key files: `bordersrc`

**`.config/btop/`**
- Purpose: btop system monitoring tool configuration
- Contains: `btop.conf` with monitoring preferences
- Key files: `btop.conf`

**`.config/ghostty/`**
- Purpose: Ghostty terminal emulator configuration
- Contains: `config` file with terminal settings, theme, fonts
- Key files: `config` (main), `config.*.bak` (backups, not committed)
- Pattern: Terminal configuration including Catppuccin Mocha theme, Maple Mono font

**`.config/nvim/`**
- Purpose: Neovim editor configuration (LazyVim starter)
- Contains: Lua initialization, lazy plugin manager lock, custom plugins
- Key files: `init.lua`, `lazy-lock.json`, `lua/` subdirectories
- Plugin system: lazy.nvim with locked versions in `lazy-lock.json`

**`.config/yazi/`**
- Purpose: Yazi file manager configuration with plugins
- Contains: Config TOML, keymap, theme, plugin definitions, installed plugins
- Key files: `yazi.toml`, `keymap.toml`, `theme.toml`, `package.toml`, `plugins/`
- Plugin system: Yazi plugin manager with package definitions

**`.config/opencode/`**
- Purpose: OpenCode IDE and GSD (Get Shit Done) workflow system configuration
- Contains: Agent definitions, command specifications, workflow templates, references, GSD tooling
- Structure:
  - `agents/`: Agent definitions for various GSD operations
  - `command/`: Command specifications
  - `get-shit-done/`: Core GSD system (bin, references, templates, workflows, hooks)
  - `cache/`: Local cache (excluded from tracking)
  - `settings.json`: OpenCode settings
- Tracked: Everything except `cache/` directory

**`.hammerspoon/`**
- Purpose: Hammerspoon automation runtime configuration
- Contains: Lua scripts for keyboard bindings, event taps, application automation
- Key files: `init.lua` (main entry point), `Spoons/` (modules)
- Behavior:
  - Cmd double-press W/Q for closing tabs/apps
  - Right Cmd remapped to F19 for other tools
  - Ctrl+Alt+T opens new Ghostty window

**`zsh/`**
- Purpose: Shell environment initialization
- Contains: `.zshrc` main shell config, CLAUDE.md guidance
- Key files: `.zshrc`
- Configuration hierarchy: `.zshrc` (tracked) → loads Oh My Zsh → sources plugins → sources Starship → sources `~/.zshrc.local` (local)
- Private content: Machine-specific vars/secrets go in `~/.zshrc.local` (not tracked)

## Key File Locations

**Entry Points:**

- `setup_mac.sh`: Main initialization script, creates symlinks for tracked configs
- `install_yazi_plugins.sh`: Installs/updates Yazi plugins from `package.toml`
- `.hammerspoon/init.lua`: Hammerspoon automation (symlinked from `~/.hammerspoon/init.lua`)
- `zsh/.zshrc`: Shell configuration (symlinked from `~/.zshrc`)
- `.config/nvim/init.lua`: Neovim bootstrap (symlinked from `~/.config/nvim/init.lua`)

**Configuration:**

- `.config/aerospace/aerospace.toml`: Window layout and app-specific rules
- `.config/ghostty/config`: Terminal emulator settings
- `.config/yazi/yazi.toml`: File manager configuration
- `.config/yazi/package.toml`: Yazi plugin definitions and versions
- `.config/nvim/lazy-lock.json`: Locked plugin versions for Neovim
- `CLAUDE.md`: Root-level agent guidance
- `.config/CLAUDE.md`: Config layer guidance

**Core Logic:**

- `.hammerspoon/init.lua`: Hotkey bindings and event tap logic
- `setup_mac.sh`: Symlink creation logic using `git ls-files` for tracking
- `install_yazi_plugins.sh`: Plugin installation logic with fallback config directories

**Testing:**

- No automated tests; manual testing required for automation changes

## Naming Conventions

**Files:**

- **Configuration files**: App-native names (`aerospace.toml`, `btop.conf`, `yazi.toml`, `config`)
- **Lock files**: Tool-specific (`lazy-lock.json`, `package.toml` for plugin manifests)
- **Setup scripts**: Kebab-case with descriptive names (`setup_mac.sh`, `install_yazi_plugins.sh`)
- **Guidance docs**: `CLAUDE.md` for agent guidance, `AGENTS.md` as symlink (per project convention)
- **Backup files**: Tool-generated names with version indicators (e.g., `config.04c41cb3.bak`)

**Directories:**

- **Config directories**: Match app names exactly (aerospace, ghostty, nvim, yazi)
- **Utility dirs**: Descriptive purpose-based names (`.hammerspoon`, `zsh`)
- **System dirs**: Hidden with dot prefix (`.config`, `.hammerspoon`, `.planning`, `.git`, `.vscode`)
- **Plugin/module dirs**: Match plugin/module names (`Spoons`, `plugins`, `lua`)

## Where to Add New Code

**New Application Configuration:**
1. Create `/.config/<app-name>/` directory
2. Add app-native config files (e.g., `<app-name>.toml`, `config`, `<app-name>.conf`)
3. Add `/.config/<app-name>/CLAUDE.md` for agent guidance
4. Update `README.md` to document the new config
5. `setup_mac.sh` will auto-detect it via `git ls-files` after commit

**New Hammerspoon Automation:**
- Add logic to `/.hammerspoon/init.lua`
- Document hotkeys and behavior in the file comments
- Test manually by reloading with Ctrl+Alt+Cmd+R hotkey
- Update `.hammerspoon/CLAUDE.md` if new patterns are established

**New Shell Configuration:**
- Public (tracked): Add to `zsh/.zshrc`
- Private/local: Instruct user to add to `~/.zshrc.local` in comments
- Environment variables: Use conditional sourcing (`[[ -f "$file" ]] && source "$file"`)

**New Yazi Plugins:**
1. Add plugin entry to `/.config/yazi/package.toml` with desired version
2. Run `install_yazi_plugins.sh` to install
3. Configure plugin in `yazi.toml` and `keymap.toml` as needed

**New Neovim Plugins:**
1. Edit `/.config/nvim/lua/` to add plugin spec (LazyVim format)
2. Run `:Lazy update` in Neovim to install and lock version in `lazy-lock.json`
3. Commit `lazy-lock.json` to track plugin state

**Utility Scripts:**
- Location: Root level (`/`) with `.sh` extension
- Naming: `<action>_<target>.sh` (e.g., `install_yazi_plugins.sh`)
- Comments: Chinese language required per project guidance
- Error handling: Use `set -euo pipefail` for robustness

## Special Directories

**`.planning/codebase/`:**
- Purpose: GSD workflow analysis documents
- Generated: Yes (by gsd-codebase-mapper agent)
- Committed: No (contains temporary planning/research output)
- Contents: ARCHITECTURE.md, STRUCTURE.md, CONVENTIONS.md, TESTING.md, CONCERNS.md

**`.config/opencode/cache/`:**
- Purpose: Runtime cache for OpenCode/GSD system
- Generated: Yes (by gsd tools at runtime)
- Committed: No (explicitly excluded in `.gitignore`)
- Contents: Metadata caches, temporary state files

**`.config/<app>/` (local-only dirs):**
- Purpose: Machine-local configuration that shouldn't be version-controlled
- Dirs: `jgit/`, `linearmouse/`, `mole/`, `op/`, `raycast/`
- Generated: Yes (by apps at runtime)
- Committed: No (excluded in `.config/.gitignore`)
- Pattern: App creates directory on first run; user adds to `.gitignore` if machine-specific

**`~/.config/<app>/` (symlink targets):**
- Purpose: System-wide location for app configs
- Created by: `setup_mac.sh` which symlinks repo `.config/<app>/` → `~/.config/<app>/`
- Behavior: Pointing to repository directory, so changes are immediately visible to all tools

---

*Structure analysis: 2026-04-01*
