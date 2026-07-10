package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"example.invalid/yamc/safety/internal/artifact"
	"example.invalid/yamc/safety/internal/contract"
	"example.invalid/yamc/safety/internal/privacy"
	"example.invalid/yamc/safety/internal/sentinel"
)

const successState = "synthetic-sentinel-passed"

func SyntheticSentinelState() string {
	return successState
}

func IsSyntheticSentinelState(state string) bool {
	return state == successState
}

type Options struct {
	BlueprintPath  string
	SurfacesPath   string
	FixtureRoot    string
	StoreRoot      string
	RepositoryRoot string
	Mode           string
}

type Summary struct {
	State          string            `json:"state"`
	ArtifactCount  int               `json:"artifact_count"`
	KindCount      int               `json:"kind_count"`
	ManifestDigest string            `json:"manifest_digest"`
	Artifacts      map[string]string `json:"artifacts"`
}

type PhaseReportOptions struct {
	SuitePath          string
	ExpectedReportPath string
	SummaryPath        string
	StoreRoot          string
	RepositoryRoot     string
}

type PhaseSurfaceEvidence struct {
	SurfaceDomain string `json:"surface_domain"`
	LogicalRef    string `json:"logical_ref"`
	BeforeToken   string `json:"before_token"`
	AfterToken    string `json:"after_token"`
}

type PhaseCurrentHostStatus struct {
	Status        string `json:"status"`
	Verdict       string `json:"verdict"`
	Reason        string `json:"reason"`
	ClaimEligible bool   `json:"claim_eligible"`
}

type PhaseReport struct {
	SchemaVersion     string                 `json:"schema_version"`
	SuiteID           string                 `json:"suite_id"`
	Tier              string                 `json:"tier"`
	EvidenceMode      string                 `json:"evidence_mode"`
	InnerStatus       string                 `json:"inner_status"`
	OuterSequence     []string               `json:"outer_sequence"`
	Verdict           string                 `json:"verdict"`
	Claim             string                 `json:"claim"`
	ArtifactKinds     []string               `json:"artifact_kinds"`
	ArtifactInstances int                    `json:"artifact_instances"`
	ArtifactDigests   map[string]string      `json:"artifact_digests"`
	ManifestDigests   map[string]string      `json:"manifest_digests"`
	SurfaceEvidence   []PhaseSurfaceEvidence `json:"surface_evidence"`
	PolicyStatuses    []string               `json:"policy_statuses"`
	Operations        []contract.Operation   `json:"operations"`
	CurrentHost       PhaseCurrentHostStatus `json:"current_host"`
}

type offlineSuite struct {
	SchemaVersion   string            `json:"schema_version"`
	SuiteID         string            `json:"suite_id"`
	Tier            string            `json:"tier"`
	EvidenceMode    string            `json:"evidence_mode"`
	TaskGroups      []suiteTaskGroup  `json:"task_groups"`
	PhaseOrder      []string          `json:"phase_order"`
	Manifests       []manifestBinding `json:"manifests"`
	ExpectedClaim   string            `json:"expected_claim"`
	CurrentHostGate string            `json:"current_host_gate"`
	NegativeMatrix  []negativeBinding `json:"negative_matrix"`
}

type suiteTaskGroup struct {
	Wave  string   `json:"wave"`
	Tasks []string `json:"tasks"`
}

type manifestBinding struct {
	ID         string `json:"id"`
	LogicalRef string `json:"logical_ref"`
	Digest     string `json:"digest"`
}

type negativeBinding struct {
	DecisionID string `json:"decision_id"`
	TaskSuite  string `json:"task_suite"`
}

type phaseReportTemplate struct {
	SchemaVersion        string                 `json:"schema_version"`
	SuiteID              string                 `json:"suite_id"`
	Tier                 string                 `json:"tier"`
	EvidenceMode         string                 `json:"evidence_mode"`
	InnerStatus          string                 `json:"inner_status"`
	OuterSequence        []string               `json:"outer_sequence"`
	Verdict              string                 `json:"verdict"`
	Claim                string                 `json:"claim"`
	ArtifactKinds        []string               `json:"artifact_kinds"`
	ArtifactInstances    int                    `json:"artifact_instances"`
	ArtifactDigestLabels []string               `json:"artifact_digest_labels"`
	ManifestBindingIDs   []string               `json:"manifest_binding_ids"`
	SurfaceEvidence      []PhaseSurfaceEvidence `json:"surface_evidence"`
	PolicyStatuses       []string               `json:"policy_statuses"`
	Operations           []contract.Operation   `json:"operations"`
	CurrentHost          PhaseCurrentHostStatus `json:"current_host"`
}

