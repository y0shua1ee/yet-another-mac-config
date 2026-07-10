# Phase 1：Safety, Privacy, and State Foundation — Research

**Researched:** 2026-07-10
**Scope:** SAFE-01 至 SAFE-08
**Audience:** phase planner / implementer
**Evidence posture:** 仅复用既有 project research 与 codebase testing map；本次未联网，也未探测或改变 live Mac。

## Research Conclusion

Phase 1 应交付一个很薄、默认离线、没有真实 apply 能力的 safety control plane。最小 vertical slice 是：CLI 接收 synthetic 输入，经 common envelope、kind-specific schema、privacy gate 与 logical-reference resolver 校验，写入 external content-addressed artifact store；同一 CLI 再在 external fixture root 中运行 fake adapter，并用 protected-surface sentinels 生成四态 verdict 与严格有范围的 readiness claim。

Phase 1 的完成标准不是“Mac 已恢复”“Nix 配置可部署”或“所有工具都没变化”，而是以下机械事实同时成立：

1. 六种 artifact 不能互相冒充，且 lineage 绑定 exact content digests；
2. 原始、未知或敏感数据在 stdout、stderr、artifact store 之前 fail closed；
3. 默认测试只使用 synthetic data 和 external fixture roots，且不会自动升级到 network、live-check 或 mutation；
4. required protected surfaces 都有完整 before/after evidence，才能给出 `covered-surfaces-unchanged-for-run`；
5. Determinate Nix、nix-darwin 与 Home Manager 在本阶段仅作为 typed contract data，不被 evaluate、build、switch 或迁移。

这与既有研究的核心边界一致：build/evaluation 不等于 activation，receipt 不等于 fresh verification，`.gitignore` 不等于安全 allowlist，extra state 只报告而不自动清理。

## Requirements and Locked-Decision Implications

| Requirement | Planner implication | Required negative proof |
|---|---|---|
| SAFE-01 | 实现 `desired-state`、`observed-state`、`generated-plan`、`applied-receipt`、`verification-evidence`、`readiness-report` 六个封闭 kind；公共 envelope 后必须进入对应 payload validator | 错 kind、错 version、未知字段、错误 lineage、缺 provenance 均整体拒绝；不生成 degraded artifact |
| SAFE-02 | 所有持久化引用使用 `repo:`、`home:`、`fixture:`、`local-state:`、`nix-output:` 或登记过的 public profile；真实 root 仅在 resolver 进程内存在 | username、hostname、serial、硬件指纹、绝对 HOME、路径 traversal 与未知 namespace 均在 write/render 前拒绝 |
| SAFE-03 | stdout、stderr 和 store 共用同一个 pre-output privacy gate；adapter raw output 只可有界地停留在内存 | secret canary、private provider ref、完整 env、private network data、未知 raw 字段、value/hash/length/basename 派生诊断均不得外泄 |
| SAFE-04 | fixture blueprint 可进 Git，但实例必须在 worktree 外的新 root；隔离 HOME、全部 XDG、TMPDIR、PATH、trust、cache、runtime 和 manager roots | 测试不得把真实 HOME、真实项目、真实 manager state 或真实 worktree 当作 writable sandbox |
| SAFE-05 | runner 默认 offline；static/pure 与 isolated integration 是不同显式 tier；integration 也默认无网络 | cache miss、工具缺失或测试失败不得触发 install、download、repair、trust mutation 或更高权限 fallback |
| SAFE-06 | `live-check` 是第三个独立 tier；probe 只有同时具备当前官方只读语义与 isolated negative evidence 才能加入 allowlist | 无法证明的 probe 返回 `unknown` 或 `manual-required`；Phase 1 phase gate 不执行 live probe |
| SAFE-07 | 每个 run 绑定 protected-surface manifest，并在同一 observation window 内取得 before/after snapshot | 缺 snapshot、超限、解析失败、optional/excluded 未声明或 manifest 漂移时不得通过 |
| SAFE-08 | schema 可以表达 `extra` / `unmanaged-present`，但 Phase 1 不提供 destructive executor | 不产生 cleanup/uninstall/zap/runtime-delete operation；不能以 synthetic receipt 暗示真实 cleanup 能力 |

Locked decisions 进一步收紧规划：

