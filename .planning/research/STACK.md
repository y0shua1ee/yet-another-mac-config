# Stack Research

**Domain:** Reproducible macOS development environment and language-toolchain ownership
**Researched:** 2026-07-10
**Confidence:** HIGH for the ownership model; MEDIUM for exact future versions and mixed direnv/mise/Nix activation until isolated fixtures are exercised

## Executive Recommendation

Use a strict **one owner per `(scope, executable)`** model:

| Scope | Owner | Responsibility |
|---|---|---|
| Machine/bootstrap | nix-darwin + Home Manager | Reproducible system and home configuration, manager binaries, shell hooks |
| macOS applications and native utilities | Homebrew, declared through nix-darwin | GUI apps and tools whose best-supported macOS distribution is Homebrew |
| Default and project language runtimes | mise | Node.js, Go, Deno, Bun, and the project-selected JDK |
| Node package manager | mise by default | Exact pnpm version; npm remains bundled with Node |
| Python | uv | Python interpreter, virtual environment, dependency resolution, and Python CLI tools |
| Rust | rustup | Rust toolchains, components, targets, and Cargo proxies |
| JVM build tools | Project wrappers | Gradle Wrapper and Maven Wrapper, never a global Gradle/Maven install |
| Fully isolated project environments | Nix devShell | Exclusive owner of every tool declared by that devShell |
| Directory activation | mise activation, or direnv/nix-direnv for an exclusive Nix devShell | Activation only; never a package owner |

This model keeps Home Manager small and stable: it installs the tool managers and their shell integration, while project files declare the language versions. It also prevents the same runtime from being installed by Homebrew, Nix, mise, Corepack, and a language-specific manager simultaneously.

The current repository already points in this direction:

- Home Manager installs `mise`, `uv`, `rustup`, `direnv`, and `nix-direnv`.
- The current global mise fallback pins Node `24.11.0` and Go `1.26.3`.
- The Home Manager Zsh route activates mise, but the fallback Zsh route does not yet do so.
- A direct Bun path remains in the shared Zsh configuration, so Bun is still transitional rather than fully mise-owned.
- Homebrew intentionally does not declare global language runtimes.

Do not change any of those live settings as part of research. Migrate them only through a later check-plan-confirm-apply-verify phase.

## Version Policy

Version numbers in this document are a research snapshot, not an instruction to upgrade the current Mac.

| Component | Current upstream reference on 2026-07-10 | Project recommendation |
|---|---:|---|
| mise | `2026.7.4` | Keep the repository's current pin until a separately reviewed upgrade |
| Node.js | v24 LTS `24.17.0`; v26 Current `26.3.1` | Keep existing global fallback `24.11.0` for now; projects pin exact versions |
| npm | `11.18.0` documentation line | Use the npm bundled with the selected Node unless a project proves it needs another version |
| Corepack | `0.35.0` | Do not enable by default |
| pnpm | `11.4.x` current release line | mise-owned, exact project pin, committed mise lock |
| Go | `1.26.5` | Keep existing fallback `1.26.3` for now; projects pin exact versions |
| uv | `0.11.28` documentation example | Binary pinned through the flake; Python versions declared per project |
| rustup | `1.29.0` | Binary pinned through the flake; Rust toolchains declared per project |
| Rust | `1.96.0` stable | Prefer exact project toolchains, not floating `stable` |
| Deno | `2.8.1` | Add only for a project that needs it, with an exact mise pin |
| Bun | `1.3.14` | Add only for a project that needs it, with an exact mise pin |
| Temurin JDK | `25.0.3+9`, JDK 25 LTS | Select only after project compatibility inventory; retain 17/21 where required |
| Gradle | `9.6.1` documentation line | Wrapper-owned per project |
| Maven | `3.9.16` stable; Maven 4 remains pre-release | Wrapper-owned per project |
| direnv | `2.37.1` | Home Manager-owned activation binary |
| nix-direnv | `3.1.1` | Home Manager-owned integration |

Exact version drift is expected. Before implementing an upgrade, refresh upstream release data, check the repository lock files, and test in an isolated fixture.

## Recommended Stack

### Core Technologies

