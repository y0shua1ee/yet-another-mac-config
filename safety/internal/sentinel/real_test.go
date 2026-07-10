package sentinel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"example.invalid/yamc/safety/internal/fixture"
)

func TestRealSentinelEnvelope(t *testing.T) {
	t.Run("requires fresh exact adapter proof", testRealAdapterRegistry)
	t.Run("keeps Git adapters read-only in an isolated repository", testReadOnlyGitAdapters)
	t.Run("keeps Go adapters read-only in isolated paths", testReadOnlyGoAdapters)
	t.Run("owns and clears the per-run key", testRealEnvelopeOwnsKey)
	t.Run("binds the exact outer sequence and scoped claim", testRealEnvelopePass)
	t.Run("preserves failures through teardown and after observation", testRealEnvelopeMonotonicity)
	t.Run("rejects synthetic implementations before the workload", testRealEnvelopeRejectsSyntheticAdapters)
}

func testRealEnvelopeOwnsKey(t *testing.T) {
	if _, exposed := reflect.TypeOf(RealEnvelopeOptions{}).FieldByName("Key"); exposed {
		t.Fatal("real envelope still accepts caller-owned key material")
	}

	manifest := loadProtectedManifest(t)
	registry := testOnlyReadyRegistry(t)
	surfaceRoot := t.TempDir()
	resolver, err := PrepareProtectedSynthetic(surfaceRoot)
	if err != nil {
		t.Fatal("isolated analog setup failed")
	}
	root, _ := createEnvelopeFixture(t, surfaceRoot, false)
	var failedKey []byte
	workloadCalled := false
	result := runRealEnvelope(t, RealEnvelopeOptions{
		Manifest: manifest, Registry: registry, Adapters: isolatedRealAdapters(manifest, registry, resolver, ""), Retention: root.Retention(),
		Workload: func(_ context.Context) (string, error) {
			workloadCalled = true
			return innerSyntheticSuccess, nil
		},
		Clock: envelopeClock(), WindowID: "real-envelope-key-failure",
		secretFactory: func(destination []byte) error {
			failedKey = destination
			destination[0] = 0x7f
			return errors.New("synthetic entropy failure")
		},
	})
	if result.Evaluation.Verdict != VerdictHarnessError || result.Evaluation.Reason != "real-envelope-key-rejected" || workloadCalled || !reflect.DeepEqual(result.Sequence, []string{"proof-gate"}) {
		t.Fatal("entropy failure did not stop before observation and workload")
	}
	assertZeroedSecret(t, failedKey)
}

func testRealAdapterRegistry(t *testing.T) {
	manifest := loadProtectedManifest(t)
	data := readRealAdapterManifest(t)
	testSource, err := os.ReadFile(filepath.Join(safetyRoot(t), "internal", "sentinel", "real_test.go"))
	if err != nil || sha256Digest(testSource) != negativeTestSourceDigest {
		t.Fatal("negative-suite digest is not bound to the tracked isolated test source")
	}
	implementationSource, err := os.ReadFile(filepath.Join(safetyRoot(t), "internal", "sentinel", "real.go"))
	implementationStart := bytes.Index(implementationSource, []byte("func LoadRealAdapterRegistry("))
	if err != nil || implementationStart < 0 || sha256Digest(implementationSource[implementationStart:]) != realImplementationSourceDigest {
		t.Fatal("adapter implementation digest is not bound to the tracked real envelope source")
	}
	reviewed := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	registry, err := LoadRealAdapterRegistry(data, reviewed)
	if err != nil {
		t.Fatal("tracked real adapter registry was rejected")
	}
	assessment := registry.Assess(manifest)
	if assessment.Status != "manual-required" || assessment.Verdict != VerdictIndeterminate || assessment.ExitCode != 32 || assessment.Reason != "required-real-adapter-proof-unavailable" || assessment.ClaimEligible || assessment.SurfaceDomain != "service" || assessment.LogicalRef != "profile:sentinel/service/homebrew-mxcl-nginx" {
		t.Fatal("missing launchctl proof did not stop before real observation")
	}
	readyOnly := RequireControlledRealEnvelope(RealProofAssessment{Status: "ready", Verdict: VerdictPassed, ExitCode: 0, Reason: "all-required-real-adapters-proven", ClaimEligible: true})
	if readyOnly.Status != "manual-required" || readyOnly.Verdict != VerdictIndeterminate || readyOnly.ExitCode != 32 || readyOnly.Reason != "controlled-real-envelope-runner-required" || readyOnly.ClaimEligible || readyOnly.LogicalRef != "" || readyOnly.SurfaceDomain != "" {
		t.Fatal("proof metadata alone retained CLI claim capability")
	}
	if len(registry.definitions) != len(realAdapterSpecs) || len(registry.usable) != len(realAdapterSpecs) {
		t.Fatal("real adapter registry is not closed")
	}
	for adapterID := range realAdapterSpecs {
		wantUsable := adapterID != "launchctl-print-service-v1"
		if registry.usable[adapterID] != wantUsable {
			t.Fatalf("unexpected tracked proof state for %s", adapterID)
		}
		definition := registry.definitions[adapterID]
		if definition.ImplementationDigest != realImplementationSourceDigest || definition.NegativeSuite.TestSourceDigest != negativeTestSourceDigest || (wantUsable && definition.NegativeSuite.Digest != expectedNegativeDigest(definition)) {
			t.Fatalf("negative evidence is not exact for %s: want=%s got=%s", adapterID, expectedNegativeDigest(definition), definition.NegativeSuite.Digest)
		}
	}

	stale, err := LoadRealAdapterRegistry(data, time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal("stale proof should remain parseable and fail at the proof gate")
	}
	staleAssessment := stale.Assess(manifest)
	if staleAssessment.Status != "manual-required" || staleAssessment.Verdict != VerdictIndeterminate || staleAssessment.ExitCode == 0 || staleAssessment.ClaimEligible {
		t.Fatal("stale proof returned ready")
	}

	var parsed RealAdapterManifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal("real adapter negative setup failed")
	}
	badDigest := parsed
	badDigest.Adapters = append([]RealAdapterDefinition(nil), parsed.Adapters...)
	badDigest.Adapters[0].NegativeSuite.Digest = "sha256:" + strings.Repeat("0", 64)
	badDigestData, _ := json.Marshal(badDigest)
	badDigestRegistry, err := LoadRealAdapterRegistry(badDigestData, reviewed)
	if err != nil {
		t.Fatal("failed negative evidence should be represented as unusable proof")
	}
	if result := badDigestRegistry.Assess(manifest); result.Status != "manual-required" || result.ExitCode == 0 || result.ClaimEligible {
		t.Fatal("mismatched negative-suite digest returned ready")
	}

	badInvocation := parsed
	badInvocation.Adapters = append([]RealAdapterDefinition(nil), parsed.Adapters...)
	badInvocation.Adapters[0].Invocations = append([]AdapterInvocation(nil), parsed.Adapters[0].Invocations...)
	badInvocation.Adapters[0].Invocations[0].Argv = append([]string(nil), parsed.Adapters[0].Invocations[0].Argv...)
	badInvocation.Adapters[0].Invocations[0].Argv = append(badInvocation.Adapters[0].Invocations[0].Argv, "--ignored")
	badInvocationData, _ := json.Marshal(badInvocation)
	if _, err := LoadRealAdapterRegistry(badInvocationData, reviewed); err == nil {
		t.Fatal("adapter argv substitution was accepted")
	}

	badSource := parsed
	badSource.Adapters = append([]RealAdapterDefinition(nil), parsed.Adapters...)
	badSource.Adapters[0].Official = append([]OfficialSource(nil), parsed.Adapters[0].Official...)
	badSource.Adapters[0].Official[0].URL = "https://example.invalid/substituted"
	badSourceData, _ := json.Marshal(badSource)
	if _, err := LoadRealAdapterRegistry(badSourceData, reviewed); err == nil {
		t.Fatal("official-source substitution was accepted")
	}

	duplicate := bytes.Replace(data, []byte(`"schema_version": "1.0.0"`), []byte(`"schema_version": "1.0.0", "schema_version": "1.0.0"`), 1)
	if _, err := LoadRealAdapterRegistry(duplicate, reviewed); err == nil {
		t.Fatal("duplicate registry key was accepted")
	}
}

