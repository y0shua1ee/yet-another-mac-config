package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"example.invalid/yamc/safety/internal/fixture"
	"example.invalid/yamc/safety/internal/sentinel"
	"example.invalid/yamc/safety/internal/workflow"
)

type phaseReport struct {
	Status            string             `json:"status"`
	SchemaVersion     string             `json:"schema_version"`
	SuiteID           string             `json:"suite_id"`
	Tier              string             `json:"tier"`
	EvidenceMode      string             `json:"evidence_mode"`
	InnerStatus       string             `json:"inner_status"`
	OuterSequence     []string           `json:"outer_sequence"`
	Verdict           string             `json:"verdict"`
	Claim             string             `json:"claim"`
	ArtifactKinds     []string           `json:"artifact_kinds"`
	ArtifactInstances int                `json:"artifact_instances"`
	ArtifactDigests   map[string]string  `json:"artifact_digests"`
	ManifestDigests   map[string]string  `json:"manifest_digests"`
	SurfaceEvidence   []phaseSurface     `json:"surface_evidence"`
	PolicyStatuses    []string           `json:"policy_statuses"`
	Operations        []any              `json:"operations"`
	CurrentHost       currentHostStatus  `json:"current_host"`
	ClaimBinding      *phaseClaimBinding `json:"claim_binding,omitempty"`
}

type phaseSurface struct {
	SurfaceID     string `json:"surface_id"`
	SurfaceDomain string `json:"surface_domain"`
	LogicalRef    string `json:"logical_ref"`
	Policy        string `json:"policy"`
	BeforeStatus  string `json:"before_status"`
	AfterStatus   string `json:"after_status"`
	BeforeToken   string `json:"before_token"`
	AfterToken    string `json:"after_token"`
}

type phaseClaimBinding struct {
	EvidenceDigest string                     `json:"evidence_digest"`
	SuiteDigest    string                     `json:"suite_digest"`
	ManifestDigest string                     `json:"manifest_digest"`
	Window         sentinel.ObservationWindow `json:"window"`
	WindowDigest   string                     `json:"window_digest"`
}