| Technology | Role | Why this owner | Confidence |
|---|---|---|---|
| nix-darwin | Machine-level declarative orchestration | Already the repository's activation boundary; integrates macOS settings, Home Manager, and Homebrew declarations | HIGH |
| Home Manager | User-level manager binaries and shell integration | Reproducibly installs the manager entrypoints without claiming ownership of every runtime | HIGH |
| Homebrew through nix-darwin | GUI apps and macOS-native utilities | Broad macOS support while keeping declarations in the canonical repository | HIGH |
| mise | Node, Go, Deno, Bun, and JDK runtime selection | One cross-language project contract, exact pins, activation, and lock support | HIGH |
| uv | Python runtime and dependency stack | Python-specific, fast, lockfile-aware, and able to own interpreter plus environment coherently | HIGH |
| rustup | Rust toolchains and components | The Rust project's recommended toolchain installer and the canonical owner for targets/components | HIGH |
| Gradle Wrapper / Maven Wrapper | JVM build-tool distribution | Keeps build-tool version with the project and avoids global tool ambiguity | HIGH |
| Nix devShell | Opt-in fully isolated project environment | Strong reproducibility when a project needs native libraries or an environment wider than language versions | HIGH |
| direnv + nix-direnv | Optional activation for Nix devShells | Convenient directory activation, provided it does not also manipulate mise's tool PATH | MEDIUM |

### Supporting Contracts

| Ecosystem | Required project contract | Lock/frozen contract |
|---|---|---|
| Node + pnpm | `mise.toml` or `.mise.toml` with exact Node and pnpm; `package.json` with exact package-manager expectation | `mise.lock` and `pnpm-lock.yaml`; frozen install |
| Node + npm | Exact Node in mise; `package.json` | `mise.lock` and `package-lock.json`; `npm ci` |
| Go | Exact Go in mise; `go.mod`; tracked `GOTOOLCHAIN=local` policy | `mise.lock` and `go.sum` |
| Python | `pyproject.toml` with `requires-python`; exact `.python-version` | `uv.lock`; `uv sync --frozen` |
| Rust | Exact `rust-toolchain.toml`; `rust-version` in `Cargo.toml` | `Cargo.lock`; locked Cargo operations |
| Deno | Exact Deno in mise; `deno.json` | `mise.lock` and `deno.lock`; `deno ci` |
| Bun | Exact Bun in mise; `package.json` and optional `bunfig.toml` | `mise.lock` and `bun.lock`; `bun ci` |
| Gradle | Exact JDK in mise; project language/toolchain target; wrapper files | Wrapper distribution checksum plus dependency locking where the project uses it |
| Maven | Exact JDK in mise; compiler `release`/toolchain contract; wrapper files | Wrapper version plus dependency reproducibility controls appropriate to the project |
| Nix devShell | `flake.nix` or equivalent shell expression | `flake.lock` |

### Development Tools

| Tool | Purpose | Ownership rule |
|---|---|---|
| Repository doctor | Read-only ownership, path, pin, lockfile, and drift report | Implemented in this repository; must never install or switch |
| Static ownership lint | Detect duplicate managers and undeclared mutable paths before activation | Implemented in this repository and suitable for CI |
| Isolated ecosystem fixtures | Prove hydration and frozen installs without touching the real home | Temporary state directories only |
| `mise exec` | Reproducible non-interactive execution | mise-owned projects only |
| `uv run` / `uv sync` | Python execution and environment synchronization | uv-owned projects only |
| Cargo locked operations | Rust metadata, build, and test verification | rustup-selected project toolchain |
| Gradle Wrapper / Maven Wrapper | JVM build entrypoints | Committed and invoked from the project |
| `nix develop` | Explicit entry into an exclusive Nix environment | Nix-devShell projects only |

## Ownership Matrix

This matrix is normative. If implementation discovery finds a second owner, migration should remove or disable the duplicate only after isolated verification.