type fact struct {
	Ref   string `json:"ref"`
	State string `json:"state"`
}

type blueprint struct {
	SchemaVersion          string `json:"schema_version"`
	RunID                  string `json:"run_id"`
	SuiteID                string `json:"suite_id"`
	Profile                string `json:"profile"`
	Desired                []fact `json:"desired"`
	Observed               []fact `json:"observed"`
	ExpectedPostconditions []fact `json:"expected_postconditions"`
	OperationID            string `json:"operation_id"`
}

type preparedArtifact struct {
	canonical []byte
	envelope  artifact.Envelope
}

func RunSynthetic(options Options) (Summary, error) {
	if options.Mode != "synthetic" {
		return Summary{}, errors.New("synthetic mode required")
	}
	repositoryRoot, blueprintPath, surfacesPath, rawSamplePath, fixtureRoot, storeRoot, err := preflight(options)
	if err != nil {
		return Summary{}, err
	}

	blueprintBytes, err := readBounded(blueprintPath)
	if err != nil {
		return Summary{}, err
	}
	input, err := parseBlueprint(blueprintBytes)
	if err != nil {
		return Summary{}, err
	}
	policyDecision, err := contract.Phase1Policy().Evaluate(contract.PolicyRequest{
		SchemaVersion: contract.PolicySchemaVersion,
		Provenance:    "synthetic",
		Intent:        contract.IntentSyntheticFixture,
		Status:        contract.StatusSyntheticFixture,
		Operations: []contract.Operation{{
			Kind:   contract.OperationFixtureFakeWrite,
			Target: input.OperationID,
			Mode:   "synthetic",
		}},
	})
	if err != nil || len(policyDecision.Operations) != 1 {
		return Summary{}, errors.New("synthetic operation policy rejected")
	}
	operationID := policyDecision.Operations[0].Target
	surfacesBytes, err := readBounded(surfacesPath)
	if err != nil {
		return Summary{}, err
	}
	manifest, err := sentinel.ParseManifest(surfacesBytes)
	if err != nil {
		return Summary{}, err
	}
	if err := os.MkdirAll(fixtureRoot, 0o700); err != nil {
		return Summary{}, errors.New("fixture root unavailable")
	}
	if err := sentinel.PrepareSynthetic(manifest, fixtureRoot); err != nil {
		return Summary{}, err
	}
	before, err := sentinel.ObserveSynthetic(manifest, fixtureRoot)
	if err != nil {
		return Summary{}, err
	}

	blueprintDigest, err := artifact.DigestValue(input)
	if err != nil {
		return Summary{}, err
	}
	run := artifact.RunMetadata{RunID: input.RunID, Tier: "offline-static", SuiteID: input.SuiteID}
	desired, err := makeArtifact(artifact.DesiredState, run, []string{blueprintDigest}, struct {
		Profile      string `json:"profile"`
		Declarations []fact `json:"declarations"`
	}{input.Profile, input.Desired})
	if err != nil {
		return Summary{}, err
	}
	observed, err := makeArtifact(artifact.ObservedState, run, []string{blueprintDigest}, struct {
		Scope string `json:"scope"`
		Facts []fact `json:"facts"`
	}{"fixture:scope/walking-skeleton", input.Observed})
	if err != nil {
		return Summary{}, err
	}
	expectedDigest, err := artifact.DigestValue(input.ExpectedPostconditions)
	if err != nil {
		return Summary{}, err
	}
	plan, err := makeArtifact(artifact.GeneratedPlan, run, []string{desired.envelope.ContentDigest, observed.envelope.ContentDigest}, struct {
		DesiredDigest                string   `json:"desired_digest"`
		ObservedDigest               string   `json:"observed_digest"`
		ExpectedPostconditionsDigest string   `json:"expected_postconditions_digest"`
		OperationIDs                 []string `json:"operation_ids"`
	}{desired.envelope.ContentDigest, observed.envelope.ContentDigest, expectedDigest, []string{operationID}})
	if err != nil {
		return Summary{}, err
	}
	receipt, err := makeArtifact(artifact.AppliedReceipt, run, []string{plan.envelope.ContentDigest}, struct {
		PlanDigest   string   `json:"plan_digest"`
		Mode         string   `json:"mode"`
		OperationIDs []string `json:"operation_ids"`
		Outcome      string   `json:"outcome"`
	}{plan.envelope.ContentDigest, "synthetic", []string{operationID}, "fixture:outcome/completed"})
	if err != nil {
		return Summary{}, err
	}

	rawSample, err := readBounded(rawSamplePath)
	if err != nil {
		return Summary{}, err
	}
	registry, err := privacy.MaterializeFixtureAdapter(fixtureRoot, rawSample)
	if err != nil {
		return Summary{}, errors.New("synthetic adapter unavailable")
	}
	captured, rejection := privacy.Capture(context.Background(), registry, privacy.CommandFixtureFake, privacy.Limits{})
	if rejection != nil {
		return Summary{}, rejection
	}
	normalizedFacts, ok := capturedFacts(captured, input.ExpectedPostconditions)
	if !ok {
		return Summary{}, errors.New("synthetic adapter normalization rejected")
	}
	freshObservedArtifact, err := makeArtifact(artifact.ObservedState, run, []string{receipt.envelope.ContentDigest}, struct {
		Scope string `json:"scope"`
		Facts []fact `json:"facts"`
	}{"fixture:scope/walking-skeleton", normalizedFacts})
	if err != nil {
		return Summary{}, err
	}
	fresh := artifact.FreshObserved{
		Scope:               "fixture:scope/walking-skeleton",
		State:               input.ExpectedPostconditions[0].State,
		SourceReceiptDigest: receipt.envelope.ContentDigest,
		ContentDigest:       freshObservedArtifact.envelope.ContentDigest,
	}
	after, err := sentinel.ObserveSynthetic(manifest, fixtureRoot)
	if err != nil || !sentinel.Equal(before, after) {
		return Summary{}, errors.New("synthetic sentinel rejected run")
	}
	evidence, err := makeArtifact(artifact.VerificationEvidence, run, []string{plan.envelope.ContentDigest, receipt.envelope.ContentDigest, expectedDigest, freshObservedArtifact.envelope.ContentDigest}, struct {
		PlanDigest                   string                 `json:"plan_digest"`
		ReceiptDigest                string                 `json:"receipt_digest"`
		ExpectedPostconditionsDigest string                 `json:"expected_postconditions_digest"`
		FreshObservedDigest          string                 `json:"fresh_observed_digest"`
		FreshObserved                artifact.FreshObserved `json:"fresh_observed"`
		ManifestDigest               string                 `json:"manifest_digest"`
		SentinelBeforeDigest         string                 `json:"sentinel_before_digest"`
		SentinelAfterDigest          string                 `json:"sentinel_after_digest"`
	}{plan.envelope.ContentDigest, receipt.envelope.ContentDigest, expectedDigest, freshObservedArtifact.envelope.ContentDigest, fresh, manifest.Digest, before.StateDigest, after.StateDigest})
	if err != nil {
		return Summary{}, err
	}
	report, err := makeArtifact(artifact.ReadinessReport, run, []string{evidence.envelope.ContentDigest}, struct {
		EvidenceDigest string `json:"evidence_digest"`
		State          string `json:"state"`
	}{evidence.envelope.ContentDigest, successState})
	if err != nil {
		return Summary{}, err
	}

	store, err := artifact.NewStore(storeRoot, repositoryRoot)
	if err != nil {
		return Summary{}, err
	}
	graph := artifact.LineageGraph{
		Desired:                      desired.canonical,
		Observed:                     observed.canonical,
		Plan:                         plan.canonical,
		Receipt:                      receipt.canonical,
		FreshObserved:                freshObservedArtifact.canonical,
		Evidence:                     evidence.canonical,
		Report:                       report.canonical,
		ExpectedPostconditionsDigest: expectedDigest,
	}
	digests, err := store.WriteGraph(artifact.LineageApply, graph)
	if err != nil || len(digests) != 7 || digests[artifact.FreshObservedKey] != freshObservedArtifact.envelope.ContentDigest {
		return Summary{}, errors.New("artifact store write rejected")
	}
	kinds := map[artifact.Kind]struct{}{}
	for _, prepared := range []preparedArtifact{desired, observed, plan, receipt, freshObservedArtifact, evidence, report} {
		kinds[prepared.envelope.Kind] = struct{}{}
	}
	return Summary{
		State:          successState,
		ArtifactCount:  len(digests),
		KindCount:      len(kinds),
		ManifestDigest: manifest.Digest,
		Artifacts:      digests,
	}, nil
}

