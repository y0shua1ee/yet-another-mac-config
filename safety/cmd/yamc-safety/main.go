package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"example.invalid/yamc/safety/internal/artifact"
	"example.invalid/yamc/safety/internal/fixture"
	"example.invalid/yamc/safety/internal/privacy"
	"example.invalid/yamc/safety/internal/sentinel"
	"example.invalid/yamc/safety/internal/workflow"
)

const maxArtifactBytes = 1 << 20

type fixtureRunFlags struct {
	blueprintPath  string
	surfacesPath   string
	fixtureRoot    string
	fixtureBase    string
	fixtureID      string
	fixtureTTL     time.Duration
	keepFixture    bool
	storeRoot      string
	repositoryRoot string
	mode           string
	managed        bool
}

type validateFlags struct {
	expectedKind string
	artifactPath string
}

type storeFlags struct {
	mode              string
	storeRoot         string
	repositoryRoot    string
	desiredPath       string
	observedPath      string
	freshObservedPath string
	planPath          string
	receiptPath       string
	evidencePath      string
	reportPath        string
}

type testPolicyFlags struct {
	tier                string
	networkManifestPath string
	repositoryRoot      string
	networkTestID       string
	liveProbeID         string
}

type sentinelVerifyFlags struct {
	mode         string
	manifestPath string
	fixtureRoot  string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		writeSafeError(stderr, "UNSUPPORTED_COMMAND")
		return 64
	}
	switch arguments[0] {
	case "fixture":
		if len(arguments) < 2 || arguments[1] != "run" {
			writeSafeError(stderr, "UNSUPPORTED_COMMAND")
			return 64
		}
		return runFixture(arguments[2:], stdout, stderr)
	case "validate":
		return runValidate(arguments[1:], stdout, stderr)
	case "store":
		return runStore(arguments[1:], stdout, stderr)
	case "test-policy":
		return runTestPolicy(arguments[1:], stdout, stderr)
	case "sentinel":
		if len(arguments) < 2 || arguments[1] != "verify" {
			writeSafeError(stderr, "UNSUPPORTED_COMMAND")
			return 64
		}
		return runSentinelVerify(arguments[2:], stdout, stderr)
	default:
		writeSafeError(stderr, "UNSUPPORTED_COMMAND")
		return 64
	}
}

func runSentinelVerify(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseSentinelVerifyFlags(arguments)
	if err != nil {
		writeSafeError(stderr, "SENTINEL_ARGUMENTS_REJECTED")
		return 64
	}
	data, err := readBoundedArtifact(parsed.manifestPath)
	if err != nil {
		writeSafeError(stderr, "SENTINEL_MANIFEST_REJECTED")
		return 2
	}
	manifest, err := sentinel.ParseProtectedManifest(data)
	if err != nil {
		writeSafeError(stderr, "SENTINEL_MANIFEST_REJECTED")
		return 2
	}
	frozen, err := sentinel.FreezeProtectedManifest(manifest)
	if err != nil {
		writeSafeError(stderr, "SENTINEL_MANIFEST_REJECTED")
		return 2
	}
	resolver, err := sentinel.NewSyntheticResolver(parsed.fixtureRoot)
	if err != nil {
		writeSafeError(stderr, "SENTINEL_RESOLVER_REJECTED")
		return 2
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		writeSafeError(stderr, "SENTINEL_KEY_REJECTED")
		return 70
	}
	snapshot, err := sentinel.SnapshotProtected(frozen, manifest, resolver, key, sentinel.SnapshotOptions{})
	for index := range key {
		key[index] = 0
	}
	if err != nil {
		writeSafeError(stderr, "SENTINEL_SNAPSHOT_REJECTED")
		return 70
	}
	complete := true
	for _, surface := range snapshot.Surfaces {
		if surface.Status != sentinel.ObservationComplete {
			complete = false
		}
	}
	status := "synthetic-sentinel-passed"
	exitCode := 0
	if !complete {
		status = "indeterminate"
		exitCode = 21
	}
	result := struct {
		Status   string                     `json:"status"`
		Snapshot sentinel.ProtectedSnapshot `json:"snapshot"`
	}{Status: status, Snapshot: snapshot}
	if err := renderSafe(stdout, result); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return exitCode
}