type currentHostStatus struct {
	Status        string `json:"status"`
	Verdict       string `json:"verdict"`
	Reason        string `json:"reason"`
	ClaimEligible bool   `json:"claim_eligible"`
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

func TestPhaseE2E(t *testing.T) {
	t.Run("reconstructs the exact seven-object graph and bounded report", testPhaseReportRoundTrip)
	t.Run("keeps current-host production proof fail-closed", testPhaseCurrentHostProofGate)
	t.Run("binds every locked decision to a named negative suite", testPhaseDecisionMatrix)
	t.Run("fixes phase order and compositional deadlines", testPhaseRunnerContract)
}

func testPhaseReportRoundTrip(t *testing.T) {
	safetyRoot, repositoryRoot := projectRoots(t)
	suitePath := filepath.Join(safetyRoot, "manifests", "offline-suite.v1.json")
	expectedPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "expected-report.json")
	if _, err := os.Stat(suitePath); errors.Is(err, os.ErrNotExist) {
		t.Fatal("EXPECTED_RED: phase-integration-behavior-missing")
	}
	if _, err := os.Stat(expectedPath); errors.Is(err, os.ErrNotExist) {
		t.Fatal("EXPECTED_RED: phase-integration-behavior-missing")
	}
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil || bytes.Contains(expectedBytes, []byte(`"verdict":"passed"`)) || bytes.Contains(expectedBytes, []byte(sentinel.ScopedUnchangedClaim)) || bytes.Contains(expectedBytes, []byte("hmac-sha256:")) {
		t.Fatal("checked-in report expectation contains claim evidence")
	}

	base := t.TempDir()
	root, err := fixture.Create(fixture.CreateOptions{
		Base:           base,
		RepositoryRoot: repositoryRoot,
		LogicalID:      "fixture:phase-e2e/run",
	})
	if err != nil {
		t.Fatal("isolated phase fixture unavailable")
	}
	physicalRoot := root.Paths().Root
	summary, err := workflow.RunSynthetic(workflow.Options{
		BlueprintPath:  filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "input.json"),
		SurfacesPath:   filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "protected-surfaces.json"),
		FixtureRoot:    physicalRoot,
		StoreRoot:      root.Paths().ArtifactStore,
		RepositoryRoot: repositoryRoot,
		Mode:           "synthetic",
	})
	if err != nil || summary.State != wantSuccessState || summary.ArtifactCount != 7 || summary.KindCount != 6 {
		t.Fatal("isolated synthetic workload did not produce the exact graph")
	}
	summaryData, err := json.Marshal(summary)
	if err != nil || os.WriteFile(filepath.Join(root.Paths().Temporary, "summary.json"), summaryData, 0o600) != nil {
		t.Fatal("isolated phase summary unavailable")
	}
	summaryPath := filepath.Join(root.Paths().Temporary, "summary.json")

	stdout, stderr, runErr := runCLI(safetyRoot,
		"report",
		"--suite", suitePath,
		"--expected", expectedPath,
		"--summary", summaryPath,
		"--store-root", root.Paths().ArtifactStore,
		"--repo-root", repositoryRoot,
	)
	if runErr != nil {
		t.Fatal("EXPECTED_RED: phase-integration-behavior-missing")
	}
	assertBoundedAndPrivate(t, stdout, stderr, repositoryRoot, physicalRoot)
	var report phaseReport
	decodeStrict(t, stdout, &report)
	assertPhaseReport(t, report, summary)
	replayBase, err := workflow.BuildPhaseReport(workflow.PhaseReportOptions{
		SuitePath: suitePath, ExpectedReportPath: expectedPath, SummaryPath: summaryPath,
		StoreRoot: root.Paths().ArtifactStore, RepositoryRoot: repositoryRoot,
	})
	if err != nil {
		t.Fatal("standalone report replay setup failed")
	}
	if _, _, err := workflow.BindPhaseReport(replayBase, &sentinel.Evidence{}, sentinel.Evaluation{}, nil); err == nil {
		t.Fatal("standalone report acquired a claim from replayable input")
	}

	if _, err := workflow.BuildPhaseReport(workflow.PhaseReportOptions{
		SuitePath:          suitePath,
		ExpectedReportPath: filepath.Join(safetyRoot, "manifests", "network-tests.v1.json"),
		SummaryPath:        summaryPath,
		StoreRoot:          root.Paths().ArtifactStore,
		RepositoryRoot:     repositoryRoot,
	}); err == nil {
		t.Fatal("substituted expected-report binding was accepted")
	}
	staleSummary := summary
	staleSummary.Artifacts = make(map[string]string, len(summary.Artifacts))
	for label, digest := range summary.Artifacts {
		staleSummary.Artifacts[label] = digest
	}
	staleSummary.Artifacts["readiness-report"] = "sha256:" + strings.Repeat("0", 64)
	staleData, err := json.Marshal(staleSummary)
	if err != nil || os.WriteFile(summaryPath, staleData, 0o600) != nil {
		t.Fatal("stale-lineage negative setup failed")
	}
	if _, err := workflow.BuildPhaseReport(workflow.PhaseReportOptions{
		SuitePath:          suitePath,
		ExpectedReportPath: expectedPath,
		SummaryPath:        summaryPath,
		StoreRoot:          root.Paths().ArtifactStore,
		RepositoryRoot:     repositoryRoot,
	}); err == nil {
		t.Fatal("stale report lineage was accepted")
	}

	frozen, err := fixture.FreezePrimary(fixture.VerdictPassed)
	if err != nil {
		t.Fatal("phase verdict did not freeze")
	}
	final := root.Retention().Finalize(frozen)
	if final.Verdict != fixture.VerdictPassed || final.Teardown.Status != fixture.TeardownRemoved {
		t.Fatal("marker-owned phase fixture was not removed after the frozen verdict")
	}
	if _, err := os.Lstat(physicalRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("marker-owned phase fixture remained after teardown")
	}
	if _, err := os.Stat(base); err != nil {
		t.Fatal("fixture teardown reached the external retention base")
	}
}

