# Codebase Concerns

**Analysis Date:** 2026-04-01

## Symlink Management

**Dual AGENTS.md/CLAUDE.md symlink pattern:**
- Issue: Multiple AGENTS.md and CLAUDE.md files exist across `.config/nvim/`, `.config/yazi/`, `.config/`, `zsh/`, and `.hammerspoon/`, with 6 of 7 being symlinks pointing back to root versions
- Files: `AGENTS.md` (6 symlinks), `CLAUDE.md` (6 symlinks at `.config/nvim/CLAUDE.md`, `.config/yazi/CLAUDE.md`, `.config/CLAUDE.md`, `zsh/CLAUDE.md`, `.hammerspoon/CLAUDE.md`, and root `/CLAUDE.md`)
- Impact: Creates confusion about single source of truth. If guidance documents are modified in subdirectories, changes may not persist. Symlinks are brittle when moving config directories or during setup
- Fix approach: Keep one physical CLAUDE.md and one physical AGENTS.md in root, symlink all others to root versions. Verify symlink targets exist after any directory reorganization. Consider adding a validation script to `.planning/` that checks symlink integrity

## Setup Script Issues

**setup_mac.sh relies on Git for tracking detection:**
- Issue: Script uses `git ls-files` to detect tracked directories (line 22), but fails gracefully only if current dir is not a Git repo. In line 30, it falls back to filesystem scan, which could inadvertently include `.gitignore`d directories locally before they're .gitignore'd
- Files: `/setup_mac.sh` (line 21-26)
- Impact: If a user adds a directory to `.config/` locally before committing to git, the fallback scan may include it in symlinks, creating inconsistency between machines
- Fix approach: Always prioritize Git-tracked files. Add explicit validation that detected config names match `.gitignore` exceptions. Consider adding a dry-run mode with `-n` flag to preview what will be linked

**Hardcoded user-specific paths:**
- Issue: Script prompts for username but doesn't validate it exists as system user (line 34-38). It checks if path exists but doesn't verify it's actually a valid user home directory
- Files: `/setup_mac.sh` (line 43-46)
- Impact: Could create symlinks in arbitrary directories if user provides wrong name. Could mask permission errors
- Fix approach: Add validation using `dscl . -read /Users/$username` or similar to verify user exists in system

**Redundant symlink existence checks:**
- Issue: Multiple conditionals check `[[ -e "$target_path" || -L "$target_path" ]]` (lines 67, 89, 115, 141) but logic is repeated identically. Not a bug, but maintenance burden
- Files: `/setup_mac.sh` (lines 67, 89, 115, 141)
- Impact: If symlink handling logic needs updating (e.g., to support relative symlinks), it must be updated in 4 places
- Fix approach: Extract to a helper function `check_target_exists()` to reduce duplication

## Plugin Management

**Yazi plugin zoom.lua compatibility TODO:**
- Issue: `/.config/yazi/plugins/full-border.yazi/main.lua` line 23 contains `-- TODO: remove this compatibility hack` referencing conditional fs.unique check
- Files: `/.config/yazi/plugins/full-border.yazi/main.lua` (line 23-24)
- Impact: Plugin may have undefined behavior on Yazi versions where fs.unique is not stable. Could break padding logic
- Fix approach: Monitor full-border.yazi repository for when fs.unique becomes stable, update plugin

**Yazi plugins missing dependency documentation:**
- Issue: `install_yazi_plugins.sh` warns about missing dependencies (ImageMagick, 7zz, lazygit, starship) but doesn't specify which plugins actually require them
- Files: `/install_yazi_plugins.sh` (lines 79-83)
- Impact: User may install dependencies unnecessarily, or worse, skip them thinking they're optional when they're critical for certain plugins
- Fix approach: Add a mapping in README.md or in script: zoom.yazi→magick, compress.yazi→7zz, lazygit.yazi→lazygit, starship.yazi→starship

**Plugin installation idempotency not documented:**
- Issue: Script assumes `ya pkg install` is idempotent but doesn't document what happens if a plugin fails to install mid-operation
- Files: `/install_yazi_plugins.sh` (line 114)
- Impact: If network fails or plugin repository is unavailable, incomplete plugin state may persist. User may not know which plugins actually installed
- Fix approach: Add explicit failure handling; check `ya pkg list` output before and after to verify all expected plugins installed

## Configuration Consistency

**LazyVim and Neovim configuration version pinning:**
- Issue: `/.config/nvim/lazy-lock.json` and `/.config/nvim/lazyvim.json` track exact plugin versions, but there's no documented strategy for updating these when new plugin versions are available
- Files: `/.config/nvim/lazy-lock.json`, `/.config/nvim/lazyvim.json`
- Impact: Plugins become stale, security patches are missed, Neovim behavior drifts from upstream
- Fix approach: Document in README or CLAUDE.md when and how to run `:Lazy update`, and when to commit updated lock files. Consider quarterly review of plugin versions

## Hammerspoon Automation Risks

