# Coding Conventions

**Analysis Date:** 2026-04-01

## Naming Patterns

**Files:**
- JavaScript/CommonJS modules: `kebab-case.cjs` or `kebab-case.js` (e.g., `gsd-check-update.js`, `frontmatter.cjs`)
- Shell scripts: `snake_case.sh` with executable permission (e.g., `setup_mac.sh`, `install_yazi_plugins.sh`)
- Configuration files: lowercase with extension (e.g., `.zshrc`, `config.toml`)

**Functions:**
- camelCase for exported functions: `extractFrontmatter`, `reconstructFrontmatter`, `cmdStateGet`
- Functions prefixed with `cmd` indicate CLI command handlers: `cmdVerifySummary`, `cmdStateLoad`
- Helper functions may be prefixed with underscore for private scope conceptually: `_staleHooks` (arrays/objects), internal helpers unnamed

**Variables:**
- camelCase for local variables and parameters: `frontmatter`, `mentionedFiles`, `exitCode`
- CONSTANT_CASE for module-level constants: `ZSH_THEME`, `FRONTMATTER_SCHEMAS`
- Suffixes indicate type: `*List` for arrays, `*Map` for Maps, `*Str` for strings (e.g., `hookFiles`, `commitHashPattern`)
- snake_case used in object keys matching domain concepts: `files_modified`, `depends_on`, `must_haves` (YAML/JSON frontmatter fields)

**Types:**
- No TypeScript in this codebase. Plain JavaScript with JSDoc type hints in comments.
- Objects use descriptive keys: `{ passed: boolean, checks: object, errors: array }`

## Code Style

**Formatting:**
- Indentation: 2 spaces (consistent across .cjs, .js, .sh files)
- Line length: No strict limit enforced, but files tend to stay under 120 characters for readability
- No explicit formatter config file detected (no .prettierrc, eslint config) — style is manual/conventional

**Linting:**
- No linter config detected (.eslintrc, biome.json, etc.)
- Code follows conventional patterns suggesting manual review process
- Error messages use imperative tone: `error('file path required')` vs `throw new Error(...)`

**Comment Style:**
- JSDoc-style block comments for function documentation:
  ```javascript
  /**
   * Remove stale gsd-* temp files/dirs older than maxAgeMs (default: 5 minutes).
   * Runs opportunistically before each new temp file write to prevent unbounded accumulation.
   * @param {string} prefix - filename prefix to match (e.g., 'gsd-')
   * @param {object} opts
   * @param {number} opts.maxAgeMs - max age in ms before removal (default: 5 min)
   * @param {boolean} opts.dirsOnly - if true, only remove directories (default: false)
   */
  ```
- Inline comments for complex logic (as seen in `frontmatter.cjs` parsing logic)
- Section dividers using ASCII: `// ─── Path helpers ────────────────────────────────────────────────────────────`
- Shell scripts use Chinese comments: `# 初始化脚本。` (per CLAUDE.md requirement)

## Import Organization

**Order (Node.js/CommonJS):**
1. Built-in modules: `const fs = require('fs');`, `const path = require('path');`
2. External dependencies: `const { exec } = require('child_process');`
3. Local module imports: `const { core } = require('./core.cjs');`

**Example from `frontmatter.cjs`:**
```javascript
const fs = require('fs');
const path = require('path');
const { safeReadFile, normalizeMd, output, error } = require('./core.cjs');
```

**Module Exports:**
- CommonJS: `module.exports = { funcName, anotherFunc };` (multiple exports)
- All major modules export multiple utility functions as object

**Shell Scripts:**
- No imports (shell). Use `source` for sourcing external scripts if needed (seen in `.zshrc`)

## Error Handling

**Patterns:**
- CommonJS: Use custom `error()` function from core module for fatal errors:
  ```javascript
  if (!filePath) { error('file path required'); }
  ```
  This function terminates execution and outputs to stderr with non-zero exit code.

- Recoverable errors: Return error objects in output:
  ```javascript
  output({ error: 'File not found', path: filePath }, raw); return;
  ```

- Exception catching: Try/catch with silent fallback common:
  ```javascript
  try {
    parsedValue = JSON.parse(value);
  } catch {
    parsedValue = value;  // Fall back to string
  }
  ```

- Shell scripts: Use `set -euo pipefail` to fail fast on errors (seen in `setup_mac.sh` line 6)
  - Check file/directory existence before operations
  - Use `command -v` to check if executables exist before running them

