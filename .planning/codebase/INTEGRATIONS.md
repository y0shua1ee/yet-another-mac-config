# External Integrations

**Analysis Date:** 2026-04-01

## APIs & External Services

**Code Generation & Image Creation:**
- ray.so - Create beautiful code snippet images
  - SDK/Client: Raycast extension
  - Location: `.config/raycast/extensions/30667be5-d294-4921-a450-034ce776cbc0/`
  - Integration: Visual code-to-image conversion

**1Password Integration:**
- 1Password - Password manager and credential storage
  - SDK/Client: Raycast extension + CLI (`op`)
  - Location: `.config/raycast/extensions/ba9ecf89-7162-4f6a-a417-5087d8d48a98/`
  - Auth: CLI configured at `.config/op/` (local-only, not tracked)
  - Purpose: Credentials and secure information management

**Code Editor Integration:**
- Visual Studio Code - Code editor control and workspace management
  - SDK/Client: Raycast extension
  - Location: `.config/raycast/extensions/95e41a2e-a943-4d49-b0df-152c3db2f7e0/`
  - Purpose: Open projects, recent files, command execution from Raycast

**Git Integration:**
- lazygit - Interactive Git UI
  - Type: External command-line tool
  - Integration: Yazi file manager plugin (`lazygit.yazi`)
  - Config location: `$HOME/Library/Application Support/lazygit/config.yml` (macOS)
  - Environment variable: `LG_CONFIG_FILE` (set by `install_yazi_plugins.sh`)
  - Purpose: Git operations within file manager

**System Utilities Integration:**
- GitHub (implicit) - Repository hosting and version control
  - Used via: Git CLI for syncing dotfiles repository
  - No explicit SDK, command-line only

## Data Storage

**Databases:**
- None detected in application layer

**Configuration Storage:**
- Local file system only - All configurations stored as TOML, JSON, Lua files
  - Neovim: `~/.config/nvim/lazy-lock.json` (plugin lockfile)
  - Yazi: `~/.config/yazi/package.toml` (plugin lockfile)
  - Neovim: `~/.neoconf.json` (project-specific settings)

**File Storage:**
- Local filesystem only - No cloud storage integration detected
- Caches managed locally:
  - Neovim plugin cache: `~/.local/share/nvim/`
  - Yazi plugin cache: Plugin-specific locations
  - OpenCode cache: `.config/opencode/cache/` (local-only, not tracked)

**Caching:**
- Plugin manager caching (lazy.nvim, Yazi package manager)
- Neovim LSP/treesitter caches
- Raycast extension caches (local-only, not tracked)

## Authentication & Identity

**Auth Provider:**
- None centralized; multiple local patterns:
  - 1Password - Local credential storage via CLI
  - Git - SSH keys or credential helper (configured in `~/.zshrc.local`)
  - Application-specific tokens stored in `~/.zshrc.local` (not tracked)

**Implementation:**
- Private environment variables: `~/.zshrc.local` sourced at shell startup
- Hammerspoon: No authentication required (system-level automation)
- Raycast: Extension-specific auth stored in Raycast's local app cache
- 1Password CLI: Device-specific auth stored in `.config/op/` (local-only)

## Monitoring & Observability

**Error Tracking:**
- None detected

**Logs:**
- Hammerspoon: Alert notifications for startup and configuration reload
- System tools: Standard output/error handling via shell scripts
- OpenCode (GSD): Integrated logging to state files and task progress tracking

**Performance Monitoring:**
- btop - System resource visualization (terminal UI, no external service)

## CI/CD & Deployment

**Hosting:**
- GitHub - Remote repository hosting for dotfiles

**CI Pipeline:**
- None detected - Configuration-driven setup only
- Manual execution: `setup_mac.sh` for initial setup
- Plugin updates: Lazy.nvim checks for plugin updates periodically
- Manual update: `ya pkg install` for Yazi plugins

**Deployment Model:**
- Symbolic linking from repository to system config directories
- No automated deployment pipeline
- Updates manual via `git pull` followed by setup script re-execution

## Environment Configuration

**Required env vars (Public - in `.zshrc`):**
- `ZSH` - Oh My Zsh installation path
- `ZSH_THEME` - Zsh theme name (robbyrussell)
- `ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE` - Suggestion styling
- Path adjustments for Homebrew
- Starship shell initialization

**Required env vars (Private - in `~/.zshrc.local`, not tracked):**
- API keys for external services
- Git credentials (if using token-based auth)
- Project-specific variables
- Machine-specific paths

**Secrets Location:**
- `~/.zshrc.local` - Shell-level secrets (not tracked, sourced at end of `.zshrc`)
- `~/.codex/config.toml` - Codex-specific config (local-only, not tracked)
- `.config/op/` - 1Password device info (local-only, not tracked)
- `.config/raycast/` - Extension caches and local data (not tracked)
- Application-specific storage not exposed to Git

## Webhooks & Callbacks

**Incoming:**
- None detected

**Outgoing:**
- None detected

## External Raycast Extensions

**Installed Extensions:**
- Kill Process - Process management by CPU/memory usage
- Visual Studio Code - VS Code workspace and project integration
- ray.so - Code snippet image generation
- 1Password - Password manager integration
- Additional extensions tracked in `.config/raycast/extensions/` (7 total)

**Integration Pattern:**
- Extensions are stored locally in `.config/raycast/`
- Each extension has its own `package.json` following Raycast schema
- Extensions not synced to repository (per `.gitignore` for `.config/raycast/`)
- User can re-install from Raycast store after setup

## OpenCode Framework Integration

**OpenCode (GSD - Get Shit Done):**
- Location: `.config/opencode/`
- Type: Local development workflow automation
- Components:
  - Agents: LLM-powered workflow orchestrators (`.config/opencode/agents/`)
  - Commands: GSD workflow definitions (`.config/opencode/command/`)
  - Workflows: Execution templates (`.config/opencode/workflows/`)
  - Hooks: Pre/post-execution checks (`.config/opencode/hooks/`)
  - Templates: Phase, project, and state templates
  - References: Documentation for features and patterns
- Configuration: `opencode.json` (permissions for file access)
- Cache: `.config/opencode/cache/` (local-only, not tracked)
- Integration: Called manually via `gsd` CLI commands from shell

**No External API Integration:** OpenCode is a local framework; all execution happens locally without external service calls detected.

---

*Integration audit: 2026-04-01*