func preflight(options Options) (string, string, string, string, string, string, error) {
	repositoryRoot, err := filepath.EvalSymlinks(options.RepositoryRoot)
	if err != nil {
		return "", "", "", "", "", "", errors.New("repository root rejected")
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return "", "", "", "", "", "", errors.New("repository root rejected")
	}
	blueprintPath, err := validateTrackedInput(options.BlueprintPath, repositoryRoot)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	surfacesPath, err := validateTrackedInput(options.SurfacesPath, repositoryRoot)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	rawSamplePath, err := validateTrackedInput(filepath.Join(repositoryRoot, "safety", "testdata", "raw", "fake-adapter.json"), repositoryRoot)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	fixtureRoot, err := artifact.ValidateExternalRoot(options.FixtureRoot, repositoryRoot)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	storeRoot, err := artifact.ValidateExternalRoot(options.StoreRoot, repositoryRoot)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	return repositoryRoot, blueprintPath, surfacesPath, rawSamplePath, fixtureRoot, storeRoot, nil
}

func validateTrackedInput(path, repositoryRoot string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("tracked input rejected")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.New("tracked input rejected")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("tracked input rejected")
	}
	relative, err := filepath.Rel(repositoryRoot, resolved)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("tracked input rejected")
	}
	return resolved, nil
}

