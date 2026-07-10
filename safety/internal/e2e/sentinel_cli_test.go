package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.invalid/yamc/safety/internal/sentinel"
)

type sentinelEvaluationOutput struct {
	Status     string              `json:"status"`
	Evaluation sentinel.Evaluation `json:"evaluation"`
}

func TestSentinelCLI(t *testing.T) {
	t.Run("returns synthetic pass without real claim", testSyntheticVerdictCLI)
	t.Run("returns exact non-pass exits", testSentinelNonPassCLI)
	t.Run("rejects claim selection and crafted provenance", testSentinelClaimCLI)
	t.Run("binds exact verdict runner pairs", testSentinelVerdictRunner)
}

func testSyntheticVerdictCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	manifest, evidence := cliSentinelEvidence(t, safetyRoot)
	evidencePath := writeEvidence(t, evidence)
	stdout, stderr, err := runCLI(safetyRoot, "sentinel", "evaluate", "--manifest", filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json"), "--evidence", evidencePath)
	if err != nil || len(stderr) != 0 {
		t.Fatal("synthetic verdict CLI failed")
	}
	output := decodeSentinelOutput(t, stdout)
	if output.Status != "synthetic-sentinel-passed" || output.Evaluation.Verdict != sentinel.VerdictPassed || output.Evaluation.ExitCode != 0 || output.Evaluation.Claim != "" || output.Evaluation.EvidenceDigest == "" {
		t.Fatal("synthetic verdict CLI overclaimed or changed")
	}
	if bytes.Contains(stdout, []byte(evidencePath)) || bytes.Contains(stdout, []byte(filepath.Dir(evidencePath))) || bytes.Contains(stdout, []byte(sentinel.ScopedUnchangedClaim)) {
		t.Fatal("synthetic verdict CLI leaked a path or real claim")
	}
	if result := sentinel.Evaluate(manifest, evidence); result.Verdict != sentinel.VerdictPassed {
		t.Fatal("CLI fixture evidence was not valid")
	}
}

func testSentinelNonPassCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	_, evidence := cliSentinelEvidence(t, safetyRoot)
	manifestPath := filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json")
	cases := []struct {
		name       string
		mutate     func(*sentinel.Evidence)
		exit       int
		verdict    sentinel.Verdict
		changeCode string
	}{
		{name: "violation", exit: sentinel.ExitViolation, verdict: sentinel.VerdictViolation, changeCode: sentinel.ChangeDetectedCode, mutate: func(value *sentinel.Evidence) {
			value.Surfaces[0].AfterToken = "hmac-sha256:" + strings.Repeat("a", 64)
		}},
		{name: "indeterminate", exit: sentinel.ExitIndeterminate, verdict: sentinel.VerdictIndeterminate, mutate: func(value *sentinel.Evidence) {
			value.Surfaces[0].AfterStatus = sentinel.ObservationIncomplete
			value.Surfaces[0].AfterToken = ""
			value.Surfaces[0].AfterReason = sentinel.ReasonUnreadable
		}},
		{name: "harness", exit: sentinel.ExitHarnessError, verdict: sentinel.VerdictHarnessError, mutate: func(value *sentinel.Evidence) {
			value.ManifestDigest = sentinel.SuiteDigest("substituted", "offline-static")
		}},
	}
	for _, testCase := range cases {
		candidate := evidence
		candidate.Surfaces = append([]sentinel.SurfaceEvidence(nil), evidence.Surfaces...)
		testCase.mutate(&candidate)
		stdout, stderr, err := runCLI(safetyRoot, "sentinel", "evaluate", "--manifest", manifestPath, "--evidence", writeEvidence(t, candidate))
		if err == nil || !isGoRunExit(stderr, testCase.exit) {
			t.Fatalf("%s verdict did not use exact non-zero exit", testCase.name)
		}
		output := decodeSentinelOutput(t, stdout)
		if output.Evaluation.Verdict != testCase.verdict || output.Evaluation.ExitCode != testCase.exit || output.Evaluation.Claim != "" || output.Evaluation.ChangeCode != testCase.changeCode {
			t.Fatalf("%s verdict output changed", testCase.name)
		}
		if testCase.verdict == sentinel.VerdictViolation {
			for _, forbidden := range []string{"restore", "retry", "ignore", "attribution"} {
				if bytes.Contains(stdout, []byte(forbidden)) {
					t.Fatalf("violation output attributed or mutated state: %s", forbidden)
				}
			}
		}
	}
}

func testSentinelClaimCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	_, evidence := cliSentinelEvidence(t, safetyRoot)
	manifestPath := filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json")
	for _, claim := range []string{sentinel.ScopedUnchangedClaim, "whole-Mac-unchanged", "recovery-ready-on-current-host", "multi-host-verified", "fresh-install-verified"} {
		stdout, stderr, err := runCLI(safetyRoot, "sentinel", "evaluate", "--manifest", manifestPath, "--evidence", writeEvidence(t, evidence), "--claim", claim)
		if err == nil || !isGoRunExit(stderr, sentinel.ExitHarnessError) || bytes.Contains(stdout, []byte(claim)) {
			t.Fatalf("synthetic CLI selected forbidden claim: %s", claim)
		}
		output := decodeSentinelOutput(t, stdout)
		if output.Evaluation.Verdict != sentinel.VerdictHarnessError || output.Evaluation.Claim != "" {
			t.Fatal("claim rejection did not fail closed")
		}
	}
	crafted := evidence
	crafted.Provenance = "real"
	stdout, stderr, err := runCLI(safetyRoot, "sentinel", "evaluate", "--manifest", manifestPath, "--evidence", writeEvidence(t, crafted))
	if err == nil || !isGoRunExit(stderr, sentinel.ExitIndeterminate) {
		t.Fatal("crafted real provenance returned zero")
	}
	output := decodeSentinelOutput(t, stdout)
	if output.Evaluation.Reason != "real-envelope-binding-missing" || output.Evaluation.Claim != "" {
		t.Fatal("crafted real provenance acquired claim capability")
	}
}

func testSentinelVerdictRunner(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	text := string(data)
	for _, required := range []string{"'./internal/sentinel'", "'^TestSentinelVerdicts$'", "'TestSentinelVerdicts'", "'./internal/e2e'", "'^TestSentinelCLI$'", "'TestSentinelCLI'", "task:sentinel-verdicts)"} {
		if !strings.Contains(text, required) {
			t.Fatalf("sentinel verdict runner literal missing: %s", required)
		}
	}
	if strings.Count(text, "task:sentinel-verdicts)") != 1 {
		t.Fatal("sentinel verdict runner label is not unique")
	}
}

func cliSentinelEvidence(t *testing.T, safetyRoot string) (sentinel.ProtectedManifest, sentinel.Evidence) {
	t.Helper()
	manifestData, err := os.ReadFile(filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json"))
	if err != nil {
		t.Fatal("protected manifest unavailable")
	}
	manifest, err := sentinel.ParseProtectedManifest(manifestData)
	if err != nil {
		t.Fatal("protected manifest rejected")
	}
	root := t.TempDir()
	resolver, err := sentinel.PrepareProtectedSynthetic(root)
	if err != nil {
		t.Fatal("synthetic sentinel fixture unavailable")
	}
	frozen, err := sentinel.FreezeProtectedManifest(manifest)
	if err != nil {
		t.Fatal("protected manifest did not freeze")
	}
	key := bytes.Repeat([]byte{0x27}, 32)
	before, err := sentinel.SnapshotProtected(frozen, manifest, resolver, key, sentinel.SnapshotOptions{})
	if err != nil {
		t.Fatal("before snapshot failed")
	}
	after, err := sentinel.SnapshotProtected(frozen, manifest, resolver, key, sentinel.SnapshotOptions{})
	if err != nil {
		t.Fatal("after snapshot failed")
	}
	opened := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	evidence, err := sentinel.BuildEvidence(manifest, before, after, sentinel.EvidenceOptions{SuiteID: manifest.SuiteID, Tier: "offline-static", WindowID: "synthetic-window-cli", OpenedAt: opened, ClosedAt: opened.Add(time.Second), Provenance: "synthetic"})
	if err != nil {
		t.Fatal("synthetic evidence build failed")
	}
	return manifest, evidence
}

func writeEvidence(t *testing.T, evidence sentinel.Evidence) string {
	t.Helper()
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal("evidence encode failed")
	}
	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal("evidence write failed")
	}
	return path
}

func decodeSentinelOutput(t *testing.T, data []byte) sentinelEvaluationOutput {
	t.Helper()
	var output sentinelEvaluationOutput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatal("sentinel CLI output is invalid")
	}
	return output
}
