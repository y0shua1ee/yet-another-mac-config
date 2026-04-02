# Project guidance for agents

## Introduction and Structure
- This is my Mac configuration, synced via GitHub.
- The README file, along with all the setup and installation scripts, is placed at the outermost layer.
- Some system configurations or application configuration files may be symbolic links to the corresponding directories under this project. You can find these by consulting the setup script.

## Workflow
- After modifying any configuration under this project, always check whether the corresponding `CLAUDE.md` (in the same directory or the root) needs to be updated, and apply changes if necessary.

## Style
- Please communicate with me in Chinese.
- Please use English when committing with git.
- The generated file structure must conform to the above description.
- When generating or modifying scripts, comments in Chinese are required.
- Maintain simple and easy-to-understand naming conventions.
- Before installing software, you should search and read relevant documentation online. Prioritize using nanobrew (`nb`) for package installation; Homebrew (`brew`) is kept as a fallback only.

