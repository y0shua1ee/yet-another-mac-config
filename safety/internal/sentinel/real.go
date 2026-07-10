package sentinel

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"example.invalid/yamc/safety/internal/fixture"
	"example.invalid/yamc/safety/internal/privacy"
)

const (
	realAdapterSchema     = "1.0.0"
	realAdapterVersion    = "1.0.0"
	innerSyntheticSuccess = "synthetic-sentinel-passed"
	maxCommandOutput      = 64 << 10
)

type ProofState string

const (
	ProofCurrent ProofState = "current"
	ProofMissing ProofState = "missing"
)

type OfficialSource struct {
	URL      string `json:"url"`
	Boundary string `json:"boundary"`
}

type AdapterInvocation struct {
	Kind        string   `json:"kind"`
	API         string   `json:"api,omitempty"`
	Executable  string   `json:"executable,omitempty"`
	Argv        []string `json:"argv"`
	Environment []string `json:"environment"`
}

type NegativeSuite struct {
	SuiteID          string `json:"suite_id"`
	EvidenceContract string `json:"evidence_contract"`
	TestSourceDigest string `json:"test_source_digest"`
	Digest           string `json:"digest,omitempty"`
	Status           string `json:"status"`
}

type AdapterLimits struct {
	MaxFiles   int `json:"max_files"`
	MaxBytes   int `json:"max_bytes"`
	TimeoutMS  int `json:"timeout_ms"`
	MaxProcess int `json:"max_processes"`
}

type RealAdapterDefinition struct {
	AdapterID            string                `json:"adapter_id"`
	Version              string                `json:"version"`
	ImplementationDigest string                `json:"implementation_digest"`
	SurfaceDomain        privacy.SurfaceDomain `json:"surface_domain"`
	LogicalRefs          []string              `json:"logical_refs"`
	ProofState           ProofState            `json:"proof_state"`
	Official             []OfficialSource      `json:"official_sources"`
	ReviewedAt           string                `json:"reviewed_at"`
	ValidUntil           *string               `json:"valid_until"`
	NegativeSuite        NegativeSuite         `json:"negative_suite"`
	Invocations          []AdapterInvocation   `json:"invocations"`
	Limits               AdapterLimits         `json:"limits"`
	Statuses             []string              `json:"statuses"`
	NetworkPolicy        string                `json:"network_policy"`
	ShellPolicy          string                `json:"shell_policy"`
}

type RealAdapterManifest struct {
	SchemaVersion string                  `json:"schema_version"`
	Adapters      []RealAdapterDefinition `json:"adapters"`
}

type RealAdapterRegistry struct {
	definitions  map[string]RealAdapterDefinition
	usable       map[string]bool
	capabilities map[string]*realAdapterCapability
	now          time.Time
}

type realAdapterCapability struct {
	registry             *RealAdapterRegistry
	adapterID            string
	implementationDigest string
}

type RealProofAssessment struct {
	Status        string                `json:"status"`
	Verdict       Verdict               `json:"verdict"`
	ExitCode      int                   `json:"exit_code"`
	Reason        string                `json:"reason"`
	ClaimEligible bool                  `json:"claim_eligible"`
	SurfaceDomain privacy.SurfaceDomain `json:"surface_domain,omitempty"`
	LogicalRef    string                `json:"logical_ref,omitempty"`
}

func RequireControlledRealEnvelope(assessment RealProofAssessment) RealProofAssessment {
	if !assessment.ClaimEligible {
		return assessment
	}
	return RealProofAssessment{
		Status:        "manual-required",
		Verdict:       VerdictIndeterminate,
		ExitCode:      32,
		Reason:        "controlled-real-envelope-runner-required",
		ClaimEligible: false,
	}
}

type realAdapterSpec struct {
	domain               privacy.SurfaceDomain
	logicalRefs          []string
	invocations          []AdapterInvocation
	sources              []string
	limits               AdapterLimits
	implementationDigest string
	negativeSuiteID      string
	negativeContract     string
}

const (
	negativeTestSourceDigest       = "sha256:88bd26e6beba5c79cf2491b6c4f50af6d2ce811f175119081e57f77894ad5887"
	realImplementationSourceDigest = "sha256:45b786a3a32d2f1288c94209c49aa6c7f0e1b2f140ccb605771ff1b252a8d040"
)

