package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const (
	wantSuccessState = "synthetic-sentinel-passed"
	maxCLIOutput     = 64 << 10
)

var forbiddenSyntheticClaims = []string{
	"covered-surfaces-unchanged-for-run",
	"whole-Mac-unchanged",
	"recovery-ready-on-current-host",
	"multi-host-verified",
	"fresh-install-verified",
}

type runSummary struct {
	State          string            `json:"state"`
	ArtifactCount  int               `json:"artifact_count"`
	KindCount      int               `json:"kind_count"`
	ManifestDigest string            `json:"manifest_digest"`
	Artifacts      map[string]string `json:"artifacts"`
}

type envelope struct {
	Kind          string          `json:"kind"`
	SchemaVersion string          `json:"schema_version"`
	Run           json.RawMessage `json:"run"`
	Producer      json.RawMessage `json:"producer"`
	Provenance    json.RawMessage `json:"provenance"`
	Payload       json.RawMessage `json:"payload"`
	ContentDigest string          `json:"content_digest"`
}

func TestWalkingSkeletonContract(t *testing.T) {
	safetyRoot, repoRoot := projectRoots(t)
	externalBase := t.TempDir()
	fixtureRoot := filepath.Join(externalBase, "fixture")
	storeRoot := filepath.Join(externalBase, "store")
	blueprintPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "input.json")
	surfacesPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "protected-surfaces.json")

	for _, path := range []string{blueprintPath, surfacesPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("SETUP_FAILURE: tracked synthetic input unavailable")
		}
	}

	stdout, stderr, err := runCLI(safetyRoot,
		"fixture", "run",
		"--blueprint", blueprintPath,
		"--surfaces", surfacesPath,
		"--fixture-root", fixtureRoot,
		"--store-root", storeRoot,
		"--repo-root", repoRoot,
		"--mode", "synthetic",
	)
	if err != nil {
		entrypoint := filepath.Join(safetyRoot, "cmd", "yamc-safety", "main.go")
		if errors.Is(statError(entrypoint), os.ErrNotExist) {
			t.Fatalf("EXPECTED_RED: round-trip-capability-missing")
		}
		t.Fatalf("TOOLCHAIN_FAILURE: implemented round trip exited non-zero")
	}

	assertBoundedAndPrivate(t, stdout, stderr, repoRoot, externalBase)
	var summary runSummary
	decodeStrict(t, stdout, &summary)
	if summary.State != wantSuccessState {
		t.Fatalf("unexpected synthetic success state")
	}
	if summary.ArtifactCount != 7 || summary.KindCount != 6 || len(summary.Artifacts) != 7 {
		t.Fatalf("expected seven artifacts across exactly six distinct kinds")
	}
	if summary.ManifestDigest == "" {
		t.Fatalf("missing protected-surface manifest digest")
	}
	assertNoOverclaim(t, stdout)

	artifacts := readStoredArtifacts(t, storeRoot)
	assertSixKindsAndStoreKeys(t, artifacts, summary)
	assertExactLineage(t, artifacts, summary, summary.ManifestDigest)
	assertNegativeRoutes(t, safetyRoot, repoRoot, blueprintPath, surfacesPath, externalBase)
}

func projectRoots(t *testing.T) (string, string) {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("SETUP_FAILURE: caller path unavailable")
	}
	safetyRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	repoRoot := filepath.Dir(safetyRoot)
	return safetyRoot, repoRoot
}

func statError(path string) error {
	_, err := os.Stat(path)
	return err
}

func runCLI(safetyRoot string, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command("go", append([]string{"run", "./cmd/yamc-safety"}, args...)...)
	cmd.Dir = safetyRoot
	cmd.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if stdout.Len() > maxCLIOutput || stderr.Len() > maxCLIOutput {
		return nil, nil, fmt.Errorf("bounded output exceeded")
	}
	return stdout.Bytes(), stderr.Bytes(), err
}