| Executable / state | Global fallback owner | Project owner | Explicitly excluded owners |
|---|---|---|---|
| `node` | mise | mise | Homebrew, Nix profile packages, Corepack |
| `npm` | Selected Node distribution | Selected Node distribution, unless explicitly pinned in mise for compatibility | Homebrew, Corepack interception |
| `pnpm` | Prefer no global fallback; otherwise exact mise fallback | mise | Corepack by default, Homebrew, pnpm self-management |
| `corepack` | None by default | Compatibility variant only | Node-bundled Corepack as an assumed permanent dependency |
| `go` | mise | mise | Homebrew Go, Nix profile Go, automatic Go toolchain download |
| `python` / `python3` | None unless explicitly declared later | uv-managed interpreter | mise, pyenv, Homebrew Python, global user pip |
| `uv` / `uvx` | Home Manager | Same binary | Self-updated installer copy, Homebrew |
| `rustup` | Home Manager | Same binary | Homebrew |
| `rustc` / `cargo` | rustup | rustup selected by `rust-toolchain.toml` | mise, Homebrew Rust, Nix profile Rust |
| `deno` | None initially | mise | Homebrew, `deno upgrade` |
| `bun` | None after migration | mise | Homebrew, direct `~/.bun` install, `bun upgrade` |
| `java` / `javac` | None until need is proven | mise-selected JDK | Homebrew JDK, Nix profile JDK |
| `gradle` | None | Gradle Wrapper | Homebrew, mise, standalone Nix profile Gradle |
| `mvn` | None | Maven Wrapper | Homebrew, mise, standalone Nix profile Maven |
| `direnv` | Home Manager | Same binary | Homebrew |
| devShell tools | None | Nix devShell, exclusively | Concurrent mise ownership for the same project tools |

## Ecosystem Decisions

### Node.js, npm, pnpm, and Corepack

#### Recommended owner

- mise owns Node.js and pnpm.
- The selected Node distribution owns npm.
- Corepack is disabled or simply absent by default.

#### Project contract

For a pnpm project:

1. Declare exact Node and pnpm versions in `mise.toml` or `.mise.toml`.
2. Commit `mise.lock` so resolved artifacts, URLs, and checksums are reviewable where the backend supports them.
3. Commit `pnpm-lock.yaml`.
4. On pnpm 11 projects, declare an exact `devEngines.packageManager` expectation with `onFail: "error"`. Use `engines.pnpm` when older tooling requires it.
5. Do not declare pnpm's `devEngines.runtime` or `engines.runtime` download feature; mise already owns Node, Deno, and Bun runtimes.

For an npm project:

1. Pin exact Node in mise.
2. Use the npm bundled with that Node by default.
3. Commit `package-lock.json` and use `npm ci` in reproducible checks.
4. Pin a separate npm version in mise only when a demonstrated lockfile or CLI compatibility issue requires it.

#### Why Corepack is not the default

- Corepack remains experimental.
- Node stopped distributing Corepack beginning with Node 25.
- Current Corepack `0.35.0` declares a Node engine of `^22.22.2 || ^24.15.0 || >=26.0.0`, while the repository's current fallback is Node `24.11.0`.
- Corepack maintains Known Good Releases and a cache that can mutate unless carefully constrained.
- mise already provides a direct, exact, lockable pnpm installation path through its registry.

#### Compatibility variant

If a project explicitly requires Corepack behavior:

- Install a standalone, exact Corepack version declaratively through Nix/Home Manager.
- Remove pnpm ownership from mise for that project.
- Commit an exact `packageManager` value, including the integrity hash where supported.
- Set `COREPACK_DEFAULT_TO_LATEST=0` to prevent implicit Known Good Release updates.
- Verify the Corepack binary's Node runtime compatibility in an isolated fixture.

Do not rely on the copy historically bundled in Node as a long-term contract.

#### Activation and verification

- Interactive shells: full `mise activate zsh` integration.
- Non-interactive commands: `mise exec -- ...` or deliberately configured mise shims.
- Prefer full activation for normal terminal use because shims do not reproduce all environment variables or hooks.
- Verify `command -v`, `mise which node`, `mise which pnpm`, version output, exact project declarations, and frozen dependency installation in an isolated cache.

### Go

#### Recommended owner

mise is the sole owner of the Go runtime.

#### Project contract

- Exact Go version in mise plus committed `mise.lock`.
- `go.mod` and `go.sum` remain the Go module contract.
- Treat the `go` directive as the minimum language/toolchain requirement.
- If a `toolchain` directive is used, keep it consistent with the exact mise version.
- Declare `GOTOOLCHAIN=local` in tracked project environment configuration so the Go command fails clearly instead of downloading a second toolchain.

Never use `go env -w` for repository policy because it writes hidden machine-global state.

#### Fallback

Keep the existing `1.26.3` fallback until a separately approved upgrade. It is in the same supported Go 1.26 line as the researched current `1.26.5`.

#### Activation and verification

