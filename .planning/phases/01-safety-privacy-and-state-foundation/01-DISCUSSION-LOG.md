# Phase 1: Safety, Privacy, and State Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-10
**Phase:** 1-Safety, Privacy, and State Foundation
**Areas discussed:** Artifact identity and lifecycle, Determinate Nix/Home Manager control plane clarification, Privacy violation handling, Fixture and opt-in integration experience, Sentinel proof and final verdict

---

## Artifact Identity and Lifecycle

### Artifact structure contract

| Option | Description | Selected |
|--------|-------------|----------|
| Common envelope plus kind-specific schemas | Share identity/version/provenance handling while validating each artifact payload against a distinct schema. | ✓ |
| Six independent root structures | Maximize visible separation but duplicate common metadata and validation behavior. | |
| One generic schema with optional fields | Minimize files but weaken type boundaries and allow ambiguous partial records. | |

**User's choice:** Common envelope plus kind-specific schemas.
**Notes:** Similar fields must never let desired, observed, plan, receipt, verification, or report substitute for one another.

### Persistence lifecycle

| Option | Description | Selected |
|--------|-------------|----------|
| Kind-specific lifecycle | Keep real run artifacts in ignored local state; retain each kind according to its evidence semantics; permit only synthetic privacy-checked fixtures in Git. | ✓ |
| Persist a complete bundle for every run | Maximize audit continuity but accumulate machine-state artifacts and privacy burden. | |
| Keep all generated artifacts ephemeral unless explicitly exported | Minimize persistence but risk losing exact plans, receipts, and evidence. | |

**User's choice:** Kind-specific lifecycle.
**Notes:** Public desired sources remain tracked; generated machine/run artifacts do not become Git truth.

### Invalid artifact handling

| Option | Description | Selected |
|--------|-------------|----------|
| Reject the complete artifact and stop | Return non-zero with bounded structured diagnostics; produce no partial success evidence. | ✓ |
| Keep valid fields and produce a degraded artifact | Continue with partial/unknown content, increasing the chance downstream code trusts incomplete evidence. | |
| Reject but preserve the raw artifact in quarantine | Aid debugging while persisting unvalidated paths, identity, or secrets. | |

**User's choice:** Reject the complete artifact and stop.
**Notes:** Raw invalid content is not retained; diagnostics must be independently safe.

### Evidence lineage

| Option | Description | Selected |
|--------|-------------|----------|
| Strict digest lineage graph | Bind each downstream artifact to exact allowed upstream digests and use a separate explicit lineage for no-apply runs. | ✓ |
| Run ID association only | Easy to read but cannot detect replacement or mutation within one run directory. | |
| Select the latest compatible artifact automatically | Convenient but may mix repository revisions, profiles, or machine states. | |

**User's choice:** Strict digest lineage graph.
**Notes:** `run_id` is informational. Latest-file discovery and directory co-location are never integrity evidence.

---

## Determinate Nix and Home Manager Control Plane Clarification

The discussion was temporarily paused when the user captured a project-wide clarification: the Mac should be managed overall through Determinate Nix and Home Manager. Current repository code and official tool boundaries were checked before resuming.

### Meaning of overall management

| Option | Description | Selected |
|--------|-------------|----------|
| Primary Nix-based control plane with delegated unique owners | Determinate Nix, nix-darwin, and Home Manager own composition and manager entrypoints; Homebrew/mise/uv/rustup/wrappers may own explicitly delegated payloads. | ✓ |
| Direct Nix/Home Manager ownership of every package and runtime | Eliminate delegated managers and put every executable/payload directly in the Nix store. | |
| Custom split | Define another boundary per component. | |

**User's choice:** Primary Nix-based control plane with delegated unique owners.
**Notes:** The user asked whether Home Manager modules for mise/uv and Nix modules for Homebrew imply the second option. The distinction was clarified: a module may install/configure a manager or invoke it during activation while the mutable downstream payload remains owned by that manager. Ownership must be recorded separately for declaration, manager binary, payload, selected executable, and activation context.

### Module implications reviewed

| Component | Declarative layer | Downstream ownership conclusion |
|-----------|-------------------|---------------------------------|
| Homebrew | nix-darwin declares inventory and activation; optional nix-homebrew can manage the Homebrew installation itself. | Homebrew still installs and owns formula/cask payload state. |
| mise | Home Manager can install the Nix-built binary, write global config, and add shell activation. | mise still owns runtimes it downloads/selects unless a scope explicitly chooses a Nix package/devShell instead. |
| uv | The repository's locked Home Manager module installs uv and writes `uv.toml`; newer upstream modules can also invoke uv install/prune during activation. | uv remains payload owner; activation downloads/pruning remain a separate mutable write boundary. |