- artifact 校验必须 fail closed；不保留 invalid raw artifact 或 quarantine 副本，只允许 schema-validated、安全且有界的 error envelope。
- `run_id` 只用于检索，不能承担完整性；不得通过目录位置、mtime、文件名或 `latest` 自动拼 lineage。
- integration、network 与 live-check 都是逐项 opt-in，不存在一个“允许所有网络/所有 live test”的总开关。
- sentinel 检出变化只能陈述 `change-detected-during-window`，不得归因给测试，也不得 auto-restore、auto-ignore 或 auto-retry。
- 本阶段最强 claim 只能是 `covered-surfaces-unchanged-for-run`；`recovery-ready-on-current-host` 与 `fresh-install-verified` 分别保留给后续 phase 和 clean-host evidence。

## Recommended Minimal Offline Implementation Stack

> 以下为实现推荐，而不是既有事实。选择依据是：当前仓库没有统一 runner/assertion/fixture system；Phase 1 需要 typed JSON、严格 parsing、SHA-256、bounded subprocess capture、filesystem isolation 与 deterministic tests，同时必须避免 package-manager auto-download。

推荐采用 **一个 repository-owned Go module，且 Phase 1 只使用 Go standard library**：

- `encoding/json` 配合 `Decoder.DisallowUnknownFields` 和每-kind 显式 validation；
- `crypto/sha256`、`crypto/hmac` 实现 content digest 与 per-run sentinel fingerprint；
- `context`、`os/exec`、`io.LimitedReader` 实现 timeout 和 bounded raw capture；
- `os`, `path/filepath`, `io/fs` 实现 external fixture/store、marker 与 symlink-safe containment；
- `testing` 实现 unit、negative、integration tests；
- 一个 CLI binary 同时提供 validate/store/fixture/sentinel 命令，但 apply executor 不存在于 module dependency graph。