- mise activation or `mise exec` selects Go.
- Verify the resolved binary path, `go env GOTOOLCHAIN`, `go version`, module integrity, and tests with isolated `GOCACHE`, `GOMODCACHE`, and `GOPATH`.
- A doctor/check phase must not allow Go to auto-download an undeclared toolchain.

### Python and uv

#### Recommended owner

- Home Manager owns the `uv` and `uvx` entrypoint binaries.
- uv exclusively owns Python interpreters, virtual environments, locked dependencies, and uv-managed Python CLI tools.

Do not also manage Python through mise. A single Python-specific manager is clearer than sharing interpreter ownership between mise and uv.

#### Project contract

- `pyproject.toml` with an explicit `requires-python` range.
- Exact `.python-version` matching the intended interpreter.
- Committed `uv.lock` for applications and other projects that need reproducible dependency resolution.
- Local `.venv` remains untracked.
- Set `python-preference = "only-managed"` and `python-downloads = "manual"` at the appropriate declarative scope. This prevents unexpected use of system Python and unexpected interpreter downloads during a read-only check.

#### Fallback

Do not add an unversioned global Python fallback initially. Python projects should declare their own version; one-off tools should use uv's tool mechanism with declarative documentation where persistence matters.

#### Activation and verification

- Use `uv run` and `uv sync --frozen` rather than manually activating and mutating a shared virtual environment.
- Verify interpreter source, exact version, lock consistency, and frozen sync using isolated uv cache, Python install, and virtual-environment directories.
- No global `pip install --user`, pyenv, Homebrew Python, or mise Python.

### Rust and rustup

#### Recommended owner

- Home Manager owns the rustup executable.
- rustup owns Rust toolchains, `rustc`, Cargo, components, targets, and proxy selection.

#### Project contract

- Commit `rust-toolchain.toml` with an exact toolchain version and only required components/targets.
- Set `rust-version` in `Cargo.toml` as the project's minimum supported Rust version.
- Commit `Cargo.lock` for binaries/applications, and follow the project's library policy for libraries.
- Prefer rustup's `default` profile; avoid `complete`.

Do not use a floating `stable` channel when bit-for-bit recovery matters.

#### Fallback

Do not add a new default Rust toolchain merely because rustup is installed. Introduce an exact fallback only after inventory proves that toolchainless directories need it.

#### Activation and verification

- rustup selects the project toolchain from `rust-toolchain.toml`.
- The rustup executable itself is updated through the flake/Home Manager, not `rustup self update`.
- Toolchain downloads are allowed only in a confirmed apply/hydration phase, never in check or doctor.
- Verify with isolated `RUSTUP_HOME` and `CARGO_HOME`, then run locked metadata/build/test commands.

### Deno

#### Recommended owner

mise owns Deno when a project needs it.

#### Project contract

- Exact Deno in mise and committed `mise.lock`.
- `deno.json` for project configuration.
- Committed `deno.lock` with an appropriate frozen lock setting.
- Use `deno ci` for reproducible dependency installation/checking.

#### Fallback and exclusions

No global Deno fallback initially. Do not install it with Homebrew and do not use `deno upgrade` when mise owns the binary.

#### Verification

Verify the mise-resolved path, version, config discovery, and frozen lock behavior with an isolated `DENO_DIR`.

### Bun

#### Recommended owner

mise owns Bun when a project needs it.

#### Project contract

- Exact Bun in mise and committed `mise.lock`.
- `package.json` plus optional `bunfig.toml`.
- Committed text `bun.lock`.
- Use `bun ci` or the documented frozen install path.

#### Migration note

The repository currently exposes a direct Bun path. Treat it as legacy state to inventory and safely migrate later. Do not delete or replace it during research.

#### Fallback and exclusions

No global Bun fallback initially. After migration, avoid Homebrew Bun, a separately installed `~/.bun` distribution, and `bun upgrade`.

#### Verification

Verify the mise-resolved path and frozen install with isolated `BUN_INSTALL` and cache directories. The fixture must not read or modify the user's existing `~/.bun`.

### Java, JVM, Maven, and Gradle

#### Recommended owner

- mise owns project JDKs.
- Prefer an explicit vendor and exact patch, such as Temurin, rather than an ambiguous major-only Java pin.
- Gradle Wrapper owns Gradle.
- Maven Wrapper owns Maven.