func testReadOnlyGitAdapters(t *testing.T) {
	repository := t.TempDir()
	runIsolatedGit(t, repository, "init", "--quiet")
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("tracked-state-v1\n"), 0o600); err != nil {
		t.Fatal("isolated Git fixture unavailable")
	}
	if err := os.Symlink("tracked.txt", filepath.Join(repository, "tracked-link")); err != nil {
		t.Fatal("isolated Git symlink fixture unavailable")
	}
	runIsolatedGit(t, repository, "add", "--", "tracked.txt", "tracked-link")

	before := captureTreeState(t, repository)
	manifest := loadProtectedManifest(t)
	resolver := &realResolver{repositoryRoot: repository}
	key := bytes.Repeat([]byte{0x31}, 32)
	tracked := surfaceByRef(t, manifest, "repo:sentinel/worktree/tracked")
	index := surfaceByRef(t, manifest, "repo:sentinel/worktree/index")
	gitCapability := newGitVersionCapability(repository)
	trackedSnapshot := snapshotGitWorktree(context.Background(), tracked, resolver, key, gitCapability)
	indexSnapshot := snapshotGitIndex(context.Background(), index, resolver, key, gitCapability)
	after := captureTreeState(t, repository)

	if !reflect.DeepEqual(before, after) {
		t.Fatal("read-only Git observation changed the isolated repository")
	}
	if gitCapability.probes != 1 {
		t.Fatal("Git capability was probed more than once for one adapter set")
	}
	if gitCapability.supported {
		if trackedSnapshot.Status != ObservationComplete || indexSnapshot.Status != ObservationComplete || !validOpaqueToken(trackedSnapshot.OpaqueState) || !validOpaqueToken(indexSnapshot.OpaqueState) {
			t.Fatal("supported isolated Git adapters did not produce bounded opaque observations")
		}
	} else if trackedSnapshot.Status != ObservationIncomplete || indexSnapshot.Status != ObservationIncomplete {
		t.Fatal("unsupported Git version did not fail closed")
	}
	for _, snapshot := range []SurfaceSnapshot{trackedSnapshot, indexSnapshot} {
		encoded, _ := json.Marshal(snapshot)
		if bytes.Contains(encoded, []byte(repository)) || bytes.Contains(encoded, []byte("tracked.txt")) || bytes.Contains(encoded, []byte("tracked-state-v1")) {
			t.Fatal("Git adapter exposed a physical path or raw tracked state")
		}
	}
}

