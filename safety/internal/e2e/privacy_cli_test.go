package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const privacyRawMarker = "synthetic-raw-boundary-canary"

func TestPrivacyCLI(t *testing.T) {
	safetyRoot, repoRoot := projectRoots(t)
	externalRoot := t.TempDir()
	fixtureRoot := filepath.Join(externalRoot, "fixture")
	storeRoot := filepath.Join(externalRoot, "store")
	blueprintPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "input.json")
	surfacesPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "protected-surfaces.json")
	rawPath := filepath.Join(safetyRoot, "testdata", "raw", "fake-adapter.json")

	rawSample, err := os.ReadFile(rawPath)
	if err != nil || !bytes.Contains(rawSample, []byte(privacyRawMarker)) {
		t.Fatal("tracked synthetic raw sample unavailable")
	}
	stdout, stderr, err := runCLI(safetyRoot, fixtureArgs(blueprintPath, surfacesPath, fixtureRoot, storeRoot, repoRoot, "synthetic")...)
	if err != nil {
		t.Fatal("privacy-boundary fixture run failed")
	}
	if len(stderr) != 0 || bytes.Contains(stdout, []byte(privacyRawMarker)) {
		t.Fatal("CLI exposed raw adapter data")
	}
	var summary runSummary
	decodeStrict(t, stdout, &summary)
	if summary.State != wantSuccessState || summary.ArtifactCount != 7 || summary.KindCount != 6 {
		t.Fatal("privacy-boundary fixture summary is incomplete")
	}
	artifacts := readStoredArtifacts(t, storeRoot)
	assertNormalizedFreshObservation(t, artifacts, summary)
	assertNoRawRetention(t, fixtureRoot, storeRoot)
	assertPrivacyRunnerContract(t, safetyRoot)
}

func assertNormalizedFreshObservation(t *testing.T, artifacts map[string]envelope, summary runSummary) {
	t.Helper()
	fresh := summaryArtifact(t, artifacts, summary, freshObservedKey)
	payload := payloadMap(t, fresh)
	facts, ok := payload["facts"].([]any)
	if !ok || len(facts) != 1 {
		t.Fatal("fresh observation does not contain one normalized fact")
	}
	fact, ok := facts[0].(map[string]any)
	if !ok || len(fact) != 2 || fact["ref"] != "fixture:observed/shell-policy" || fact["state"] != "declared" {
		t.Fatal("fresh observation contains non-normalized adapter data")
	}
	encoded, err := json.Marshal(payload)
	if err != nil || bytes.Contains(encoded, []byte(privacyRawMarker)) {
		t.Fatal("fresh observation retained transport data")
	}
}

func assertNoRawRetention(t *testing.T, fixtureRoot, storeRoot string) {
	t.Helper()
	adapterPath := filepath.Join(fixtureRoot, "path", "bin", "yamc-fixture-adapter-v1")
	info, err := os.Lstat(adapterPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o100 == 0 {
		t.Fatal("fixture adapter executable was not materialized in fixture:path/bin")
	}
	for _, root := range []string{fixtureRoot, storeRoot} {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			name := strings.ToLower(entry.Name())
			if strings.Contains(name, "stdout") || strings.Contains(name, "stderr") || strings.HasSuffix(name, ".log") || strings.Contains(name, "raw-output") {
				return errors.New("raw retention path found")
			}
			if !entry.Type().IsRegular() || path == adapterPath {
				return nil
			}
			info, err := entry.Info()
			if err != nil || info.Size() > 1<<20 {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if bytes.Contains(data, []byte(privacyRawMarker)) {
				return errors.New("raw transport data retained")
			}
			return nil
		}); err != nil {
			t.Fatal("fixture or canonical store retained raw adapter data")
		}
	}
}

func assertPrivacyRunnerContract(t *testing.T, safetyRoot string) {
	t.Helper()
	scriptPath := filepath.Join(safetyRoot, "scripts", "test.sh")
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal("privacy runner unavailable")
	}
	text := string(script)
	for _, required := range []string{
		"'./internal/privacy'", "'^TestPrivacyBoundary$'", "'^TestBoundedCapture$'",
		"'./internal/e2e'", "'^TestPrivacyCLI$'", "run_privacy_wave", "test-selection-not-exact",
	} {
		if !strings.Contains(text, required) {
			t.Fatal("privacy runner package or pattern pair is not fixed")
		}
	}
	for _, future := range []string{"task:fixture-lifecycle", "task:tier-network-policy", "wave:fixture-policy", "task:sentinel-manifest"} {
		if strings.Contains(text, future) {
			t.Fatal("Phase 4+ runner route registered early")
		}
	}
	for _, arguments := range [][]string{
		{},
		{"task", "unknown-suite"},
		{"task", "fixture-lifecycle"},
		{"wave", "fixture-policy"},
		{"phase"},
	} {
		command := exec.Command("/bin/bash", append([]string{scriptPath}, arguments...)...)
		command.Env = os.Environ()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err == nil {
			t.Fatal("unknown or future runner route succeeded")
		}
		if stdout.Len() > maxCLIOutput || stderr.Len() > maxCLIOutput {
			t.Fatal("rejected runner route exceeded bounded output")
		}
	}
}