Go toolchain availability 是 Phase 1 的前置能力，不是安装授权。runner 必须设置 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOENV=off`、`GOWORK=off` 与 external `GOCACHE`/`GOMODCACHE`；本地 toolchain 缺失时返回清晰的 `manual-required`，不得让 mise、Nix、Homebrew 或 Go 自动下载。

若 planner 在实施前发现 stdlib-only Go 与当前 tracked toolchain contract 不相容，允许改用另一种已有、可离线验证的 runtime；但必须保留同样的无依赖、无自动 bootstrap、strict parsing 和 physical read/write separation。不要为了“标准 schema”临时引入未经审计且需联网获取的 validator dependency。

### Recommended File Layout

```text
safety/
├── go.mod
├── cmd/yamc-safety/main.go              # CLI interaction surface only
├── internal/
│   ├── artifact/
│   │   ├── envelope.go                  # common envelope + closed kind registry
│   │   ├── desired.go                   # kind-specific payload contract
│   │   ├── observed.go
│   │   ├── plan.go
│   │   ├── receipt.go
│   │   ├── evidence.go
│   │   ├── report.go
│   │   ├── canonical.go                 # restricted canonical JSON
│   │   ├── digest.go                    # content digest + lineage checks
│   │   └── store.go                     # content-addressed read/write
│   ├── privacy/
│   │   ├── gate.go                      # one gate for store/stdout/stderr
│   │   ├── errors.go                    # stable safe error envelope
│   │   ├── logicalref.go                # namespace parser + ephemeral resolver
│   │   └── capture.go                   # bounded allowlisted subprocess capture
│   ├── fixture/
│   │   ├── root.go                      # external root creation and isolation
│   │   ├── environment.go               # minimal allowlisted environment
│   │   ├── retention.go                 # marker, TTL, safe cleanup
│   │   └── network.go                   # exact-ID authorization; deny by default
│   ├── sentinel/
│   │   ├── manifest.go
│   │   ├── snapshot.go
│   │   ├── verdict.go
│   │   └── adapters/                    # worktree, named-home, manager, service, named-target
│   └── contract/
│       └── controlplane.go              # Determinate/Nix/HM ownership data only
├── manifests/
│   ├── protected-surfaces.v1.json
│   └── network-tests.v1.json
├── testdata/
│   ├── blueprints/walking-skeleton/     # entirely synthetic full-stack fixture
│   ├── artifacts/                       # valid/invalid golden artifacts
│   ├── raw/                             # synthetic adapter bytes only
│   └── canaries/                        # synthetic privacy failures
└── scripts/test.sh                      # stable task/wave/phase entrypoint
```

The apply package is intentionally absent. Future write adapters must live in a physically separate module or binary so `validate`, `fixture`, `sentinel`, `check`, `plan` and `verify` cannot import or dispatch them accidentally.

## Artifact Model: Envelope, Kind Schemas, and Digest Lineage

### Common envelope

推荐使用一个 closed envelope；精确字段名可由 planner 调整，但语义不应减少：

```json
{
  "kind": "observed-state",
  "schema_version": "1.0.0",
  "run": {
    "run_id": "synthetic-run-001",
    "tier": "offline-static",
    "suite_id": "phase-1"
  },
  "producer": {
    "id": "yamc-safety",
    "version": "0.1.0"
  },
  "provenance": {
    "mode": "synthetic",
    "input_digests": []
  },
  "payload": {},
  "content_digest": "sha256:..."
}
```

推荐的 digest domain 是 envelope 中除 `content_digest` 外的全部字段，先经 project-defined restricted canonical JSON，再做 SHA-256。Canonical form 应排序 object keys、拒绝重复 key、拒绝 float/NaN/Infinity、保留 array 顺序、使用 UTF-8 且无无意义空白。`content_digest` 必须在 read 时重新计算；store key 必须等于 digest，不能信任调用者给出的文件名。

### Kind-specific payload contracts

| Kind | Minimum payload | Explicitly forbidden substitution |
|---|---|---|
| `desired-state` | logical profile、declarations、expected ownership/postconditions、source provenance | 不能包含 observed result、真实 identity 或 secret value |
| `observed-state` | named scope、normalized facts、adapter ID/status、logical refs、observation time window | raw stdout/stderr、unknown field、完整环境与绝对路径不得进入 payload |
| `generated-plan` | exact desired digest、exact observed digest、data-only operation enum、preconditions、expected postconditions、risk/rollback metadata | 任意 shell、动态 discovery、秘密、绝对路径、destructive default operation |
| `applied-receipt` | exact plan digest、每 operation ID/outcome/checkpoint、synthetic/live mode 明示 | exit 0 不能冒充 verification；Phase 1 只产生 fake-adapter synthetic receipt |
| `verification-evidence` | expected-postconditions digest、fresh observed digest、receipt/plan digest（若 apply path）、comparison result | 旧 observed snapshot 或 receipt 不能替代 fresh observation |
| `readiness-report` | exact evidence digest list、statuses、excluded/optional scopes、bounded claim | 不能依据目录中的“最新文件”组装，也不能升级为 current-host/fresh-install claim |

所有 schema 都应 `additionalProperties: false` 等价处理。Unknown kind、unknown schema version、unknown operation enum、缺少 required field 或 provenance 不完整均拒绝整份 artifact。Schema/version migration 必须是显式、纯函数、可测试的转换；Phase 1 不需要先实现 migration framework。

### Required lineage graph

```text
desired digest ─┐
                ├──> generated-plan digest ───> applied-receipt digest ─┐
observed digest ┘                                                        ├──> verification-evidence digest ───> readiness-report digest
expected-postconditions digest ─────────────────────────────────────────┘
fresh observed digest ──────────────────────────────────────────────────┘
```

- Plan 必须同时绑定 exact desired 与 observed digests。
- Receipt 必须绑定 exact plan digest，且 receipt operation IDs 必须是 plan operation IDs 的有序子集/结果集合，不能新增 operation。
- Apply-path evidence 必须绑定 plan、receipt、expected postconditions 与 fresh observed digest。
- Read-only path 使用显式 `lineage_mode: read-only`，绑定 desired（如适用）、fresh observed 与 expected postconditions，但不能伪造 receipt。
- Report 只能列出它实际读取并校验过的 evidence digests。
- Store 采用 `local-state:artifacts/sha256/<digest>` 的逻辑布局；不存在 `latest` alias、mtime selection 或 directory guessing。

## Privacy Gate, Logical References, and Bounded Raw Capture

### One pre-output gate

所有 machine-readable writer 和 human renderer 都必须调用同一 gate，顺序推荐为：

```text
bounded capture in memory
  -> strict adapter parse
  -> reject unknown fields
  -> registered normalization only
  -> logical-reference conversion
  -> kind schema validation
  -> forbidden key/category/canary validation
  -> canonical serialization
  -> content digest
  -> atomic store write OR bounded renderer