func assertPhaseReport(t *testing.T, report phaseReport, summary workflow.Summary) {
	t.Helper()
	wantKinds := []string{"applied-receipt", "desired-state", "generated-plan", "observed-state", "readiness-report", "verification-evidence"}
	if report.Status != "synthetic-report-claim-ineligible" || report.SchemaVersion != "1.0.0" || report.SuiteID != "phase-01-offline-safety-v1" || report.Tier != "offline-static" || report.EvidenceMode != "replay-claim-ineligible" {
		t.Fatal("phase report identity is not exact")
	}
	if report.InnerStatus != wantSuccessState || report.Verdict != "indeterminate" || report.Claim != "" || len(report.OuterSequence) != 0 || report.ClaimBinding != nil {
		t.Fatal("standalone phase report recovered a real claim or evidence binding")
	}
	if report.ArtifactInstances != 7 || !reflect.DeepEqual(report.ArtifactKinds, wantKinds) || !reflect.DeepEqual(report.ArtifactDigests, summary.Artifacts) || len(report.ManifestDigests) != 4 {
		t.Fatal("phase report did not reverse-bind the exact artifact and manifest digests")
	}
	if len(report.SurfaceEvidence) != 0 || !reflect.DeepEqual(report.PolicyStatuses, []string{"extra", "unmanaged-present"}) || len(report.Operations) != 0 {
		t.Fatal("standalone report copied surface proof or added convergence authority")
	}
	if report.CurrentHost.Status != "manual-required" || report.CurrentHost.Verdict != "indeterminate" || report.CurrentHost.Reason != "required-real-adapter-proof-unavailable" || report.CurrentHost.ClaimEligible {
		t.Fatal("isolated proof report overstates current-host readiness")
	}
	encoded, _ := json.Marshal(report)
	for _, forbidden := range []string{"/Users/", "whole-Mac-unchanged", "recovery-ready-on-current-host", "multi-host-verified", "fresh-install-verified", "effective_uid", "ownership_nonce", "resolver_mapping", "service_output", "raw_output", "hmac_key"} {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("phase report leaked process data or a stronger claim: %s", forbidden)
		}
	}
}

func testPhaseCurrentHostProofGate(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	stdout, stderr, err := runCLI(safetyRoot,
		"sentinel", "verify",
		"--mode", "real",
		"--manifest", filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json"),
		"--adapter-manifest", filepath.Join(safetyRoot, "manifests", "real-adapters.v1.json"),
	)
	if err == nil || !isGoRunExit(stderr, 32) {
		t.Fatal("current-host proof gate did not stop with manual-required")
	}
	var assessment sentinel.RealProofAssessment
	decodeStrict(t, stdout, &assessment)
	if assessment.Status != "manual-required" || assessment.Verdict != sentinel.VerdictIndeterminate || assessment.ExitCode != 32 || assessment.ClaimEligible || assessment.Reason != "required-real-adapter-proof-unavailable" {
		t.Fatal("current-host proof gate reached an adapter, workload, or claim")
	}
	if bytes.Contains(stdout, []byte(sentinel.ScopedUnchangedClaim)) {
		t.Fatal("manual-required current-host path emitted a scoped claim")
	}
}