func runFixture(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseFixtureRunFlags(arguments)
	if err != nil {
		writeSafeError(stderr, "FIXTURE_ARGUMENTS_REJECTED")
		return 64
	}
	if parsed.managed {
		return runManagedFixture(parsed, stdout, stderr)
	}
	summary, err := workflow.RunSynthetic(workflow.Options{
		BlueprintPath:  parsed.blueprintPath,
		SurfacesPath:   parsed.surfacesPath,
		FixtureRoot:    parsed.fixtureRoot,
		StoreRoot:      parsed.storeRoot,
		RepositoryRoot: parsed.repositoryRoot,
		Mode:           parsed.mode,
	})
	if err != nil {
		writeRejected(stderr, err, "FIXTURE_RUN_REJECTED")
		return 2
	}
	if err := renderSafe(stdout, summary); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return 0
}

func runTestPolicy(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseTestPolicyFlags(arguments)
	if err != nil {
		writeSafeError(stderr, "TEST_POLICY_ARGUMENTS_REJECTED")
		return 64
	}
	tier, err := fixture.ParseTier(parsed.tier)
	if err != nil {
		writeSafeError(stderr, "TEST_POLICY_TIER_REJECTED")
		return 64
	}
	switch tier {
	case fixture.TierOfflineStatic:
		return renderPolicyDecision(stdout, stderr, fixture.DefaultPolicyDecision(tier), 0)
	case fixture.TierIsolatedIntegration:
		if parsed.networkTestID == "" {
			return renderPolicyDecision(stdout, stderr, fixture.DefaultPolicyDecision(tier), 0)
		}
		policy, loadErr := fixture.LoadNetworkPolicy(parsed.networkManifestPath, parsed.repositoryRoot)
		if loadErr != nil {
			decision := fixture.PolicyDecision{
				Status:        fixture.PolicyManualRequired,
				Tier:          tier,
				NetworkPolicy: "denied",
				Reason:        "network-manifest-rejected",
			}
			return renderPolicyDecision(stdout, stderr, decision, 32)
		}
		decision := policy.Authorize(parsed.networkTestID, tier, presentPolicyEnvironmentKeys())
		return renderPolicyDecision(stdout, stderr, decision, 32)
	case fixture.TierLiveCheck:
		decision := (fixture.LivePolicy{}).Evaluate(parsed.liveProbeID)
		return renderPolicyDecision(stdout, stderr, decision, 32)
	default:
		writeSafeError(stderr, "TEST_POLICY_TIER_REJECTED")
		return 64
	}
}

func renderPolicyDecision(stdout, stderr io.Writer, decision fixture.PolicyDecision, exitCode int) int {
	if err := renderSafe(stdout, decision); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return exitCode
}

func runManagedFixture(parsed fixtureRunFlags, stdout, stderr io.Writer) int {
	root, err := fixture.Create(fixture.CreateOptions{
		Base:           parsed.fixtureBase,
		RepositoryRoot: parsed.repositoryRoot,
		LogicalID:      parsed.fixtureID,
		KeepFixture:    parsed.keepFixture,
		TTL:            parsed.fixtureTTL,
	})
	if err != nil {
		writeSafeError(stderr, "FIXTURE_ROOT_REJECTED")
		return 2
	}
	paths := root.Paths()
	summary, runErr := workflow.RunSynthetic(workflow.Options{
		BlueprintPath:  parsed.blueprintPath,
		SurfacesPath:   parsed.surfacesPath,
		FixtureRoot:    paths.Root,
		StoreRoot:      paths.ArtifactStore,
		RepositoryRoot: parsed.repositoryRoot,
		Mode:           parsed.mode,
	})
	primary := fixture.VerdictPassed
	if runErr != nil {
		primary = fixture.VerdictHarnessError
	}
	frozen, freezeErr := fixture.FreezePrimary(primary)
	if freezeErr != nil {
		writeSafeError(stderr, "FIXTURE_VERDICT_REJECTED")
		return 70
	}
	final := root.Retention().Finalize(frozen)
	if final.Teardown.Status == fixture.TeardownFailed {
		writeSafeError(stderr, "FIXTURE_TEARDOWN_REJECTED")
		return 70
	}
	if runErr != nil {
		writeRejected(stderr, runErr, "FIXTURE_RUN_REJECTED")
		return 2
	}
	result := struct {
		Summary        workflow.Summary       `json:"summary"`
		LogicalRef     string                 `json:"logical_ref"`
		Retention      fixture.TeardownStatus `json:"retention_status"`
		ExpiryCategory string                 `json:"expiry_category"`
	}{
		Summary:        summary,
		LogicalRef:     root.LogicalID(),
		Retention:      final.Teardown.Status,
		ExpiryCategory: final.Teardown.ExpiryCategory,
	}
	if err := renderSafe(stdout, result); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return 0
}