JDK 25 is an LTS candidate for new compatible projects, but it must not become the universal fallback before existing projects are inventoried. Java 17 and 21 may still be required.

#### Project contract

For both Maven and Gradle:

- Exact JDK vendor/version in mise plus committed `mise.lock`.
- Build configuration declares the Java language/release target.
- Commit wrapper scripts, wrapper metadata, and the wrapper JAR according to official wrapper guidance.
- Review and pin the wrapper distribution checksum.

For Gradle:

- Use `./gradlew`, never a global `gradle`.
- Disable Gradle's JDK auto-download when mise owns the JDK.
- Full mise activation is required because mise sets `JAVA_HOME`; shims alone do not.
- Current compatibility guidance says Java 25 can run Gradle from 9.1 and Java 26 from 9.4, but every existing wrapper must be checked before selecting a JDK.

For Maven:

- Use `./mvnw`, never a global `mvn`.
- Use the compiler plugin's `release` setting.
- Maven 3.9.x remains the stable baseline; do not standardize on Maven 4 while it is pre-release.

#### Multi-JDK exception

Single-JDK projects should rely on the mise-activated `JAVA_HOME`. For a project that genuinely builds against several JDKs:

1. Declare all exact JDKs in project mise configuration and generate any host-path mapping into an ignored local file from a non-secret tracked manifest; or
2. Move the entire project environment to an exclusive Nix devShell.

Never commit absolute paths from `~/.m2/toolchains.xml` or machine-local Gradle installation paths.

#### Fallback

No global Java, Gradle, or Maven fallback initially. Add an exact JDK fallback only after inventory proves a machine-level need; never add global Gradle or Maven.

### direnv, nix-direnv, and Nix devShell

#### Default mise project

- mise activation owns runtime PATH changes.
- Do not use direnv to add mise runtimes to PATH.
- Do not use the deprecated `use mise` integration.
- direnv may still load unrelated, non-secret project environment variables, but a native mise environment file is simpler when mise already owns activation.

#### Nix devShell project

A Nix devShell is an **exclusive project variant**:

- Nix owns every runtime and build tool declared by the devShell.
- The same project must not also declare those executables in mise.
- `flake.lock` pins the environment.
- Explicit `nix develop` is the clearest initial activation and easiest to debug.

If automatic activation is desired:

- Use Home Manager's direnv + nix-direnv integration.
- Keep `.envrc` minimal, normally `use flake`.
- Require explicit `direnv allow`.
- Prefer manual reload behavior and disallow stale fallback environments.
- Consider project-level mise `enable_tools = []` to suppress inherited global mise runtimes, but validate hook ordering in an isolated fixture first.

mise officially discourages combining mise and direnv for PATH management. Therefore automatic nix-direnv plus globally activated mise is a MEDIUM-confidence integration until the project fixture proves that Nix's tools exclusively win for the project and are removed cleanly on exit.

#### Ownership

direnv is an activation mechanism, not a package manager. It must never become a second owner of runtimes.

## Global Fallback Policy

The global fallback is for entering an undeclared directory, not for overriding a project's contract.

1. Keep it minimal.
2. Pin exact versions; do not use `latest`, `lts`, `stable`, or major-only aliases.
3. Commit the mise config and lock data.
4. A project declaration always overrides the global fallback.
5. Do not add a fallback merely because a manager binary is present.
6. Do not silently auto-install a missing project version during check, doctor, shell startup, or verification.
7. Hydration is an apply-stage action that requires the normal confirmation gate.

Initial fallback recommendation:

| Tool | Fallback |
|---|---|
| Node | Preserve current `24.11.0` until controlled migration |
| Go | Preserve current `1.26.3` until controlled migration |
| pnpm | Prefer none; add exact version only if frequent undeclared-directory use is proven |
| Python | None |
| Rust toolchain | None |
| Deno | None |
| Bun | None after safe migration of the current direct install |
| Java | None |
| Gradle / Maven | Never |

## Activation Model

| Context | Activation | Notes |
|---|---|---|
| Home Manager-managed interactive Zsh | `mise activate zsh` from declarative shell config | Canonical interactive route |
| Repository fallback Zsh | Add the same mise activation only in a later tested migration | Current route does not activate mise |
| Script / CI | `mise exec -- command ...` or intentionally provisioned shims | Do not assume interactive shell hooks |
| Python project | `uv run` / `uv sync` | Avoid shared global activation |
| Rust project | rustup proxy resolution from `rust-toolchain.toml` | Exact toolchain contract |
| Explicit Nix project | `nix develop` | Nix exclusively owns project tools |
| Automatic Nix project | direnv + nix-direnv after isolated validation | Keep `.envrc` minimal |

