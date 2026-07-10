# Technology Stack

**Analysis Date:** 2026-07-10

## Languages

**Primary:**
- Nix expression language (version not pinned) - Defines the Apple Silicon host, flake inputs, nix-darwin system modules, Home Manager user modules, Homebrew inventory, and macOS defaults in `flake.nix`, `nix/darwin/default.nix`, `nix/darwin/defaults.nix`, `nix/darwin/homebrew.nix`, `nix/home/default.nix`, and `nix/modules/zsh.nix`.
- Bash and Zsh (versions supplied by macOS/Homebrew, not pinned) - Drive interactive setup, Yazi plugin synchronization, the fallback shell entrypoint, shared shell behavior, and the JankyBorders launcher in `setup_mac.sh`, `install_yazi_plugins.sh`, `zsh/.zshrc`, `zsh/shared.zsh`, and `.config/borders/bordersrc`.

**Secondary:**
- Lua (hosted runtime versions not pinned) - Configures Neovim/LazyVim, Yazi plugins, and Hammerspoon automation in `.config/nvim/init.lua`, `.config/nvim/lua/config/lazy.lua`, `.config/yazi/init.lua`, `.config/yazi/plugins/`, and `.hammerspoon/init.lua`.
- TOML, YAML, JSON, and application-specific key/value formats - Configure AeroSpace, mise, Yazi, GitHub CLI, Ghostty, mpv, btop, and tmux in `.config/aerospace/aerospace.toml`, `.config/mise/config.toml`, `.config/yazi/`, `.config/gh/config.yml`, `.config/ghostty/config`, `.config/mpv/mpv.conf`, `.config/btop/btop.conf`, and `.config/tmux/tmux.conf.local`.
- GLSL (Ghostty shader dialect; language version not declared) - Supplies the vendored animated terminal effects under `.config/ghostty/shaders/`, with `cursor_blaze.glsl` selected by `.config/ghostty/config`.
- CSS - Implements the custom Typora theme in `.config/typora/themes/spring.css`.
- Markdown - Carries setup, operational constraints, rollback guidance, and component-specific maintenance procedures in `README.md`, `nix/README.md`, `nix/CLAUDE.md`, and the component `CLAUDE.md` files under `.config/`, `.hammerspoon/`, and `zsh/`.

## Runtime

**Environment:**
- Apple Silicon macOS is the sole declared host platform: `aarch64-darwin` with the `AresdeMacBook-Air` flake target in `flake.nix` and `nix/darwin/default.nix`.
- Determinate Nix is the expected Nix runtime; nix-darwin is explicitly prevented from managing the Nix daemon with `nix.enable = false` in `nix/darwin/default.nix`, as documented in `nix/README.md`.
- nix-darwin provides system activation and Home Manager provides the user profile and generated zsh configuration through `flake.nix`, `nix/darwin/default.nix`, and `nix/home/default.nix`.
- `system.stateVersion = 5` and `home.stateVersion = "24.11"` are compatibility state versions, not dependency releases, in `nix/darwin/default.nix` and `nix/home/default.nix`.
- Global fallback toolchains are Node.js `24.11.0` and Go `1.26.3`, installed through mise according to `.config/mise/config.toml` and activated after the private shell override by `nix/modules/zsh.nix`.
- Neovim, Yazi, and Hammerspoon supply their own Lua hosts; no standalone Lua runtime version is declared in `.config/nvim/lua/config/lazy.lua`, `.config/yazi/init.lua`, or `.hammerspoon/init.lua`.

