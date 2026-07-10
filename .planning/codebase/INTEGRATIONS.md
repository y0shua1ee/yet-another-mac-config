# External Integrations

**Analysis Date:** 2026-07-10

## APIs & External Services

**Source and package delivery:**
- GitHub hosts all three flake inputs: nixpkgs, nix-darwin, and Home Manager are fetched through Nix's GitHub flake transport from the URLs in `flake.nix` and immutable revisions in `flake.lock`.
  - SDK/Client: Nix flake fetcher invoked by `nix flake` and `darwin-rebuild` from `flake.nix` and `nix/README.md`.
  - Auth: No repository auth variable is declared; the tracked inputs are public GitHub repositories in `flake.nix`.
- GitHub hosts the Neovim plugin ecosystem: lazy.nvim is cloned with Git when absent, and LazyVim plus its dependency graph are pinned by commit in `.config/nvim/lua/config/lazy.lua` and `.config/nvim/lazy-lock.json`.
  - SDK/Client: `git clone` followed by lazy.nvim's plugin manager in `.config/nvim/lua/config/lazy.lua`.
  - Auth: No credential variable is declared for these public repositories in `.config/nvim/lua/config/lazy.lua`.
- GitHub-hosted Yazi plugins are revision-pinned and installed through `ya pkg install` from `.config/yazi/package.toml` and `install_yazi_plugins.sh`.
  - SDK/Client: Yazi `ya pkg` plus Git, both required by `install_yazi_plugins.sh`.
  - Auth: No credential variable is declared for the public plugin sources in `.config/yazi/package.toml` or `install_yazi_plugins.sh`.
- oh-my-tmux is cloned from GitHub, and its plugin loader installs tmux-resurrect and tmux-continuum from GitHub namespaces in `setup_mac.sh` and `.config/tmux/tmux.conf.local`.
  - SDK/Client: Git plus the oh-my-tmux/TPM-compatible plugin flow in `setup_mac.sh` and `.config/tmux/tmux.conf.local`.
  - Auth: No credential variable is declared for these public repositories in `setup_mac.sh` or `.config/tmux/tmux.conf.local`.
- The Ghostty shader collection is vendored from the public `0xhckr/ghostty-shaders` repository at source commit `aa6121ba2ddd5251ac75b92729c758fe41256e55` in `.config/ghostty/shaders/README.md`.
  - SDK/Client: Manual vendoring of top-level GLSL files, not a runtime SDK, according to `.config/ghostty/shaders/README.md` and `.config/ghostty/CLAUDE.md`.
  - Auth: Not applicable for the public source recorded in `.config/ghostty/shaders/README.md`.
- Homebrew and five third-party taps deliver macOS-native packages: `nikitabobko/tap`, `felixkratz/formulae`, `antoniorodr/memo`, `steipete/tap`, and `xdevplatform/tap` in `nix/darwin/homebrew.nix`.
  - SDK/Client: nix-darwin's Homebrew module and Homebrew Bundle activation generated from `nix/darwin/homebrew.nix`.
  - Auth: No Homebrew credential variable is declared in `nix/darwin/homebrew.nix`; taps are public package sources.

**Developer and account-backed services:**
- GitHub CLI is configured to use HTTPS for Git operations and provides a `pr checkout` alias in `.config/gh/config.yml`.
  - SDK/Client: Homebrew `gh` formula from `nix/darwin/homebrew.nix` with preferences from `.config/gh/config.yml`.
  - Auth: GitHub CLI login state is deliberately stored only in ignored `.config/gh/hosts.yml`, as documented without values in `.gitignore` and `README.md`.
- X API access is available through the `xurl` cask in `nix/darwin/homebrew.nix`.
  - SDK/Client: Homebrew-installed `xurl` CLI declared in `nix/darwin/homebrew.nix`.
  - Auth: Credentials remain in local `~/.xurl` and are not tracked, as explicitly documented in `nix/darwin/homebrew.nix`.
- Apple Notes and Reminders are exposed through the `memo` and `remindctl` Homebrew formulae in `nix/darwin/homebrew.nix`.
  - SDK/Client: `antoniorodr/memo/memo` and `steipete/tap/remindctl` command-line clients in `nix/darwin/homebrew.nix`.
  - Auth: No repository credential is declared; access depends on the local macOS user/account state described by the workstation scope in `README.md`.