func testReadOnlyGoAdapters(t *testing.T) {
	root := t.TempDir()
	exact := filepath.Join(root, "exact")
	tree := filepath.Join(root, "tree")
	if err := os.WriteFile(exact, []byte("exact-state-v1"), 0o600); err != nil {
		t.Fatal("isolated exact-file fixture unavailable")
	}
	if err := os.MkdirAll(filepath.Join(tree, "versions"), 0o700); err != nil {
		t.Fatal("isolated tree fixture unavailable")
	}
	if err := os.WriteFile(filepath.Join(tree, "versions", "state"), []byte("tree-state-v1"), 0o600); err != nil {
		t.Fatal("isolated tree fixture unavailable")
	}
	if err := os.Symlink("state", filepath.Join(tree, "versions", "current")); err != nil {
		t.Fatal("isolated tree symlink fixture unavailable")
	}
	manifest := loadProtectedManifest(t)
	key := bytes.Repeat([]byte{0x42}, 32)
	fileSurface := surfaceByRef(t, manifest, "home:.zshrc")
	treeSurface := surfaceByRef(t, manifest, "home:sentinel/manager/mise-data")
	if reason := validateTreeSymlinkTarget(filepath.Join(tree, "versions", "current"), "state", tree); reason != "" {
		t.Fatalf("internal tree symlink target validator rejected containment: %s", reason)
	}
	observedLink, observedReason := observeExactSymlink(filepath.Join(tree, "versions", "current"), root, treeSurface.Bounds.MaxBytes, time.Now().Add(time.Second))
	if observedReason != "" || validateTreeSymlinkTarget(filepath.Join(tree, "versions", "current"), observedLink.target, tree) != "" {
		t.Fatalf("internal tree symlink observation rejected containment: %s", observedReason)
	}
	before := captureTreeState(t, root)
	fileSnapshot := snapshotExactFile(fileSurface, exact, root, "", key)
	treeSnapshot := snapshotExactTree(treeSurface, tree, root, key)
	after := captureTreeState(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("bounded Go observation wrote to isolated paths")
	}
	if fileSnapshot.Status != ObservationComplete || treeSnapshot.Status != ObservationComplete {
		t.Fatalf("bounded Go observation failed closed unexpectedly: file=%s/%s tree=%s/%s", fileSnapshot.Status, fileSnapshot.Reason, treeSnapshot.Status, treeSnapshot.Reason)
	}

	overflowSurface := fileSurface
	overflowSurface.Bounds.MaxBytes = 1
	if snapshot := snapshotExactFile(overflowSurface, exact, root, "", key); snapshot.Status != ObservationIncomplete || snapshot.Reason != ReasonOverflow || snapshot.OpaqueState != "" {
		t.Fatal("bounded file overflow did not fail closed")
	}
	outside := t.TempDir()
	if snapshot := snapshotExactFile(fileSurface, filepath.Join(outside, "not-allowed"), root, "", key); snapshot.Status != ObservationIncomplete || snapshot.OpaqueState != "" {
		t.Fatal("exact-file resolver escape produced a token")
	}
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside-state"), 0o600); err != nil {
		t.Fatal("symlink escape setup failed")
	}
	for _, testCase := range []struct {
		name   string
		setup  func(string) error
		reason IncompleteReason
	}{
		{
			name: "internal relative escape",
			setup: func(treeRoot string) error {
				relative, err := filepath.Rel(treeRoot, filepath.Join(outside, "secret"))
				if err != nil {
					return err
				}
				return os.Symlink(relative, filepath.Join(treeRoot, "current"))
			},
			reason: ReasonSymlinkEscape,
		},
		{
			name: "internal absolute escape",
			setup: func(treeRoot string) error {
				return os.Symlink(filepath.Join(outside, "secret"), filepath.Join(treeRoot, "current"))
			},
			reason: ReasonSymlinkEscape,
		},
		{
			name: "internal chain escape",
			setup: func(treeRoot string) error {
				if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(treeRoot, "outside-hop")); err != nil {
					return err
				}
				return os.Symlink("outside-hop", filepath.Join(treeRoot, "current"))
			},
			reason: ReasonSymlinkEscape,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			escapingTree := filepath.Join(root, strings.ReplaceAll(testCase.name, " ", "-"))
			if err := os.Mkdir(escapingTree, 0o700); err != nil || testCase.setup(escapingTree) != nil {
				t.Fatal("internal tree symlink escape setup failed")
			}
			snapshot := snapshotExactTree(treeSurface, escapingTree, root, key)
			if snapshot.Status != ObservationIncomplete || snapshot.Reason != testCase.reason || snapshot.OpaqueState != "" {
				t.Fatal("internal tree symlink escape produced a complete token")
			}
		})
	}
	absoluteEscapeTree := filepath.Join(root, "external-change-tree")
	if err := os.Mkdir(absoluteEscapeTree, 0o700); err != nil || os.Symlink(filepath.Join(outside, "secret"), filepath.Join(absoluteEscapeTree, "current")) != nil {
		t.Fatal("external target change setup failed")
	}
	escapeBefore := snapshotExactTree(treeSurface, absoluteEscapeTree, root, key)
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside-state-tree-change"), 0o600); err != nil {
		t.Fatal("external target change setup failed")
	}
	escapeAfter := snapshotExactTree(treeSurface, absoluteEscapeTree, root, key)
	if escapeBefore.Status != ObservationIncomplete || escapeAfter.Status != ObservationIncomplete || escapeBefore.Reason != ReasonSymlinkEscape || escapeAfter.Reason != ReasonSymlinkEscape || escapeBefore.OpaqueState != "" || escapeAfter.OpaqueState != "" {
		t.Fatal("external target change completed a manager-tree observation")
	}
	beforeSurfaces := make([]SurfaceSnapshot, 0, len(manifest.Surfaces))
	afterSurfaces := make([]SurfaceSnapshot, 0, len(manifest.Surfaces))
	for _, surface := range manifest.Surfaces {
		if surface.LogicalRef == treeSurface.LogicalRef {
			beforeSurfaces = append(beforeSurfaces, escapeBefore)
			afterSurfaces = append(afterSurfaces, escapeAfter)
			continue
		}
		beforeSurfaces = append(beforeSurfaces, completeSurface(surface, key, []byte("stable")))
		afterSurfaces = append(afterSurfaces, completeSurface(surface, key, []byte("stable")))
	}
	opened := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	evidence, err := BuildEvidence(
		manifest,
		ProtectedSnapshot{ManifestDigest: manifest.Digest, WindowState: "closed", Surfaces: beforeSurfaces},
		ProtectedSnapshot{ManifestDigest: manifest.Digest, WindowState: "closed", Surfaces: afterSurfaces},
		EvidenceOptions{SuiteID: manifest.SuiteID, Tier: "offline-static", WindowID: "synthetic-window-tree-escape", OpenedAt: opened, ClosedAt: opened.Add(time.Second), Provenance: "synthetic"},
	)
	if err != nil {
		t.Fatal("tree escape evidence setup failed")
	}
	evaluation := Evaluate(manifest, evidence)
	if evaluation.Verdict != VerdictIndeterminate || evaluation.ExitCode == 0 {
		t.Fatal("tree escape evidence was not indeterminate")
	}
	if claim, err := RequestClaim(&evidence, evaluation, ScopedUnchangedClaim); err == nil || claim != "" {
		t.Fatal("tree escape evidence produced a claim")
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal("symlink escape setup failed")
	}
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root, "named-link")); err != nil {
		t.Fatal("named symlink identity setup failed")
	}
	if snapshot := snapshotExactFile(fileSurface, filepath.Join(root, "named-link"), root, "", key); snapshot.Status != ObservationIncomplete || snapshot.OpaqueState != "" {
		t.Fatal("unregistered external named symlink target produced a token")
	}
	linkBefore := snapshotExactFile(fileSurface, filepath.Join(root, "named-link"), root, outside, key)
	if linkBefore.Status != ObservationComplete || !validOpaqueToken(linkBefore.OpaqueState) {
		t.Fatal("registered external named symlink target was not fingerprinted")
	}
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside-state-v2"), 0o600); err != nil {
		t.Fatal("named symlink drift setup failed")
	}
	linkAfter := snapshotExactFile(fileSurface, filepath.Join(root, "named-link"), root, outside, key)
	if linkAfter.Status != ObservationComplete || linkAfter.OpaqueState == linkBefore.OpaqueState {
		t.Fatal("registered external named symlink target content drift was not detected")
	}
	if snapshot := snapshotExactFile(fileSurface, filepath.Join(root, "escape", "secret"), root, "", key); snapshot.Status != ObservationIncomplete || snapshot.OpaqueState != "" {
		t.Fatal("intermediate symlink escape produced a token")
	}
	if snapshot := snapshotExactTree(treeSurface, filepath.Join(root, "escape"), root, key); snapshot.Status != ObservationIncomplete || snapshot.OpaqueState != "" {
		t.Fatal("tree symlink escape produced a token")
	}
	if err := os.WriteFile(filepath.Join(root, ".zshrc"), []byte("frozen-resolver-state"), 0o600); err != nil {
		t.Fatal("frozen resolver setup failed")
	}
	registry := testOnlyReadyRegistry(t)
	callerResolver := &realResolver{repositoryRoot: root, homeRoot: root, managerRoot: tree, systemShells: exact}
	defaultAdapters := defaultRealAdapters(registry, callerResolver)
	callerResolver.repositoryRoot = outside
	callerResolver.homeRoot = outside
	callerResolver.managerRoot = outside
	callerResolver.systemShells = filepath.Join(outside, "secret")
	if snapshot := defaultAdapters["go-lstat-file-v1"].snapshot(context.Background(), fileSurface, nil, key); snapshot.Status != ObservationComplete || !validOpaqueToken(snapshot.OpaqueState) {
		t.Fatal("caller resolver mutation redirected an authorized default adapter")
	}
	fifo := filepath.Join(root, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal("FIFO negative setup failed")
	}
	if snapshot := snapshotExactFile(fileSurface, fifo, root, "", key); snapshot.Status != ObservationIncomplete || snapshot.OpaqueState != "" {
		t.Fatal("FIFO produced a token or blocked the exact-file adapter")
	}
	if _, reason := readExactRegularFile(exact, root, fileSurface.Bounds.MaxBytes, time.Now().Add(-time.Second)); reason != ReasonWindow {
		t.Fatal("expired exact-file window was not typed as window-exceeded")
	}
	if _, err := newRealResolver(root, root, outside); err == nil {
		t.Fatal("manager mapping outside the named home was accepted")
	}
	if _, err := newRealResolver(root, root, filepath.Join(root, "escape")); err == nil {
		t.Fatal("manager mapping through an escaping symlink was accepted")
	}
}