func runValidate(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseValidateFlags(arguments)
	if err != nil || !knownKind(artifact.Kind(parsed.expectedKind)) {
		writeSafeError(stderr, "VALIDATE_ARGUMENTS_REJECTED")
		return 64
	}
	canonical, err := readBoundedArtifact(parsed.artifactPath)
	if err != nil {
		writeSafeError(stderr, "ARTIFACT_READ_REJECTED")
		return 2
	}
	envelope, err := artifact.Validate(artifact.Kind(parsed.expectedKind), canonical)
	if err != nil {
		writeSafeError(stderr, "ARTIFACT_VALIDATION_REJECTED")
		return 2
	}
	if _, rejection := privacy.Gate(privacy.Candidate{
		ArtifactKind: privacy.ArtifactKind(envelope.Kind),
		AdapterID:    privacy.AdapterCLIRenderer,
		Canonical:    canonical,
	}); rejection != nil {
		writePrivacyError(stderr, *rejection)
		return 2
	}
	result := struct {
		Status string        `json:"status"`
		Kind   artifact.Kind `json:"kind"`
		Digest string        `json:"digest"`
	}{Status: "valid", Kind: envelope.Kind, Digest: envelope.ContentDigest}
	if err := renderSafe(stdout, result); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return 0
}

func runStore(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseStoreFlags(arguments)
	if err != nil {
		writeSafeError(stderr, "STORE_ARGUMENTS_REJECTED")
		return 64
	}
	graph, err := readLineageGraph(parsed)
	if err != nil {
		writeSafeError(stderr, "ARTIFACT_READ_REJECTED")
		return 2
	}
	mode := artifact.LineageMode(parsed.mode)
	store, err := artifact.NewStore(parsed.storeRoot, parsed.repositoryRoot)
	if err != nil {
		writeSafeError(stderr, "STORE_ROOT_REJECTED")
		return 2
	}
	digests, err := store.WriteGraph(mode, graph)
	if err != nil {
		writeRejected(stderr, err, "ARTIFACT_STORE_REJECTED")
		return 2
	}
	result := struct {
		Status  string               `json:"status"`
		Mode    artifact.LineageMode `json:"mode"`
		Digests map[string]string    `json:"digests"`
	}{Status: "stored", Mode: mode, Digests: digests}
	if err := renderSafe(stdout, result); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return 0
}