var gitVersionInvocation = AdapterInvocation{Kind: "executable", Executable: "git", Argv: []string{"--no-lazy-fetch", "--version"}, Environment: []string{"GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "LC_ALL=C", "LANG=C", "PATH=/usr/bin:/bin"}}

var gitReadOnlyEnvironment = []string{"GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "LC_ALL=C", "LANG=C", "PATH=/usr/bin:/bin", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0"}

var realAdapterSpecs = map[string]realAdapterSpec{
	"git-worktree-readonly-v1": {
		domain:      privacy.SurfaceWorktree,
		logicalRefs: []string{"repo:sentinel/worktree/tracked"},
		invocations: []AdapterInvocation{
			gitVersionInvocation,
			{Kind: "executable", Executable: "git", Argv: []string{"--no-lazy-fetch", "-c", "core.fsmonitor=false", "status", "--porcelain=v2", "--untracked-files=no", "--ignore-submodules=all"}, Environment: gitReadOnlyEnvironment},
			{Kind: "executable", Executable: "git", Argv: []string{"--no-lazy-fetch", "ls-files", "-z", "--stage"}, Environment: gitReadOnlyEnvironment},
		},
		sources:              []string{"https://git-scm.com/docs/git-status", "https://git-scm.com/docs/git", "https://git-scm.com/docs/git-config"},
		limits:               AdapterLimits{MaxFiles: 4096, MaxBytes: 16 << 20, TimeoutMS: 5000, MaxProcess: 3},
		implementationDigest: realImplementationSourceDigest,
		negativeSuiteID:      "negative.git-status.no-write.v1",
		negativeContract:     "isolated-fixture-byte-identical-v1",
	},
	"git-index-readonly-v1": {
		domain:      privacy.SurfaceWorktree,
		logicalRefs: []string{"repo:sentinel/worktree/index"},
		invocations: []AdapterInvocation{
			gitVersionInvocation,
			{Kind: "executable", Executable: "git", Argv: []string{"--no-lazy-fetch", "ls-files", "-z", "--stage"}, Environment: gitReadOnlyEnvironment},
		},
		sources:              []string{"https://git-scm.com/docs/git-ls-files", "https://git-scm.com/docs/git"},
		limits:               AdapterLimits{MaxFiles: 4096, MaxBytes: 16 << 20, TimeoutMS: 5000, MaxProcess: 2},
		implementationDigest: realImplementationSourceDigest,
		negativeSuiteID:      "negative.git-ls-files.no-write.v1",
		negativeContract:     "isolated-fixture-byte-identical-v1",
	},
	"go-lstat-file-v1": {
		domain:               privacy.SurfaceNamedHome,
		logicalRefs:          []string{"home:.zshrc"},
		invocations:          []AdapterInvocation{{Kind: "go-api", API: "os.OpenRoot+Root.Lstat+Root.OpenFile+Root.Readlink+File.Stat+os.SameFile+io.LimitReader", Argv: []string{}, Environment: []string{}}},
		sources:              []string{"https://pkg.go.dev/os#OpenRoot", "https://pkg.go.dev/os#Lstat", "https://pkg.go.dev/os#Open", "https://pkg.go.dev/os#SameFile"},
		limits:               AdapterLimits{MaxFiles: 1, MaxBytes: 1 << 20, TimeoutMS: 1000, MaxProcess: 0},
		implementationDigest: realImplementationSourceDigest,
		negativeSuiteID:      "negative.go-lstat-file.no-write.v1",
		negativeContract:     "isolated-fixture-byte-identical-v1",
	},
	"go-bounded-tree-v1": {
		domain:               privacy.SurfaceManagerRoot,
		logicalRefs:          []string{"home:sentinel/manager/mise-data"},
		invocations:          []AdapterInvocation{{Kind: "go-api", API: "os.OpenRoot+io/fs.WalkDir+Root.Lstat+Root.OpenFile+Root.Readlink+File.Stat+os.SameFile+io.LimitReader", Argv: []string{}, Environment: []string{}}},
		sources:              []string{"https://pkg.go.dev/os#OpenRoot", "https://pkg.go.dev/io/fs#WalkDir", "https://pkg.go.dev/os#Lstat", "https://pkg.go.dev/os#Open"},
		limits:               AdapterLimits{MaxFiles: 2048, MaxBytes: 16 << 20, TimeoutMS: 5000, MaxProcess: 0},
		implementationDigest: realImplementationSourceDigest,
		negativeSuiteID:      "negative.go-bounded-tree.no-write.v1",
		negativeContract:     "isolated-fixture-byte-identical-v1",
	},
	"launchctl-print-service-v1": {
		domain:               privacy.SurfaceService,
		logicalRefs:          []string{"profile:sentinel/service/homebrew-mxcl-nginx"},
		invocations:          []AdapterInvocation{{Kind: "executable", Executable: "launchctl", Argv: []string{"print", "gui/<uid>/homebrew.mxcl.nginx"}, Environment: []string{}}},
		sources:              []string{"https://github.com/apple-oss-distributions/launchd/blob/main/man/launchctl.1"},
		limits:               AdapterLimits{MaxFiles: 1, MaxBytes: 64 << 10, TimeoutMS: 2000, MaxProcess: 1},
		implementationDigest: realImplementationSourceDigest,
		negativeSuiteID:      "negative.launchctl-print.no-write.v1",
		negativeContract:     "isolated-fixture-byte-identical-v1",
	},
	"go-system-shells-file-v1": {
		domain:               privacy.SurfaceNamedTarget,
		logicalRefs:          []string{"profile:sentinel/named-target/system-shells"},
		invocations:          []AdapterInvocation{{Kind: "go-api", API: "os.OpenRoot+Root.Lstat+Root.OpenFile+Root.Readlink+File.Stat+os.SameFile+io.LimitReader", Argv: []string{}, Environment: []string{}}},
		sources:              []string{"https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man3/getusershell.3.html", "https://pkg.go.dev/os#OpenRoot", "https://pkg.go.dev/os#Lstat", "https://pkg.go.dev/os#Open"},
		limits:               AdapterLimits{MaxFiles: 1, MaxBytes: 1 << 20, TimeoutMS: 1000, MaxProcess: 0},
		implementationDigest: realImplementationSourceDigest,
		negativeSuiteID:      "negative.go-system-shells.no-write.v1",
		negativeContract:     "isolated-fixture-byte-identical-v1",
	},
}

func LoadRealAdapterRegistry(data []byte, now time.Time) (*RealAdapterRegistry, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, errors.New("real adapter manifest rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest RealAdapterManifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, errors.New("real adapter manifest rejected")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("real adapter manifest rejected")
	}
	if manifest.SchemaVersion != realAdapterSchema || len(manifest.Adapters) != len(realAdapterSpecs) {
		return nil, errors.New("real adapter manifest rejected")
	}
	// reviewed_at/valid_until 使用 UTC 日历日期；生产入口先归一化为 UTC，测试只注入明确的 UTC 时刻。
	registry := &RealAdapterRegistry{definitions: make(map[string]RealAdapterDefinition, len(manifest.Adapters)), usable: make(map[string]bool, len(manifest.Adapters)), capabilities: make(map[string]*realAdapterCapability, len(manifest.Adapters)), now: now}
	for _, definition := range manifest.Adapters {
		spec, ok := realAdapterSpecs[definition.AdapterID]
		if !ok || definition.Version != realAdapterVersion || definition.ImplementationDigest != spec.implementationDigest || definition.SurfaceDomain != spec.domain || !equalStrings(definition.LogicalRefs, spec.logicalRefs) || !equalInvocations(definition.Invocations, spec.invocations) {
			return nil, errors.New("real adapter definition rejected")
		}
		if _, duplicate := registry.definitions[definition.AdapterID]; duplicate {
			return nil, errors.New("duplicate real adapter rejected")
		}
		if definition.NetworkPolicy != "forbidden" || definition.ShellPolicy != "forbidden" || definition.Limits != spec.limits || definition.Limits.MaxProcess != executableInvocationCount(definition.Invocations) || !equalStrings(definition.Statuses, []string{"complete", "indeterminate", "manual-required"}) {
			return nil, errors.New("real adapter limits rejected")
		}
		if !containsExactSources(definition.Official, spec.sources) {
			return nil, errors.New("real adapter official source rejected")
		}
		usable := proofUsable(definition, registry.now)
		if definition.AdapterID == "launchctl-print-service-v1" {
			usable = false
		}
		registry.definitions[definition.AdapterID] = definition
		registry.usable[definition.AdapterID] = usable
		if usable {
			registry.capabilities[definition.AdapterID] = &realAdapterCapability{registry: registry, adapterID: definition.AdapterID, implementationDigest: definition.ImplementationDigest}
		}
	}
	return registry, nil
}

func (registry *RealAdapterRegistry) Assess(manifest ProtectedManifest) RealProofAssessment {
	canonical, canonicalErr := canonicalProtectedManifest(manifest)
	if registry == nil || manifest.Digest == "" || validateProtectedManifest(manifest) != nil || canonicalErr != nil || sha256Digest(canonical) != manifest.Digest {
		return RealProofAssessment{Status: "harness-error", Verdict: VerdictHarnessError, ExitCode: ExitHarnessError, Reason: "real-adapter-registry-rejected"}
	}
	for _, surface := range manifest.Surfaces {
		definition, ok := registry.definitions[surface.AdapterID]
		if !ok || definition.SurfaceDomain != surface.SurfaceDomain || !containsString(definition.LogicalRefs, surface.LogicalRef) {
			return RealProofAssessment{Status: "manual-required", Verdict: VerdictIndeterminate, ExitCode: 32, Reason: "required-real-adapter-unavailable", SurfaceDomain: surface.SurfaceDomain, LogicalRef: surface.LogicalRef}
		}
		if surface.Bounds.MaxFiles > definition.Limits.MaxFiles || surface.Bounds.MaxBytes > definition.Limits.MaxBytes || surface.Bounds.Timeout > definition.Limits.TimeoutMS || !registry.usable[surface.AdapterID] {
			if surface.Policy == PolicyRequired {
				return RealProofAssessment{Status: "manual-required", Verdict: VerdictIndeterminate, ExitCode: 32, Reason: "required-real-adapter-proof-unavailable", SurfaceDomain: surface.SurfaceDomain, LogicalRef: surface.LogicalRef}
			}
		}
	}
	return RealProofAssessment{Status: "ready", Verdict: VerdictPassed, ExitCode: ExitPassed, Reason: "all-required-real-adapters-proven", ClaimEligible: true}
}

func expectedNegativeDigest(definition RealAdapterDefinition) string {
	spec, ok := realAdapterSpecs[definition.AdapterID]
	if !ok {
		return ""
	}
	canonical, err := json.Marshal(struct {
		AdapterID            string                `json:"adapter_id"`
		Version              string                `json:"version"`
		ImplementationDigest string                `json:"implementation_digest"`
		SurfaceDomain        privacy.SurfaceDomain `json:"surface_domain"`
		LogicalRefs          []string              `json:"logical_refs"`
		Invocations          []AdapterInvocation   `json:"invocations"`
		Limits               AdapterLimits         `json:"limits"`
		Official             []OfficialSource      `json:"official_sources"`
		ReviewedAt           string                `json:"reviewed_at"`
		ValidUntil           *string               `json:"valid_until"`
		SuiteID              string                `json:"suite_id"`
		EvidenceContract     string                `json:"evidence_contract"`
		TestSourceDigest     string                `json:"test_source_digest"`
		Result               string                `json:"result"`
	}{
		AdapterID: definition.AdapterID, Version: realAdapterVersion, ImplementationDigest: spec.implementationDigest,
		SurfaceDomain: spec.domain, LogicalRefs: spec.logicalRefs, Invocations: spec.invocations, Limits: spec.limits,
		Official: definition.Official, ReviewedAt: definition.ReviewedAt, ValidUntil: definition.ValidUntil,
		SuiteID: spec.negativeSuiteID, EvidenceContract: spec.negativeContract, TestSourceDigest: negativeTestSourceDigest, Result: "passed",
	})
	if err != nil {
		return ""
	}
	return sha256Digest(canonical)
}

func proofUsable(definition RealAdapterDefinition, now time.Time) bool {
	spec, ok := realAdapterSpecs[definition.AdapterID]
	if !ok || definition.ProofState != ProofCurrent || definition.ValidUntil == nil || definition.ReviewedAt == "" || definition.NegativeSuite.Status != "passed" || definition.NegativeSuite.SuiteID != spec.negativeSuiteID || definition.NegativeSuite.EvidenceContract != spec.negativeContract || definition.NegativeSuite.TestSourceDigest != negativeTestSourceDigest || definition.NegativeSuite.Digest != expectedNegativeDigest(definition) {
		return false
	}
	reviewed, err := time.Parse("2006-01-02", definition.ReviewedAt)
	if err != nil {
		return false
	}
	validUntil, err := time.Parse("2006-01-02", *definition.ValidUntil)
	if err != nil || validUntil.Before(reviewed) || validUntil.Sub(reviewed) > 30*24*time.Hour {
		return false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return !today.Before(reviewed) && !today.After(validUntil)
}

func equalInvocation(left, right AdapterInvocation) bool {
	return left.Kind == right.Kind && left.API == right.API && left.Executable == right.Executable && equalStrings(left.Argv, right.Argv) && equalStrings(left.Environment, right.Environment)
}

func equalInvocations(left, right []AdapterInvocation) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !equalInvocation(left[index], right[index]) {
			return false
		}
	}
	return true
}

func executableInvocationCount(invocations []AdapterInvocation) int {
	count := 0
	for _, invocation := range invocations {
		if invocation.Kind == "executable" {
			count++
		}
	}
	return count
}

func containsExactSources(got []OfficialSource, expected []string) bool {
	if len(got) != len(expected) {
		return false
	}
	urls := make([]string, 0, len(got))
	for _, source := range got {
		if source.Boundary == "" {
			return false
		}
		urls = append(urls, source.URL)
	}
	sort.Strings(urls)
	want := append([]string(nil), expected...)
	sort.Strings(want)
	return equalStrings(urls, want)
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

type realResolver struct {
	repositoryRoot string
	homeRoot       string
	managerRoot    string
	systemShells   string
}

type realAdapter struct {
	id         string
	capability *realAdapterCapability
	snapshot   func(context.Context, ProtectedSurface, *realResolver, []byte) SurfaceSnapshot
}

type RealEnvelopeOptions struct {
	Manifest      ProtectedManifest
	Registry      *RealAdapterRegistry
	Adapters      map[string]realAdapter
	Retention     *fixture.Retention
	Context       context.Context
	Workload      func(context.Context) (string, error)
	Clock         func() time.Time
	WindowID      string
	ClaimConsumer func(*Evidence, Evaluation, []string) (string, error)
	secretFactory func([]byte) error
}

type RealEnvelope struct {
	Status         string                 `json:"status"`
	Sequence       []string               `json:"sequence"`
	Evaluation     Evaluation             `json:"evaluation"`
	Evidence       *Evidence              `json:"evidence,omitempty"`
	TeardownStatus fixture.TeardownStatus `json:"teardown_status,omitempty"`
}

func RunRealEnvelope(options RealEnvelopeOptions) RealEnvelope {
	frozen, err := FreezeProtectedManifest(options.Manifest)
	if err != nil {
		return harnessEnvelope("real-manifest-freeze-rejected", []string{"proof-gate"})
	}
	manifest := frozen.Manifest()
	assessment := options.Registry.Assess(manifest)
	if !assessment.ClaimEligible {
		return RealEnvelope{Status: assessment.Status, Sequence: []string{"proof-gate"}, Evaluation: Evaluation{Verdict: assessment.Verdict, ExitCode: assessment.ExitCode, Reason: assessment.Reason}}
	}
	if options.Context == nil || options.Workload == nil || !publicManifestID.MatchString(options.WindowID) {
		return harnessEnvelope("real-envelope-input-rejected", []string{"proof-gate"})
	}
	if _, ok := options.Context.Deadline(); !ok {
		return harnessEnvelope("real-envelope-input-rejected", []string{"proof-gate"})
	}
	adapters := make(map[string]realAdapter, len(manifest.Surfaces))
	for _, surface := range manifest.Surfaces {
		adapter, ok := options.Adapters[surface.AdapterID]
		if !ok || adapter.id != surface.AdapterID || !options.Registry.authorizes(adapter) || adapter.snapshot == nil {
			return harnessEnvelope("real-adapter-implementation-rejected", []string{"proof-gate"})
		}
		adapters[surface.AdapterID] = adapter
	}
	secretFactory := options.secretFactory
	if secretFactory == nil {
		secretFactory = func(destination []byte) error {
			_, err := io.ReadFull(rand.Reader, destination)
			return err
		}
	}
	key := make([]byte, 32)
	if err := secretFactory(key); err != nil || allZeroBytes(key) {
		clearSecret(key)
		return harnessEnvelope("real-envelope-key-rejected", []string{"proof-gate"})
	}
	defer clearSecret(key)
	clock := options.Clock
	if clock == nil {
		clock = time.Now
	}
	sequence := []string{"real-before"}
	opened := clock().UTC()
	before := snapshotRealSurfaces(options.Context, manifest, adapters, key)
	primary := snapshotsVerdict(manifest, before)
	sequence = append(sequence, "isolated-workload")
	if options.Context.Err() != nil {
		primary = MonotonicCombine(primary, VerdictIndeterminate)
	} else {
		state, workloadErr := options.Workload(options.Context)
		if options.Context.Err() != nil {
			primary = MonotonicCombine(primary, VerdictIndeterminate)
		} else if workloadErr != nil || state != innerSyntheticSuccess {
			primary = MonotonicCombine(primary, VerdictHarnessError)
		}
	}
	sequence = append(sequence, "freeze-primary")
	if options.Context.Err() != nil {
		primary = MonotonicCombine(primary, VerdictIndeterminate)
	}
	teardownStatus := fixture.TeardownFailed
	if options.Retention == nil {
		primary = MonotonicCombine(primary, VerdictHarnessError)
	} else {
		frozenPrimary, err := fixture.FreezePrimary(toFixtureVerdict(primary))
		if err != nil {
			primary = MonotonicCombine(primary, VerdictHarnessError)
		} else {
			finalized := options.Retention.Finalize(frozenPrimary)
			teardownStatus = finalized.Teardown.Status
			primary = fromFixtureVerdict(finalized.Verdict)
		}
	}
	sequence = append(sequence, "fixture-finalize", "real-after")
	if options.Context.Err() != nil {
		primary = MonotonicCombine(primary, VerdictIndeterminate)
	}
	after := snapshotRealSurfaces(options.Context, manifest, adapters, key)
	if options.Context.Err() != nil {
		primary = MonotonicCombine(primary, VerdictIndeterminate)
	}
	closed := clock().UTC()
	evidence, evidenceErr := BuildEvidence(manifest, before, after, EvidenceOptions{SuiteID: manifest.SuiteID, Tier: "real-sentinel-envelope", WindowID: options.WindowID, OpenedAt: opened, ClosedAt: closed, Provenance: "real"})
	evaluation := Evaluation{Verdict: VerdictHarnessError, ExitCode: ExitHarnessError, Reason: "real-evidence-build-rejected"}
	if evidenceErr == nil {
		if primary == VerdictPassed && snapshotsVerdict(manifest, before) == VerdictPassed && snapshotsVerdict(manifest, after) == VerdictPassed {
			_ = bindRealEvidence(&evidence)
		}
		evaluation = Evaluate(manifest, evidence)
	}
	finalVerdict := MonotonicCombine(primary, evaluation.Verdict)
	// 真实 after 证据一旦确认 required surface 发生变化，D-15 的 bounded violation 不能被较早的 workload/teardown 错误遮蔽。
	if evaluation.Verdict == VerdictViolation {
		finalVerdict = VerdictViolation
	}
	if finalVerdict != evaluation.Verdict {
		evaluation = evaluationForVerdict(finalVerdict, evaluation.EvidenceDigest)
	}
	sequence = append(sequence, "monotonic-combine")
	if finalVerdict != VerdictPassed {
		evaluation.Claim = ""
	} else if evidenceErr == nil {
		var claim string
		var claimErr error
		if options.ClaimConsumer != nil {
			claim, claimErr = options.ClaimConsumer(&evidence, evaluation, append([]string(nil), sequence...))
		} else {
			claim, claimErr = RequestClaim(&evidence, evaluation, ScopedUnchangedClaim)
		}
		if claimErr != nil || claim != ScopedUnchangedClaim || evidence.realBinding != nil {
			finalVerdict = VerdictHarnessError
			evaluation = evaluationForVerdict(VerdictHarnessError, evaluation.EvidenceDigest)
		} else {
			evaluation.Claim = claim
		}
	}
	if evidenceErr == nil {
		evidence.realBinding = nil
	}
	status := string(finalVerdict)
	if evaluation.Claim != "" {
		status = evaluation.Claim
	}
	result := RealEnvelope{Status: status, Sequence: sequence, Evaluation: evaluation, TeardownStatus: teardownStatus}
	if evidenceErr == nil {
		result.Evidence = &evidence
	}
	return result
}

func clearSecret(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

func allZeroBytes(value []byte) bool {
	for _, item := range value {
		if item != 0 {
			return false
		}
	}
	return true
}

func (registry *RealAdapterRegistry) authorizes(adapter realAdapter) bool {
	if registry == nil || adapter.id == "" || adapter.capability == nil {
		return false
	}
	capability := registry.capabilities[adapter.id]
	definition, ok := registry.definitions[adapter.id]
	return ok && registry.usable[adapter.id] && capability != nil && adapter.capability == capability && capability.registry == registry && capability.adapterID == adapter.id && capability.implementationDigest == definition.ImplementationDigest
}

func harnessEnvelope(reason string, sequence []string) RealEnvelope {
	return RealEnvelope{Status: string(VerdictHarnessError), Sequence: sequence, Evaluation: Evaluation{Verdict: VerdictHarnessError, ExitCode: ExitHarnessError, Reason: reason}}
}

func evaluationForVerdict(verdict Verdict, digest string) Evaluation {
	switch verdict {
	case VerdictViolation:
		return Evaluation{Verdict: verdict, ExitCode: ExitViolation, ChangeCode: ChangeDetectedCode, EvidenceDigest: digest}
	case VerdictIndeterminate:
		return Evaluation{Verdict: verdict, ExitCode: ExitIndeterminate, Reason: "primary-run-indeterminate", EvidenceDigest: digest}
	case VerdictHarnessError:
		return Evaluation{Verdict: verdict, ExitCode: ExitHarnessError, Reason: "primary-run-harness-error", EvidenceDigest: digest}
	default:
		return Evaluation{Verdict: VerdictPassed, ExitCode: ExitPassed, EvidenceDigest: digest}
	}
}

func snapshotRealSurfaces(ctx context.Context, manifest ProtectedManifest, adapters map[string]realAdapter, key []byte) ProtectedSnapshot {
	result := ProtectedSnapshot{ManifestDigest: manifest.Digest, WindowState: "closed", Surfaces: make([]SurfaceSnapshot, 0, len(manifest.Surfaces))}
	for _, surface := range manifest.Surfaces {
		if ctx == nil || ctx.Err() != nil {
			result.Surfaces = append(result.Surfaces, incompleteSurface(surface, ReasonWindow))
			continue
		}
		adapter := adapters[surface.AdapterID]
		snapshot := adapter.snapshot(ctx, surface, nil, key)
		result.Surfaces = append(result.Surfaces, snapshot)
	}
	return result
}

func snapshotsVerdict(manifest ProtectedManifest, snapshot ProtectedSnapshot) Verdict {
	byRef, err := snapshotsByRef(snapshot)
	if err != nil {
		return VerdictHarnessError
	}
	for _, surface := range manifest.Surfaces {
		if surface.Policy != PolicyRequired {
			continue
		}
		observed, ok := byRef[surface.LogicalRef]
		if !ok || observed.Status != ObservationComplete || !validOpaqueToken(observed.OpaqueState) {
			return VerdictIndeterminate
		}
	}
	return VerdictPassed
}

func toFixtureVerdict(verdict Verdict) fixture.PrimaryVerdict {
	switch verdict {
	case VerdictPassed:
		return fixture.VerdictPassed
	case VerdictViolation:
		return fixture.VerdictViolation
	case VerdictIndeterminate:
		return fixture.VerdictIndeterminate
	default:
		return fixture.VerdictHarnessError
	}
}

func fromFixtureVerdict(verdict fixture.PrimaryVerdict) Verdict {
	switch verdict {
	case fixture.VerdictPassed:
		return VerdictPassed
	case fixture.VerdictViolation:
		return VerdictViolation
	case fixture.VerdictIndeterminate:
		return VerdictIndeterminate
	default:
		return VerdictHarnessError
	}
}

func newRealResolver(repositoryRoot, homeRoot, managerRoot string) (*realResolver, error) {
	repository, err := canonicalDirectory(repositoryRoot)
	if err != nil {
		return nil, errors.New("real repository mapping rejected")
	}
	home, err := canonicalDirectory(homeRoot)
	if err != nil {
		return nil, errors.New("real home mapping rejected")
	}
	if managerRoot == "" || !filepath.IsAbs(managerRoot) {
		return nil, errors.New("real manager mapping rejected")
	}
	manager := filepath.Clean(managerRoot)
	inside, err := withinRoot(home, manager)
	if err != nil || !inside {
		return nil, errors.New("real manager mapping rejected")
	}
	relativeManager, reason := rootedRelative(home, manager)
	if reason != "" {
		return nil, errors.New("real manager mapping rejected")
	}
	homeHandle, err := os.OpenRoot(home)
	if err != nil {
		return nil, errors.New("real manager mapping rejected")
	}
	managerInfo, managerErr := homeHandle.Lstat(relativeManager)
	_ = homeHandle.Close()
	if managerErr != nil && !errors.Is(managerErr, os.ErrNotExist) {
		return nil, errors.New("real manager mapping rejected")
	}
	if managerErr == nil && (!managerInfo.IsDir() || managerInfo.Mode()&os.ModeSymlink != 0) {
		return nil, errors.New("real manager mapping rejected")
	}
	return &realResolver{repositoryRoot: repository, homeRoot: home, managerRoot: manager, systemShells: "/etc/shells"}, nil
}

func defaultRealAdapters(registry *RealAdapterRegistry, resolver *realResolver) map[string]realAdapter {
	if registry == nil || resolver == nil {
		return nil
	}
	frozenResolver := *resolver
	resolver = &frozenResolver
	gitCapability := newGitVersionCapability(resolver.repositoryRoot)
	return map[string]realAdapter{
		"git-worktree-readonly-v1": {id: "git-worktree-readonly-v1", capability: registry.capabilities["git-worktree-readonly-v1"], snapshot: func(ctx context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
			return snapshotGitWorktree(ctx, surface, resolver, key, gitCapability)
		}},
		"git-index-readonly-v1": {id: "git-index-readonly-v1", capability: registry.capabilities["git-index-readonly-v1"], snapshot: func(ctx context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
			return snapshotGitIndex(ctx, surface, resolver, key, gitCapability)
		}},
		"go-lstat-file-v1": {id: "go-lstat-file-v1", capability: registry.capabilities["go-lstat-file-v1"], snapshot: func(ctx context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
			return snapshotExactFileWithContext(ctx, surface, filepath.Join(resolver.homeRoot, ".zshrc"), resolver.homeRoot, resolver.repositoryRoot, key)
		}},
		"go-bounded-tree-v1": {id: "go-bounded-tree-v1", capability: registry.capabilities["go-bounded-tree-v1"], snapshot: func(ctx context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
			return snapshotExactTreeWithContext(ctx, surface, resolver.managerRoot, resolver.homeRoot, key)
		}},
		"launchctl-print-service-v1": {id: "launchctl-print-service-v1", capability: registry.capabilities["launchctl-print-service-v1"], snapshot: func(_ context.Context, surface ProtectedSurface, _ *realResolver, _ []byte) SurfaceSnapshot {
			return incompleteSurface(surface, ReasonUnreadable)
		}},
		"go-system-shells-file-v1": {id: "go-system-shells-file-v1", capability: registry.capabilities["go-system-shells-file-v1"], snapshot: func(ctx context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
			return snapshotExactFileWithContext(ctx, surface, resolver.systemShells, filepath.Dir(resolver.systemShells), "", key)
		}},
	}
}

type gitVersionCapability struct {
	once      sync.Once
	directory string
	supported bool
	probes    int
}

func newGitVersionCapability(directory string) *gitVersionCapability {
	return &gitVersionCapability{directory: directory}
}

func (capability *gitVersionCapability) supports(ctx context.Context) bool {
	if capability == nil || capability.directory == "" {
		return false
	}
	capability.once.Do(func() {
		capability.probes++
		capability.supported = supportedGitVersion(ctx, capability.directory)
	})
	return capability.supported
}

func snapshotGitWorktree(ctx context.Context, surface ProtectedSurface, resolver *realResolver, key []byte, capability *gitVersionCapability) SurfaceSnapshot {
	if resolver == nil || surface.LogicalRef != "repo:sentinel/worktree/tracked" || capability == nil || capability.directory != resolver.repositoryRoot {
		return incompleteSurface(surface, ReasonUnreadable)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(surface.Bounds.Timeout)*time.Millisecond)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if !capability.supports(ctx) {
		return incompleteSurface(surface, commandIncompleteReason(ctx))
	}
	status, err := runGit(ctx, resolver.repositoryRoot, surface.Bounds, []string{"--no-lazy-fetch", "-c", "core.fsmonitor=false", "status", "--porcelain=v2", "--untracked-files=no", "--ignore-submodules=all"})
	if err != nil {
		return incompleteSurface(surface, commandIncompleteReason(ctx))
	}
	lines := bytes.Split(bytes.TrimSuffix(status, []byte{'\n'}), []byte{'\n'})
	if len(lines) == 1 && len(lines[0]) == 0 {
		lines = nil
	}
	for _, line := range lines {
		if len(line) < 2 || (line[0] != '1' && line[0] != '2' && line[0] != 'u') || line[1] != ' ' {
			return incompleteSurface(surface, ReasonUnreadable)
		}
	}
	sort.Slice(lines, func(i, j int) bool { return bytes.Compare(lines[i], lines[j]) < 0 })
	indexOutput, err := runGit(ctx, resolver.repositoryRoot, surface.Bounds, []string{"--no-lazy-fetch", "ls-files", "-z", "--stage"})
	if err != nil {
		return incompleteSurface(surface, commandIncompleteReason(ctx))
	}
	records, paths, err := parseGitIndex(indexOutput)
	if err != nil {
		return incompleteSurface(surface, ReasonUnreadable)
	}
	facts, reason := fingerprintTrackedPaths(resolver.repositoryRoot, paths, surface.Bounds, deadline)
	if reason != "" {
		return incompleteSurface(surface, reason)
	}
	canonical, _ := json.Marshal(struct {
		Status  [][]byte `json:"status"`
		Index   []string `json:"index"`
		Tracked []string `json:"tracked"`
	}{lines, records, facts})
	return completeSurface(surface, key, canonical)
}

func snapshotGitIndex(ctx context.Context, surface ProtectedSurface, resolver *realResolver, key []byte, capability *gitVersionCapability) SurfaceSnapshot {
	if resolver == nil || surface.LogicalRef != "repo:sentinel/worktree/index" || capability == nil || capability.directory != resolver.repositoryRoot {
		return incompleteSurface(surface, ReasonUnreadable)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(surface.Bounds.Timeout)*time.Millisecond)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if !capability.supports(ctx) {
		return incompleteSurface(surface, commandIncompleteReason(ctx))
	}
	output, err := runGit(ctx, resolver.repositoryRoot, surface.Bounds, []string{"--no-lazy-fetch", "ls-files", "-z", "--stage"})
	if err != nil {
		return incompleteSurface(surface, commandIncompleteReason(ctx))
	}
	records, _, err := parseGitIndex(output)
	if err != nil {
		return incompleteSurface(surface, ReasonUnreadable)
	}
	if len(records) > surface.Bounds.MaxFiles {
		return incompleteSurface(surface, ReasonOverflow)
	}
	indexPath := filepath.Join(resolver.repositoryRoot, ".git", "index")
	indexCanonical, reason := readExactRegularFile(indexPath, filepath.Join(resolver.repositoryRoot, ".git"), surface.Bounds.MaxBytes, deadline)
	if reason != "" {
		return incompleteSurface(surface, reason)
	}
	canonical, _ := json.Marshal(struct {
		Entries []string `json:"entries"`
		Index   []byte   `json:"index"`
	}{records, indexCanonical})
	return completeSurface(surface, key, canonical)
}

func supportedGitVersion(ctx context.Context, directory string) bool {
	limits := SurfaceBounds{MaxFiles: 1, MaxBytes: 256, Timeout: 1000}
	output, err := runBoundedCommand(ctx, "/usr/bin/git", gitVersionInvocation.Argv, directory, gitVersionInvocation.Environment, limits)
	if err != nil {
		return false
	}
	text := strings.TrimSpace(string(output))
	parts := strings.Fields(text)
	if len(parts) < 3 || parts[0] != "git" || parts[1] != "version" {
		return false
	}
	versionParts := strings.Split(parts[2], ".")
	if len(versionParts) < 2 {
		return false
	}
	major, majorErr := strconv.Atoi(versionParts[0])
	minor, minorErr := strconv.Atoi(versionParts[1])
	return majorErr == nil && minorErr == nil && (major > 2 || (major == 2 && minor >= 49))
}

func runGit(ctx context.Context, directory string, bounds SurfaceBounds, argv []string) ([]byte, error) {
	return runBoundedCommand(ctx, "/usr/bin/git", argv, directory, gitReadOnlyEnvironment, bounds)
}

func commandIncompleteReason(ctx context.Context) IncompleteReason {
	if ctx != nil && ctx.Err() != nil {
		return ReasonWindow
	}
	return ReasonUnreadable
}

type boundedCommandBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
}

func (buffer *boundedCommandBuffer) Write(data []byte) (int, error) {
	if buffer.buffer.Len()+len(data) > buffer.limit {
		remaining := buffer.limit - buffer.buffer.Len()
		if remaining > 0 {
			_, _ = buffer.buffer.Write(data[:remaining])
		}
		buffer.overflow = true
		return len(data), nil
	}
	return buffer.buffer.Write(data)
}

func runBoundedCommand(parent context.Context, executable string, argv []string, directory string, environment []string, bounds SurfaceBounds) ([]byte, error) {
	timeout := time.Duration(bounds.Timeout) * time.Millisecond
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	stdout := &boundedCommandBuffer{limit: minInt(bounds.MaxBytes, maxCommandOutput)}
	stderr := &boundedCommandBuffer{limit: minInt(bounds.MaxBytes, maxCommandOutput)}
	command := exec.CommandContext(ctx, executable, argv...)
	command.Dir = directory
	command.Env = append([]string(nil), environment...)
	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr
	err := command.Run()
	if ctx.Err() != nil || stdout.overflow || stderr.overflow || err != nil || stderr.buffer.Len() != 0 {
		return nil, errors.New("bounded read-only command rejected")
	}
	return bytes.Clone(stdout.buffer.Bytes()), nil
}

func parseGitIndex(data []byte) ([]string, []string, error) {
	if len(data) == 0 {
		return []string{}, []string{}, nil
	}
	parts := bytes.Split(data, []byte{0})
	if len(parts[len(parts)-1]) != 0 {
		return nil, nil, errors.New("git index output rejected")
	}
	parts = parts[:len(parts)-1]
	records := make([]string, 0, len(parts))
	paths := make([]string, 0, len(parts))
	for _, record := range parts {
		metadata, pathBytes, ok := bytes.Cut(record, []byte{'\t'})
		fields := bytes.Fields(metadata)
		if !ok || len(fields) != 3 || len(pathBytes) == 0 || bytes.ContainsRune(pathBytes, 0) {
			return nil, nil, errors.New("git index output rejected")
		}
		mode := string(fields[0])
		object := string(fields[1])
		stage := string(fields[2])
		if _, err := strconv.ParseUint(mode, 8, 32); err != nil || (len(object) != 40 && len(object) != 64) || !isHex(object) || (stage != "0" && stage != "1" && stage != "2" && stage != "3") {
			return nil, nil, errors.New("git index output rejected")
		}
		path := string(pathBytes)
		if filepath.IsAbs(path) || strings.ContainsRune(path, '\x00') || path == "." || path == ".." || strings.HasPrefix(filepath.ToSlash(path), "../") {
			return nil, nil, errors.New("git index path rejected")
		}
		records = append(records, mode+" "+object+" "+stage+"\t"+path)
		paths = append(paths, path)
	}
	sort.Strings(records)
	sort.Strings(paths)
	return records, paths, nil
}

func fingerprintTrackedPaths(root string, paths []string, bounds SurfaceBounds, deadline time.Time) ([]string, IncompleteReason) {
	facts := make([]string, 0, len(paths))
	total := 0
	if len(paths) > bounds.MaxFiles {
		return nil, ReasonOverflow
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, ReasonUnreadable
	}
	defer rootHandle.Close()
	for _, path := range paths {
		if time.Now().After(deadline) {
			return nil, ReasonWindow
		}
		candidate := filepath.Join(root, filepath.FromSlash(path))
		relative, reason := rootedRelative(root, candidate)
		if reason != "" {
			return nil, reason
		}
		info, err := rootHandle.Lstat(relative)
		if errors.Is(err, os.ErrNotExist) {
			facts = append(facts, path+"\x00absent")
			continue
		}
		if err != nil {
			return nil, ReasonUnreadable
		}
		remaining := bounds.MaxBytes - total
		if info.Mode()&os.ModeSymlink != 0 {
			canonical, reason := readExactSymlink(candidate, root, remaining, deadline)
			if reason != "" {
				return nil, reason
			}
			current, err := rootHandle.Lstat(relative)
			if err != nil || !sameEntryState(info, current) {
				return nil, ReasonRace
			}
			total += int(info.Size())
			facts = append(facts, path+"\x00symlink\x00"+hex.EncodeToString(canonical))
			continue
		}
		if !info.Mode().IsRegular() {
			return nil, ReasonUnreadable
		}
		canonical, reason := readExactRegularFile(candidate, root, remaining, deadline)
		if reason != "" {
			return nil, reason
		}
		current, err := rootHandle.Lstat(relative)
		if err != nil || !sameEntryState(info, current) {
			return nil, ReasonRace
		}
		total += int(info.Size())
		facts = append(facts, path+"\x00"+hex.EncodeToString(canonical))
	}
	sort.Strings(facts)
	return facts, ""
}

func snapshotExactFile(surface ProtectedSurface, path, root, allowedSymlinkRoot string, key []byte) SurfaceSnapshot {
	return snapshotExactFileWithContext(context.Background(), surface, path, root, allowedSymlinkRoot, key)
}

func snapshotExactFileWithContext(ctx context.Context, surface ProtectedSurface, path, root, allowedSymlinkRoot string, key []byte) SurfaceSnapshot {
	if ctx == nil {
		return incompleteSurface(surface, ReasonWindow)
	}
	child, cancel := context.WithTimeout(ctx, time.Duration(surface.Bounds.Timeout)*time.Millisecond)
	defer cancel()
	deadline, ok := boundedObservationDeadline(child, surface.Bounds)
	if !ok {
		return incompleteSurface(surface, ReasonWindow)
	}
	canonical, reason := readExactNamedEntry(path, root, allowedSymlinkRoot, surface.Bounds.MaxBytes, deadline)
	if reason != "" {
		return incompleteSurface(surface, reason)
	}
	if child.Err() != nil {
		return incompleteSurface(surface, ReasonWindow)
	}
	second, reason := readExactNamedEntry(path, root, allowedSymlinkRoot, surface.Bounds.MaxBytes, deadline)
	if reason != "" {
		return incompleteSurface(surface, reason)
	}
	if !bytes.Equal(canonical, second) {
		return incompleteSurface(surface, ReasonRace)
	}
	if child.Err() != nil {
		return incompleteSurface(surface, ReasonWindow)
	}
	return completeSurface(surface, key, canonical)
}

func snapshotExactTree(surface ProtectedSurface, path, root string, key []byte) SurfaceSnapshot {
	return snapshotExactTreeWithContext(context.Background(), surface, path, root, key)
}

func snapshotExactTreeWithContext(ctx context.Context, surface ProtectedSurface, path, root string, key []byte) SurfaceSnapshot {
	if ctx == nil {
		return incompleteSurface(surface, ReasonWindow)
	}
	child, cancel := context.WithTimeout(ctx, time.Duration(surface.Bounds.Timeout)*time.Millisecond)
	defer cancel()
	deadline, ok := boundedObservationDeadline(child, surface.Bounds)
	if !ok {
		return incompleteSurface(surface, ReasonWindow)
	}
	canonical, reason := fingerprintExactTree(surface, path, root, deadline)
	if reason != "" {
		return incompleteSurface(surface, reason)
	}
	if child.Err() != nil {
		return incompleteSurface(surface, ReasonWindow)
	}
	second, reason := fingerprintExactTree(surface, path, root, deadline)
	if reason != "" {
		return incompleteSurface(surface, reason)
	}
	if !bytes.Equal(canonical, second) {
		return incompleteSurface(surface, ReasonRace)
	}
	if child.Err() != nil {
		return incompleteSurface(surface, ReasonWindow)
	}
	return completeSurface(surface, key, canonical)
}

func boundedObservationDeadline(ctx context.Context, bounds SurfaceBounds) (time.Time, bool) {
	if ctx == nil || ctx.Err() != nil || bounds.Timeout < 1 {
		return time.Time{}, false
	}
	deadline := time.Now().Add(time.Duration(bounds.Timeout) * time.Millisecond)
	if outer, ok := ctx.Deadline(); ok && outer.Before(deadline) {
		deadline = outer
	}
	if !time.Now().Before(deadline) {
		return time.Time{}, false
	}
	return deadline, true
}

func fingerprintExactTree(surface ProtectedSurface, path, root string, deadline time.Time) ([]byte, IncompleteReason) {
	relativeRoot, reason := rootedRelative(root, path)
	if reason != "" {
		return nil, reason
	}
	treeRoot, err := canonicalDirectory(path)
	if err != nil {
		return nil, ReasonUnreadable
	}
	canonicalObservationRoot, err := canonicalDirectory(root)
	if err != nil {
		return nil, ReasonUnreadable
	}
	if inside, err := withinRoot(canonicalObservationRoot, treeRoot); err != nil || !inside {
		return nil, ReasonSymlinkEscape
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, ReasonUnreadable
	}
	defer rootHandle.Close()
	if time.Now().After(deadline) {
		return nil, ReasonWindow
	}
	rootInfo, err := rootHandle.Lstat(relativeRoot)
	if errors.Is(err, os.ErrNotExist) {
		return []byte(`{"kind":"absent"}`), ""
	}
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, ReasonUnreadable
	}
	facts := make([]string, 0)
	files := 0
	bytesRead := 0
	walkErr := fs.WalkDir(rootHandle.FS(), relativeRoot, func(current string, _ fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.New(string(ReasonUnreadable))
		}
		if time.Now().After(deadline) {
			return errors.New(string(ReasonWindow))
		}
		files++
		if files > surface.Bounds.MaxFiles {
			return errors.New(string(ReasonOverflow))
		}
		info, err := rootHandle.Lstat(current)
		if err != nil {
			return errors.New(string(ReasonUnreadable))
		}
		if info.Mode()&os.ModeSymlink != 0 {
			remaining := surface.Bounds.MaxBytes - bytesRead
			candidate := filepath.Join(root, filepath.FromSlash(current))
			observation, reason := observeExactSymlink(candidate, root, remaining, deadline)
			if reason != "" {
				return errors.New(string(reason))
			}
			if reason := validateTreeSymlinkTarget(candidate, observation.target, treeRoot); reason != "" {
				return errors.New(string(reason))
			}
			entryAfter, err := rootHandle.Lstat(current)
			if err != nil || !sameEntryState(info, entryAfter) {
				return errors.New(string(ReasonRace))
			}
			bytesRead += int(info.Size())
			relative, err := filepath.Rel(filepath.FromSlash(relativeRoot), filepath.FromSlash(current))
			if err != nil {
				return errors.New(string(ReasonUnreadable))
			}
			facts = append(facts, filepath.ToSlash(relative)+"\x00symlink\x00"+hex.EncodeToString(observation.canonical))
			return nil
		}
		relative, err := filepath.Rel(filepath.FromSlash(relativeRoot), filepath.FromSlash(current))
		if err != nil {
			return errors.New(string(ReasonUnreadable))
		}
		if info.IsDir() {
			facts = append(facts, filepath.ToSlash(relative)+"\x00directory\x00"+strconv.FormatUint(uint64(info.Mode()), 10))
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New(string(ReasonUnreadable))
		}
		remaining := surface.Bounds.MaxBytes - bytesRead
		candidate := filepath.Join(root, filepath.FromSlash(current))
		canonical, reason := readExactRegularFile(candidate, root, remaining, deadline)
		if reason != "" {
			return errors.New(string(reason))
		}
		entryAfter, err := rootHandle.Lstat(current)
		if err != nil || !sameEntryState(info, entryAfter) {
			return errors.New(string(ReasonRace))
		}
		bytesRead += int(info.Size())
		facts = append(facts, filepath.ToSlash(relative)+"\x00regular\x00"+hex.EncodeToString(canonical))
		return nil
	})
	if walkErr != nil {
		reason := IncompleteReason(walkErr.Error())
		switch reason {
		case ReasonUnreadable, ReasonRace, ReasonOverflow, ReasonSymlinkEscape, ReasonWindow:
			return nil, reason
		default:
			return nil, ReasonUnreadable
		}
	}
	sort.Strings(facts)
	canonical, _ := json.Marshal(facts)
	return canonical, ""
}