func testRealEnvelopePass(t *testing.T) {
	manifest := loadProtectedManifest(t)
	callerManifest := manifest
	callerManifest.Surfaces = append([]ProtectedSurface(nil), manifest.Surfaces...)
	registry := testOnlyReadyRegistry(t)
	surfaceRoot := t.TempDir()
	resolver, err := PrepareProtectedSynthetic(surfaceRoot)
	if err != nil {
		t.Fatal("isolated real-envelope analogs unavailable")
	}
	workloadFixture, _ := createEnvelopeFixture(t, surfaceRoot, false)
	workloadRoot := workloadFixture.Paths().Root
	var keyBuffer []byte
	keyMaterial := sha256.Sum256([]byte("real-envelope-window-01"))
	var claimMaterial ClaimMaterial
	result := runRealEnvelope(t, RealEnvelopeOptions{
		Manifest:  callerManifest,
		Registry:  registry,
		Adapters:  isolatedRealAdapters(manifest, registry, resolver, ""),
		Retention: workloadFixture.Retention(),
		Workload: func(_ context.Context) (string, error) {
			if _, err := os.Lstat(workloadRoot); err != nil {
				return "", errors.New("isolated workload fixture unavailable")
			}
			// 调用方在窗口期间修改自己的 slice，不能改变哨兵启动时冻结的 scope。
			callerManifest.Surfaces[0].Policy = PolicyOptional
			return innerSyntheticSuccess, nil
		},
		Clock:         envelopeClock(),
		WindowID:      "real-envelope-window-01",
		secretFactory: deterministicSecretFactory("real-envelope-window-01", &keyBuffer),
		ClaimConsumer: func(evidence *Evidence, evaluation Evaluation, sequence []string) (string, error) {
			if !reflect.DeepEqual(sequence, []string{"real-before", "isolated-workload", "freeze-primary", "fixture-finalize", "real-after", "monotonic-combine"}) {
				return "", errors.New("claim consumer sequence rejected")
			}
			material, err := ConsumeClaim(evidence, evaluation, ScopedUnchangedClaim)
			if err == nil {
				claimMaterial = material
			}
			return material.Claim, err
		},
	})
	wantSequence := []string{"real-before", "isolated-workload", "freeze-primary", "fixture-finalize", "real-after", "monotonic-combine"}
	if !reflect.DeepEqual(result.Sequence, wantSequence) || result.Evaluation.Verdict != VerdictPassed || result.Evaluation.ExitCode != ExitPassed || result.Evaluation.Claim != ScopedUnchangedClaim || result.Status != ScopedUnchangedClaim || result.TeardownStatus != fixture.TeardownRemoved || result.Evidence == nil || len(result.Evidence.Surfaces) != 6 {
		t.Fatal("complete isolated outer envelope did not produce the exact scoped claim")
	}
	if callerManifest.Surfaces[0].Policy != PolicyOptional {
		t.Fatal("manifest-freeze mutation setup did not run")
	}
	if _, err := os.Lstat(workloadRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("successful marker-owned fixture was not removed")
	}
	if _, err := os.Lstat(surfaceRoot); err != nil {
		t.Fatal("fixture teardown reached the protected analog root")
	}
	if claimMaterial.Claim != ScopedUnchangedClaim || claimMaterial.EvidenceDigest != result.Evaluation.EvidenceDigest || claimMaterial.ManifestDigest != result.Evidence.ManifestDigest || claimMaterial.SuiteDigest != result.Evidence.SuiteDigest || claimMaterial.Window != result.Evidence.Window || claimMaterial.WindowDigest != result.Evidence.WindowDigest || len(claimMaterial.Surfaces) != len(result.Evidence.Surfaces) {
		t.Fatal("controlled claim consumer did not bind the actual evidence window")
	}
	assertZeroedSecret(t, keyBuffer)
	for _, surface := range result.Evidence.Surfaces {
		if surface.BeforeStatus != ObservationComplete || surface.AfterStatus != ObservationComplete || surface.BeforeToken == "" || surface.BeforeToken != surface.AfterToken {
			t.Fatal("same-run before/after observations did not share one ephemeral key")
		}
	}
	if claim, err := RequestClaim(result.Evidence, result.Evaluation, ScopedUnchangedClaim); err == nil || claim != "" {
		t.Fatal("returned evidence retained a replayable claim capability")
	}
	for _, overclaim := range []string{"whole-Mac-unchanged", "recovery-ready-on-current-host", "multi-host-verified", "fresh-install-verified"} {
		if claim, err := RequestClaim(result.Evidence, result.Evaluation, overclaim); err == nil || claim != "" {
			t.Fatalf("real envelope authorized overclaim %s", overclaim)
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal("real envelope render setup failed")
	}
	for _, forbidden := range []string{surfaceRoot, workloadRoot, "/Users/", "effective_uid", "ownership_nonce", "resolver_mapping", "service_output", "raw_output", "hmac_key", hex.EncodeToString(keyMaterial[:])} {
		if forbidden != "" && bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("real envelope exposed process-only data: %s", forbidden)
		}
	}
	parsed, err := ParseEvidence(mustMarshal(t, *result.Evidence))
	if err != nil {
		t.Fatal("public real evidence did not parse")
	}
	if evaluation := Evaluate(manifest, parsed); evaluation.Verdict != VerdictIndeterminate || evaluation.Claim != "" || evaluation.Reason != "real-envelope-binding-missing" {
		t.Fatal("persisted real evidence retained process-only claim capability")
	}

	secondSurfaceRoot := t.TempDir()
	secondResolver, err := PrepareProtectedSynthetic(secondSurfaceRoot)
	if err != nil {
		t.Fatal("second isolated real-envelope analog unavailable")
	}
	secondFixture, _ := createEnvelopeFixture(t, secondSurfaceRoot, false)
	second := runRealEnvelope(t, RealEnvelopeOptions{
		Manifest: manifest, Registry: registry, Adapters: isolatedRealAdapters(manifest, registry, secondResolver, ""), Retention: secondFixture.Retention(),
		Workload: func(_ context.Context) (string, error) { return innerSyntheticSuccess, nil }, Clock: envelopeClock(), WindowID: "real-envelope-window-02",
	})
	if second.Evaluation.Verdict != VerdictPassed || second.Evidence == nil || len(second.Evidence.Surfaces) != len(result.Evidence.Surfaces) {
		t.Fatal("second real envelope did not complete")
	}
	for index := range result.Evidence.Surfaces {
		if result.Evidence.Surfaces[index].LogicalRef != second.Evidence.Surfaces[index].LogicalRef || result.Evidence.Surfaces[index].BeforeToken == second.Evidence.Surfaces[index].BeforeToken {
			t.Fatal("same surface reused a stable token across real runs")
		}
	}
}