func parseBlueprint(data []byte) (blueprint, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var input blueprint
	if err := decoder.Decode(&input); err != nil {
		return blueprint{}, errors.New("synthetic blueprint rejected")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return blueprint{}, errors.New("synthetic blueprint rejected")
	}
	if input.SchemaVersion != "1.0.0" ||
		!validLogicalRef(input.RunID) || !strings.HasPrefix(input.RunID, "fixture:") ||
		!validLogicalRef(input.SuiteID) || !strings.HasPrefix(input.SuiteID, "fixture:") ||
		!validLogicalRef(input.Profile) || !strings.HasPrefix(input.Profile, "profile:") ||
		!validLogicalRef(input.OperationID) || !strings.HasPrefix(input.OperationID, "fixture:") ||
		len(input.Desired) == 0 || len(input.Observed) == 0 || len(input.ExpectedPostconditions) == 0 {
		return blueprint{}, errors.New("synthetic blueprint rejected")
	}
	for _, group := range [][]fact{input.Desired, input.Observed, input.ExpectedPostconditions} {
		for _, item := range group {
			if !validLogicalRef(item.Ref) || item.State == "" {
				return blueprint{}, errors.New("synthetic blueprint rejected")
			}
		}
	}
	return input, nil
}

func validLogicalRef(value string) bool {
	allowed := []string{"repo:", "home:", "fixture:", "local-state:", "nix-output:", "profile:"}
	prefix := ""
	for _, candidate := range allowed {
		if strings.HasPrefix(value, candidate) {
			prefix = candidate
			break
		}
	}
	if prefix == "" {
		return false
	}
	relative := strings.TrimPrefix(value, prefix)
	if relative == "" || strings.HasPrefix(relative, "/") || strings.Contains(relative, "\\") || strings.ContainsRune(relative, '\x00') {
		return false
	}
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func makeArtifact(kind artifact.Kind, run artifact.RunMetadata, inputs []string, payload any) (preparedArtifact, error) {
	canonical, envelope, err := artifact.New(kind, run, artifact.Provenance{Mode: "synthetic", InputDigests: inputs}, payload)
	return preparedArtifact{canonical: canonical, envelope: envelope}, err
}

func capturedFacts(observation privacy.Observation, expected []fact) ([]fact, bool) {
	if observation.Status != "normalized" || len(observation.Facts) != len(expected) {
		return nil, false
	}
	result := make([]fact, len(observation.Facts))
	for index, captured := range observation.Facts {
		if captured.Ref != expected[index].Ref || captured.State != expected[index].State {
			return nil, false
		}
		result[index] = fact{Ref: captured.Ref, State: captured.State}
	}
	return result, true
}

func readBounded(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 64<<10 {
		return nil, errors.New("tracked input rejected")
	}
	return data, nil
}

func BuildPhaseReport(options PhaseReportOptions) (PhaseReport, error) {
	repositoryRoot, err := filepath.EvalSymlinks(options.RepositoryRoot)
	if err != nil {
		return PhaseReport{}, errors.New("phase repository rejected")
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return PhaseReport{}, errors.New("phase repository rejected")
	}
	suitePath, err := validateTrackedInput(options.SuitePath, repositoryRoot)
	if err != nil || filepath.ToSlash(mustRelative(repositoryRoot, suitePath)) != "safety/manifests/offline-suite.v1.json" {
		return PhaseReport{}, errors.New("phase suite rejected")
	}
	expectedPath, err := validateTrackedInput(options.ExpectedReportPath, repositoryRoot)
	if err != nil || filepath.ToSlash(mustRelative(repositoryRoot, expectedPath)) != "safety/testdata/blueprints/walking-skeleton/expected-report.json" {
		return PhaseReport{}, errors.New("phase expected report rejected")
	}

	suiteData, err := readBoundedNoSymlink(suitePath)
	if err != nil {
		return PhaseReport{}, errors.New("phase suite rejected")
	}
	var suite offlineSuite
	if err := decodeClosedPhase(suiteData, &suite); err != nil || validateOfflineSuite(suite) != nil {
		return PhaseReport{}, errors.New("phase suite rejected")
	}
	expectedData, err := readBoundedNoSymlink(expectedPath)
	if err != nil {
		return PhaseReport{}, errors.New("phase expected report rejected")
	}
	var expected phaseReportTemplate
	if err := decodeClosedPhase(expectedData, &expected); err != nil {
		return PhaseReport{}, errors.New("phase expected report rejected")
	}

	manifestDigests, err := validateManifestBindings(suite, repositoryRoot, expectedPath)
	if err != nil {
		return PhaseReport{}, err
	}
	currentHost, err := assessCurrentHostProof(repositoryRoot)
	if err != nil {
		return PhaseReport{}, err
	}
	if err := validatePhaseTemplate(expected, suite, currentHost); err != nil {
		return PhaseReport{}, err
	}

	summaryData, err := readBoundedNoSymlink(options.SummaryPath)
	if err != nil {
		return PhaseReport{}, errors.New("phase summary rejected")
	}
	var summary Summary
	if err := decodeClosedPhase(summaryData, &summary); err != nil || summary.State != successState || summary.ArtifactCount != 7 || summary.KindCount != 6 || len(summary.Artifacts) != 7 {
		return PhaseReport{}, errors.New("phase summary rejected")
	}
	if err := validateSyntheticManifestDigest(repositoryRoot, summary.ManifestDigest); err != nil {
		return PhaseReport{}, err
	}
	if err := validatePhasePolicies(expected.PolicyStatuses); err != nil {
		return PhaseReport{}, err
	}
	if err := validateArtifactGraph(options.StoreRoot, repositoryRoot, summary); err != nil {
		return PhaseReport{}, err
	}

	report := PhaseReport{
		SchemaVersion:     expected.SchemaVersion,
		SuiteID:           expected.SuiteID,
		Tier:              expected.Tier,
		EvidenceMode:      expected.EvidenceMode,
		InnerStatus:       expected.InnerStatus,
		OuterSequence:     append([]string(nil), expected.OuterSequence...),
		Verdict:           expected.Verdict,
		Claim:             expected.Claim,
		ArtifactKinds:     append([]string(nil), expected.ArtifactKinds...),
		ArtifactInstances: expected.ArtifactInstances,
		ArtifactDigests:   cloneStrings(summary.Artifacts),
		ManifestDigests:   manifestDigests,
		SurfaceEvidence:   append([]PhaseSurfaceEvidence(nil), expected.SurfaceEvidence...),
		PolicyStatuses:    append([]string(nil), expected.PolicyStatuses...),
		Operations:        make([]contract.Operation, 0),
		CurrentHost:       currentHost,
	}
	approved, rejection := privacy.Gate(privacy.Candidate{ArtifactKind: privacy.KindCommandResult, AdapterID: privacy.AdapterCLIRenderer, Value: report})
	if rejection != nil || len(approved) == 0 {
		return PhaseReport{}, errors.New("phase report privacy rejected")
	}
	return report, nil
}

func validateOfflineSuite(suite offlineSuite) error {
	if suite.SchemaVersion != "1.0.0" || suite.SuiteID != "phase-01-offline-safety-v1" || suite.Tier != "offline-static" || suite.EvidenceMode != "isolated-proof-double" || suite.ExpectedClaim != sentinel.ScopedUnchangedClaim || suite.CurrentHostGate != "manual-required" {
		return errors.New("phase suite identity rejected")
	}
	wantGroups := []suiteTaskGroup{
		{Wave: "skeleton", Tasks: []string{"walking-skeleton"}},
		{Wave: "artifact-contracts", Tasks: []string{"artifact-kinds", "artifact-lineage"}},
		{Wave: "privacy", Tasks: []string{"privacy-boundary", "bounded-capture"}},
		{Wave: "fixture-policy", Tasks: []string{"fixture-lifecycle", "tier-network-policy"}},
		{Wave: "sentinels", Tasks: []string{"sentinel-manifest", "sentinel-verdicts", "real-sentinel-envelope"}},
		{Wave: "controlplane", Tasks: []string{"controlplane-contract", "no-destructive-defaults"}},
		{Wave: "phase-integration", Tasks: []string{"phase-e2e"}},
	}
	wantOrder := []string{"wave:skeleton", "wave:artifact-contracts", "wave:privacy", "wave:fixture-policy", "wave:sentinels", "wave:controlplane", "task:phase-e2e"}
	if !samePhaseValue(suite.TaskGroups, wantGroups) || !equalPhaseStrings(suite.PhaseOrder, wantOrder) || len(suite.Manifests) != 4 || len(suite.NegativeMatrix) != 19 {
		return errors.New("phase suite composition rejected")
	}
	seen := make(map[string]struct{}, 19)
	for _, binding := range suite.NegativeMatrix {
		if binding.TaskSuite == "" {
			return errors.New("phase decision binding rejected")
		}
		seen[binding.DecisionID] = struct{}{}
	}
	for index := 1; index <= 19; index++ {
		identifier := "D-" + twoDigit(index)
		if _, ok := seen[identifier]; !ok {
			return errors.New("phase decision binding rejected")
		}
	}
	return nil
}

func validateManifestBindings(suite offlineSuite, repositoryRoot, expectedPath string) (map[string]string, error) {
	want := map[string]string{
		"protected-surfaces": "repo:safety/manifests/protected-surfaces.v1.json",
		"real-adapters":      "repo:safety/manifests/real-adapters.v1.json",
		"network-contract":   "repo:safety/manifests/network-tests.v1.json",
		"expected-report":    "repo:safety/testdata/blueprints/walking-skeleton/expected-report.json",
	}
	result := make(map[string]string, len(want))
	for _, binding := range suite.Manifests {
		logical, err := privacy.ParseLogicalRef(binding.LogicalRef)
		if err != nil || logical.Namespace != privacy.NamespaceRepo || want[binding.ID] != binding.LogicalRef || !artifact.IsDigest(binding.Digest) {
			return nil, errors.New("phase manifest binding rejected")
		}
		physical, err := validateTrackedInput(filepath.Join(repositoryRoot, filepath.FromSlash(logical.ID)), repositoryRoot)
		if err != nil {
			return nil, errors.New("phase manifest binding rejected")
		}
		if binding.ID == "expected-report" && filepath.Clean(physical) != filepath.Clean(expectedPath) {
			return nil, errors.New("phase expected report binding rejected")
		}
		data, err := readBoundedNoSymlink(physical)
		if err != nil || digestPhaseBytes(data) != binding.Digest {
			return nil, errors.New("phase manifest digest rejected")
		}
		if _, duplicate := result[binding.ID]; duplicate {
			return nil, errors.New("phase manifest binding rejected")
		}
		result[binding.ID] = binding.Digest
	}
	if len(result) != len(want) {
		return nil, errors.New("phase manifest binding rejected")
	}
	return result, nil
}

func assessCurrentHostProof(repositoryRoot string) (PhaseCurrentHostStatus, error) {
	protectedData, err := readBoundedNoSymlink(filepath.Join(repositoryRoot, "safety", "manifests", "protected-surfaces.v1.json"))
	if err != nil {
		return PhaseCurrentHostStatus{}, errors.New("phase protected manifest rejected")
	}
	manifest, err := sentinel.ParseProtectedManifest(protectedData)
	if err != nil {
		return PhaseCurrentHostStatus{}, errors.New("phase protected manifest rejected")
	}
	adapterData, err := readBoundedNoSymlink(filepath.Join(repositoryRoot, "safety", "manifests", "real-adapters.v1.json"))
	if err != nil {
		return PhaseCurrentHostStatus{}, errors.New("phase adapter manifest rejected")
	}
	registry, err := sentinel.LoadRealAdapterRegistry(adapterData, time.Now().UTC())
	if err != nil {
		return PhaseCurrentHostStatus{}, errors.New("phase adapter manifest rejected")
	}
	assessment := registry.Assess(manifest)
	if assessment.Status != "manual-required" || assessment.Verdict != sentinel.VerdictIndeterminate || assessment.ExitCode != 32 || assessment.Reason != "required-real-adapter-proof-unavailable" || assessment.ClaimEligible {
		return PhaseCurrentHostStatus{}, errors.New("phase current-host proof boundary rejected")
	}
	return PhaseCurrentHostStatus{Status: assessment.Status, Verdict: string(assessment.Verdict), Reason: assessment.Reason, ClaimEligible: false}, nil
}

func validatePhaseTemplate(expected phaseReportTemplate, suite offlineSuite, currentHost PhaseCurrentHostStatus) error {
	wantSequence := []string{"real-before", "isolated-workload", "freeze-primary", "fixture-finalize", "real-after", "monotonic-combine"}
	wantKinds := []string{"applied-receipt", "desired-state", "generated-plan", "observed-state", "readiness-report", "verification-evidence"}
	wantLabels := []string{"applied-receipt", "desired-state", "fresh-observed-state", "generated-plan", "observed-state", "readiness-report", "verification-evidence"}
	bindingIDs := make([]string, 0, len(suite.Manifests))
	for _, binding := range suite.Manifests {
		bindingIDs = append(bindingIDs, binding.ID)
	}
	sort.Strings(bindingIDs)
	if expected.SchemaVersion != suite.SchemaVersion || expected.SuiteID != suite.SuiteID || expected.Tier != suite.Tier || expected.EvidenceMode != suite.EvidenceMode || expected.InnerStatus != successState || expected.Verdict != "passed" || expected.Claim != suite.ExpectedClaim || expected.ArtifactInstances != 7 {
		return errors.New("phase expected report identity rejected")
	}
	if !equalPhaseStrings(expected.OuterSequence, wantSequence) || !equalPhaseStrings(expected.ArtifactKinds, wantKinds) || !equalPhaseStrings(expected.ArtifactDigestLabels, wantLabels) || !equalPhaseStrings(expected.ManifestBindingIDs, bindingIDs) || len(expected.Operations) != 0 {
		return errors.New("phase expected report composition rejected")
	}
	if !samePhaseValue(expected.CurrentHost, currentHost) || !equalPhaseStrings(expected.PolicyStatuses, []string{"extra", "unmanaged-present"}) || len(expected.SurfaceEvidence) != 6 {
		return errors.New("phase expected report policy rejected")
	}
	wantSurfaces := map[string]privacy.SurfaceDomain{
		"repo:sentinel/worktree/tracked":               privacy.SurfaceWorktree,
		"repo:sentinel/worktree/index":                 privacy.SurfaceWorktree,
		"home:.zshrc":                                  privacy.SurfaceNamedHome,
		"home:sentinel/manager/mise-data":              privacy.SurfaceManagerRoot,
		"profile:sentinel/service/homebrew-mxcl-nginx": privacy.SurfaceService,
		"profile:sentinel/named-target/system-shells":  privacy.SurfaceNamedTarget,
	}
	seen := make(map[string]struct{}, len(wantSurfaces))
	for _, surface := range expected.SurfaceEvidence {
		domain, ok := wantSurfaces[surface.LogicalRef]
		if !ok || string(domain) != surface.SurfaceDomain || privacy.ValidateSurface(domain, surface.LogicalRef) != nil || !validOpaquePhaseToken(surface.BeforeToken) || surface.BeforeToken != surface.AfterToken {
			return errors.New("phase expected surface rejected")
		}
		if _, duplicate := seen[surface.LogicalRef]; duplicate {
			return errors.New("phase expected surface rejected")
		}
		seen[surface.LogicalRef] = struct{}{}
	}
	return nil
}

func validateSyntheticManifestDigest(repositoryRoot, expectedDigest string) error {
	data, err := readBoundedNoSymlink(filepath.Join(repositoryRoot, "safety", "testdata", "blueprints", "walking-skeleton", "protected-surfaces.json"))
	if err != nil {
		return errors.New("phase synthetic manifest rejected")
	}
	manifest, err := sentinel.ParseManifest(data)
	if err != nil || manifest.Digest != expectedDigest {
		return errors.New("phase synthetic manifest digest rejected")
	}
	return nil
}

func validatePhasePolicies(statuses []string) error {
	for _, status := range statuses {
		request := contract.PolicyRequest{SchemaVersion: contract.PolicySchemaVersion, Provenance: "synthetic", Intent: contract.IntentReportOnly, Status: contract.PolicyStatus(status), Operations: make([]contract.Operation, 0)}
		decision, err := contract.Phase1Policy().Evaluate(request)
		if err != nil || string(decision.Status) != status || len(decision.Operations) != 0 {
			return errors.New("phase report-only policy rejected")
		}
	}
	return nil
}

func validateArtifactGraph(storeRoot, repositoryRoot string, summary Summary) error {
	wantKinds := map[string]artifact.Kind{
		"desired-state":         artifact.DesiredState,
		"observed-state":        artifact.ObservedState,
		"generated-plan":        artifact.GeneratedPlan,
		"applied-receipt":       artifact.AppliedReceipt,
		"fresh-observed-state":  artifact.ObservedState,
		"verification-evidence": artifact.VerificationEvidence,
		"readiness-report":      artifact.ReadinessReport,
	}
	if len(summary.Artifacts) != len(wantKinds) {
		return errors.New("phase artifact set rejected")
	}
	store, err := artifact.NewStore(storeRoot, repositoryRoot)
	if err != nil {
		return errors.New("phase artifact store rejected")
	}
	canonical := make(map[string][]byte, len(wantKinds))
	seenDigests := make(map[string]struct{}, len(wantKinds))
	for label, kind := range wantKinds {
		digest, ok := summary.Artifacts[label]
		if !ok || !artifact.IsDigest(digest) {
			return errors.New("phase artifact digest rejected")
		}
		if _, duplicate := seenDigests[digest]; duplicate {
			return errors.New("phase artifact digest rejected")
		}
		seenDigests[digest] = struct{}{}
		data, envelope, err := store.Read(digest)
		if err != nil || envelope.Kind != kind || envelope.StorageClass != artifact.ExternalLocalState || envelope.ContentDigest != digest {
			return errors.New("phase artifact reload rejected")
		}
		if approved, rejection := privacy.Gate(privacy.Candidate{ArtifactKind: privacy.ArtifactKind(envelope.Kind), AdapterID: privacy.AdapterArtifactStore, Canonical: data}); rejection != nil || len(approved) == 0 {
			return errors.New("phase artifact privacy rejected")
		}
		canonical[label] = data
	}
	entries, err := os.ReadDir(filepath.Join(storeRoot, "sha256"))
	if err != nil || len(entries) != len(wantKinds) {
		return errors.New("phase artifact object set rejected")
	}
	var evidencePayload artifact.VerificationEvidencePayload
	evidenceEnvelope, err := artifact.Validate(artifact.VerificationEvidence, canonical["verification-evidence"])
	if err != nil || json.Unmarshal(evidenceEnvelope.Payload, &evidencePayload) != nil {
		return errors.New("phase evidence reload rejected")
	}
	graph := artifact.LineageGraph{
		Desired: canonical["desired-state"], Observed: canonical["observed-state"], Plan: canonical["generated-plan"],
		Receipt: canonical["applied-receipt"], FreshObserved: canonical["fresh-observed-state"], Evidence: canonical["verification-evidence"],
		Report: canonical["readiness-report"], ExpectedPostconditionsDigest: evidencePayload.ExpectedPostconditionsDigest,
	}
	if err := artifact.ValidateLineage(artifact.LineageApply, graph); err != nil {
		return errors.New("phase artifact lineage rejected")
	}
	return nil
}

func decodeClosedPhase(data []byte, target any) error {
	canonical, err := artifact.Canonicalize(data)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("phase json trailing value rejected")
	}
	return nil
}

func readBoundedNoSymlink(path string) ([]byte, error) {
	if path == "" || !filepath.IsAbs(path) {
		return nil, errors.New("phase input rejected")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Size() > 64<<10 {
		return nil, errors.New("phase input rejected")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 64<<10 {
		return nil, errors.New("phase input rejected")
	}
	after, err := os.Lstat(path)
	if err != nil || !after.Mode().IsRegular() || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, after) || before.Size() != after.Size() || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) {
		return nil, errors.New("phase input rejected")
	}
	return data, nil
}

func digestPhaseBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validOpaquePhaseToken(value string) bool {
	if !strings.HasPrefix(value, "hmac-sha256:") {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "hmac-sha256:"))
	return err == nil && len(decoded) == sha256.Size
}

func mustRelative(root, candidate string) string {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return ""
	}
	return relative
}

func samePhaseValue(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func equalPhaseStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneStrings(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func twoDigit(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10))
}