- Terminal email is available through Himalaya in `nix/darwin/homebrew.nix`.
  - SDK/Client: Homebrew `himalaya` formula declared in `nix/darwin/homebrew.nix`.
  - Auth: Account configuration is kept in the ignored local `.config/himalaya/` directory and never synchronized, as documented in `.gitignore` and `README.md`.
- Claude Code, Codex, and CC Switch are provisioned as Homebrew casks for local AI-assisted development in `nix/darwin/homebrew.nix`.
  - SDK/Client: Their installed command-line applications; this repository contains no API SDK integration in `nix/darwin/homebrew.nix` or the tracked application configs.
  - Auth: Provider login state and tokens are intentionally outside this repository according to the local-state policy in `README.md` and `.gitignore`.

**macOS and application services:**
- Hammerspoon integrates with macOS event taps, global hotkeys, timers, Accessibility, application lookup, task launch, alerts, and its IPC message port in `.hammerspoon/init.lua`.
  - SDK/Client: Hammerspoon's `hs.*` Lua APIs, including `hs.ipc`, `hs.eventtap`, `hs.hotkey`, `hs.timer`, `hs.application`, and `hs.task`, in `.hammerspoon/init.lua`.
  - Auth: Manual macOS Accessibility permission is mandatory and cannot be declared by Nix, as documented in `.hammerspoon/CLAUDE.md` and `nix/README.md`.
- Hammerspoon launches or activates Ghostty for the `Ctrl+Alt+T` hotkey in `.hammerspoon/init.lua`.
  - SDK/Client: `hs.application.get`, `/usr/bin/open -na Ghostty`, and synthetic Cmd+N input in `.hammerspoon/init.lua`.
  - Auth: No application credential; Ghostty must be installed from `nix/darwin/homebrew.nix` and Hammerspoon must have Accessibility permission from `.hammerspoon/CLAUDE.md`.
- AeroSpace integrates with macOS application/window metadata to float matched settings windows, communication apps, AI clients, media players, and utilities in `.config/aerospace/aerospace.toml`.
  - SDK/Client: AeroSpace's `[[on-window-detected]]` rules and command callbacks in `.config/aerospace/aerospace.toml`.
  - Auth: No credential; app identifiers must be obtained through AeroSpace inspection commands documented in `.config/aerospace/CLAUDE.md`.
- mpv delegates remote URL extraction to yt-dlp and uses macOS VideoToolbox for hardware decoding in `.config/mpv/mpv.conf`.
  - SDK/Client: mpv's built-in ytdl hook plus the Homebrew `yt-dlp` executable declared in `nix/darwin/homebrew.nix`.
  - Auth: No media-service credential is declared in `.config/mpv/mpv.conf`; authenticated media sessions, if any, are outside the repository.
- Colima provides a local Linux VM behind Docker CLI and Docker Compose according to `README.md` and `nix/darwin/homebrew.nix`.
  - SDK/Client: Homebrew `colima`, `docker`, and `docker-compose` formulae in `nix/darwin/homebrew.nix`.
  - Auth: Container registry authentication belongs to local `~/.docker/config.json`, which is explicitly treated as potentially sensitive and excluded from repository management in `README.md`.

## Data Storage

**Databases:**
- Not detected; the tracked repository is declarative workstation configuration and contains no database schema, connection client, or database-backed service definition in `flake.nix`, `nix/`, `setup_mac.sh`, or `.config/`.
  - Connection: Not applicable; no database environment variable or connection file is declared in `nix/home/shell-env.nix`, `zsh/shared.zsh`, or the tracked app configs.
  - Client: Not applicable; dependencies in `nix/darwin/homebrew.nix`, `nix/home/packages.nix`, and `.config/nvim/lazy-lock.json` are workstation tools rather than database clients used by repository code.