func testRealEnvelopeMonotonicity(t *testing.T) {
	manifest := loadProtectedManifest(t)
	registry := testOnlyReadyRegistry(t)
	wantSequence := []string{"real-before", "isolated-workload", "freeze-primary", "fixture-finalize", "real-after", "monotonic-combine"}

	t.Run("workload failure still finalizes and observes after", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, _ := createEnvelopeFixture(t, surfaceRoot, false)
		physicalRoot := root.Paths().Root
		var failedKey []byte
		result := runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: registry, Adapters: isolatedRealAdapters(manifest, registry, resolver, ""), Retention: root.Retention(), Workload: func(_ context.Context) (string, error) {
			return "", errors.New("isolated workload failed")
		}, Clock: envelopeClock(), WindowID: "real-envelope-workload-failure", secretFactory: deterministicSecretFactory("real-envelope-workload-failure", &failedKey)})
		if !reflect.DeepEqual(result.Sequence, wantSequence) || result.Evaluation.Verdict != VerdictHarnessError || result.Evaluation.ExitCode == 0 || result.Evaluation.Claim != "" || result.TeardownStatus != fixture.TeardownRemoved {
			t.Fatal("workload failure was masked or skipped teardown/after")
		}
		if _, err := os.Lstat(physicalRoot); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("failed workload fixture was not safely finalized")
		}
		assertZeroedSecret(t, failedKey)
	})

	t.Run("teardown failure cannot improve the primary verdict", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, _ := createEnvelopeFixture(t, surfaceRoot, false)
		if err := os.WriteFile(filepath.Join(root.Paths().Root, ".yamc-fixture-marker.json"), []byte("{\n"), 0o600); err != nil {
			t.Fatal("teardown failure setup failed")
		}
		result := runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: registry, Adapters: isolatedRealAdapters(manifest, registry, resolver, ""), Retention: root.Retention(), Workload: func(_ context.Context) (string, error) {
			return innerSyntheticSuccess, nil
		}, Clock: envelopeClock(), WindowID: "real-envelope-teardown-failure"})
		if !reflect.DeepEqual(result.Sequence, wantSequence) || result.Evaluation.Verdict != VerdictHarnessError || result.Evaluation.ExitCode == 0 || result.Evaluation.Claim != "" || result.TeardownStatus != fixture.TeardownFailed {
			t.Fatal("teardown failure improved or short-circuited the frozen primary verdict")
		}
	})

	t.Run("pre-run keep retains then expires only the owned child", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, advance := createEnvelopeFixture(t, surfaceRoot, true)
		physicalRoot := root.Paths().Root
		result := runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: registry, Adapters: isolatedRealAdapters(manifest, registry, resolver, ""), Retention: root.Retention(), Workload: func(_ context.Context) (string, error) {
			return innerSyntheticSuccess, nil
		}, Clock: envelopeClock(), WindowID: "real-envelope-retained"})
		if result.Evaluation.Verdict != VerdictPassed || result.Evaluation.Claim != ScopedUnchangedClaim || result.TeardownStatus != fixture.TeardownRetained {
			t.Fatal("pre-run keep did not retain a passing owned fixture")
		}
		if _, err := os.Lstat(physicalRoot); err != nil {
			t.Fatal("retained fixture disappeared before expiry")
		}
		advance(25 * time.Hour)
		frozen, err := fixture.FreezePrimary(fixture.VerdictPassed)
		if err != nil {
			t.Fatal("retained cleanup verdict did not freeze")
		}
		final := root.Retention().TeardownExpiredOwnedFixture(frozen)
		if final.Verdict != fixture.VerdictPassed || final.Teardown.Status != fixture.TeardownRemoved {
			t.Fatal("expired retained fixture was not safely removed")
		}
		if _, err := os.Lstat(physicalRoot); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("expired retained child remains")
		}
	})

	t.Run("after observation failure remains indeterminate", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, _ := createEnvelopeFixture(t, surfaceRoot, false)
		result := runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: registry, Adapters: isolatedRealAdapters(manifest, registry, resolver, "home:.zshrc"), Retention: root.Retention(), Workload: func(_ context.Context) (string, error) {
			return innerSyntheticSuccess, nil
		}, Clock: envelopeClock(), WindowID: "real-envelope-after-incomplete"})
		if !reflect.DeepEqual(result.Sequence, wantSequence) || result.Evaluation.Verdict != VerdictIndeterminate || result.Evaluation.ExitCode == 0 || result.Evaluation.Claim != "" || result.TeardownStatus != fixture.TeardownRemoved {
			t.Fatal("after observation failure was masked")
		}
	})

	t.Run("freezes authorized adapter implementations across the workload", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, _ := createEnvelopeFixture(t, surfaceRoot, false)
		adapters := isolatedRealAdapters(manifest, registry, resolver, "")
		authorizedCalls := 0
		substitutedCalls := 0
		for adapterID, adapter := range adapters {
			original := adapter.snapshot
			adapter.snapshot = func(ctx context.Context, surface ProtectedSurface, resolver *realResolver, key []byte) SurfaceSnapshot {
				authorizedCalls++
				return original(ctx, surface, resolver, key)
			}
			adapters[adapterID] = adapter
		}
		result := runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: registry, Adapters: adapters, Retention: root.Retention(), Workload: func(_ context.Context) (string, error) {
			for adapterID, adapter := range adapters {
				adapter.snapshot = func(_ context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
					substitutedCalls++
					return completeSurface(surface, key, []byte("post-gate-substitution"))
				}
				adapters[adapterID] = adapter
			}
			return innerSyntheticSuccess, nil
		}, Clock: envelopeClock(), WindowID: "real-envelope-frozen-adapters"})
		if result.Evaluation.Verdict != VerdictPassed || result.Evaluation.Claim != ScopedUnchangedClaim || result.TeardownStatus != fixture.TeardownRemoved {
			t.Fatal("post-gate adapter substitution changed the authorized envelope")
		}
		if authorizedCalls != 2*len(manifest.Surfaces) || substitutedCalls != 0 {
			t.Fatal("after observation did not use the frozen authorized adapter set")
		}
	})

	t.Run("required drift is not hidden by an earlier workload failure", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, _ := createEnvelopeFixture(t, surfaceRoot, false)
		adapters := isolatedRealAdapters(manifest, registry, resolver, "")
		logicalRef := "home:.zshrc"
		adapterID := surfaceByRef(t, manifest, logicalRef).AdapterID
		adapter := adapters[adapterID]
		calls := 0
		adapter.snapshot = func(_ context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
			calls++
			state := []byte("before-state")
			if calls > 1 {
				state = []byte("after-state")
			}
			return completeSurface(surface, key, state)
		}
		adapters[adapterID] = adapter
		result := runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: registry, Adapters: adapters, Retention: root.Retention(), Workload: func(_ context.Context) (string, error) {
			return "", errors.New("isolated workload failed before drift")
		}, Clock: envelopeClock(), WindowID: "real-envelope-failure-and-drift"})
		if result.Evaluation.Verdict != VerdictViolation || result.Evaluation.ExitCode != ExitViolation || result.Evaluation.ChangeCode != ChangeDetectedCode || result.Evaluation.Claim != "" || result.TeardownStatus != fixture.TeardownRemoved {
			t.Fatal("required drift was hidden by an earlier non-pass")
		}
	})

	t.Run("shares one outer deadline across every stage", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, _ := createEnvelopeFixture(t, surfaceRoot, false)
		physicalRoot := root.Paths().Root
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		adapters := isolatedRealAdapters(manifest, registry, resolver, "")
		adapterCalls := 0
		contextMismatch := false
		for adapterID, adapter := range adapters {
			original := adapter.snapshot
			adapter.snapshot = func(got context.Context, surface ProtectedSurface, resolver *realResolver, key []byte) SurfaceSnapshot {
				if got != ctx {
					contextMismatch = true
				}
				adapterCalls++
				return original(got, surface, resolver, key)
			}
			adapters[adapterID] = adapter
		}
		workloadCalls := 0
		result := RunRealEnvelope(RealEnvelopeOptions{
			Manifest: manifest, Registry: registry, Adapters: adapters, Retention: root.Retention(), Context: ctx,
			Workload: func(got context.Context) (string, error) {
				workloadCalls++
				if got != ctx {
					contextMismatch = true
				}
				cancel()
				return innerSyntheticSuccess, nil
			},
			Clock: envelopeClock(), WindowID: "real-envelope-shared-deadline", secretFactory: deterministicSecretFactory("real-envelope-shared-deadline", nil),
		})
		if !reflect.DeepEqual(result.Sequence, wantSequence) || result.Evaluation.Verdict != VerdictIndeterminate || result.Evaluation.ExitCode != ExitIndeterminate || result.Evaluation.Claim != "" || result.TeardownStatus != fixture.TeardownRemoved || result.Evidence == nil {
			t.Fatal("shared deadline did not remain monotonic through finalization and after observation")
		}
		if contextMismatch || workloadCalls != 1 || adapterCalls != len(manifest.Surfaces) {
			t.Fatal("outer deadline was replaced or later adapters ran after cancellation")
		}
		for _, surface := range result.Evidence.Surfaces {
			if surface.AfterStatus != ObservationIncomplete || surface.AfterReason != ReasonWindow || surface.AfterToken != "" {
				t.Fatal("after observation ignored the expired shared deadline")
			}
		}
		if _, err := os.Lstat(physicalRoot); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("shared deadline skipped marker-owned fixture finalization")
		}
	})

	t.Run("claim consumer rejection zeroes the run key", func(t *testing.T) {
		surfaceRoot := t.TempDir()
		resolver, err := PrepareProtectedSynthetic(surfaceRoot)
		if err != nil {
			t.Fatal("isolated analog setup failed")
		}
		root, _ := createEnvelopeFixture(t, surfaceRoot, false)
		var rejectedKey []byte
		result := runRealEnvelope(t, RealEnvelopeOptions{
			Manifest: manifest, Registry: registry, Adapters: isolatedRealAdapters(manifest, registry, resolver, ""), Retention: root.Retention(),
			Workload: func(_ context.Context) (string, error) { return innerSyntheticSuccess, nil }, Clock: envelopeClock(), WindowID: "real-envelope-consumer-rejected",
			secretFactory: deterministicSecretFactory("real-envelope-consumer-rejected", &rejectedKey),
			ClaimConsumer: func(*Evidence, Evaluation, []string) (string, error) {
				return "", errors.New("claim consumer rejected")
			},
		})
		if result.Evaluation.Verdict != VerdictHarnessError || result.Evaluation.Claim != "" || result.TeardownStatus != fixture.TeardownRemoved {
			t.Fatal("claim consumer rejection did not fail closed")
		}
		assertZeroedSecret(t, rejectedKey)
	})
}