Shell startup must remain non-destructive: no downloads, installs, upgrades, lockfile rewrites, or trust prompts.

## Safe Verification Strategy

Verification must prove reproducibility without changing the current Mac's environment.

### 1. Static ownership lint

Parse the repository's declarative sources and fail if:

- The same executable is declared by more than one global owner.
- A project declares an exclusive Nix devShell tool and the same tool in mise.
- Homebrew gains a language runtime, package manager, Gradle, or Maven without an explicit exception.
- A project version is floating rather than exact.
- A required project lockfile is missing or unexpectedly modified.
- A shell file adds direct manager-owned install paths such as a legacy Bun path.

This phase is read-only.

### 2. Live read-only doctor

Report, without installing anything:

- `command -v` and `type -a` results for each managed executable.
- mise's configured and resolved versions using non-installing inspection commands.
- uv, rustup, direnv, Nix, and Homebrew binary origins.
- Duplicate paths and unexpected Homebrew or user-profile runtimes.
- Whether interactive and non-interactive activation agree.

If an inspection command could download or install a missing tool, do not run it. Report the unverified item instead.

### 3. Isolated fixture

Use a temporary home and dedicated state/cache roots, including as applicable:

- `HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, and `XDG_CACHE_HOME`
- `MISE_DATA_DIR` and `MISE_CACHE_DIR`
- uv Python, cache, and tool directories
- `RUSTUP_HOME` and `CARGO_HOME`
- `COREPACK_HOME` only for the compatibility variant
- `GOCACHE`, `GOMODCACHE`, and `GOPATH`
- `DENO_DIR`
- `BUN_INSTALL` and Bun cache
- `GRADLE_USER_HOME`
- an isolated Maven home/config

The fixture may hydrate tools only as an explicitly confirmed test/apply action. It must never read, replace, or clean the user's existing manager state.

### 4. Ecosystem checks in the fixture

| Ecosystem | Verification |
|---|---|
| npm | `npm ci` against a tiny committed fixture |
| pnpm | Frozen-lockfile install with exact mise-resolved pnpm |
| Go | `go mod verify` and tests with `GOTOOLCHAIN=local` |
| Python | `uv sync --frozen` and `uv run` |
| Rust | locked Cargo metadata/build/test with isolated rustup/Cargo homes |
| Deno | `deno ci` plus project checks |
| Bun | `bun ci` plus project checks |
| Gradle | Wrapper checksum/version and a small wrapper build with the declared JDK |
| Maven | Wrapper version and a small wrapper build with the declared JDK |
| Nix | Locked devShell evaluation and PATH ownership assertion |

### 5. Activation smoke test

Use a spawned shell with temporary startup files to prove:

- The Home Manager path activates mise once.
- The fallback path behaves as documented.
- `JAVA_HOME` matches the selected mise JDK.
- Leaving a Nix devShell restores the prior environment.
- direnv trust remains explicit.
- No startup path triggers network access or writes outside the temporary fixture.

Only after these checks pass should a later plan propose changing the real machine.

## Installation

### Declarative Provisioning Target

The desired implementation shape is:

1. `flake.lock` pins nix-darwin, Home Manager, nixpkgs, and manager binaries.
2. Home Manager declares only manager binaries and shell hooks.
3. nix-darwin's Homebrew module declares GUI/native packages and retains conservative cleanup behavior.
4. A tracked global mise config provides the minimal fallback.
5. Each project commits its runtime contract and ecosystem lockfiles.
6. A machine-local private layer contains credentials and secrets only; public configuration references environment-variable names or secret-manager entries, never values.
7. Check and doctor commands are read-only.
8. Apply performs approved hydration and activation.
9. Verification uses isolated state before real-home activation.

No package-manager self-update command should be part of the steady-state workflow for a binary owned by Nix, Home Manager, mise, uv, or rustup.

## Alternatives Considered

### Let mise own every language, including Python and Rust

**Not recommended.** It simplifies the number of manager entrypoints but loses the strongest ecosystem-native ownership:

- uv coherently owns Python downloads, lock resolution, environments, and tools.
- rustup coherently owns Rust components, targets, profiles, and project overrides.

Using mise for those runtimes while uv/rustup remain installed creates ambiguous state and harder recovery.

### Let Nix own every global runtime

**Not the default.** It is highly reproducible, but project developers and upstream tooling often expect native managers and project files. Use Nix devShell as an exclusive opt-in where native libraries or complete isolation justify it.

### Let Homebrew own language runtimes

**Rejected.** Homebrew version availability and upgrade cadence are machine-global and do not naturally encode per-project versions. Keep Homebrew focused on macOS-native software.

### Use Corepack as the universal Node package-manager router

**Rejected as the default.** Its experimental status, removal from Node distributions beginning with Node 25, runtime compatibility constraints, and mutable Known Good Release behavior add a second ownership layer. Keep it as a documented compatibility variant only.

### Use direnv to activate mise

**Rejected.** mise's official guidance says not to combine the two for PATH management, and `use mise` is deprecated. Use mise's own shell activation.

### Install Gradle and Maven globally through mise

**Rejected.** Official wrappers are the project-level distribution contract and avoid global build-tool drift.

## What Not to Use

- Floating runtime aliases such as `latest`, `lts`, `stable`, or major-only Java pins in reproducibility-critical config.
- Homebrew language runtimes, global Gradle, or global Maven without a documented exception.
- Duplicate Node, Go, Python, Rust, Deno, Bun, or Java owners.
- pnpm runtime auto-download features when mise owns runtimes.
- Corepack's implicit latest/Known Good Release mutation.
- Go's automatic toolchain download for managed projects.
- uv's automatic Python download during a read-only check.
- `rustup self update` for a Nix-owned rustup binary.
- `deno upgrade` or `bun upgrade` for mise-owned binaries.
- `go env -w` for repository policy.
- `pip install --user` or shared mutable global Python environments.
- Global `gradle` or `mvn` commands instead of wrappers.
- Committed absolute Java installation paths.
- Automatic direnv trust, stale fallback devShells, or mixed mise/direnv PATH ownership.
- Shell-startup downloads, installs, upgrades, cleanup, or lockfile generation.
- Tests that reuse or clean the user's actual manager caches and homes.

## Compatibility Notes

### Corepack and Node

The current standalone Corepack `0.35.0` engine range excludes Node `24.11.0` but accepts Node `24.15.0` and newer within Node 24. This is a concrete reason not to assume Corepack can run under the repository's current Node fallback. If the compatibility variant is implemented, test the actual Nix-packaged binary/runtime pair.

### pnpm and Node

pnpm 11 supports Node 22, 24, and 26. Exact project pins are still required because package-manager lockfile behavior can change between pnpm releases.

### Java and Gradle

Gradle's compatibility matrix distinguishes running Gradle from compiling/testing with a Java toolchain. Current documentation says Gradle 9.1 can run on Java 25 and Gradle 9.4 can run on Java 26. Older wrappers may require an older JDK even when source code targets a newer language level.

### mise Java activation

mise sets `JAVA_HOME` under full shell activation. Shims alone are insufficient for tools that discover Java through `JAVA_HOME`, including common Gradle workflows.

### uv Python support

uv currently treats Python 3.10 through 3.14 as Tier 1 and older 3.6 through 3.9 as Tier 2/EOL. Existing projects on older Python require an explicit compatibility exception rather than a silent global fallback.

### nix-direnv requirements

The researched nix-direnv release expects modern Bash, Nix, and direnv versions. The repository's flake pins must be checked together rather than upgrading only one integration component.

## Sources

All sources are official project documentation or upstream repositories.

### mise

- [mise configuration](https://mise.jdx.dev/configuration.html)
- [mise activation](https://mise.jdx.dev/cli/activate.html)
- [mise shims](https://mise.jdx.dev/dev-tools/shims.html)
- [mise lockfiles](https://mise.jdx.dev/dev-tools/mise-lock.html)
- [mise settings](https://mise.jdx.dev/configuration/settings.html)
- [mise registry](https://mise.jdx.dev/registry.html)
- [mise Node](https://mise.jdx.dev/lang/node.html)
- [mise Go](https://mise.jdx.dev/lang/go.html)
- [mise Bun](https://mise.jdx.dev/lang/bun.html)
- [mise Java](https://mise.jdx.dev/lang/java.html)
- [mise and direnv](https://mise.jdx.dev/direnv.html)

### Node ecosystem

- [Node.js release schedule and current lines](https://nodejs.org/en/about/previous-releases)
- [Node.js Corepack API documentation](https://nodejs.org/download/release/v25.8.0/docs/api/corepack.html)
- [Corepack upstream repository](https://github.com/nodejs/corepack)
- [Corepack package metadata](https://github.com/nodejs/corepack/blob/main/package.json)
- [nixpkgs Corepack package](https://github.com/NixOS/nixpkgs/blob/master/pkgs/by-name/co/corepack/package.nix)
- [pnpm installation and compatibility](https://pnpm.io/installation)
- [pnpm package.json configuration](https://pnpm.io/package_json)
- [pnpm upstream repository](https://github.com/pnpm/pnpm)
- [npm package-lock documentation](https://docs.npmjs.com/cli/v11/configuring-npm/package-lock-json/)
- [npm ci](https://docs.npmjs.com/cli/commands/npm-ci/)

### Go

- [Go downloads](https://go.dev/dl/)
- [Go toolchain selection](https://go.dev/doc/toolchain)
- [Go module reference](https://go.dev/ref/mod)

### Python

- [uv Python versions](https://docs.astral.sh/uv/concepts/python-versions/)
- [uv project guide](https://docs.astral.sh/uv/guides/projects/)
- [uv project configuration](https://docs.astral.sh/uv/concepts/projects/config/)
- [uv Python support policy](https://docs.astral.sh/uv/reference/policies/python/)
- [uv installation and releases](https://docs.astral.sh/uv/getting-started/installation/)

### Rust

- [Rust installation and rustup recommendation](https://www.rust-lang.org/tools/install)
- [rustup overrides](https://rust-lang.github.io/rustup/overrides.html)
- [rustup profiles](https://rust-lang.github.io/rustup/concepts/profiles.html)
- [Cargo rust-version](https://doc.rust-lang.org/stable/cargo/reference/rust-version.html)
- [rustup changelog](https://github.com/rust-lang/rustup/blob/main/CHANGELOG.md)
- [Rust releases](https://github.com/rust-lang/rust/releases/latest)

### Deno and Bun

- [Deno configuration](https://docs.deno.com/runtime/reference/deno_json/)
- [Deno packages and lockfile](https://docs.deno.com/runtime/packages/)
- [Deno installation and upgrades](https://docs.deno.com/runtime/getting_started/installation/)
- [Deno releases](https://github.com/denoland/deno/releases)
- [Bun install and CI](https://bun.sh/docs/pm/cli/install)
- [Bun releases](https://github.com/oven-sh/bun/releases)

### Java and JVM build tools

- [Eclipse Temurin support roadmap](https://adoptium.net/support/)
- [Gradle Wrapper](https://docs.gradle.org/current/userguide/gradle_wrapper.html)
- [Gradle Java compatibility](https://docs.gradle.org/current/userguide/compatibility.html)
- [Gradle toolchains](https://docs.gradle.org/current/userguide/toolchains.html)
- [Maven Wrapper](https://maven.apache.org/tools/wrapper.html)
- [Maven release history](https://maven.apache.org/docs/history)
- [Maven compiler release setting](https://maven.apache.org/plugins/maven-compiler-plugin/examples/set-compiler-release.html)
- [Maven toolchains](https://maven.apache.org/guides/mini/guide-using-toolchains)

### Nix, Home Manager, and direnv

- [Home Manager introduction](https://nix-community.github.io/home-manager/introduction.html)
- [nix-darwin manual](https://nix-darwin.github.io/nix-darwin/manual/)
- [Nix develop](https://nix.dev/manual/nix/2.34/command-ref/new-cli/nix3-develop.html)
- [Nix and direnv recipe](https://nix.dev/guides/recipes/direnv.html)
- [direnv documentation](https://direnv.net/)
- [direnv releases](https://github.com/direnv/direnv/releases)
- [nix-direnv](https://github.com/nix-community/nix-direnv)
- [Homebrew FAQ](https://docs.brew.sh/FAQ)

---

*Research completed for the project's new-project planning phase. Implementation must preserve the current Mac until an isolated verification and explicit apply decision are complete.*