**Package Manager:**
- Nix flakes are the reproducible dependency manager; `flake.lock` is present with lock schema version 7 and pins `nixpkgs`, `nix-darwin`, and `home-manager` by Git revision from `flake.nix`.
- Home Manager installs stable user-level CLI and toolchain entrypoints from nixpkgs in `nix/home/packages.nix` and `nix/home/dev-toolchains.nix`.
- Homebrew is enabled through nix-darwin and acts as the macOS-native formula, cask, font, and service inventory in `nix/darwin/homebrew.nix`; the repository intentionally uses this Nix module instead of a root `Brewfile`.
- mise manages the concrete Node.js and Go fallback versions in `.config/mise/config.toml`; the mise executable itself comes from Home Manager in `nix/home/dev-toolchains.nix`.
- lazy.nvim manages Neovim plugins; Git commits are pinned in `.config/nvim/lazy-lock.json`, while `.config/nvim/lua/config/lazy.lua` bootstraps the stable lazy.nvim branch when it is missing.
- Yazi's `ya pkg` manager installs revision-pinned plugins from `.config/yazi/package.toml`, orchestrated by `install_yazi_plugins.sh`.
- oh-my-tmux is installed as a local Git clone by `setup_mac.sh`, while tmux-resurrect and tmux-continuum are managed by the oh-my-tmux plugin flow declared in `.config/tmux/tmux.conf.local`.
- Lockfiles/manifests are present at `flake.lock`, `.config/nvim/lazy-lock.json`, and `.config/yazi/package.toml`; there is no conventional application-level dependency manifest alongside `flake.nix`.

## Frameworks

**Core:**
- nixpkgs unstable, pinned to revision `b3c092d3c36d91e2f61f3dfb39a159f180a56659`, supplies Nix packages and module evaluation through `flake.lock` and `flake.nix`.
- nix-darwin master, pinned to revision `06648f4902343228ce2de79f291dd5a58ee12146`, applies the macOS system, defaults, Homebrew, and service configuration through `flake.lock`, `flake.nix`, and `nix/darwin/`.
- Home Manager master, pinned to revision `4bfce11ea820df0359f73736fd59c7e8f53641a6`, owns the user profile, zsh, session variables, direnv, and toolchain entrypoints through `flake.lock`, `flake.nix`, and `nix/home/`.
- LazyVim configuration schema 8 with lazy.nvim provides the Neovim distribution and plugin framework in `.config/nvim/lazyvim.json`, `.config/nvim/lua/config/lazy.lua`, and `.config/nvim/lazy-lock.json`.
- Hammerspoon supplies the macOS automation APIs used for event taps, hotkeys, timers, application control, task launch, alerts, and IPC in `.hammerspoon/init.lua`.
- AeroSpace config schema 2 supplies tiling-window rules, workspaces, monitor assignment, and modal keybindings in `.config/aerospace/aerospace.toml`.
- Yazi's Lua plugin framework supplies Git metadata, borders, Starship prompt integration, smart-enter behavior, compression, lazygit, and image zoom in `.config/yazi/init.lua`, `.config/yazi/package.toml`, and `.config/yazi/keymap.toml`.
- oh-my-tmux supplies the base tmux configuration and plugin loader; the tracked override enables tmux-resurrect and tmux-continuum in `.config/tmux/tmux.conf.local`, while `setup_mac.sh` installs the upstream base.

**Testing:**
- No unit-test runner or assertion framework is part of this configuration repository; system-level validation is `nix flake check` followed by `darwin-rebuild build --flake .#AresdeMacBook-Air`, documented in `nix/README.md` and `nix/CLAUDE.md`.
- Neovim uses headless sync and health checks (`Lazy! sync`, `checkhealth lazy`, `checkhealth vim.treesitter`, and a clean headless start) documented in `.config/nvim/README.md` and `.config/nvim/CLAUDE.md`.
- Ghostty configuration validation uses its bundled `+validate-config` command documented in `.config/ghostty/CLAUDE.md` and `.config/ghostty/shaders/README.md`.
- Hammerspoon uses its bundled `hs` IPC client for config-directory checks and reload smoke tests documented in `.hammerspoon/CLAUDE.md`.
- The repository includes gitleaks configuration with narrow false-positive allowlists in `.gitleaks.toml`; `nix/language-stack-plan.md` records gitleaks and `git diff --check` as validation examples. The mandatory root-level pre-commit control is the manual diff review for privacy leaks required by `AGENTS.md`, not an automatically enforced gitleaks gate.