func testRealEnvelopeRejectsSyntheticAdapters(t *testing.T) {
	manifest := loadProtectedManifest(t)
	registry := testOnlyReadyRegistry(t)
	root := t.TempDir()
	resolver, err := PrepareProtectedSynthetic(root)
	if err != nil {
		t.Fatal("isolated analog setup failed")
	}
	adapters := isolatedRealAdapters(manifest, registry, resolver, "")
	workloadCalled := false
	tampered := manifest
	tampered.Surfaces = append([]ProtectedSurface(nil), manifest.Surfaces...)
	tampered.Surfaces[0].Bounds.MaxFiles--
	result := runRealEnvelope(t, RealEnvelopeOptions{Manifest: tampered, Registry: registry, Adapters: adapters, Workload: func(_ context.Context) (string, error) {
		workloadCalled = true
		return innerSyntheticSuccess, nil
	}, Clock: envelopeClock(), WindowID: "real-envelope-stale-manifest"})
	if result.Evaluation.Verdict != VerdictHarnessError || result.Evaluation.ExitCode == 0 || !reflect.DeepEqual(result.Sequence, []string{"proof-gate"}) || workloadCalled {
		t.Fatal("stale manifest digest reached an adapter or workload")
	}

	first := manifest.Surfaces[0].AdapterID
	adapter := adapters[first]
	adapter.capability = &realAdapterCapability{registry: registry, adapterID: first, implementationDigest: registry.definitions[first].ImplementationDigest}
	adapters[first] = adapter
	workloadCalled = false
	result = runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: registry, Adapters: adapters, Workload: func(_ context.Context) (string, error) {
		workloadCalled = true
		return innerSyntheticSuccess, nil
	}, Clock: envelopeClock(), WindowID: "real-envelope-synthetic-rejected"})
	if result.Evaluation.Verdict != VerdictHarnessError || result.Evaluation.ExitCode == 0 || result.Evaluation.Claim != "" || !reflect.DeepEqual(result.Sequence, []string{"proof-gate"}) || workloadCalled || result.Evidence != nil {
		t.Fatal("synthetic adapter reached the real workload or claim path")
	}

	tracked, err := LoadRealAdapterRegistry(readRealAdapterManifest(t), time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal("tracked registry rejected")
	}
	adapterCalls := 0
	for adapterID, candidate := range adapters {
		candidate.snapshot = func(_ context.Context, surface ProtectedSurface, _ *realResolver, _ []byte) SurfaceSnapshot {
			adapterCalls++
			return incompleteSurface(surface, ReasonUnreadable)
		}
		adapters[adapterID] = candidate
	}
	result = runRealEnvelope(t, RealEnvelopeOptions{Manifest: manifest, Registry: tracked, Adapters: adapters, Workload: func(_ context.Context) (string, error) {
		workloadCalled = true
		return innerSyntheticSuccess, nil
	}, Clock: envelopeClock(), WindowID: "real-envelope-proof-missing"})
	if result.Status != "manual-required" || result.Evaluation.Verdict != VerdictIndeterminate || result.Evaluation.ExitCode != 32 || !reflect.DeepEqual(result.Sequence, []string{"proof-gate"}) || workloadCalled || adapterCalls != 0 {
		t.Fatal("missing proof did not stop before adapter or workload execution")
	}
}

