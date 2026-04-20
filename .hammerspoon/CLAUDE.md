# Guidance for agents

## Introduction and Structure
- This folder contains the Hammerspoon automation synced to the system `~/.hammerspoon` directory.
- `init.lua` is the main entry point and currently contains the active hotkeys and automation logic.

## External dependencies (must be satisfied on any new Mac)
- **Hammerspoon app itself** — installed via the `hammerspoon` cask. From Phase 4 minimum onward this cask is declared in `nix/darwin/homebrew.nix`, so `darwin-rebuild switch` will install it. On machines that don't use Nix, install manually with `brew install --cask hammerspoon`.
- **macOS Accessibility permission** — Hammerspoon's event taps (`hs.eventtap.new(...)`) and most hotkeys (double-tap Cmd+W/Q, right-Cmd → F19 remap, etc.) silently do nothing without this permission. Grant it under **System Settings → Privacy & Security → Accessibility**, toggle on Hammerspoon. macOS may also prompt automatically on first launch; if declined, you must re-enable it manually — there is no fallback in `init.lua`.
- **Ghostty app** — the `Ctrl+Alt+T` hotkey uses `hs.application.get("Ghostty")` and launches `open -na Ghostty`. Missing Ghostty → the hotkey is a silent no-op. Ghostty is also declared in `nix/darwin/homebrew.nix`, so the Nix path will install it automatically; otherwise `brew install --cask ghostty`.
- None of the above are installed or granted by `setup_mac.sh` — that script only symlinks `.hammerspoon` into `~/.hammerspoon`. The dependencies listed here are **in addition to** running the sync script.

## Workflow
- Also consult the Hammerspoon API reference for the specific modules involved.
- Keep hotkeys, event taps, and automation behavior explicit and easy to trace, because conflicts and recursive triggers are easy to introduce here.
- If a change depends on an external application or macOS permission, keep the dependency clear and update the project README + this file when needed. The root `README.md` has a dedicated "Hammerspoon 激活说明" section describing the full activation flow on a fresh Mac.
- Prefer low-noise automation and avoid adding unnecessary alerts, logs, or background behavior unless the feature clearly needs them.
- Phase 4 minimum only added the `hammerspoon` cask to the Homebrew inventory. The script itself (`init.lua`, any Spoons) is still managed the same way as before — do **not** try to rewrite it into a Home Manager module or move it under `nix/`.