**Exit Codes:**
- CommonJS: 0 for success, non-zero for errors (via `error()` function or `process.exit(n)`)
- Shell: Same convention, check results with `$?`

## Logging

**Framework:** Console via `console.error()` for warnings/errors, no dedicated logger

**Patterns in CommonJS:**
- Custom `output()` function handles structured logging:
  ```javascript
  output(result, raw, plaintext);
  // output(jsonObject, shouldOutputRaw, alternateTextOutput)
  ```
- Conditional logging with guards: `catch (e) {}` — silent on error for non-critical operations
- Warnings printed to stderr in background processes: `console.error('Failed to fetch CPU performance data:')`

**Patterns in Shell:**
- Echo to stdout for user info: `echo "已创建: $target_path"`
- Use `>&2` to redirect errors to stderr when needed

## Comments

**When to Comment:**
- Complex algorithms (YAML parsing in `frontmatter.cjs`, regex patterns for git detection)
- Non-obvious intent (e.g., why we check parent config first, then global)
- Caveats or edge cases: `// Shell may not support process memory on Windows`
- Section headers for logical grouping

**JSDoc/TSDoc Usage:**
- Function-level JSDoc with parameter types: `@param {string} prefix`, `@param {object} opts`
- No return type annotation seen (inferring from code)
- Used for all exported functions in library modules

**Inline Documentation:**
- Regex patterns get inline explanation: `// Find must_haves: first to detect its indentation level`
- Logic flow explained for multi-step processes (git repo detection in `core.cjs`)

## Function Design

**Size:**
- Range: 10–50 lines typical, up to 150+ for complex operations (e.g., `findProjectRoot`)
- Longer functions justified by single responsibility (e.g., YAML parsing state machine)

**Parameters:**
- Positional for required params: `function cmdStateLoad(cwd, raw)`
- Options objects for many optional settings:
  ```javascript
  function reapStaleTempFiles(prefix = 'gsd-', { maxAgeMs = 5 * 60 * 1000, dirsOnly = false } = {})
  ```
- Path/directory as first param (convention): `cwd` always first

**Return Values:**
- Objects with structured data (not tuples or mixed types):
  ```javascript
  return {
    update_available: boolean,
    installed: string,
    latest: string,
    checked: number,
    stale_hooks: array | undefined
  };
  ```
- Functions may also mutate/output side effects (write files, call `output()`)

## Module Design

**Exports:**
- All .cjs files export a module object with multiple functions:
  ```javascript
  module.exports = {
    extractFrontmatter,
    reconstructFrontmatter,
    spliceFrontmatter,
    parseMustHavesBlock,
    FRONTMATTER_SCHEMAS,
    cmdFrontmatterGet,
    cmdFrontmatterSet,
    // ...
  };
  ```
- No default exports used

**Barrel Files:** Not used. Each module is self-contained.

**File Organization:**
- Each `.cjs` file has one primary responsibility: `frontmatter.cjs` handles frontmatter, `state.cjs` handles state, etc.
- Internal helpers grouped near top of file (path helpers, output helpers)
- Command functions (cmd*) grouped at bottom or middle
- Constants and schemas defined near usage

**Interdependencies:**
- Common shared module: `core.cjs` — contains utilities imported by all others
- Minimal circular dependencies (checked in code structure)

## Patterns and Idioms

**Config Detection:**
- Multi-path detection with fallback cascade:
  ```javascript
  for (const dir of ['.config/opencode', '.opencode', '.gemini', '.config', 'opencode']) {
    if (fs.existsSync(path.join(baseDir, dir, 'get-shit-done', 'VERSION'))) {
      return path.join(baseDir, dir);
    }
  }
  return envDir || path.join(baseDir, '.config', 'opencode');
  ```

**Regex Patterns:**
- Inline regex with multiline flag for markdown parsing: `/(?:^|\n)\s*---\r?\n([\s\S]+?)\r?\n---/g`
- Escaped patterns for special chars: `escapeRegex(fieldName)` utility wraps field names

**Async/Concurrency:**
- Node.js callback-based: `execSync()` for synchronous shell commands, `spawn()` for detached processes
- Promises not used (older Node.js compatibility or deliberate choice)
- Child process spawning with `detached: true` on Windows for proper cleanup

**String Handling:**
- Path normalization to POSIX: `p.split(path.sep).join('/')`
- CRLF/LF handling in multiline strings: `\r?\n` patterns account for both

---

*Convention analysis: 2026-04-01*
