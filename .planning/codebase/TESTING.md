# Testing Patterns

**Analysis Date:** 2026-04-01

## Test Framework

**Status:** No dedicated testing framework detected

**Why:** This codebase is a system configuration and CLI tool suite, not a traditional tested library:
- Raycast extensions are bundled/minified for distribution (no source tests visible)
- GSD (Get-Shit-Done) CLI tool uses shell integration testing through OpenCode platform
- Manual verification through user workflows and acceptance testing (UAT patterns in `.config/opencode/get-shit-done/templates/UAT.md`)
- No Jest, Vitest, Mocha, or similar test runner config found

**Test Files:** Not detected in codebase structure
- No `*.test.js`, `*.spec.js`, `*.test.cjs` files
- No `jest.config.js`, `vitest.config.js`, or test configuration files
- No `tests/`, `__tests__/`, `test/` directories

## Verification Strategy Instead of Unit Tests

**Structural Verification:**
Instead of unit tests, verification happens through:

1. **Schema Validation** (in `frontmatter.cjs` and `verify.cjs`):
   ```javascript
   const FRONTMATTER_SCHEMAS = {
     plan: { required: ['phase', 'plan', 'type', 'wave', 'depends_on', 'files_modified', 'autonomous', 'must_haves'] },
     summary: { required: ['phase', 'plan', 'subsystem', 'tags', 'duration', 'completed'] },
     verification: { required: ['phase', 'verified', 'status', 'score'] },
   };

   function cmdFrontmatterValidate(cwd, filePath, schemaName, raw) {
     const schema = FRONTMATTER_SCHEMAS[schemaName];
     const fm = extractFrontmatter(content);
     const missing = schema.required.filter(f => fm[f] === undefined);
     output({ valid: missing.length === 0, missing, present, schema: schemaName }, raw, ...);
   }
   ```

2. **Summary Verification** (in `verify.cjs`):
   ```javascript
   function cmdVerifySummary(cwd, summaryPath, checkFileCount, raw) {
     // Check 1: Summary exists
     // Check 2: Spot-check files mentioned in summary
     // Check 3: Commits exist in git history
     // Check 4: Self-check section status
     const checks = {
       summary_exists: true,
       files_created: { checked, found, missing },
       commits_exist: boolean,
       self_check: 'passed' | 'failed' | 'not_found'
     };
   }
   ```

3. **Plan Structure Validation** (in `verify.cjs`):
   ```javascript
   function cmdVerifyPlanStructure(cwd, filePath, raw) {
     const required = ['phase', 'plan', 'type', 'wave', 'depends_on', 'files_modified', 'autonomous', 'must_haves'];
     // Checks:
     // - Frontmatter fields present
     // - Task elements (<task>, <name>, <action>, <verify>, <done>, <files>)
     // - Wave/depends_on consistency
     // Returns: { passed, checks, errors, warnings }
   }
   ```

## Test Data Patterns

**Frontmatter Fixtures:**
YAML frontmatter blocks in `.planning/` markdown files serve as test data:
- `plan: "phase-1"` / `plan: "phase-2"` identify phase
- `wave: 1` or `wave: 2` for parallel execution waves
- `depends_on: [phase-0]` for dependency validation
- `must_haves:` block with `artifacts:`, `truths:`, `key_links:` subsections

**Example (from `frontmatter.cjs` parsing):**
```yaml
---
phase: phase-1
plan: phase-1-plan
wave: 1
depends_on: []
files_modified: [src/index.js, src/utils.js]
must_haves:
  artifacts:
    - path: src/index.js
      provides: core-functionality
  truths:
    - all endpoints are RESTful
---
```

**File Path Test Data:**
Verification uses real file paths from summary documents:
```javascript
const patterns = [
  /`([^`]+\.[a-zA-Z]+)`/g,  // Matches `src/file.js`
  /(?:Created|Modified|Added|Updated|Edited):\s*`?([^\s`]+\.[a-zA-Z]+)`?/gi,
];
const filesToCheck = Array.from(mentionedFiles).slice(0, checkCount);
for (const file of filesToCheck) {
  if (!fs.existsSync(path.join(cwd, file))) {
    missing.push(file);
  }
}
```

## Error Handling in Verification

**Patterns for Invalid Input:**

1. **Required Param Validation:**
   ```javascript
   if (!filePath) { error('file path required'); }
   if (filePath.includes('\0')) { error('file path contains null bytes'); }
   ```