func assertBoundedAndPrivate(t *testing.T, stdout, stderr []byte, repoRoot, externalRoot string) {
	t.Helper()
	combined := string(append(append([]byte{}, stdout...), stderr...))
	for _, forbidden := range []string{repoRoot, externalRoot} {
		if forbidden != "" && strings.Contains(combined, forbidden) {
			t.Fatalf("physical root leaked to output")
		}
	}
}

func assertNoOverclaim(t *testing.T, data []byte) {
	t.Helper()
	for _, claim := range forbiddenSyntheticClaims {
		if strings.Contains(string(data), claim) {
			t.Fatalf("OVERCLAIM_ACCEPTED: synthetic output contained a real-machine claim")
		}
	}
}

func decodeStrict(t *testing.T, data []byte, target any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("invalid bounded JSON output")
	}
	if decoder.Decode(&struct{}{}) == nil {
		t.Fatalf("multiple JSON values in output")
	}
}

func readStoredArtifacts(t *testing.T, storeRoot string) map[string]envelope {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(storeRoot, "sha256"))
	if err != nil {
		t.Fatalf("artifact store unavailable")
	}
	if len(entries) != 7 {
		t.Fatalf("expected seven content-addressed objects")
	}
	result := make(map[string]envelope, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("unexpected directory in artifact object set")
		}
		data, err := os.ReadFile(filepath.Join(storeRoot, "sha256", entry.Name()))
		if err != nil {
			t.Fatalf("stored artifact unreadable")
		}
		var artifact envelope
		decodeStrict(t, data, &artifact)
		if artifact.ContentDigest != "sha256:"+entry.Name() {
			t.Fatalf("store key does not equal content digest")
		}
		if _, err := hex.DecodeString(entry.Name()); err != nil || len(entry.Name()) != sha256.Size*2 {
			t.Fatalf("invalid digest-addressed store key")
		}
		if _, exists := result[artifact.ContentDigest]; exists {
			t.Fatalf("duplicate content digest collapsed a stored artifact")
		}
		result[artifact.ContentDigest] = artifact
	}
	return result
}

func assertSixKindsAndStoreKeys(t *testing.T, artifacts map[string]envelope, summary runSummary) {
	t.Helper()
	wantSummaryKeys := []string{
		"applied-receipt",
		"desired-state",
		"fresh-observed-state",
		"generated-plan",
		"observed-state",
		"readiness-report",
		"verification-evidence",
	}
	sort.Strings(wantSummaryKeys)
	gotSummaryKeys := make([]string, 0, len(summary.Artifacts))
	seenDigests := make(map[string]struct{}, len(summary.Artifacts))
	for label, digest := range summary.Artifacts {
		gotSummaryKeys = append(gotSummaryKeys, label)
		artifact, exists := artifacts[digest]
		if !exists {
			t.Fatalf("summary references an artifact absent from the exact store key")
		}
		if artifact.SchemaVersion != "1.0.0" {
			t.Fatalf("unexpected schema version")
		}
		if digest != artifact.ContentDigest {
			t.Fatalf("summary digest does not match stored artifact")
		}
		if _, duplicate := seenDigests[digest]; duplicate {
			t.Fatalf("two summary keys alias the same stored artifact")
		}
		seenDigests[digest] = struct{}{}
	}
	sort.Strings(gotSummaryKeys)
	if strings.Join(gotSummaryKeys, "\n") != strings.Join(wantSummaryKeys, "\n") || len(seenDigests) != len(artifacts) {
		t.Fatalf("summary artifact keys are incomplete or non-exact")
	}

	wantKindCounts := map[string]int{
		"desired-state":         1,
		"observed-state":        2,
		"generated-plan":        1,
		"applied-receipt":       1,
		"verification-evidence": 1,
		"readiness-report":      1,
	}
	gotKindCounts := make(map[string]int, len(wantKindCounts))
	for _, artifact := range artifacts {
		gotKindCounts[artifact.Kind]++
	}
	if len(gotKindCounts) != len(wantKindCounts) {
		t.Fatalf("artifact kind registry is incomplete or open")
	}
	for kind, wantCount := range wantKindCounts {
		if gotKindCounts[kind] != wantCount {
			t.Fatalf("artifact kind multiplicity is invalid")
		}
	}
}