**Event tap startup suppression logic:**
- Issue: Hammerspoon config uses `during-aerospace-startup = false` in AeroSpace window detection rules (e.g., line 101, 108, 114) but Hammerspoon has no knowledge of this property—it only applies to AeroSpace
- Files: `/.config/aerospace/aerospace.toml` (lines 101, 108, 114, 122, 129, 136, 142)
- Impact: Window floating logic may not reliably suppress during Hammerspoon startup, potentially causing race conditions where windows are laid out before Hammerspoon finishes initialization
- Fix approach: Verify AeroSpace version requirement for `during-aerospace-startup` flag. Add explicit wait/delay in Hammerspoon init if critical window layout depends on AeroSpace readiness

**Hammerspoon right Cmd remapping conflicts:**
- Issue: Remapping right Cmd to F19 (line 103-112 in init.lua) could conflict with Karabiner-Elements or other key remapping tools if they're installed later
- Files: `/.hammerspoon/init.lua` (line 103-112)
- Impact: User installs Karabiner later and gets unexpected key behavior due to competing remappers. No fallback if F19 is already bound
- Fix approach: Check if Karabiner or other remappers are installed in setup script. Document this incompatibility in README. Consider making F19 configurable in Hammerspoon settings

## Codebase Sync Gaps

**Codex configuration intentionally not tracked:**
- Issue: `.codex/config.toml` is explicitly excluded from git tracking (README line 44) but setup_mac.sh still offers to symlink it (line 86)
- Files: `/setup_mac.sh` (line 86), `.gitignore` (line 5)
- Impact: Codex is useful but not synced across machines, creating fragmentation. User must manually configure Codex on each machine. Setup script silently fails if .codex doesn't exist locally
- Fix approach: Clarify if Codex should be machine-local or shared. If local, remove setup_mac.sh prompts. If sharable, document which parts are private vs public and create a split configuration

**Raycast extensions fully local:**
- Issue: Raycast extensions are in `.config/raycast/` but git-ignored completely (line 14), yet AeroSpace config has no Raycast rules, suggesting no integration attempted
- Files: `/.config/raycast/`, `.gitignore` (line 14)
- Impact: Raycast setup is completely manual per machine. No visibility into what extensions are installed where
- Fix approach: Either version-control essential Raycast extension list, or update README to document which extensions to install manually

## Local-Only Configuration

**No validation that local-only dirs remain untracked:**
- Issue: `.gitignore` lists local-only directories (`.codex/`, `.config/op/`, `.config/linearmouse/`, etc.) but there's no CI or pre-commit hook to verify they stay ignored
- Files: `.gitignore` (lines 5, 13-16, 21-22, 25)
- Impact: User could accidentally commit secrets (1Password tokens in `.config/op/`), session data, or personal data if they move files around
- Fix approach: Add a `.git/hooks/pre-commit` script to verify no ignored files are staged

**Missing .zshrc.local documentation:**
- Issue: README mentions `.zshrc.local` for secrets (line 27) but provides no example or template of what should go there
- Files: `README.md` (line 27), `zsh/.zshrc` (line 35)
- Impact: User may not know what's appropriate to put in `.zshrc.local`, leading to either missing configuration or security risks (putting secrets in tracked `.zshrc`)
- Fix approach: Create `zsh/.zshrc.local.template` with examples of common machine-specific settings (PATH, API keys, project variables)

## Testing and Validation

**No validation suite for symlink setup:**
- Issue: `setup_mac.sh` creates multiple symlinks but has no post-execution validation that all symlinks are valid and point to correct locations
- Files: `/setup_mac.sh`
- Impact: Silent failures if symlinks are created but targets become invalid later (e.g., user moves repo). User won't discover issue until they try to access config
- Fix approach: Add a validation script (e.g., `validate_config.sh`) that checks all expected symlinks exist and are readable

**No pre-commit validation of config file syntax:**
- Issue: TOML files (aerospace.toml, package.toml) are committed without syntax validation. Invalid TOML could break user's machine when applied
- Files: All `.toml` files in `.config/`
- Impact: Commit with malformed TOML, setup_mac.sh succeeds, but applications fail to read config
- Fix approach: Add `.git/hooks/pre-commit` to validate TOML syntax using `toml-cli` or similar

## Documentation Gaps

**No troubleshooting guide:**
- Issue: README explains setup but doesn't document what to do if symlinks fail, apps crash, or setup script fails partway through
- Files: `README.md`
- Impact: User unable to debug issues, may resort to manual configuration instead of using setup script
- Fix approach: Add TROUBLESHOOTING.md with common issues and recovery steps

**Version compatibility not documented:**
- Issue: No requirement specs for macOS version, homebrew version, or nanobrew version. AeroSpace config assumes specific features
- Files: `README.md`, `setup_mac.sh`
- Impact: Setup fails on older macOS or incompatible tool versions without clear error messages
- Fix approach: Add `REQUIREMENTS.md` specifying minimum macOS version, tool versions, and compatibility matrix

---

*Concerns audit: 2026-04-01*