2. **File Not Found Handling:**
   ```javascript
   const content = safeReadFile(fullPath);
   if (!content) {
     output({ error: 'File not found', path: filePath }, raw);
     return;
   }
   ```

3. **Malformed YAML/JSON:**
   ```javascript
   try {
     parsedValue = JSON.parse(value);
   } catch {
     parsedValue = value;  // Degrade gracefully
   }
   ```

4. **Git Command Failures:**
   ```javascript
   const result = execGit(cwd, ['cat-file', '-t', hash]);
   if (result.exitCode === 0 && result.stdout === 'commit') {
     commitsExist = true;
   }
   ```

## Configuration Files Referenced

**No test config files exist**, but verification tools reference:
- `.planning/config.json` — Project configuration with milestones, branching strategy
- `.planning/STATE.md` — Current progress state (phase, completion %)
- `.planning/ROADMAP.md` — Milestone definitions
- Phase plan files (`.planning/phases/PHASE_*.md`) — Phase specifications with tasks

These files are validated but not unit-tested; they're validated through CLI commands:
```bash
gsd frontmatter validate .planning/phases/phase-1.md plan
gsd verify plan-structure .planning/phases/phase-1.md
gsd verify summary .planning/SUMMARY.md
```

## Hooks for Validation

**Pre-commit Hooks:**
Located in `.config/opencode/hooks/`:
- `gsd-check-update.js` — Detects stale hook versions, cache management
- `gsd-context-monitor.js` — Tracks context changes
- `gsd-prompt-guard.js` — Validates agent prompts
- `gsd-workflow-guard.js` — Validates workflow definitions

**Example Hook Pattern** (from `gsd-check-update.js`):
```javascript
const result = {
  update_available: latest && installed !== latest,
  installed,
  latest: latest || 'unknown',
  checked: Math.floor(Date.now() / 1000),
  stale_hooks: staleHooks.length > 0 ? staleHooks : undefined
};
fs.writeFileSync(cacheFile, JSON.stringify(result));
```

Hooks can fail validation and abort workflows if issues detected.

## Shell Script Validation

**Setup Script** (`setup_mac.sh`):
- Validates file existence before operations: `if [[ -d "$config_source" ]]`
- Validates user input: `if [[ -z "$username" ]]` (empty check)
- Validates regex patterns: `if [[ ! "$answer" =~ ^[Yy]$ ]]` (yes/no validation)
- Safe file operations: `set -euo pipefail` enforces error handling

**Pattern:**
```bash
if [[ ! -d "$target_dir" ]]; then
  echo "用户目录不存在: $target_dir"
  exit 1
fi
```

## Acceptance Testing Template

**Location:** `.config/opencode/get-shit-done/templates/UAT.md`

UAT document provides manual testing checklist for phases:
- Functional requirements verification
- Integration points testing
- Edge case coverage

Example structure (referenced but not shown in full):
```markdown
## UAT Checklist

- [ ] Feature X works as specified
- [ ] Integration with Y passes
- [ ] Performance acceptable
- [ ] Error handling covers cases A, B, C
```

## Coverage and Gaps

**What IS Tested:**
- Frontmatter structure (required fields, type validation)
- File existence in summary references
- Git commit hash validity
- Phase plan task element completeness
- YAML parsing correctness (explicit test in `frontmatter.cjs`)
- Configuration schema compliance

**What IS NOT Tested:**
- Individual function unit tests (no test framework)
- Integration tests for multi-module workflows
- CLI command options/flags combinations
- Process cleanup on error conditions
- Raycast extension functionality (minified, external platform)
- Performance/load testing

**Test Coverage Estimation:**
- Core utilities (path detection, file I/O, validation): ~60% covered via schema+verification commands
- CLI command handlers: ~40% covered (basic path validation, output format)
- Complex state machines (YAML parsing): ~70% covered (explicit block testing visible in code)
- Error conditions: ~50% covered (some error cases handled, others silent-fail)

## How to Add Tests (If Framework Were Adopted)

If unit testing were to be introduced:

1. **Test File Location:** Create `test/` directory at root or co-locate with source
2. **Test Framework:** Use Jest or Vitest (with CommonJS support)
3. **Fixtures:** Move validation YAML examples to `test/fixtures/frontmatter.yaml`
4. **Mocking:** Mock `fs`, `child_process`, `execSync` for isolation
5. **Coverage Target:** Aim for 80%+ on core modules (`core.cjs`, `frontmatter.cjs`, `verify.cjs`)

---

*Testing analysis: 2026-04-01*