func testPhaseDecisionMatrix(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	suitePath := filepath.Join(safetyRoot, "manifests", "offline-suite.v1.json")
	data, err := os.ReadFile(suitePath)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatal("EXPECTED_RED: phase-integration-behavior-missing")
	}
	var suite offlineSuite
	decodeStrict(t, data, &suite)
	if suite.SchemaVersion != "1.0.0" || suite.SuiteID != "phase-01-offline-safety-v1" || suite.Tier != "offline-static" || suite.EvidenceMode != "isolated-proof-double" || suite.ExpectedClaim != sentinel.ScopedUnchangedClaim || suite.CurrentHostGate != "manual-required" {
		t.Fatal("offline suite identity or claim ceiling changed")
	}
	wantOrder := []string{"wave:skeleton", "wave:artifact-contracts", "wave:privacy", "wave:fixture-policy", "wave:sentinels", "wave:controlplane", "task:phase-e2e"}
	if len(suite.TaskGroups) != 7 || !reflect.DeepEqual(suite.PhaseOrder, wantOrder) || len(suite.Manifests) != 4 {
		t.Fatal("offline suite component or manifest bindings are incomplete")
	}
	manifestIDs := make(map[string]string, len(suite.Manifests))
	for _, binding := range suite.Manifests {
		manifestIDs[binding.ID] = binding.Digest
	}
	if !reflect.DeepEqual(sortedKeys(manifestIDs), []string{"expected-report", "network-contract", "protected-surfaces", "real-adapters"}) {
		t.Fatal("offline suite manifest bindings changed")
	}
	if len(suite.NegativeMatrix) != 19 {
		t.Fatal("D-01..D-19 negative matrix is incomplete")
	}
	seen := make(map[string]struct{}, 19)
	for _, binding := range suite.NegativeMatrix {
		seen[binding.DecisionID] = struct{}{}
		if binding.TaskSuite == "" {
			t.Fatal("decision matrix contains an empty task suite")
		}
	}
	for index := 1; index <= 19; index++ {
		decisionID := "D-" + twoDigits(index)
		if _, ok := seen[decisionID]; !ok {
			t.Fatalf("decision matrix omitted %s", decisionID)
		}
	}
	runner, err := os.ReadFile(filepath.Join(safetyRoot, "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	for _, binding := range suite.NegativeMatrix {
		if !bytes.Contains(runner, []byte("task:"+binding.TaskSuite+")")) && binding.TaskSuite != "phase-e2e" {
			t.Fatalf("decision %s points to an undeclared task suite", binding.DecisionID)
		}
	}
}

func testPhaseRunnerContract(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	text := string(data)
	for _, required := range []string{
		"task:phase-e2e)",
		"phase:phase)",
		"'./internal/e2e'",
		"'^TestPhaseE2E$'",
		"'TestPhaseE2E'",
		"RUNNER_BUDGET_SECONDS=15",
		"RUNNER_BUDGET_SECONDS=47",
		"RUNNER_BUDGET_SECONDS=305",
		"remaining}\" -lt 47",
		"remaining}\" -lt 15",
		"runner-deadline-exceeded",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("phase runner literal missing: %s", required)
		}
	}
	if strings.Count(text, "task:phase-e2e)") != 1 || strings.Count(text, "phase:phase)") != 1 {
		t.Fatal("phase runner labels are not unique literals")
	}
	start := strings.Index(text, "run_phase_gate()")
	if start < 0 {
		t.Fatal("phase gate handler unavailable")
	}
	end := strings.Index(text[start:], "\n}\n")
	if end < 0 {
		t.Fatal("phase gate handler unavailable")
	}
	body := text[start : start+end]
	want := []string{"skeleton", "artifact-contracts", "privacy", "fixture-policy", "sentinels", "controlplane"}
	position := -1
	for _, wave := range want {
		next := strings.Index(body, "run_phase_wave_child "+wave)
		if next <= position {
			t.Fatal("phase component waves are missing or out of order")
		}
		position = next
	}
	finalTask := strings.Index(body, "run_phase_task_child phase-e2e")
	if finalTask <= position || strings.Count(body, "run_phase_wave_child ") != 6 || strings.Count(body, "run_phase_task_child ") != 1 {
		t.Fatal("phase gate child set is not exactly six waves plus phase-e2e")
	}
	for _, forbidden := range []string{"run_with_deadline", "run_with_runner_deadline", " launchctl ", " darwin-rebuild ", " brew ", " mise ", " uv ", " rustup ", " curl ", " eval "} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("phase gate contains a forbidden child or nested capability: %s", forbidden)
		}
	}
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return "" + string(rune('0'+value/10)) + string(rune('0'+value%10))
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
