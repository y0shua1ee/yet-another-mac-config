# Guidance for agents

## Introduction and Structure
- This folder contains the Neovim configuration for this project, based on LazyVim.
- `init.lua` is the entry point. `lua/config` stores core configuration. `lua/plugins` stores custom plugin specs. `lazy-lock.json` pins plugin versions.

## Workflow
- When changing Neovim behavior or adding plugins, first consult the official Neovim documentation, the LazyVim documentation, and the relevant plugin documentation.
- Prefer small, targeted changes under `lua/config` and `lua/plugins` instead of restructuring the whole setup.
- Keep machine-specific state, caches, sessions, and temporary files out of this folder and out of Git.
- If a new plugin requires installation, lockfiles, or extra setup steps, update the tracked configuration files accordingly and document the required steps in the project README when needed.