func validateTreeSymlinkTarget(linkPath, target, treeRoot string) IncompleteReason {
	canonicalTreeRoot, err := filepath.EvalSymlinks(treeRoot)
	if err != nil {
		return ReasonUnreadable
	}
	canonicalTreeRoot, err = filepath.Abs(canonicalTreeRoot)
	if err != nil {
		return ReasonUnreadable
	}
	targetPath := target
	if !filepath.IsAbs(targetPath) {
		canonicalParent, err := filepath.EvalSymlinks(filepath.Dir(linkPath))
		if err != nil {
			return ReasonUnreadable
		}
		canonicalParent, err = filepath.Abs(canonicalParent)
		if err != nil {
			return ReasonUnreadable
		}
		targetPath = filepath.Join(canonicalParent, filepath.FromSlash(targetPath))
		targetPath = filepath.Clean(targetPath)
		inside, err := withinRoot(canonicalTreeRoot, targetPath)
		if err != nil || !inside {
			return ReasonSymlinkEscape
		}
	}
	targetPath = filepath.Clean(targetPath)
	resolved, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return ReasonUnreadable
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return ReasonUnreadable
	}
	inside, err := withinRoot(canonicalTreeRoot, resolved)
	if err != nil || !inside {
		return ReasonSymlinkEscape
	}
	return ""
}