func readRealAdapterManifest(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(safetyRoot(t), "manifests", "real-adapters.v1.json"))
	if err != nil {
		t.Fatal("tracked real adapter manifest unavailable")
	}
	return data
}

func testOnlyReadyRegistry(t *testing.T) *RealAdapterRegistry {
	t.Helper()
	registry, err := LoadRealAdapterRegistry(readRealAdapterManifest(t), time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal("tracked registry rejected")
	}
	for adapterID := range registry.usable {
		registry.usable[adapterID] = true
		definition := registry.definitions[adapterID]
		registry.capabilities[adapterID] = &realAdapterCapability{registry: registry, adapterID: adapterID, implementationDigest: definition.ImplementationDigest}
	}
	if assessment := registry.Assess(loadProtectedManifest(t)); !assessment.ClaimEligible {
		t.Fatal("test-only isolated registry did not open the controlled envelope")
	}
	return registry
}

func isolatedRealAdapters(manifest ProtectedManifest, registry *RealAdapterRegistry, resolver *SyntheticResolver, failAfterRef string) map[string]realAdapter {
	callCount := make(map[string]int, len(manifest.Surfaces))
	adapters := make(map[string]realAdapter, len(manifest.Surfaces))
	for _, declared := range manifest.Surfaces {
		adapterID := declared.AdapterID
		adapters[adapterID] = realAdapter{id: adapterID, capability: registry.capabilities[adapterID], snapshot: func(_ context.Context, surface ProtectedSurface, _ *realResolver, key []byte) SurfaceSnapshot {
			callCount[surface.LogicalRef]++
			if surface.LogicalRef == failAfterRef && callCount[surface.LogicalRef] > 1 {
				return incompleteSurface(surface, ReasonUnreadable)
			}
			path, err := resolver.resolve(surface.LogicalRef)
			if err != nil {
				return incompleteSurface(surface, ReasonSymlinkEscape)
			}
			start := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
			canonical, reason := fingerprintSurface(resolver.root, path, surface.Bounds, start, func() time.Time { return start }, nil)
			if reason != "" {
				return incompleteSurface(surface, reason)
			}
			return completeSurface(surface, key, canonical)
		}}
	}
	return adapters
}

