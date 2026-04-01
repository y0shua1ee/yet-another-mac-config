# Technology Stack

**Analysis Date:** 2026-04-01

## Languages

**Primary:**
- Lua - Configuration scripting for Hammerspoon and Neovim (init.lua files)
- TOML - Configuration files for Yazi, AeroSpace, and other applications
- Shell (Bash/Zsh) - Setup and initialization scripts
- JSON - Neovim lock files, configuration metadata
- Markdown - Documentation and configuration guides

**Secondary:**
- JavaScript/TypeScript - Raycast extension configurations and OpenCode framework

## Runtime

**Environment:**
- macOS (specific to Apple Silicon based on reference to `/opt/homebrew/bin/brew` and native apps)
- Zsh shell (primary shell configuration)
- Bash (setup scripts, fallback compatibility)

**Package Manager:**
- Homebrew (`brew`) - Primary macOS package manager
- nanobrew (`nb`) - Lightweight alternative package manager (preferred per project guidelines)
- npm/Node.js - For OpenCode and Raycast extension ecosystem

## Frameworks

**Core Applications:**
- Neovim - Text editor based on LazyVim framework
  - Entry: `~/.config/nvim/init.lua`
  - Plugin manager: lazy.nvim
  - Configuration language: Lua

- Hammerspoon - macOS automation framework
  - Entry: `~/.hammerspoon/init.lua`
  - Hotkey bindings and event tapping automation

- Yazi - Terminal file manager
  - Configuration: TOML-based
  - Plugin system with custom plugins
  - Supports custom themes

**Terminal & Shell:**
- Oh My Zsh - Zsh configuration framework
- Starship - Cross-shell prompt generator
- Ghostty - Terminal emulator (integrated with Hammerspoon)

**Application Integrations:**
- AeroSpace - macOS window manager (TOML config)
- Raycast - macOS command launcher with extensibility
- VS Code - Code editor with project-level settings
- Typora - Markdown editor with custom themes
- Borders (JankyBorders) - Window border decoration

**System Tools:**
- btop - System resource monitor (TOML config)
- Linear Mouse - Mouse/trackpad configuration
- Mole - Cleanup tool
- Codex - Local development configuration tool

## Key Dependencies

**Neovim Ecosystem:**
- lazy.nvim - Plugin manager
- LazyVim - Neovim distribution and plugin collection
- mason.nvim - LSP/tool installer
- mason-lspconfig.nvim - LSP configuration manager
- nvim-lspconfig - LSP configuration framework
- nvim-treesitter - Syntax tree parsing for highlights and text objects
- conform.nvim - Code formatter integration
- nvim-lint - Linter integration
- blink.cmp - Completion engine
- Colorscheme: catppuccin, tokyonight
- UI: bufferline, lualine, noice.nvim, snacks.nvim, nui.nvim
- Utilities: gitsigns.nvim, flash.nvim, grug-far.nvim, todo-comments.nvim, trouble.nvim, which-key.nvim
- Language extras: ESLint, Prettier, TypeScript, JSON, Tailwind CSS

**Yazi File Manager Plugins:**
- smart-enter - Smart directory navigation
- git - Git status integration
- starship - Prompt integration
- compress - Archive compression utilities
- lazygit - Git operations
- full-border - Enhanced border display
- zoom - Image preview scaling with ImageMagick

**System Dependencies (Optional/Conditional):**
- git - Version control (required by Yazi plugins)
- starship - Prompt generation (required for starship.yazi)
- lazygit - Interactive Git UI (required for lazygit.yazi)
- 7zz - Archive extraction (required for compress.yazi)
- magick/ImageMagick - Image processing (required for zoom.yazi)

**Shell Plugins:**
- zsh-syntax-highlighting - Syntax highlighting
- zsh-autosuggestions - Command suggestions
- Git plugin (Oh My Zsh)

## Configuration

**Environment:**
- Configured via symlinked dotfiles from Git repository to system locations
- Public configs tracked in Git: general settings, themes, plugins
- Private configs local only: API keys, project variables in `~/.zshrc.local` (not tracked)
- Environment variables set in Zsh configuration with `export` statements

**Build:**
- No traditional build system (configuration-driven)
- Lazy.nvim handles Neovim plugin installation and lazy-loading
- Yazi package manager (`ya pkg install`) manages plugin versions via `package.toml` lockfile
- Setup script (`setup_mac.sh`) creates symbolic links between repo and system config directories

**Key Configuration Files:**
- `.zshrc` - Shell configuration (symlinked to `~/.zshrc`)
- `.hammerspoon/init.lua` - Hammerspoon automation (symlinked to `~/.hammerspoon/`)
- `.config/nvim/` - Neovim configuration (symlinked to `~/.config/nvim/`)
- `.config/yazi/` - Yazi file manager (symlinked to `~/.config/yazi/`)
- `.config/aerospace/aerospace.toml` - AeroSpace window manager
- `.config/ghostty/` - Ghostty terminal configuration
- `.config/opencode/` - OpenCode framework and GSD workflow definitions

## Platform Requirements

**Development:**
- macOS (latest versions with Apple Silicon or Intel)
- Terminal/command-line access
- Git installed for repository operations
- Homebrew or nanobrew for package installation

**Runtime Requirements:**
- Zsh shell environment
- Neovim (latest recommended)
- Yazi file manager
- Hammerspoon application installed via `nb install --cask hammerspoon`
- Ghostty terminal installed via `nb install --cask ghostty`
- Raycast application (not installed by setup script)
- Other optional tools: starship, lazygit, 7zz, ImageMagick

**System Capabilities:**
- Hotkey binding and event tapping (Hammerspoon)
- Application window manipulation (AeroSpace)
- Terminal emulation and customization
- Configuration file symlink support

---

*Stack analysis: 2026-04-01*