func readExactNamedEntry(path, root, allowedSymlinkRoot string, maxBytes int, deadline time.Time) ([]byte, IncompleteReason) {
	relative, reason := rootedRelative(root, path)
	if reason != "" {
		return nil, reason
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, ReasonUnreadable
	}
	defer rootHandle.Close()
	if time.Now().After(deadline) {
		return nil, ReasonWindow
	}
	info, err := rootHandle.Lstat(relative)
	if errors.Is(err, os.ErrNotExist) {
		return []byte(`{"kind":"absent"}`), ""
	}
	if err != nil {
		return nil, ReasonUnreadable
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, reason := observeExactSymlink(path, root, maxBytes, deadline)
		if reason != "" {
			return nil, reason
		}
		resolvedTarget := link.target
		if !filepath.IsAbs(resolvedTarget) {
			resolvedTarget = filepath.Join(filepath.Dir(path), filepath.FromSlash(resolvedTarget))
		}
		resolvedTarget = filepath.Clean(resolvedTarget)
		targetRoot := ""
		if inside, err := withinRoot(root, resolvedTarget); err == nil && inside {
			targetRoot = root
		} else if allowedSymlinkRoot != "" {
			if inside, err := withinRoot(allowedSymlinkRoot, resolvedTarget); err == nil && inside {
				targetRoot = allowedSymlinkRoot
			}
		}
		if targetRoot == "" {
			return nil, ReasonSymlinkEscape
		}
		remaining := maxBytes - len(link.target)
		target, reason := readExactRegularFile(resolvedTarget, targetRoot, remaining, deadline)
		if reason != "" {
			return nil, reason
		}
		canonical, _ := json.Marshal(struct {
			Link   json.RawMessage `json:"link"`
			Target json.RawMessage `json:"target"`
		}{json.RawMessage(link.canonical), json.RawMessage(target)})
		return canonical, ""
	}
	if !info.Mode().IsRegular() {
		return nil, ReasonUnreadable
	}
	return readExactRegularFile(path, root, maxBytes, deadline)
}

