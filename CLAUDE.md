# Project guidance for agents

## Introduction and Structure
- This is my Mac configuration, synced via GitHub.
- The README file, along with all the setup and installation scripts, is placed at the outermost layer.
- Some system configurations or application configuration files may be symbolic links to the corresponding directories under this project. You can find these by consulting the setup script.

## Workflow
- Before changing any app configuration, always consult the official documentation for that app (and its plugins/framework if applicable).
- Prefer small, targeted changes over restructuring an entire config.
- After adding or modifying any configuration, you MUST complete this documentation checklist:
  1. **README.md** (repo root): update the config table, setup instructions, and gitignore notes as needed.
  2. **CLAUDE.md** (sub-directory): create one if adding a new config directory, or update the existing one. Always symlink `AGENTS.md -> CLAUDE.md` alongside it.
  3. **README.md / CLAUDE.md at higher levels**: check whether they need updates too (usually not, unless global conventions change).
- This checklist is non-optional. Do not consider a configuration change complete until all relevant documentation is in sync.

## Style
- Please communicate with me in Chinese.
- Please use English when committing with git.
- The generated file structure must conform to the above description.
- When generating or modifying scripts, comments in Chinese are required.
- Maintain simple and easy-to-understand naming conventions.
- Before installing software, you should search and read relevant documentation online. Prioritize using nanobrew (`nb`) for package installation; Homebrew (`brew`) is kept as a fallback only.