---

## Privacy Violation Handling

### Default privacy response

| Option | Description | Selected |
|--------|-------------|----------|
| Allowlisted structural conversion, otherwise hard fail | Convert only registered identities/roots to logical references; stop before output for secret, raw, unknown, or unclassifiable data. | ✓ |
| Mask sensitive fragments and continue | Preserve flow with `[REDACTED]`-style substitutions but risk incomplete masking and invalid data presented as success. | |
| Drop fields silently and downgrade to unknown | Reduce failures but hide adapter privacy regressions and lose operator awareness. | |

**User's choice:** Allowlisted structural conversion, otherwise hard fail.
**Notes:** Generic masking is not an authorization to persist or treat data as valid.

### Rejection diagnostics

| Option | Description | Selected |
|--------|-------------|----------|
| Stable error code and logical location only | Include artifact kind, adapter, logical pointer, violation category, and remediation without any content-derived detail. | ✓ |
| Include a masked sample or content hash | Improve debugging but retain correlatable prefixes, lengths, or fingerprints. | |
| Show only a generic privacy error | Minimize disclosure but provide insufficient localization for safe maintenance. | |

**User's choice:** Stable error code and logical location only.
**Notes:** The diagnostic envelope itself must pass schema and privacy validation.

### Raw stdout/stderr

| Option | Description | Selected |
|--------|-------------|----------|
| Bounded capture and in-memory parse | Capture through pipes with time/size limits, emit normalized facts only, and discard raw bytes. | ✓ |
| Persist raw failures in an ignored debug directory | Ease debugging but accumulate private machine data outside Git. | |
| Allow raw terminal output but prohibit it in artifacts | Avoid artifact leaks while still exposing secrets to terminal scrollback and wrappers. | |

**User's choice:** Bounded capture and in-memory parse.
**Notes:** Only synthetic fixtures may retain synthetic raw samples. Parse/size failures return `unknown` or privacy error without raw output.

### Logical paths and identity

| Option | Description | Selected |
|--------|-------------|----------|
| Registered logical namespaces | Resolve real roots locally and persist only `repo:`, `home:`, `fixture:`, `local-state:`, `nix-output:`, or public logical-profile references. | ✓ |
| Tilde/basename/path trimming | Improve readability but retain project/path clues and introduce collisions. | |
| Stable hashes or salted fingerprints | Hide source text but create cross-report identity/linkability and salt-management problems. | |

**User's choice:** Registered logical namespaces.
**Notes:** Unknown absolute references fail closed; no real-value hashes are persisted.

---

## Fixture and Opt-in Integration Experience

### Test tiers

| Option | Description | Selected |
|--------|-------------|----------|
| Three explicit tiers with no automatic escalation | Separate default offline synthetic tests, explicit isolated integration, and separate allowlisted live-check. | ✓ |
| Include live probes in the default test | Provide more evidence in one command while coupling default tests to real machine state. | |
| Smart runner chooses the strongest available tier | Simplify UX but execute different side-effect levels on different Macs. | |

**User's choice:** Three explicit tiers with no automatic escalation.
**Notes:** Failure or missing capability never upgrades to a more privileged tier.

### Fixture root construction

| Option | Description | Selected |
|--------|-------------|----------|
| Tracked blueprint plus fresh external temporary root | Keep synthetic definitions in Git and create a minimal isolated HOME/XDG/TMPDIR/PATH/manager environment per run. | ✓ |
| Gitignored repository-local `.tmp` sandbox | Make failures easy to inspect but disturb worktree-adjacent state and watchers. | |
| Container or VM for every test | Maximize isolation but lose native macOS representativeness or incur excessive cost. | |

**User's choice:** Tracked blueprint plus fresh external temporary root.
**Notes:** The real worktree supplies tracked inputs only and is not a writable fixture.

### Fixture retention

| Option | Description | Selected |
|--------|-------------|----------|
| Delete by default; retain only with pre-run opt-in | Keep only explicitly requested synthetic/integration roots with logical ID, TTL, and ownership marker. | ✓ |
| Retain failures by default | Aid debugging but accumulate caches, downloads, and temporary state. | |
| Always delete immediately | Minimize state but make isolated integration failures difficult to inspect safely. | |