func readExactSymlink(path, root string, maxBytes int, deadline time.Time) ([]byte, IncompleteReason) {
	observation, reason := observeExactSymlink(path, root, maxBytes, deadline)
	return observation.canonical, reason
}

type symlinkObservation struct {
	canonical []byte
	target    string
}

func observeExactSymlink(path, root string, maxBytes int, deadline time.Time) (symlinkObservation, IncompleteReason) {
	relative, reason := rootedRelative(root, path)
	if reason != "" {
		return symlinkObservation{}, reason
	}
	if maxBytes < 1 {
		return symlinkObservation{}, ReasonOverflow
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return symlinkObservation{}, ReasonUnreadable
	}
	defer rootHandle.Close()
	if time.Now().After(deadline) {
		return symlinkObservation{}, ReasonWindow
	}
	before, err := rootHandle.Lstat(relative)
	if err != nil || before.Mode()&os.ModeSymlink == 0 {
		return symlinkObservation{}, ReasonRace
	}
	if before.Size() > int64(maxBytes) {
		return symlinkObservation{}, ReasonOverflow
	}
	target, err := rootHandle.Readlink(relative)
	if err != nil || len(target) > maxBytes {
		if len(target) > maxBytes {
			return symlinkObservation{}, ReasonOverflow
		}
		return symlinkObservation{}, ReasonUnreadable
	}
	after, err := rootHandle.Lstat(relative)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return symlinkObservation{}, ReasonRace
	}
	if time.Now().After(deadline) {
		return symlinkObservation{}, ReasonWindow
	}
	targetDigest := sha256.Sum256([]byte(target))
	canonical, _ := json.Marshal(struct {
		Kind   string `json:"kind"`
		Mode   uint32 `json:"mode"`
		Size   int64  `json:"size"`
		Target string `json:"target_digest"`
	}{"symlink", uint32(after.Mode()), after.Size(), hex.EncodeToString(targetDigest[:])})
	return symlinkObservation{canonical: canonical, target: target}, ""
}