func assertExactLineage(t *testing.T, artifacts map[string]envelope, summary runSummary, manifestDigest string) {
	t.Helper()
	desired := summaryArtifact(t, artifacts, summary, "desired-state")
	plan := summaryArtifact(t, artifacts, summary, "generated-plan")
	receipt := summaryArtifact(t, artifacts, summary, "applied-receipt")
	evidence := summaryArtifact(t, artifacts, summary, "verification-evidence")
	report := summaryArtifact(t, artifacts, summary, "readiness-report")

	planPayload := payloadMap(t, plan)
	observedDigest, ok := planPayload["observed_digest"].(string)
	observed, observedExists := artifacts[observedDigest]
	if planPayload["desired_digest"] != desired.ContentDigest || !ok || !observedExists || observed.Kind != "observed-state" || summary.Artifacts["observed-state"] != observedDigest {
		t.Fatalf("plan does not bind exact desired and observed digests")
	}
	expectedPostconditionsDigest, ok := planPayload["expected_postconditions_digest"].(string)
	if !ok || expectedPostconditionsDigest == "" {
		t.Fatalf("plan does not bind expected postconditions")
	}

	receiptPayload := payloadMap(t, receipt)
	if receiptPayload["plan_digest"] != plan.ContentDigest || receiptPayload["mode"] != "synthetic" {
		t.Fatalf("synthetic receipt lineage is invalid")
	}

	evidencePayload := payloadMap(t, evidence)
	if evidencePayload["plan_digest"] != plan.ContentDigest ||
		evidencePayload["receipt_digest"] != receipt.ContentDigest ||
		evidencePayload["expected_postconditions_digest"] != expectedPostconditionsDigest ||
		evidencePayload["manifest_digest"] != manifestDigest {
		t.Fatalf("verification evidence lineage is incomplete")
	}
	freshDigest, ok := evidencePayload["fresh_observed_digest"].(string)
	freshObservedArtifact, freshExists := artifacts[freshDigest]
	if !ok || !freshExists || freshObservedArtifact.Kind != "observed-state" || summary.Artifacts["fresh-observed-state"] != freshDigest || freshDigest == observed.ContentDigest {
		t.Fatalf("evidence does not bind the separately stored fresh observation")
	}
	freshObserved, ok := evidencePayload["fresh_observed"].(map[string]any)
	if !ok || freshObserved["content_digest"] != freshDigest {
		t.Fatalf("fresh observation descriptor does not reference the stored object")
	}
	if len(freshObserved) != 4 {
		t.Fatalf("embedded fresh observation descriptor is not reference-only metadata")
	}
	for key := range freshObserved {
		switch key {
		case "scope", "state", "source_receipt_digest", "content_digest":
		default:
			t.Fatalf("embedded fresh observation descriptor contains payload data")
		}
	}
	if freshObserved["source_receipt_digest"] != receipt.ContentDigest {
		t.Fatalf("fresh observation descriptor does not reference the exact receipt")
	}
	if scope, ok := freshObserved["scope"].(string); !ok || scope == "" {
		t.Fatalf("fresh observation descriptor is missing its logical scope")
	}
	if state, ok := freshObserved["state"].(string); !ok || state == "" {
		t.Fatalf("fresh observation descriptor is missing its normalized state")
	}
	freshProvenance := provenanceMap(t, freshObservedArtifact)
	inputs, ok := freshProvenance["input_digests"].([]any)
	if !ok || len(inputs) != 1 || inputs[0] != receipt.ContentDigest {
		t.Fatalf("fresh observed artifact provenance does not bind exactly the receipt")
	}
	if evidencePayload["sentinel_before_digest"] != evidencePayload["sentinel_after_digest"] {
		t.Fatalf("synthetic protected surface changed during the window")
	}

	reportPayload := payloadMap(t, report)
	if reportPayload["evidence_digest"] != evidence.ContentDigest || reportPayload["state"] != wantSuccessState {
		t.Fatalf("readiness report does not bind exact evidence")
	}
}