**User's choice:** Delete by default; retain only with pre-run opt-in.
**Notes:** Cleanup can delete only marked content inside the dedicated local fixture root; live raw output is never retained.

### Network and downloads

| Option | Description | Selected |
|--------|-------------|----------|
| Exact-test authorization with manifest | Predeclare purpose, integrity, byte/time limits, and isolated cache; authorize only the exact test ID. | ✓ |
| One global allow-network switch | Easy to use but grants future/new tests broad inherited access. | |
| Auto-download on cache miss | Match normal manager behavior but make cold and warm runs differ and reintroduce implicit install. | |

**User's choice:** Exact-test authorization with manifest.
**Notes:** No credential, Keychain, proxy, or token inheritance. Unenforceable egress/integrity returns `manual-required`.

---

## Sentinel Proof and Final Verdict

### Protected-surface scope

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit protected-surface manifest with domain sentinels | Observe approved privacy-safe metadata/fingerprints for named worktree, HOME, manager, service, and external targets. | ✓ |
| Recursively hash the whole HOME and repository | Appear comprehensive while reading secrets, caches, and noisy mutable state. | |
| Check Git worktree only | Detect repository changes but miss HOME, manager, service, and external state. | |

**User's choice:** Explicit protected-surface manifest with domain sentinels.
**Notes:** Every test declares surfaces it could touch. Unlisted surfaces are not covered by the proof.

### Verdict for missing or unsafe observation

| Option | Description | Selected |
|--------|-------------|----------|
| Strict four-state verdict | `passed`, `violation`, `indeterminate`, or `harness-error`; required uncertainty blocks success. | ✓ |
| Warn on unknown and otherwise pass | Reduce false failures but allow success without required real-state evidence. | |
| Report coverage percentage | Offer a simple number that incorrectly treats surfaces as equal and resembles a safety probability. | |

**User's choice:** Strict four-state verdict.
**Notes:** Only optional sentinels may warn without blocking; every non-passed required state returns non-zero.

### Concurrent/background changes

| Option | Description | Selected |
|--------|-------------|----------|
| Change means violation without attribution | Fail the current proof window but do not claim the test caused the change; never auto-restore/ignore/retry. | ✓ |
| Attribute by PID/process tree | Try to ignore unrelated writes despite incomplete helper/IPC/filesystem attribution. | |
| Retry until a stable window | Hide the initial changed run and execute potentially unsafe behavior multiple times. | |

**User's choice:** Change means violation without attribution.
**Notes:** Noisy targets must be narrowed, optional, or excluded before the run, never after observing a failure.

### Strongest Phase 1 claim

| Option | Description | Selected |
|--------|-------------|----------|
| `covered-surfaces-unchanged-for-run` | Bind the claim to exact suite/tier/manifest/window/snapshots and preserve exclusions. | ✓ |
| “The test did not change the Mac” | Overclaim across unobserved or excluded surfaces. | |
| Permanently certify a test after one pass | Ignore drift in code, tools, official semantics, and host behavior. | |

**User's choice:** `covered-surfaces-unchanged-for-run`.
**Notes:** `recovery-ready-on-current-host` is reserved for Phase 13; clean-host claims remain future work.

---

## Agent's Discretion

- Exact implementation language and minimal dependency set for schemas, harness, adapters, and sentinels.
- Digest algorithm, exact envelope field names, schema file split, error code names, and ignored local-state directory name.
- Fixture TTL, concurrency layout, byte/time ceilings, and fake-binary implementation.
- Exact minimal protected-surface adapters and privacy-safe fingerprint algorithms, within the locked coverage/verdict rules.
- Final CLI command names and human-readable renderer, while preserving the three explicit tiers and machine-readable contracts.

## Deferred Ideas

- Evaluate optional Determinate nix-darwin module and `nix-homebrew` in a later host/recovery phase.
- Evaluate `programs.mise` and `programs.uv` migrations in their ecosystem phases, preserving one writer per config and treating downloads/pruning as mutable operations.
- Build the ownership inspector in Phase 2, ecosystem contracts in Phases 3–8, multi-host composition in Phase 9, readiness in Phase 10, safe links in Phase 11, recovery apply in Phase 12, and current-host claim promotion in Phase 13.
- Validate `fresh-install-verified` only on a future clean macOS VM or second physical Mac.