func readExactRegularFile(path, root string, maxBytes int, deadline time.Time) ([]byte, IncompleteReason) {
	relative, reason := rootedRelative(root, path)
	if reason != "" {
		return nil, ReasonSymlinkEscape
	}
	if maxBytes < 1 {
		return nil, ReasonOverflow
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, ReasonUnreadable
	}
	defer rootHandle.Close()
	if time.Now().After(deadline) {
		return nil, ReasonWindow
	}
	before, err := rootHandle.Lstat(relative)
	if errors.Is(err, os.ErrNotExist) {
		return []byte(`{"kind":"absent"}`), ""
	}
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Size() > int64(maxBytes) || before.Mode().Perm()&0o444 == 0 {
		if err == nil && before.Size() > int64(maxBytes) {
			return nil, ReasonOverflow
		}
		return nil, ReasonUnreadable
	}
	file, err := rootHandle.OpenFile(relative, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ReasonUnreadable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, ReasonRace
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, ReasonUnreadable
	}
	if len(data) > maxBytes {
		return nil, ReasonOverflow
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(opened, after) || opened.Size() != after.Size() || opened.Mode() != after.Mode() || !opened.ModTime().Equal(after.ModTime()) {
		return nil, ReasonRace
	}
	namedAfter, err := rootHandle.Lstat(relative)
	if err != nil || !os.SameFile(after, namedAfter) || time.Now().After(deadline) {
		if time.Now().After(deadline) {
			return nil, ReasonWindow
		}
		return nil, ReasonRace
	}
	content := sha256.Sum256(data)
	canonical, _ := json.Marshal(struct {
		Kind    string `json:"kind"`
		Mode    uint32 `json:"mode"`
		Size    int64  `json:"size"`
		Content string `json:"content"`
	}{"regular", uint32(after.Mode()), after.Size(), hex.EncodeToString(content[:])})
	return canonical, ""
}

func rootedRelative(root, candidate string) (string, IncompleteReason) {
	inside, err := withinRoot(root, candidate)
	if err != nil || !inside {
		return "", ReasonSymlinkEscape
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", ReasonSymlinkEscape
	}
	relative = filepath.ToSlash(relative)
	if !fs.ValidPath(relative) {
		return "", ReasonSymlinkEscape
	}
	return relative, ""
}

func sameEntryState(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

func completeSurface(surface ProtectedSurface, key, canonical []byte) SurfaceSnapshot {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(surface.SurfaceID))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write(canonical)
	return SurfaceSnapshot{SurfaceID: surface.SurfaceID, SurfaceDomain: string(surface.SurfaceDomain), LogicalRef: surface.LogicalRef, Status: ObservationComplete, OpaqueState: "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))}
}

func incompleteSurface(surface ProtectedSurface, reason IncompleteReason) SurfaceSnapshot {
	return SurfaceSnapshot{SurfaceID: surface.SurfaceID, SurfaceDomain: string(surface.SurfaceDomain), LogicalRef: surface.LogicalRef, Status: ObservationIncomplete, Reason: reason}
}

func canonicalDirectory(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("directory rejected")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("directory rejected")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.New("directory rejected")
	}
	return filepath.Abs(resolved)
}

func isHex(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