func parseFixtureRunFlags(arguments []string) (fixtureRunFlags, error) {
	var parsed fixtureRunFlags
	flags := flag.NewFlagSet("fixture-run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.blueprintPath, "blueprint", "", "")
	flags.StringVar(&parsed.surfacesPath, "surfaces", "", "")
	flags.StringVar(&parsed.fixtureRoot, "fixture-root", "", "")
	flags.StringVar(&parsed.fixtureBase, "fixture-base", "", "")
	flags.StringVar(&parsed.fixtureID, "fixture-id", "", "")
	flags.DurationVar(&parsed.fixtureTTL, "fixture-ttl", 0, "")
	flags.BoolVar(&parsed.keepFixture, "keep-fixture", false, "")
	flags.StringVar(&parsed.storeRoot, "store-root", "", "")
	flags.StringVar(&parsed.repositoryRoot, "repo-root", "", "")
	flags.StringVar(&parsed.mode, "mode", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return fixtureRunFlags{}, errors.New("arguments rejected")
	}
	if parsed.blueprintPath == "" || parsed.surfacesPath == "" || parsed.repositoryRoot == "" || parsed.mode != "synthetic" {
		return fixtureRunFlags{}, errors.New("arguments rejected")
	}
	legacy := parsed.fixtureRoot != "" && parsed.storeRoot != "" && parsed.fixtureBase == "" && parsed.fixtureID == "" && parsed.fixtureTTL == 0 && !parsed.keepFixture
	managed := parsed.fixtureRoot == "" && parsed.storeRoot == "" && parsed.fixtureBase != "" && parsed.fixtureID != ""
	if !legacy && !managed {
		return fixtureRunFlags{}, errors.New("arguments rejected")
	}
	parsed.managed = managed
	return parsed, nil
}

func parseValidateFlags(arguments []string) (validateFlags, error) {
	var parsed validateFlags
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.expectedKind, "expect-kind", "", "")
	flags.StringVar(&parsed.artifactPath, "artifact", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || parsed.expectedKind == "" || parsed.artifactPath == "" {
		return validateFlags{}, errors.New("arguments rejected")
	}
	return parsed, nil
}

func parseStoreFlags(arguments []string) (storeFlags, error) {
	var parsed storeFlags
	flags := flag.NewFlagSet("store", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.mode, "mode", "", "")
	flags.StringVar(&parsed.storeRoot, "root", "", "")
	flags.StringVar(&parsed.repositoryRoot, "repo-root", "", "")
	flags.StringVar(&parsed.desiredPath, "desired", "", "")
	flags.StringVar(&parsed.observedPath, "observed", "", "")
	flags.StringVar(&parsed.freshObservedPath, "fresh-observed", "", "")
	flags.StringVar(&parsed.planPath, "plan", "", "")
	flags.StringVar(&parsed.receiptPath, "receipt", "", "")
	flags.StringVar(&parsed.evidencePath, "evidence", "", "")
	flags.StringVar(&parsed.reportPath, "report", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return storeFlags{}, errors.New("arguments rejected")
	}
	if parsed.storeRoot == "" || parsed.repositoryRoot == "" || parsed.desiredPath == "" || parsed.observedPath == "" || parsed.evidencePath == "" || parsed.reportPath == "" {
		return storeFlags{}, errors.New("arguments rejected")
	}
	switch artifact.LineageMode(parsed.mode) {
	case artifact.LineageApply:
		if parsed.planPath == "" || parsed.receiptPath == "" || parsed.freshObservedPath == "" {
			return storeFlags{}, errors.New("arguments rejected")
		}
	case artifact.LineageReadOnly:
		if parsed.planPath != "" || parsed.receiptPath != "" || parsed.freshObservedPath != "" {
			return storeFlags{}, errors.New("arguments rejected")
		}
	default:
		return storeFlags{}, errors.New("arguments rejected")
	}
	return parsed, nil
}

func parseTestPolicyFlags(arguments []string) (testPolicyFlags, error) {
	var parsed testPolicyFlags
	flags := flag.NewFlagSet("test-policy", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.tier, "tier", "", "")
	flags.StringVar(&parsed.networkManifestPath, "network-manifest", "", "")
	flags.StringVar(&parsed.repositoryRoot, "repo-root", "", "")
	flags.StringVar(&parsed.networkTestID, "allow-network-test", "", "")
	flags.StringVar(&parsed.liveProbeID, "live-probe", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return testPolicyFlags{}, errors.New("arguments rejected")
	}
	tier, err := fixture.ParseTier(parsed.tier)
	if err != nil {
		return testPolicyFlags{}, errors.New("arguments rejected")
	}
	switch tier {
	case fixture.TierOfflineStatic:
		if parsed.networkManifestPath != "" || parsed.repositoryRoot != "" || parsed.networkTestID != "" || parsed.liveProbeID != "" {
			return testPolicyFlags{}, errors.New("arguments rejected")
		}
	case fixture.TierIsolatedIntegration:
		if parsed.liveProbeID != "" {
			return testPolicyFlags{}, errors.New("arguments rejected")
		}
		requested := parsed.networkManifestPath != "" || parsed.repositoryRoot != "" || parsed.networkTestID != ""
		complete := parsed.networkManifestPath != "" && parsed.repositoryRoot != "" && parsed.networkTestID != ""
		if requested && !complete {
			return testPolicyFlags{}, errors.New("arguments rejected")
		}
	case fixture.TierLiveCheck:
		if parsed.networkManifestPath != "" || parsed.repositoryRoot != "" || parsed.networkTestID != "" {
			return testPolicyFlags{}, errors.New("arguments rejected")
		}
	}
	return parsed, nil
}

func parseSentinelVerifyFlags(arguments []string) (sentinelVerifyFlags, error) {
	var parsed sentinelVerifyFlags
	flags := flag.NewFlagSet("sentinel-verify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.mode, "mode", "", "")
	flags.StringVar(&parsed.manifestPath, "manifest", "", "")
	flags.StringVar(&parsed.fixtureRoot, "fixture-root", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || parsed.mode != "synthetic" || parsed.manifestPath == "" || parsed.fixtureRoot == "" {
		return sentinelVerifyFlags{}, errors.New("arguments rejected")
	}
	return parsed, nil
}

func presentPolicyEnvironmentKeys() []string {
	keys := []string{
		"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN", "SSH_AUTH_SOCK",
	}
	present := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := os.LookupEnv(key); ok {
			present = append(present, key)
		}
	}
	return present
}

func readLineageGraph(parsed storeFlags) (artifact.LineageGraph, error) {
	desired, err := readBoundedArtifact(parsed.desiredPath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	observed, err := readBoundedArtifact(parsed.observedPath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	evidence, err := readBoundedArtifact(parsed.evidencePath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	report, err := readBoundedArtifact(parsed.reportPath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	graph := artifact.LineageGraph{Desired: desired, Observed: observed, Evidence: evidence, Report: report}
	if parsed.mode == string(artifact.LineageApply) {
		graph.Plan, err = readBoundedArtifact(parsed.planPath)
		if err != nil {
			return artifact.LineageGraph{}, err
		}
		graph.Receipt, err = readBoundedArtifact(parsed.receiptPath)
		if err != nil {
			return artifact.LineageGraph{}, err
		}
		graph.FreshObserved, err = readBoundedArtifact(parsed.freshObservedPath)
		if err != nil {
			return artifact.LineageGraph{}, err
		}
	}
	var evidenceEnvelope artifact.Envelope
	evidenceEnvelope, err = artifact.Validate(artifact.VerificationEvidence, evidence)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	var evidencePayload struct {
		ExpectedPostconditionsDigest string `json:"expected_postconditions_digest"`
	}
	if err := json.Unmarshal(evidenceEnvelope.Payload, &evidencePayload); err != nil || evidencePayload.ExpectedPostconditionsDigest == "" {
		return artifact.LineageGraph{}, errors.New("evidence rejected")
	}
	graph.ExpectedPostconditionsDigest = evidencePayload.ExpectedPostconditionsDigest
	return graph, nil
}

func readBoundedArtifact(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("artifact rejected")
	}
	var buffer bytes.Buffer
	if _, err := io.CopyN(&buffer, file, maxArtifactBytes+1); err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.New("artifact rejected")
	}
	if buffer.Len() > maxArtifactBytes {
		return nil, errors.New("artifact rejected")
	}
	return buffer.Bytes(), nil
}

func knownKind(kind artifact.Kind) bool {
	for _, candidate := range artifact.RegisteredKinds() {
		if kind == candidate {
			return true
		}
	}
	return false
}

func writeSafeError(writer io.Writer, code string) {
	errorCode := privacy.CodeOperationRejected
	category := privacy.CategoryOperation
	remediation := privacy.RemediationReview
	switch code {
	case "UNSUPPORTED_COMMAND":
		errorCode = privacy.CodeCommandRejected
		category = privacy.CategoryUnsupported
		remediation = privacy.RemediationCommand
	case "OUTPUT_REJECTED":
		errorCode = privacy.CodeOutputRejected
	}
	writePrivacyError(writer, privacy.SafeOperationError(errorCode, category, remediation))
}

func writeRejected(writer io.Writer, err error, fallback string) {
	var envelope *privacy.ErrorEnvelope
	if errors.As(err, &envelope) {
		writePrivacyError(writer, *envelope)
		return
	}
	writeSafeError(writer, fallback)
}

func writePrivacyError(writer io.Writer, envelope privacy.ErrorEnvelope) {
	_ = privacy.RenderError(writer, envelope)
}

func renderSafe(writer io.Writer, value any) error {
	if rejection := privacy.Render(writer, privacy.Candidate{
		ArtifactKind: privacy.KindCommandResult,
		AdapterID:    privacy.AdapterCLIRenderer,
		Value:        value,
	}); rejection != nil {
		return rejection
	}
	return nil
}