func createEnvelopeFixture(t *testing.T, protectedRoot string, keep bool) (*fixture.Root, func(time.Duration)) {
	t.Helper()
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	repository := t.TempDir()
	base := t.TempDir()
	root, err := fixture.Create(fixture.CreateOptions{
		Base:           base,
		RepositoryRoot: repository,
		ProtectedRoots: []string{protectedRoot},
		LogicalID:      "fixture:sentinel/real-envelope",
		KeepFixture:    keep,
		Clock:          func() time.Time { return now },
	})
	if err != nil {
		t.Fatal("isolated workload fixture unavailable")
	}
	return root, func(duration time.Duration) { now = now.Add(duration) }
}

func runRealEnvelope(t *testing.T, options RealEnvelopeOptions) RealEnvelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	options.Context = ctx
	if options.secretFactory == nil {
		options.secretFactory = deterministicSecretFactory(options.WindowID, nil)
	}
	return RunRealEnvelope(options)
}

func deterministicSecretFactory(seed string, captured *[]byte) func([]byte) error {
	return func(destination []byte) error {
		digest := sha256.Sum256([]byte(seed))
		copy(destination, digest[:])
		if captured != nil {
			*captured = destination
		}
		return nil
	}
}

func assertZeroedSecret(t *testing.T, value []byte) {
	t.Helper()
	if len(value) != 32 {
		t.Fatal("test secret factory did not receive the internal run buffer")
	}
	for _, item := range value {
		if item != 0 {
			t.Fatal("real envelope retained key material after return")
		}
	}
}

func envelopeClock() func() time.Time {
	now := time.Date(2026, 7, 11, 1, 0, 0, 0, time.UTC)
	return func() time.Time {
		current := now
		now = now.Add(time.Second)
		return current
	}
}

func surfaceByRef(t *testing.T, manifest ProtectedManifest, logicalRef string) ProtectedSurface {
	t.Helper()
	for _, surface := range manifest.Surfaces {
		if surface.LogicalRef == logicalRef {
			return surface
		}
	}
	t.Fatalf("protected surface unavailable: %s", logicalRef)
	return ProtectedSurface{}
}

type treeStateEntry struct {
	Path       string
	Mode       uint32
	Size       int64
	ModTime    int64
	ChangeTime int64
	Target     string
	Digest     string
}

func captureTreeState(t *testing.T, root string) []treeStateEntry {
	t.Helper()
	entries := make([]treeStateEntry, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		state := treeStateEntry{Path: filepath.ToSlash(relative), Mode: uint32(info.Mode()), Size: info.Size(), ModTime: info.ModTime().UnixNano()}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			state.ChangeTime = stat.Ctimespec.Sec*int64(time.Second) + int64(stat.Ctimespec.Nsec)
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			state.Target, err = os.Readlink(path)
		case info.Mode().IsRegular():
			var data []byte
			data, err = os.ReadFile(path)
			if err == nil {
				digest := sha256.Sum256(data)
				state.Digest = hex.EncodeToString(digest[:])
			}
		}
		if err != nil {
			return err
		}
		entries = append(entries, state)
		return nil
	})
	if err != nil {
		t.Fatal("isolated tree state unavailable")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries
}

func runIsolatedGit(t *testing.T, repository string, arguments ...string) {
	t.Helper()
	command := exec.Command("/usr/bin/git", arguments...)
	command.Dir = repository
	command.Env = []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"LC_ALL=C",
		"LANG=C",
		"PATH=/usr/bin:/bin",
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("isolated Git setup failed: %s", strings.TrimSpace(string(output)))
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal("JSON setup failed")
	}
	return data
}
