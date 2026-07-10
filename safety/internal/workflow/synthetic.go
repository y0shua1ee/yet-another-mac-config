package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"example.invalid/yamc/safety/internal/artifact"
	"example.invalid/yamc/safety/internal/privacy"
	"example.invalid/yamc/safety/internal/sentinel"
)

const successState = "synthetic-sentinel-passed"

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
	}{desired.envelope.ContentDigest, observed.envelope.ContentDigest, expectedDigest, []string{input.OperationID}})
	if err != nil {
		return Summary{}, err
	}
	receipt, err := makeArtifact(artifact.AppliedReceipt, run, []string{plan.envelope.ContentDigest}, struct {
		PlanDigest   string   `json:"plan_digest"`
		Mode         string   `json:"mode"`
		OperationIDs []string `json:"operation_ids"`
		Outcome      string   `json:"outcome"`
	}{plan.envelope.ContentDigest, "synthetic", []string{input.OperationID}, "fixture:outcome/completed"})
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