func payloadMap(t *testing.T, artifact envelope) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(artifact.Payload, &payload); err != nil {
		t.Fatalf("artifact payload is not structured JSON")
	}
	return payload
}

func provenanceMap(t *testing.T, artifact envelope) map[string]any {
	t.Helper()
	var provenance map[string]any
	if err := json.Unmarshal(artifact.Provenance, &provenance); err != nil {
		t.Fatalf("artifact provenance is not structured JSON")
	}
	return provenance
}

func summaryArtifact(t *testing.T, artifacts map[string]envelope, summary runSummary, label string) envelope {
	t.Helper()
	digest, ok := summary.Artifacts[label]
	if !ok || digest == "" {
		t.Fatalf("summary artifact key is missing")
	}
	artifact, ok := artifacts[digest]
	if !ok {
		t.Fatalf("summary artifact digest is absent from the store")
	}
	return artifact
}

func assertNegativeRoutes(t *testing.T, safetyRoot, repoRoot, blueprintPath, surfacesPath, externalBase string) {
	t.Helper()
	tests := []struct {
		name      string
		args      []string
		forbidden string
	}{
		{
			name:      "store-inside-repository",
			args:      fixtureArgs(blueprintPath, surfacesPath, filepath.Join(externalBase, "negative-fixture-1"), filepath.Join(safetyRoot, ".forbidden-store"), repoRoot, "synthetic"),
			forbidden: filepath.Join(safetyRoot, ".forbidden-store"),
		},
		{
			name:      "fixture-inside-repository",
			args:      fixtureArgs(blueprintPath, surfacesPath, filepath.Join(safetyRoot, ".forbidden-fixture"), filepath.Join(externalBase, "negative-store-2"), repoRoot, "synthetic"),
			forbidden: filepath.Join(safetyRoot, ".forbidden-fixture"),
		},
		{
			name:      "path-traversal",
			args:      fixtureArgs(blueprintPath, surfacesPath, filepath.Join(externalBase, "negative-fixture-3"), externalBase+string(filepath.Separator)+"store"+string(filepath.Separator)+".."+string(filepath.Separator)+"escape", repoRoot, "synthetic"),
			forbidden: filepath.Join(externalBase, "escape"),
		},
		{
			name:      "unsupported-mode",
			args:      fixtureArgs(blueprintPath, surfacesPath, filepath.Join(externalBase, "negative-fixture-4"), filepath.Join(externalBase, "negative-store-4"), repoRoot, "real"),
			forbidden: filepath.Join(externalBase, "negative-store-4"),
		},
		{
			name: "unsupported-command",
			args: []string{"apply", "run"},
		},
	}

	for _, claim := range forbiddenSyntheticClaims {
		tests = append(tests, struct {
			name      string
			args      []string
			forbidden string
		}{
			name:      "reject-overclaim-" + strings.ReplaceAll(claim, "-", "_"),
			args:      append(fixtureArgs(blueprintPath, surfacesPath, filepath.Join(externalBase, "claim-fixture"), filepath.Join(externalBase, "claim-store"), repoRoot, "synthetic"), "--claim", claim),
			forbidden: filepath.Join(externalBase, "claim-store"),
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(safetyRoot, tc.args...)
			if err == nil {
				t.Fatalf("negative route unexpectedly succeeded")
			}
			assertBoundedAndPrivate(t, stdout, stderr, repoRoot, externalBase)
			if tc.forbidden != "" {
				if _, statErr := os.Lstat(tc.forbidden); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("rejected route wrote before validation")
				}
			}
		})
	}
}

func fixtureArgs(blueprintPath, surfacesPath, fixtureRoot, storeRoot, repoRoot, mode string) []string {
	return []string{
		"fixture", "run",
		"--blueprint", blueprintPath,
		"--surfaces", surfacesPath,
		"--fixture-root", fixtureRoot,
		"--store-root", storeRoot,
		"--repo-root", repoRoot,
		"--mode", mode,
	}
}
