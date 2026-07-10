package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const privacyRawMarker = "synthetic-raw-boundary-canary"

func TestPrivacyCLI(t *testing.T) {
	safetyRoot, repoRoot := projectRoots(t)
	externalRoot := t.TempDir()
	blueprintPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "input.json")
	surfacesPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "protected-surfaces.json")
	rawPath := filepath.Join(safetyRoot, "testdata", "raw", "fake-adapter.json")

	rawSample, err := os.ReadFile(rawPath)
	if err != nil || !bytes.Contains(rawSample, []byte(privacyRawMarker)) {
		t.Fatal("tracked synthetic raw sample unavailable")
	}
	stdout, stderr, err := runCLI(safetyRoot, managedFixtureArgs(blueprintPath, surfacesPath, externalRoot, "fixture:privacy-boundary/run", repoRoot, "synthetic", true)...)
	if err != nil {
		t.Fatal("privacy-boundary fixture run failed")
	}
	if len(stderr) != 0 || bytes.Contains(stdout, []byte(privacyRawMarker)) {
		t.Fatal("CLI exposed raw adapter data")
	}
	var output managedRunOutput
	decodeStrict(t, stdout, &output)
	summary := output.Summary
	if summary.State != wantSuccessState || summary.ArtifactCount != 7 || summary.KindCount != 6 {
		t.Fatal("privacy-boundary fixture summary is incomplete")
	}
	fixtureRoot := onlyFixtureChild(t, externalRoot)
	storeRoot := filepath.Join(fixtureRoot, "artifact-store")
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
	if !ok || len(fact) != 2 || fact["ref"] != "fixture:observed/shell-policy" || fact["state"] != "fixture:state/declared" {
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
	assertLiteralDispatchLabels(t, text)

	for _, testCase := range []struct {
		arguments       []string
		wantUnsupported bool
	}{
		{arguments: []string{"task", "never-registered-task"}, wantUnsupported: true},
		{arguments: []string{"wave", "never-registered-wave"}, wantUnsupported: true},
		{arguments: []string{"never-registered-scope"}},
		{arguments: []string{"phase", "unexpected-argument"}},
	} {
		arguments := testCase.arguments
		command := exec.Command("/bin/bash", append([]string{scriptPath}, arguments...)...)
		command.Env = os.Environ()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err == nil {
			t.Fatal("generic or malformed runner route succeeded")
		}
		if stdout.Len() > maxCLIOutput || stderr.Len() > maxCLIOutput {
			t.Fatal("rejected runner route exceeded bounded output")
		}
		combined := stdout.String() + stderr.String()
		if strings.Contains(combined, "expected-red-observed") || strings.Contains(combined, safetyRoot) {
			t.Fatal("generic runner rejection leaked state or satisfied RED")
		}
		if testCase.wantUnsupported {
			if !strings.Contains(combined, `"status":"harness-error"`) || !strings.Contains(combined, `"reason":"unsupported-suite"`) {
				t.Fatal("generic task or wave rejection changed contract")
			}
		} else if !strings.Contains(combined, "usage:") {
			t.Fatal("malformed scope or phase arguments bypassed usage rejection")
		}
	}
}

func assertLiteralDispatchLabels(t *testing.T, script string) {
	t.Helper()
	const marker = `case "${SCOPE}:${SUITE}" in`
	start := strings.Index(script, marker)
	if start < 0 {
		t.Fatal("runner dispatch case is missing")
	}
	dispatch := script[start+len(marker):]
	end := strings.Index(dispatch, "\nesac")
	if end < 0 {
		t.Fatal("runner dispatch case is unterminated")
	}
	dispatch = dispatch[:end]
	literalPattern := regexp.MustCompile(`^(?:(?:task|wave):[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|phase)\)$`)
	counts := make(map[string]int)
	defaultCount := 0
	for _, line := range strings.Split(dispatch, "\n") {
		label := strings.TrimSpace(line)
		if !strings.HasSuffix(label, ")") {
			continue
		}
		if label == "*)" {
			defaultCount++
			continue
		}
		if !literalPattern.MatchString(label) || strings.ContainsAny(label, `"'$`+"`"+`{}[]?\\`) || strings.Contains(label, "|") || strings.Contains(label, "$(") {
			t.Fatal("runner dispatch label is not one complete closed literal")
		}
		counts[label]++
		if counts[label] != 1 {
			t.Fatal("runner dispatch contains a duplicate literal label")
		}
	}
	if defaultCount != 1 {
		t.Fatal("runner dispatch must contain one default rejection")
	}
	for _, required := range []string{"task:privacy-boundary)", "task:bounded-capture)", "wave:privacy)"} {
		if counts[required] != 1 {
			t.Fatal("Phase 3 runner label is missing or not unique")
		}
	}
}