**Build/Dev:**
- `nix flake check` evaluates the locked flake without activation, and `darwin-rebuild build` produces the target system before the privileged switch described in `flake.nix` and `nix/README.md`.
- `darwin-rebuild switch --flake .#AresdeMacBook-Air` is the deployment mechanism for the declared host; first activation is bootstrapped with `nix run github:nix-darwin/nix-darwin/master#darwin-rebuild` as documented in `flake.nix` and `nix/README.md`.
- `setup_mac.sh` is the non-Nix bootstrap path: it enumerates only tracked `.config` directories when Git is available, the repository is a Git worktree, and the normalized `git ls-files` result yields at least one tracked top-level app directory. If any condition fails, including an empty tracked result in a valid worktree, it enumerates physical top-level `.config` subdirectories, which can include ignored or untracked local state. It then creates user-approved symlinks, optionally links zsh/Hammerspoon/Codex config, and installs the local oh-my-tmux clone.
- `install_yazi_plugins.sh` is the application-specific provisioner for revision-pinned Yazi plugins and checks the required `ya` and `git` commands plus optional plugin helpers.
- Git, GitHub CLI, gitleaks, ast-grep, Neovim, tree-sitter-cli, and related workstation tools are declared as Homebrew formulae in `nix/darwin/homebrew.nix`.

## Key Dependencies

**Critical:**
- `nixpkgs`, `nix-darwin`, and `home-manager` are the reproducibility and activation backbone; all three are declared in `flake.nix` and pinned in `flake.lock`.
- Homebrew is required for macOS-native applications, formulae, fonts, and the two managed services; its conservative activation policy disables automatic update, upgrade, and cleanup in `nix/darwin/homebrew.nix`.
- Home Manager's stable CLI layer installs ripgrep, fd, jq, tree, and bat in `nix/home/packages.nix`, while its toolchain layer installs mise, uv, rustup, direnv, and nix-direnv in `nix/home/dev-toolchains.nix`.
- zsh initialization depends on Homebrew shell setup, Starship, Yazi, Bun's user installation, and optional local overrides in `zsh/shared.zsh` and `nix/modules/zsh.nix`; `zsh/.zshrc` preserves a guarded non-Nix fallback.
- Ghostty depends on `font-maple-mono-nf` and the vendored shader collection selected by `.config/ghostty/config`; both Ghostty and the font are declared in `nix/darwin/homebrew.nix`.
- Neovim depends on LazyVim/lazy.nvim, Treesitter, Mason-provisioned tools, and language/tooling extras for TypeScript, JSON, Markdown, Python, Rust, Go, Tailwind CSS, ESLint, and Prettier in `.config/nvim/lua/config/lazy.lua` and `.config/nvim/lazy-lock.json`.
- Yazi plugins depend on `git`, `starship`, `lazygit`, `7zz`, and ImageMagick as checked by `install_yazi_plugins.sh`; preview/open workflows also use mpv, mediainfo, exiftool, poppler, fd, ripgrep, fzf, and zoxide in `.config/yazi/yazi.toml` and `.config/yazi/keymap.toml`.

**Infrastructure:**
- `borders` and `nginx` are the only services managed declaratively with `start_service = true`; Colima, CloudDrive2, and unbound remain manual services according to `nix/darwin/homebrew.nix` and `README.md`.
- Colima, Docker CLI, and Docker Compose form the local container stack, installed by `nix/darwin/homebrew.nix` and operated manually according to `README.md`.
- mpv uses VideoToolbox hardware decoding and delegates URL playback to yt-dlp in `.config/mpv/mpv.conf`; both tools are installed through `nix/darwin/homebrew.nix`.
- Hammerspoon integrates with Ghostty and macOS Accessibility, while AeroSpace and JankyBorders provide window management and visual borders through `.hammerspoon/init.lua`, `.config/aerospace/aerospace.toml`, and `.config/borders/bordersrc`.
- GitHub CLI, Himalaya, memo, remindctl, Claude Code, Codex, CC Switch, and xurl are workstation integrations installed or configured through `.config/gh/config.yml` and `nix/darwin/homebrew.nix`.

## Configuration