**File Storage:**
- Local filesystem only: `setup_mac.sh` discovers user-selectable apps from Git-tracked `.config/<app>` paths only when Git is available, the repository is a Git worktree, and the normalized `git ls-files` result yields at least one tracked top-level app directory. If any condition fails, including an empty tracked result in a valid worktree, it falls back to physical top-level `.config` directory discovery, which can include ignored or untracked local state. It then links selected app directories into the target user's `~/.config` and can optionally link zsh, Hammerspoon, and local Codex config.
- Home Manager generates user files and profiles under the target home and Nix profile paths from `nix/home/default.nix`, `nix/home/shell-env.nix`, and `nix/modules/zsh.nix`.
- Nix stores immutable build results in the local Nix store and exposes activated generations through nix-darwin; the build/switch/rollback lifecycle is documented in `nix/README.md` and defined by `flake.nix`.
- Neovim runtime downloads and state stay in standard local XDG directories rather than Git according to `.config/nvim/README.md` and `.config/nvim/CLAUDE.md`.
- `.gitignore` and the inventory in `README.md` keep named privacy-sensitive and machine-local state out of version control, including GitHub login state, email config, 1Password device state, media watch history, and tmux runtime plugins. Those ignore rules do not filter physical-directory fallback in `setup_mac.sh`; physical `.config` subdirectories discovered whenever any tracked-discovery condition fails must be reviewed before approving a symlink.

**Caching:**
- No shared or network cache service is configured; Nix's local store and generations are the reproducible artifact cache implied by `flake.lock`, `flake.nix`, and the build flow in `nix/README.md`.
- Neovim uses local `~/.cache/nvim` plus local share/state directories according to `.config/nvim/README.md`; plugin commits remain reproducible through `.config/nvim/lazy-lock.json`.
- Yazi uses its local preview/cache behavior from `.config/yazi/yazi.toml`; the tracked configuration does not name an external cache provider.
- mpv enables a 60-second streaming cache with bounded demuxer buffers in `.config/mpv/mpv.conf`; this is local playback buffering, not a shared cache integration.

## Authentication & Identity

**Auth Provider:**
- No centralized authentication provider is implemented; identity is delegated to macOS and each installed CLI/application, while the repository declares only one local username and hostname in `flake.nix`.
  - Implementation: GitHub CLI credentials remain in ignored `.config/gh/hosts.yml`, email account details remain in ignored `.config/himalaya/`, 1Password device state remains in ignored `.config/op/`, and other local login state stays outside Git according to `.gitignore` and `README.md`.
  - Implementation: Shell secrets and machine-specific variables belong in local `~/.zshrc.local`, which is sourced by `nix/modules/zsh.nix` and `zsh/.zshrc` but is not a repository file.
  - Implementation: macOS Accessibility is a local permission gate for Hammerspoon event taps rather than an application auth provider, as documented in `.hammerspoon/CLAUDE.md` and `nix/README.md`.

## Monitoring & Observability

**Error Tracking:**
- None; no external crash reporting, metrics, tracing, or error-tracking SDK is configured in `flake.nix`, `nix/`, `setup_mac.sh`, or the tracked `.config/` application files.

**Logs:**
- Bootstrap scripts report progress and errors through terminal stdout/stderr and terminate on failure with `set -euo pipefail` in `setup_mac.sh` and `install_yazi_plugins.sh`.
- Hammerspoon intentionally minimizes logging and shows a local load alert after registering its event taps and hotkeys in `.hammerspoon/init.lua`, with the low-noise policy documented in `.hammerspoon/CLAUDE.md`.
- btop provides interactive local system monitoring with warning-level logging and two-second updates in `.config/btop/btop.conf`; it is not a central observability backend.
- Validation is command-oriented: Nix evaluation/build output, Ghostty validation, Neovim health checks, and Hammerspoon IPC checks are documented in `nix/README.md`, `.config/ghostty/CLAUDE.md`, `.config/nvim/CLAUDE.md`, and `.hammerspoon/CLAUDE.md`; `.gitleaks.toml` supplies optional repository-specific scan configuration.

## CI/CD & Deployment

**Hosting:**
- Local macOS workstation only; the repository declares `darwinConfigurations.AresdeMacBook-Air` and no hosted application target in `flake.nix` and `nix/README.md`.
- Declarative Homebrew service activation is limited to `borders` and `nginx`, which are started when absent through `nix/darwin/homebrew.nix`; Colima and other Homebrew services remain manually controlled according to `README.md`. Separately, AeroSpace starts at login and handles window-detection callbacks from `.config/aerospace/aerospace.toml`, while Hammerspoon maintains event taps and timers from `.hammerspoon/init.lua`.

