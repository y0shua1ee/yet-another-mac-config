---
created: 2026-07-10T10:10:49.439Z
title: Clarify Determinate Nix and Home Manager authority
area: planning
files:
  - .planning/PROJECT.md:3
  - .planning/ROADMAP.md:29
  - .planning/research/STACK.md:9
  - flake.nix:1
  - nix/darwin/default.nix:1
  - nix/home/default.nix:1
---

## Problem

The user clarified that the project should manage the Mac overall through Determinate Nix and Home Manager. The current repository already uses Determinate Nix as the Nix installation and daemon layer, nix-darwin as the macOS system composition layer, and Home Manager as the user-environment layer. Existing planning also assigns selected responsibilities to Homebrew through nix-darwin and to project-appropriate managers such as mise, uv, rustup, Nix devShell, and Gradle or Maven wrappers.

The project must make the word "overall" unambiguous. If it means that Determinate Nix, nix-darwin, and Home Manager form the primary declarative control plane while delegated tools retain one explicit owner per executable, the clarification is compatible with the approved requirements and roadmap. If it means that Nix or Home Manager must directly own every runtime, package manager, GUI application, and mutable service, it conflicts with the approved fit-for-purpose ownership model and would require revisiting requirements, research conclusions, and phases 2 through 10.

The repository currently uses the officially supported compatibility setting `nix.enable = false`, allowing Determinate Nix to manage Nix configuration. Current Determinate documentation also offers an optional nix-darwin module with `determinateNix.enable = true`; adopting that module is a separate implementation decision and must not be inferred or activated during planning.

## Solution

After the user confirms the intended meaning, update project-level planning language to state the hierarchy explicitly: Determinate Nix owns the Nix distribution and daemon boundary; nix-darwin is the machine-level orchestration and activation boundary; Home Manager owns reproducible user configuration and manager entrypoints; Homebrew remains declaratively inventoried through nix-darwin where appropriate; project runtimes and build tools retain their approved unique owners under repository-defined contracts.

Audit PROJECT, REQUIREMENTS, ROADMAP, and research wording for ambiguity. Preserve the existing phase structure if this is a control-plane clarification rather than an exclusive-package-ownership change. Evaluate the optional Determinate nix-darwin module separately against current official documentation, using evaluation and non-activating build checks before any explicitly confirmed switch.

Official references to re-check when planning:

- https://docs.determinate.systems/guides/nix-darwin/
- https://docs.determinate.systems/determinate-nix/
- https://nix-community.github.io/home-manager/introduction.html
- https://nix-darwin.github.io/nix-darwin/manual/