**Environment:**
- The flake currently hard-codes one user, one hostname, and one architecture in `flake.nix`; modules receive the user and hostname through `specialArgs` and use them in `nix/darwin/default.nix` and `nix/home/default.nix`.
- Home Manager declares non-sensitive `EDITOR`, `VISUAL`, and `PAGER` variables in `nix/home/shell-env.nix`; `zsh/shared.zsh` adds Homebrew, uv-tool, Bun, Starship, and Yazi behavior.
- Private or machine-specific shell configuration is deliberately sourced from untracked `~/.zshrc.local` by `nix/modules/zsh.nix` and `zsh/.zshrc`; repository policy is documented in `README.md` and `.gitignore`.
- `setup_mac.sh` offers symlinks only for tracked `.config` directories when Git is available, the repository is a Git worktree, and the normalized tracked-file query yields at least one top-level app directory. If any condition fails, its physical-directory fallback scans top-level `.config` subdirectories without applying `.gitignore`, so ignored or untracked local state can become a candidate and must be reviewed at the interactive prompt.
- Yazi plugin installation derives its target from `--config-dir`, `XDG_CONFIG_HOME`, or the tracked `.config/yazi` fallback and exports `YAZI_CONFIG_HOME` plus `LG_CONFIG_FILE` in `install_yazi_plugins.sh`.

**Build:**
- The root build definition and dependency lock are `flake.nix` and `flake.lock`; system modules live in `nix/darwin/`, user modules live in `nix/home/`, and the shared zsh module lives in `nix/modules/zsh.nix`.
- The Homebrew inventory, service policy, casks, and third-party taps are centralized in `nix/darwin/homebrew.nix`; stable macOS defaults are isolated in `nix/darwin/defaults.nix`.
- Application-level dependency locks are `.config/nvim/lazy-lock.json` for Neovim and `.config/yazi/package.toml` for Yazi; the vendored Ghostty shader source revision is recorded in `.config/ghostty/shaders/README.md`.
- Optional secret-scanning rules live in `.gitleaks.toml`, while ignored local-state and credential-bearing path classes are enumerated without values in `.gitignore` and `README.md`; `AGENTS.md` separately requires a manual privacy review of the diff before commit or push.

## Platform Requirements

**Development:**
- Use an Apple Silicon Mac capable of the `aarch64-darwin` target, with Git and Determinate Nix available for the declared path in `flake.nix`, `nix/darwin/default.nix`, and `nix/README.md`.
- Homebrew must be installed separately before the non-Nix setup path; `setup_mac.sh` only creates links and optionally clones oh-my-tmux, as stated in `README.md` and implemented in `setup_mac.sh`.
- The Nix path expects the explicit `.#AresdeMacBook-Air` target and validates with `nix flake check` plus `darwin-rebuild build` before activation according to `nix/README.md`.
- Neovim expects the Homebrew-provided Neovim and tree-sitter CLI plus its Git bootstrap path in `.config/nvim/README.md` and `.config/nvim/lua/config/lazy.lua`.
- Yazi configuration uses v26+ opener placeholders and requires the helpers checked by `install_yazi_plugins.sh`, with the compatibility rule documented in `.config/yazi/CLAUDE.md`.
- Hammerspoon automation requires the application, manual macOS Accessibility permission, and Ghostty for the terminal hotkey according to `.hammerspoon/CLAUDE.md` and `.hammerspoon/init.lua`.

**Production:**
- This repository deploys a personal workstation configuration rather than a hosted service; the declared production target is the local `AresdeMacBook-Air` generation in `flake.nix` and `nix/README.md`.
- Activation writes system and user state through `sudo darwin-rebuild switch --flake .#AresdeMacBook-Air`; rollback uses `sudo darwin-rebuild switch --rollback`, both documented in `nix/README.md`.
- Homebrew activation is intentionally non-destructive (`autoUpdate = false`, `upgrade = false`, `cleanup = "none"`) and starts only `borders` and `nginx` when absent in `nix/darwin/homebrew.nix`.
- The Ghostty Liquid Glass blur option is documented as macOS 26+ in `.config/ghostty/config`; other macOS versions require validating that option with the procedure in `.config/ghostty/CLAUDE.md`.
- Application login state, credentials, Accessibility grants, and other private state remain manual/local and outside the declared generation by policy in `README.md`, `.gitignore`, and `.hammerspoon/CLAUDE.md`.

---

*Stack analysis: 2026-07-10*