**CI Pipeline:**
- No tracked CI workflow is present; the authoritative validation and deployment sequence is manual `nix flake check`, `darwin-rebuild build --flake .#AresdeMacBook-Air`, and privileged `darwin-rebuild switch`, documented in `nix/README.md` and `nix/CLAUDE.md`.
- Configuration-specific checks are also manual: headless Neovim checks in `.config/nvim/CLAUDE.md`, Ghostty `+validate-config` in `.config/ghostty/CLAUDE.md`, and Hammerspoon IPC/reload checks in `.hammerspoon/CLAUDE.md`. Gitleaks is available through `.gitleaks.toml` and appears in language-stack validation examples, but the root workflow's required pre-commit control is the manual privacy review in `AGENTS.md`.
- The non-Nix deployment path is the interactive symlink bootstrap in `setup_mac.sh`; Yazi plugins are synchronized separately by `install_yazi_plugins.sh`.
- Rollback is generation-based through `sudo darwin-rebuild switch --rollback`, with Home Manager conflict backups using the `hm-backup` suffix configured in `flake.nix` and documented in `nix/README.md`.

## Environment Configuration

**Required env vars:**
- `HOME` is read by `install_yazi_plugins.sh` to derive Yazi and lazygit locations and by `zsh/shared.zsh` to construct user tool paths and `BUN_INSTALL`.
- `EDITOR`, `VISUAL`, and `PAGER` are supplied non-secret values by Home Manager in `nix/home/shell-env.nix`; the fallback zsh route also guards `EDITOR=nvim` in `zsh/.zshrc`.
- `XDG_CONFIG_HOME` is optional; `install_yazi_plugins.sh` falls back to `$HOME/.config/yazi` and then the tracked `.config/yazi` directory.
- `YAZI_CONFIG_HOME` and `LG_CONFIG_FILE` are set and exported by `install_yazi_plugins.sh` before it invokes `ya pkg`, so callers do not need to predefine them.
- `BUN_INSTALL` is set to `$HOME/.bun` and prepended to PATH by `zsh/shared.zsh`; Bun itself is installed outside the Nix/Homebrew inventory according to `zsh/CLAUDE.md`.
- No secret environment variable is required by tracked code; private variables are intentionally sourced from local `~/.zshrc.local` by `nix/modules/zsh.nix` and `zsh/.zshrc`.

**Secrets location:**
- Credential-bearing and private application state is local-only and ignored: `.config/gh/hosts.yml`, `.config/himalaya/`, `.config/op/`, and other listed state paths are enumerated without values in `.gitignore` and `README.md`.
- Codex and Claude project-local state is excluded through `.codex/` and `.claude/` entries in `.gitignore`; `setup_mac.sh` only offers local Codex linking when a local config already exists and never creates credentials.
- `.gitleaks.toml` provides default gitleaks rules plus narrow false-positive allowlists as an optional scan configuration. The enforced repository workflow control is the manual diff review for privacy leaks before commit or push required by `AGENTS.md`.
- Container registry credentials may live in local `~/.docker/config.json`, and X API credentials may live in local `~/.xurl`; both are documented as out-of-repository state in `README.md` and `nix/darwin/homebrew.nix`.

## Webhooks & Callbacks

**Incoming:**
- No HTTP endpoints or incoming webhooks are defined; local callbacks are limited to AeroSpace window-detection/focus events in `.config/aerospace/aerospace.toml` and Hammerspoon keyboard/timer events in `.hammerspoon/init.lua`.
- Hammerspoon exposes a local IPC message port through `require("hs.ipc")` for commands such as reload and config inspection in `.hammerspoon/init.lua` and `.hammerspoon/CLAUDE.md`; it is not a network webhook.

**Outgoing:**
- No outgoing webhook delivery is configured; network activity consists of package/source fetching from GitHub and Homebrew through `flake.nix`, `.config/nvim/lua/config/lazy.lua`, `.config/yazi/package.toml`, `setup_mac.sh`, and `nix/darwin/homebrew.nix`.
- Runtime external calls are user-initiated CLI or application actions, such as GitHub CLI operations from `.config/gh/config.yml`, mpv/yt-dlp URL playback from `.config/mpv/mpv.conf`, and local Docker commands described in `README.md`.
- Local process callbacks include Hammerspoon launching Ghostty in `.hammerspoon/init.lua` and Yazi launching editors, Finder, mpv, exiftool, and mediainfo from `.config/yazi/yazi.toml`; none send repository-defined webhooks.

---

*Integration audit: 2026-07-10*
