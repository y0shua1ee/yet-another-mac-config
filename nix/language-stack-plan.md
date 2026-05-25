# Development Language Stack Plan

> **For Hermes:** Use subagent-driven-development skill if this plan is later implemented task-by-task.

**Goal:** Bring language/toolchain management into the existing gradual Nix route without turning the Mac config repo into a fragile, all-owning runtime manager.

**Architecture:** Keep the Mac config repo responsible for stable global entrypoints and shell integration. Use `mise`/`uv`/`rustup`/`direnv` as user-facing toolchain managers, and use project-local `.mise.toml`, `uv`, or Nix devShells for per-project versions. Avoid declaring heavyweight or project-specific language runtimes directly in the global Homebrew inventory.

**Tech Stack:** nix-darwin, Home Manager, Homebrew, mise, uv, rustup, direnv/nix-direnv, optional per-project flake devShells.

---

## Current state observed on 2026-04-25 / updated 2026-04-26

Repo status:

- Current Nix route is intentionally gradual.
- `nix/home/packages.nix` owns only low-risk CLI packages: `ripgrep`, `fd`, `jq`, `tree`, `bat`.
- `nix/darwin/homebrew.nix` still avoids declaring heavyweight language runtimes directly; toolchain managers live in Home Manager.

Installed language/toolchain-related tools observed locally:

- Homebrew leaves include language/runtime-adjacent tools such as `go`, `rust`, `nvm`, `pnpm`, `uv`, `llvm@21`, plus build-adjacent tools such as `graphviz`.
- Phase 5A has switched successfully: `mise`, `rustup`, and `direnv` resolve from `/etc/profiles/per-user/areslee/bin`.
- Before Phase 5B switch, login zsh still resolves active Node from NVM: `~/.nvm/versions/node/v24.11.0/bin/node`.
- Phase 5B has prepared mise Node: `.config/mise/config.toml` pins global `node = "24.11.0"`; `mise exec -- node -v` returns `v24.11.0`, and `mise exec -- npm -v` returns `11.6.1`.
- A pilot project at `~/Documents/mise-node-pilot` verified mise Node with a minimal Claude Code run; Claude/GSD hooks that call `node` worked under mise Node.
- Current active Python path is affected by Hermes virtualenv, while `python3.13` / `python3.12` exist under `~/.local/bin`, likely uv-managed standalone Python installs.
- Current active Rust is still Homebrew `rustc` / `cargo` in login zsh until intentionally migrated.
- Current active Deno is still Homebrew `deno`.
- Current active Bun is still `~/.bun/bin/bun`.

---

## Recommended policy

### Layer 1 — Global, reproducible entrypoints managed by Home Manager

Use Home Manager for small, stable, non-GUI, non-service developer entrypoints:

- `mise` — primary multi-language version orchestrator. It can manage multiple Node versions and is the intended long-term replacement candidate for NVM, while also covering Go / Deno / Bun and possibly project-level Python selectors.
- `uv` — Python project/package/venv manager; keep it globally available.
- `rustup` — Rust toolchain manager; better than a fixed global `rust` package for components/targets/toolchains.
- `direnv` + `nix-direnv` — automatically load project-local `.envrc` / Nix devShells.
- Optional small build helpers later: `pkg-config`, `cmake`, `ninja`, `just`, only after confirming they are broadly useful across projects.

Do **not** declare secrets, tokens, registry auth, SDK license state, or project-local generated files.

### Layer 2 — Per-project version declarations

Prefer project-local files for actual runtime versions:

- Node / Go / Deno / Bun: `.mise.toml` in each project.
- Python: `pyproject.toml` + `uv.lock`; if a specific interpreter is required, let `uv` or `.mise.toml` pin it at project level.
- Rust: `rust-toolchain.toml` in Rust projects when a specific toolchain is required.
- Native dependencies: project `flake.nix` devShell when system libraries or compilers matter.

### Layer 3 — Homebrew remains for GUI, casks, and a few macOS-native formulae

Keep Homebrew for:

- Existing GUI/cask policy.
- Formulae where Homebrew integration is intentionally chosen, e.g. current terminal/editor/media tools.
- Temporary compatibility during migration.

Avoid adding global language runtimes such as `go`, `rust`, `node`, `python@*`, `deno`, `pnpm`, `llvm@*` to `nix/darwin/homebrew.nix` unless there is a specific reason and a documented boundary.

---

## Migration phases

## Phase 5A — Add toolchain managers, no runtime migration yet

**Objective:** Add management entrypoints without breaking current workflows.

**Files:**

- Create: `nix/home/dev-toolchains.nix`
- Modify: `nix/home/default.nix`
- Modify: `nix/README.md`
- Modify: `nix/CLAUDE.md`
- Modify: root `README.md` if user-facing summary changes

**Implementation shape:**