```

任何一步失败都不得写 partial artifact。临时文件也不能先写 raw 再清理；atomic write 只适用于已经通过 privacy/schema/digest gate 的 canonical bytes。

Privacy rejection 只输出安全 error envelope：

```json
{
  "error_code": "PRIVACY_UNKNOWN_ABSOLUTE_REF",
  "artifact_kind": "observed-state",
  "adapter_id": "synthetic-path-adapter",
  "pointer": "/payload/items/0/source",
  "category": "unknown-absolute-reference",
  "remediation": "register-logical-reference"
}
```

这些字段来自封闭 enum/allowlist。不得附加原值、截断值、首尾字符、长度、content hash、真实 basename、provider 名称、通用 exception text 或 raw subprocess error。

### Logical-reference contract

推荐 grammar：`<namespace>:<normalized-relative-id>`。允许 namespace 初始仅为：

- `repo:` — tracked repository logical entry；
- `home:` — manifest 明确命名的 user entry；
- `fixture:` — 当前 external fixture root 内；
- `local-state:` — Git-ignored artifact/retention root；
- `nix-output:` — contract 中的 public output identity，不是实际 store path dump；
- `profile:` — 已登记的 public logical profile。

Resolver 必须拒绝空 segment、`..`、NUL、absolute suffix、separator ambiguity、symlink escape 与未知 namespace。真实 root、UID、username、hostname 只存在于当前 resolver memory；artifact 和 renderer 只看 logical ref。不能用 basename 截断或 hash 一个未知绝对路径来“清洗”它。

### Bounded capture recommendation

仅 allowlisted executable + fixed argv template 可进入 capture layer。推荐保守默认：

- wall timeout：5 秒；manifest 可降低，单次不得高于 30 秒；
- stdout 与 stderr：各 64 KiB；单次 manifest 上限不得高于各 256 KiB；
- pipe capture in memory；禁止 inherited terminal、临时 raw file 与 combined unbounded buffer；
- overflow、timeout、invalid UTF-8（若 adapter 要求 text）、parse failure 或 unknown field：立刻终止 child、清空 raw buffer，并返回 `unknown` 或 privacy-safe error；
- adapter 成功后只输出 normalized typed facts；synthetic raw sample 只能存放于 tracked `testdata/raw/`，并通过 canary/secret scan；
- live raw stdout/stderr 永不保留，即使指定 `--keep-fixture`。

Capture API 不接受 caller-supplied arbitrary command string，也不经 shell。Phase 1 的 walking skeleton 只调用 external fixture 中的 fake binary。

## Three Test Tiers and External Fixture Lifecycle

### Tier 1 — `offline-static`（default）

覆盖 schema、canonicalization、digest、lineage、privacy gate、logical refs、error envelope、synthetic raw parser、store 与 pure sentinel comparison。不得调用 live app、manager、service 或 network；默认 `scripts/test.sh phase` 只运行这个 tier 加下面的 offline isolated synthetic run。

### Tier 2 — `isolated-integration`（explicit, still offline）

每个 test 从 tracked synthetic blueprint 实例化全新 external root，运行 fake executable、CLI、artifact store 和 fake/synthetic sentinels。不得读取真实项目、真实 HOME、global manager state、trust、Keychain、proxy 或 credentials。缺能力时失败或 `manual-required`，不得 fallback 到 live state。

### Tier 3 — `live-check`（separate allowlist, never in Phase 1 gate）

只允许 named, officially justified probe。每个 probe 还必须先有 isolated negative test，证明不会 write/install/download/execute arbitrary config。Phase 1 只实现 allowlist contract、deny/default behavior 和 synthetic tests；不以本 phase 的 validation command 执行 live probes。

### External fixture root

每次 run 在 system temp 或显式 external base 下建立新 root；启动后先证明 canonical root 不在 repository/worktree 内。推荐布局：

```text
<external-root>/
├── .yamc-fixture-marker.json
├── home/
├── xdg/{config,data,cache,state,runtime}/
├── tmp/
├── path/bin/                 # fake allowlisted executables only
├── managers/{nix,brew,mise,uv,rustup,go,cargo,node}/
├── trust/
├── network-cache/
├── artifact-store/
├── blueprint-worktree/       # copy of synthetic tracked blueprint
└── sentinel-scratch/
```

Runner 传给 child 的 environment 必须从空白 allowlist 构造，而不是删几个已知变量：isolated `HOME`、全部 XDG、`TMPDIR`、`PATH`、manager roots、`GOCACHE`、`GOMODCACHE`；明确不传 `SSH_*`、`AWS_*`、`GITHUB_*`、token/key/password 形态变量、proxy 变量或真实 shell init。PATH 只包含 fixture fake-bin 和运行 harness 所需的已解析 toolchain directory。

### Exact network manifest

Tracked `network-tests.v1.json` 每个 entry 至少包含：

```json
{
  "test_id": "fixture.download.known-archive.v1",
  "adapter_id": "repository-downloader-v1",
  "purpose": "fetch-one-public-synthetic-fixture",
  "request": {
    "method": "GET",
    "url": "https://example.invalid/exact/path/archive.tar.gz",
    "redirects": 0
  },
  "integrity": {
    "algorithm": "sha256",
    "digest": "sha256:synthetic-placeholder"
  },
  "limits": {
    "max_bytes": 1048576,
    "timeout_ms": 10000
  },
  "cache_ref": "fixture:network-cache/fixture.download.known-archive.v1",
  "credentials": "forbidden",
  "proxy_environment": "forbidden",
  "egress_policy": "exact-url-only"
}
```

上例只说明 schema，`.invalid` entry 不应实际联网。真实 entry 必须逐项授权 `--allow-network-test <exact-test-id>`；禁止 `--network`、wildcard host、generic token、Keychain credential、redirect、ambient proxy 或 shared cache。下载器必须先验证 egress enforcement 能限制 exact URL，再下载至 isolated cache，边读边限制 bytes，完成后核对 exact digest。任何一项不可实施时不运行，返回 `manual-required`。Phase 1 可以且应只测试 manifest validator、exact-ID gate 与拒绝路径，不需要一个真实 network test。

### Retention and cleanup

- success 与 failure fixture 默认全部删除；
- 只有 run 开始前显式 `--keep-fixture` 才能保留 synthetic/isolated root；
- 推荐默认 TTL 为 24 小时（实现推荐，可由 planner收紧），保留位置只在 dedicated `local-state:fixtures/retained/<logical-fixture-id>`；
- marker 至少绑定 schema version、logical fixture ID、created-at、expires-at、effective UID 与 random ownership nonce；marker 不进入 Git 或 public artifact；
- cleanup 必须验证 base containment、目标不是 symlink、marker schema/UID/nonce/TTL 全部有效，才允许删除一个 child root；任何不确定性都停止，禁止 broad recursive cleanup；
- retained fixture 仍不得包含 live raw output、credentials 或真实 HOME copy；network cache 跟随同一 TTL，且只保留 digest-verified public bytes。

## Protected-Surface Sentinels and Verdict

### Manifest-scoped protected surfaces

`protected-surfaces.v1.json` 应按 suite/test 明确列出 theoretically touchable surfaces，而不是扫描整个 Mac：

1. **worktree/index** — 当前 repo 的 tracked/index/worktree state；输出不能包含 untracked filenames；
2. **named HOME entries** — 只解析 manifest 中的 logical `home:` entry；不递归整个 HOME，不读取 secret files；
3. **global manager roots** — 只对命名 root/子范围做 bounded fingerprint；超过 file/byte/time cap 为 `indeterminate`；
4. **named services** — 只查询 allowlisted service identity；无已证明只读 adapter 时为 `manual-required`，不能扫全部 service state；
5. **repository-external named targets** — 只检查显式 logical target，symlink target 越界必须另行登记。

每个 adapter 输出 privacy-safe typed snapshot。推荐对可能含私密文件名/内容的表面使用 **per-run ephemeral-key HMAC-SHA-256**：before/after 在同一进程可精确比较，但 token 不能跨 run 形成稳定机器指纹；HMAC key 不落盘。Evidence 记录算法、opaque before/after tokens、adapter status 与 bounds，不记录输入路径、文件名或内容。Public repo tracked-tree digest 可以使用普通 SHA-256，但保持一种统一 opaque representation 会更简单。

Fingerprint 必须包含足以发现目标范围内变化的类型、存在性、mode、symlink identity 和 bounded content/tree state；不能仅比较 mtime 或 item count。遇到 unreadable、race、cap overflow、symlink escape 或 snapshot window 不闭合时，不得 pass。

### Strict four-state verdict

| Verdict | Meaning | Exit behavior |
|---|---|---|
| `passed` | 所有 required sentinels 都有完整、可解析、同 manifest 的 before/after evidence，且相等 | 0 |
| `violation` | 任一 required protected surface 在 observation window 内不同 | non-zero；推荐 20 |
| `indeterminate` | observation 缺失、超限、权限不足、race、required adapter unavailable | non-zero；推荐 21 |
| `harness-error` | manifest/schema/lineage/internal invariant 失败 | non-zero；推荐 22 |

只有 manifest 中预先标记 optional 的 sentinel 可以产生 warning 而不阻止 `passed`。Required difference 一律是 `violation`；不得事后把 noisy surface 改 optional，也不得自动 restore、retry 或忽略。

### Exact scoped claim

`passed` report 只允许输出：

```text
covered-surfaces-unchanged-for-run
```

并绑定 exact suite ID/digest、test tier、protected-surface manifest digest、observation window、每个 required before/after snapshot token、optional/excluded list 与 evidence digests。变化报告只写 `change-detected-during-window`，不声称是 test 造成。禁止“Mac unchanged”“safe on every host”“recovery-ready-on-current-host”或“fresh-install-verified”。

## Determinate Nix / nix-darwin / Home Manager Contract Boundary

既有官方资料支持下列职责边界：Determinate Nix 负责 Nix distribution/daemon/support boundary；nix-darwin 负责 machine composition/activation；Home Manager 负责 user configuration、Nix-built manager entrypoints、config files 与 shell integration。[Nix flakes/lock model](https://nix.dev/manual/nix/2.26/command-ref/new-cli/nix3-flake.html)、[Nix store secret guidance](https://nix.dev/manual/nix/2.34/store/secrets)、[nix-darwin manual](https://nix-darwin.github.io/nix-darwin/manual/) 与 [Home Manager manual](https://nix-community.github.io/home-manager/) 是既有 SUMMARY 已引用的 authoritative sources。

Phase 1 只把该边界编码为 contract data，例如：

```json
{
  "scope": "profile:synthetic-developer",
  "executable": "mise",
  "declaration_owner": "home-manager",
  "manager_binary_owner": "nix-store-via-home-manager",
  "managed_payload_owner": "mise",
  "selected_executable": "fixture:path/bin/mise",
  "activation_context": "synthetic-none"
}
```

并验证每个 `(scope, executable)` 只有一个 primary executable owner。Module 声明或 manager binary 来自 Nix store，不代表 mise/uv/rustup/Homebrew/project wrapper 的 mutable payload 已转给 Nix。任何会 download、upgrade、prune、trust、delete、switch 或写 service/defaults/link 的 activation 仍是独立 write boundary。

因此 Phase 1 明确不做：

- 不调用 `nix flake check`、`nix build`、`darwin-rebuild build/switch` 或 Home Manager activation 作为 phase gate；这些可能接触 cache/store/network，且不能证明本 phase 的 synthetic isolation；
- 不改 flake、host composition、Homebrew inventory、Home Manager module 或 shell route；
- 不把 synthetic plan/receipt 转化为真实 executor；
- 不声称 Nix generation 能原子回滚 Homebrew、service、link 或 downstream manager state。

[Homebrew Bundle documentation](https://docs.brew.sh/Brew-Bundle-and-Brewfile) 已在 SUMMARY 中用于证明 `brew bundle check` 可能执行 `system` entry；因此它不是 Phase 1 默认 probe。Extra Homebrew/tool state 只进入 `unmanaged-present`/`extra` evidence，不生成 cleanup。Secret 如果进入 Git，后续需要 rotate/history remediation；[GitHub sensitive-data removal guidance](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository) 支持“pre-write prevention 优先于事后 masking”的边界。

## Vertical MVP Walking Skeleton Domain Mapping

本项目不需要为 Phase 1 发明产品表面：

| MVP domain role | Phase 1 implementation | Explicit non-goals |
|---|---|---|
| UI / user interaction | CLI：`validate`、`store`、`fixture run`、`sentinel verify`、`report` | 无 web UI、GUI、HTTP API、daemon |
| Data plane | typed artifact store 的 read/write；content-addressed canonical JSON，位于 external/local-state root | 无 SQLite/Postgres/Redis/传统 DB，无 remote object store |
| Dev environment / full stack | external fixture local run：synthetic desired → synthetic observed → data-only plan → fake-adapter synthetic receipt → fresh synthetic observation/evidence → scoped report；before/after sentinels 包住整个 window | 无 real HOME、real project、Nix/HM/Homebrew apply、service/defaults/link mutation、network |
| Deployment | 不存在；CLI 由 local test runner 构建/运行，artifact 留在 external fixture store | 无 CI/CD、package publishing、host deployment |

Walking skeleton 的唯一 writer 是 fixture 内的 fake adapter，只能修改 `fixture:` 范围。它的 receipt 必须显式标记 `mode: synthetic`，从而证明六 kind 与 lineage 能贯穿全流程，但绝不构成真实 apply capability。

## Risks and Anti-patterns

Planner 应把以下项目写成 negative tests 或 structural guards，而不是仅写文档提醒：

- **One schema for everything：** envelope 相同不等于 payload 可互换；每-kind validator 必须封闭。
- **Trusting `run_id` or filenames：** 会把不相关或 stale artifacts 拼在一起；只接受 verified digest lineage。
- **Mask-after-write：** raw output 一旦进入 temp/log/store 就已经越界；必须先 parse/normalize/gate 再写。
- **General-purpose redactor：** masking 可能制造“看似安全”的坏 artifact；只允许预注册、无歧义转换。
- **Hashing unknown identity/path as sanitation：** stable hash 仍是机器指纹；unknown absolute ref 直接拒绝。
- **Temporary HOME only：** XDG、manager roots、trust、cache、TMPDIR、PATH、proxy 与 shell init 都可能回流真实 state。
- **Writable worktree fixture：** 测试生成物和 cache 必须 external；真实 repo 仅作 tracked input source。
- **Implicit network on cache miss：** toolchain/fixture 缺失必须 fail/manual-required，不能由 mise/uv/Go/Nix/wrapper 自动下载。
- **A broad network switch：** exact test ID、URL、digest、bytes、timeout、isolated cache 缺一不可；无法限制 egress 就不运行。
- **Live probe escalation：** isolated failure 或 adapter missing 不得自动转 live-check。
- **Whole-HOME/service scan：** 既泄露隐私又无法诚实界定 coverage；只观察 manifest named surfaces。
- **Mtime/count sentinel：** 不能可靠发现内容替换；必须使用 bounded content/tree identity，超限则 indeterminate。
- **Noisy-surface post hoc exclusion：** optional/excluded 必须在 before snapshot 之前进入 exact manifest。
- **Receipt equals verify：** fake/real command exit status都不能替代 fresh observation。
- **Nix equals all ownership：** declaration owner、manager binary、payload、selected executable 与 activation context 必须分开。
- **Synthetic receipt implies executor：** Phase 1 不得出现真实 apply import、arbitrary shell operation 或 privileged branch。
- **Cleanup as convergence：** extra/unmanaged 只报告；Homebrew cleanup/zap、runtime removal 与 broad deletion不属于本 phase。
- **Overclaiming：** sentinel 只证明 exact window + named surfaces；不能证明整个 Mac、未来时间或 clean host。

## Validation Architecture

### Runner invariants

建议由 `safety/scripts/test.sh` 作为唯一稳定入口。它必须先建立 external root、取得 protected-surface before snapshots，再以 allowlisted environment 运行 Go tests/CLI，最后无论测试成功或失败都取得 after snapshots并计算 verdict。runner 自身不得安装依赖；Go 不可用时返回 `manual-required`。

每条命令都强制：

```text
YAMC_TEST_TIER=offline-static
YAMC_NETWORK=deny
GOTOOLCHAIN=local
GOPROXY=off
GOSUMDB=off
GOENV=off
GOWORK=off
CGO_ENABLED=0
HOME/XDG_*/TMPDIR/GOCACHE/GOMODCACHE=<external fixture paths>
```

Phase 1 validation 不调用 Nix、Homebrew、mise、uv、rustup、application reload、service query、defaults、link replacement 或 live-check。Fake binaries 位于 `fixture:path/bin`，且 runner 不接受任意 executable/argv。

### Task commands

Planner 可把 implementation tasks 分成以下可独立验收单元；命令名称是推荐的稳定 UX：

```bash
./safety/scripts/test.sh task artifact-contracts
./safety/scripts/test.sh task privacy-boundary
./safety/scripts/test.sh task fixture-lifecycle
./safety/scripts/test.sh task sentinel-verdicts
./safety/scripts/test.sh task controlplane-contract
./safety/scripts/test.sh task walking-skeleton
```

Expected coverage：

- `artifact-contracts`：六 kind positive/negative、unknown field/version/kind、canonical digest、store-key mismatch、完整 lineage 与 read-only lineage；
- `privacy-boundary`：logical ref traversal/absolute refs、secret/provider/env/network canaries、safe error envelope、stdout/stderr/store 同 gate、capture timeout/overflow/parse failure；
- `fixture-lifecycle`：external containment、blank allowlisted env、全部 roots、default cleanup、explicit retention、TTL/marker/symlink/UID negative paths、network manifest exact-ID deny tests；
- `sentinel-verdicts`：五类 synthetic protected surfaces、required/optional/excluded、passed/violation/indeterminate/harness-error、window/manifest digest binding、无 overclaim；
- `controlplane-contract`：Determinate/nix-darwin/HM role data、one owner per `(scope, executable)`、payload ownership separation、所有 mutable action 被拒；
- `walking-skeleton`：external local full-stack synthetic run，六 artifact digests 可从 report 反向机械验证，fixture 外零写入。

### Wave commands

```bash
./safety/scripts/test.sh wave contracts
./safety/scripts/test.sh wave isolated-harness
```

- `wave contracts` 聚合 `artifact-contracts`、`privacy-boundary`、`controlplane-contract`；可并行运行，但每个 task 使用不同 external root/store。
- `wave isolated-harness` 在 contracts 通过后聚合 `fixture-lifecycle`、`sentinel-verdicts`、`walking-skeleton`；仍无 network/live state。

Wave runner 不复用 artifact store、fixture ID 或 sentinel key；防止 tests 通过共享 cache 或 stale `latest` 假成功。

### Phase command

```bash
./safety/scripts/test.sh phase
```

该命令顺序执行两个 wave，并额外验证：

1. 当前 exact suite/manifest digests 全部绑定；
2. external walking-skeleton report 可重新读取并完整验证六 kind lineage；
3. synthetic secret/path/host canaries 未出现在 stdout、stderr 或 retained canonical artifacts；
4. 默认 run 没有 retained fixture、network cache 或 worktree output；
5. 所有 required sentinel 都是完整 evidence；只有 verdict `passed` 时 exit 0；
6. 最终 claim 精确等于 `covered-surfaces-unchanged-for-run`。

Phase command 必须是 offline + synthetic/isolated only。任何 `indeterminate`、`violation` 或 `harness-error` 都 non-zero；不能通过 retry、optional downgrade、live-check 或 cleanup 把结果改成 pass。

### Planner sequencing recommendation

1. 先实现 envelope/kind registry、canonical digest、content-addressed store；
2. 同 wave 实现 privacy gate/logical refs/safe errors 与 control-plane contract；
3. 再实现 bounded capture、external fixture lifecycle 与 exact network manifest deny path；
4. 实现 protected-surface manifest、snapshot adapters 和四态 verdict；
5. 最后用 walking skeleton 串起六 artifact，并以 phase command 证明完整边界。

每个 plan task 都应同时包含 positive、negative、privacy-canary 与 outside-root sentinel test。不要把 privacy/sentinel 留到“最后统一补测试”，因为它们是所有 write/render API 的设计边界。

## Source Notes

本文件的外部事实只复用 `.planning/research/SUMMARY.md` 已列出的 authoritative sources：

- [Nix flakes and lock model](https://nix.dev/manual/nix/2.26/command-ref/new-cli/nix3-flake.html)；
- [Nix store secrets guidance](https://nix.dev/manual/nix/2.34/store/secrets)；
- [nix-darwin configuration options](https://nix-darwin.github.io/nix-darwin/manual/)；
- [Home Manager manual](https://nix-community.github.io/home-manager/)；
- [Homebrew Bundle documentation](https://docs.brew.sh/Brew-Bundle-and-Brewfile)；
- [Apple Privacy & Security settings](https://support.apple.com/guide/mac-help/change-privacy-security-settings-on-mac-mchl211c911f/mac)；
- [GitHub sensitive-data removal](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)。

Go stdlib、目录布局、字段名、canonical JSON 子集、默认 capture limits、24-hour TTL、exit codes、per-run HMAC 与 runner command names 均是本文件的 **inference/recommendation**，需要在 implementation plan 中以最小 prototype 与 negative tests 固化；它们不是来自外部文档的已验证 live behavior。

## Planning Readiness

Phase 1 已可进入 detailed planning。建议把 contracts/privacy 与 isolated harness/sentinel 分成两个 wave，最后以一个 external walking-skeleton phase gate 收束。无需也不应在本阶段加入 web/API、traditional database、deployment、live Nix evaluation、真实 app reload 或任何 destructive convergence。
