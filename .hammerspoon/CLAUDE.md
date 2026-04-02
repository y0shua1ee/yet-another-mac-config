# Guidance for agents

## Introduction and Structure
- This folder contains the Hammerspoon automation synced to the system `~/.hammerspoon` directory.
- `init.lua` is the main entry point and currently contains the active hotkeys and automation logic.

## Workflow
- Also consult the Hammerspoon API reference for the specific modules involved.
- Keep hotkeys, event taps, and automation behavior explicit and easy to trace, because conflicts and recursive triggers are easy to introduce here.
- If a change depends on an external application or macOS permission, keep the dependency clear and update the project README when needed.
- Prefer low-noise automation and avoid adding unnecessary alerts, logs, or background behavior unless the feature clearly needs them.