```nix
# nix/home/dev-toolchains.nix
{ pkgs, ... }:
{
  # Phase 5A: language/toolchain entrypoints only.
  # Actual per-project versions should live in project-local files:
  # .mise.toml, pyproject.toml + uv.lock, rust-toolchain.toml, or flake devShells.
  home.packages = with pkgs; [
    mise
    uv
    rustup
  ];

  programs.direnv = {
    enable = true;
    nix-direnv.enable = true;
  };
}
```

Then import it:

```nix
# nix/home/default.nix
imports = [
  ./packages.nix
  ./shell-env.nix
  ./dev-toolchains.nix
  ../modules/zsh.nix
];
```

**Verification:**

```bash
cd /Users/areslee/Documents/dev/config/yet-another-mac-config
nix flake check
darwin-rebuild build --flake .#AresdeMacBook-Air
git diff --check
/opt/homebrew/bin/gitleaks detect --no-git --redact --source .
```

**Switch:** only after review:

```bash
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

**Expected post-switch commands:**

```bash
command -v mise uv rustup direnv
mise --version
uv --version
rustup --version
direnv version
```

**Commit:**

```bash
git add nix/home/default.nix nix/home/dev-toolchains.nix nix/README.md nix/CLAUDE.md README.md
git commit -m "feat(nix): add language toolchain entrypoints"
```

## Phase 5B — Add shell integration and safe compatibility notes

**Objective:** Make mise usable as the default Node manager while preserving NVM as rollback fallback during the transition.

Implemented / current changes:

- Global mise config pins Node to the known-working NVM version: `.config/mise/config.toml` contains `node = "24.11.0"`.
- `mise install` has installed that version; `mise exec -- node -v` and `mise exec -- npm -v` match the existing NVM runtime (`v24.11.0` / `11.6.1`).
- Pilot project `~/Documents/mise-node-pilot` verified `.mise.toml` with Node `24.11.0` and a minimal Claude Code run under mise Node.
- `nix/modules/zsh.nix` enables `mise activate zsh` after sourcing `~/.zshrc.local`, so NVM remains loaded first as fallback and mise then takes over default `node` / `npm`.
- Keep current `zsh/shared.zsh` Bun PATH block initially; remove only after Bun is intentionally managed via mise.
- Do not delete NVM, Homebrew `nvm`, or Homebrew `pnpm` yet.

Current zsh integration:

```zsh
if command -v mise >/dev/null 2>&1; then
  eval "$(mise activate zsh)"
fi
```

Put this in `nix/modules/zsh.nix` after the `~/.zshrc.local` source line. Avoid putting it in `zsh/shared.zsh`: the Home Manager route can be switched/rolled back more cleanly, and the ordering relative to NVM is intentional.

## Phase 5C — Gradually migrate runtimes

**Objective:** Move from global Homebrew/NVM runtimes to project-pinned versions without breakage.

Order:

1. Node:
   - Start new projects with `.mise.toml`.
   - Prefer Corepack-managed package managers over global Homebrew `pnpm` when possible.
   - Keep NVM until all frequently used projects are verified.
2. Python:
   - Standardize on `uv` for new Python projects.
   - Avoid globally declaring `python@*` unless a system-level CLI needs it.
3. Rust:
   - Move from Homebrew `rust` to `rustup` when comfortable.
   - Use `rust-toolchain.toml` for pinned project toolchains.
4. Go:
   - If a single latest Go is enough, current Homebrew Go can remain manual temporarily.
   - If multi-version Go matters, move Go under mise per project.
5. Deno / Bun:
   - If used as general app runtimes, manage with mise per project.
   - Keep direct Bun install path until migration is tested.

## Phase 5D — Cleanup old global runtimes only after adoption

Only after Phase 5C has been tested in real projects:

- Consider removing or no longer installing Homebrew `rust`, `go`, `pnpm`, `deno`, `nvm`, `llvm@21` if they are no longer needed.
- Do **not** use Homebrew cleanup automation yet; keep `cleanup = "none"`.
- Remove one global runtime at a time, with rollback notes.

---

## Suggested decisions before implementation

Recommended defaults:

1. Use Home Manager for `mise`, `uv`, `rustup`, and `direnv`/`nix-direnv`.
2. Use mise as the long-term default Node manager; keep the global Node pinned to `24.11.0` until there is a separate reason to upgrade.
3. Keep NVM installed as fallback until the `mise activate zsh` switch has been applied and verified in daily use.
4. Do not immediately remove Homebrew `go`, `rust`, `pnpm`, `deno`, `llvm@21`.
5. Add docs first, then switch, then test real projects.
6. Keep project-specific toolchain pinning out of this Mac config repo unless the project itself lives in this repo.

---

## Rollback

If Phase 5A causes trouble:

```bash
cd /Users/areslee/Documents/dev/config/yet-another-mac-config
git revert <phase-5a-commit>
sudo darwin-rebuild switch --flake .#AresdeMacBook-Air
```

Since Phase 5A only adds entrypoints and direnv integration, it should not delete existing Homebrew/NVM/Bun/uv installs.
