package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type policyDecision struct {
	Status            string `json:"status"`
	Tier              string `json:"tier"`
	NetworkPolicy     string `json:"network_policy"`
	Reason            string `json:"reason"`
	TestID            string `json:"test_id,omitempty"`
	ProbeID           string `json:"probe_id,omitempty"`
	ContractValidated bool   `json:"contract_validated"`
}

func TestTierCLI(t *testing.T) {
	t.Run("defaults offline and keeps isolated integration offline", testTierDefaultsCLI)
	t.Run("validates exact network ID but returns manual-required without egress", testExactNetworkCLI)
	t.Run("keeps live-check bounded unknown with no execution", testLiveCheckCLI)
	t.Run("rejects broad partial and injection-shaped policy options", testPolicyOptionDenials)
	t.Run("keeps runner dispatch closed under generic and injection-shaped argv", testClosedRunnerDispatch)
	t.Run("binds exact task pairs and fresh-root wave aggregation", testTierRunnerContract)
}

func testTierDefaultsCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	for _, testCase := range []struct {
		arguments []string
		tier      string
		reason    string
	}{
		{arguments: []string{"test-policy"}, tier: "offline-static", reason: "offline-default"},
		{arguments: []string{"test-policy", "--tier", "offline-static"}, tier: "offline-static", reason: "offline-default"},
		{arguments: []string{"test-policy", "--tier", "isolated-integration"}, tier: "isolated-integration", reason: "isolated-offline"},
	} {
		stdout, stderr, err := runCLI(safetyRoot, testCase.arguments...)
		if err != nil || len(stderr) != 0 {
			t.Fatal("offline policy status failed")
		}
		decision := decodePolicyDecision(t, stdout)
		if decision.Status != "ready" || decision.Tier != testCase.tier || decision.NetworkPolicy != "denied" || decision.Reason != testCase.reason || decision.ContractValidated {
			t.Fatal("default or isolated policy escalated capability")
		}
	}
}

func testExactNetworkCLI(t *testing.T) {
	safetyRoot, repositoryRoot := projectRoots(t)
	manifestPath := filepath.Join(safetyRoot, "manifests", "network-tests.v1.json")
	arguments := []string{
		"test-policy",
		"--tier", "isolated-integration",
		"--network-manifest", manifestPath,
		"--repo-root", repositoryRoot,
		"--allow-network-test", "fixture.download.synthetic-archive.v1",
	}
	stdout, stderr, err := runCLI(safetyRoot, arguments...)
	if err == nil || !isGoRunExit(stderr, 32) {
		t.Fatal("validated network contract did not remain non-executing manual-required")
	}
	decision := decodePolicyDecision(t, stdout)
	if decision.Status != "manual-required" || decision.Tier != "isolated-integration" || decision.NetworkPolicy != "denied" || decision.Reason != "network-execution-unavailable-phase-1" || !decision.ContractValidated || decision.TestID != "fixture.download.synthetic-archive.v1" {
		t.Fatal("exact network decision changed")
	}
	if bytes.Contains(stdout, []byte(manifestPath)) || bytes.Contains(stderr, []byte(manifestPath)) {
		t.Fatal("network policy rendered a physical manifest path")
	}

	environment := withoutPolicyEnvironment(os.Environ())
	environment = append(environment, "HTTPS_PROXY=http://synthetic.invalid")
	stdout, stderr, err = runPolicyCommand(safetyRoot, environment, arguments...)
	if err == nil || !isGoRunExit(stderr, 32) {
		t.Fatal("ambient proxy did not remain non-zero")
	}
	decision = decodePolicyDecision(t, stdout)
	if decision.Status != "manual-required" || decision.Tier != "isolated-integration" || decision.Reason != "ambient-state-forbidden" || decision.ContractValidated || decision.TestID != "" {
		t.Fatal("ambient proxy was inherited or reflected")
	}
}

func testLiveCheckCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	stdout, stderr, err := runCLI(safetyRoot, "test-policy", "--tier", "live-check", "--live-probe", "fixture.live.synthetic.v1")
	if err == nil || !isGoRunExit(stderr, 32) {
		t.Fatal("live-check did not remain bounded non-zero")
	}
	decision := decodePolicyDecision(t, stdout)
	if decision.Status != "unknown" || decision.Tier != "live-check" || decision.NetworkPolicy != "denied" || decision.Reason != "live-probe-unapproved" || decision.ContractValidated || decision.ProbeID != "fixture.live.synthetic.v1" {
		t.Fatal("live-check executed, escalated, or claimed approval")
	}
}

func testPolicyOptionDenials(t *testing.T) {
	safetyRoot, repositoryRoot := projectRoots(t)
	manifestPath := filepath.Join(safetyRoot, "manifests", "network-tests.v1.json")
	invalid := [][]string{
		{"test-policy", "--network"},
		{"test-policy", "--live"},
		{"test-policy", "--tier", "isolated-integration", "--network-manifest", manifestPath, "--repo-root", repositoryRoot},
		{"test-policy", "--tier", "offline-static", "--allow-network-test", "fixture.download.synthetic-archive.v1"},
		{"test-policy", "--tier", "live-check", "--allow-network-test", "fixture.download.synthetic-archive.v1"},
		{"test-policy", "--tier", "isolated-integration", "--live-probe", "fixture.live.synthetic.v1"},
	}
	for _, arguments := range invalid {
		stdout, stderr, err := runCLI(safetyRoot, arguments...)
		if err == nil || len(stdout) != 0 || len(stderr) == 0 || len(stderr) > maxCLIOutput {
			t.Fatal("broad or partial policy option was not bounded and rejected")
		}
		combined := string(stdout) + string(stderr)
		if strings.Contains(combined, manifestPath) || strings.Contains(combined, "fixture.download.synthetic-archive.v1") || strings.Contains(combined, "fixture.live.synthetic.v1") {
			t.Fatal("policy rejection reflected caller input")
		}
	}

	injectionID := `never-registered-task$(printf runner-injection)`
	stdout, stderr, err := runCLI(
		safetyRoot,
		"test-policy",
		"--tier", "isolated-integration",
		"--network-manifest", manifestPath,
		"--repo-root", repositoryRoot,
		"--allow-network-test", injectionID,
	)
	if err == nil || !isGoRunExit(stderr, 32) {
		t.Fatal("injection-shaped network ID did not remain manual-required")
	}
	decision := decodePolicyDecision(t, stdout)
	if decision.Status != "manual-required" || decision.Tier != "isolated-integration" || decision.ContractValidated || decision.TestID != "" || bytes.Contains(stdout, []byte("runner-injection")) {
		t.Fatal("injection-shaped network ID selected or reflected capability")
	}
}

func testClosedRunnerDispatch(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	scriptPath := filepath.Join(safetyRoot, "scripts", "test.sh")
	tests := []struct {
		arguments       []string
		wantUnsupported bool
	}{
		{arguments: []string{"task", "never-registered-task"}, wantUnsupported: true},
		{arguments: []string{"task", "never-registered-task;printf-runner-injection"}, wantUnsupported: true},
		{arguments: []string{"task", `never-registered-task$(printf runner-injection)`}, wantUnsupported: true},
		{arguments: []string{"task", "never-registered-task*"}, wantUnsupported: true},
		{arguments: []string{"task", "../never-registered-task"}, wantUnsupported: true},
		{arguments: []string{"wave", "never-registered-wave"}, wantUnsupported: true},
		{arguments: []string{"wave", "never-registered-wave;printf-runner-injection"}, wantUnsupported: true},
		{arguments: []string{"wave", `never-registered-wave$(printf runner-injection)`}, wantUnsupported: true},
		{arguments: []string{"wave", "never-registered-wave*"}, wantUnsupported: true},
		{arguments: []string{"wave", "../never-registered-wave"}, wantUnsupported: true},
		{arguments: []string{"never-registered-scope"}},
		{arguments: []string{"never-registered-scope;printf-runner-injection"}},
		{arguments: []string{"phase", "unexpected-argument"}},
	}
	for _, testCase := range tests {
		command := exec.Command("/bin/bash", append([]string{scriptPath}, testCase.arguments...)...)
		command.Env = os.Environ()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err == nil {
			t.Fatal("generic or injection-shaped runner argv succeeded")
		}
		if stdout.Len() > maxCLIOutput || stderr.Len() > maxCLIOutput {
			t.Fatal("runner rejection exceeded bounded output")
		}
		combined := stdout.String() + stderr.String()
		for _, forbidden := range []string{"runner-injection", "expected-red-observed", "synthetic-sentinel-passed", "live-check", "./internal/fixture", "./internal/e2e", "TestTierNetworkPolicy", "TestTierCLI"} {
			if strings.Contains(combined, forbidden) {
				t.Fatalf("runner rejection selected or reflected capability: %s", forbidden)
			}
		}
		if testCase.wantUnsupported {
			if !strings.Contains(combined, `"status":"harness-error"`) || !strings.Contains(combined, `"reason":"unsupported-suite"`) {
				t.Fatal("generic task or wave rejection changed")
			}
		} else if !strings.Contains(combined, "usage:") {
			t.Fatal("generic scope or malformed phase bypassed usage rejection")
		}
	}
}

func testTierRunnerContract(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	text := string(data)
	for _, required := range []string{
		"'./internal/fixture'", "'^TestTierNetworkPolicy$'", "'TestTierNetworkPolicy'",
		"'./internal/e2e'", "'^TestTierCLI$'", "'TestTierCLI'",
		"run_tier_network_policy", "run_fixture_policy_wave",
		"run_wave_child fixture-lifecycle",
		"run_wave_child tier-network-policy",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("runner fixed tier pair or fresh-root aggregation missing: %s", required)
		}
	}
	if strings.Count(text, "task:tier-network-policy)") != 1 || strings.Count(text, "wave:fixture-policy)") != 1 {
		t.Fatal("tier task or fixture policy wave label is not unique")
	}
	fixtureWaveStart := strings.Index(text, "run_fixture_policy_wave()")
	if fixtureWaveStart < 0 {
		t.Fatal("fixture policy wave unavailable")
	}
	fixtureWaveEnd := strings.Index(text[fixtureWaveStart:], "\n}\n")
	if fixtureWaveEnd < 0 || strings.Contains(text[fixtureWaveStart:fixtureWaveStart+fixtureWaveEnd], "run_with_runner_deadline /bin/bash") {
		t.Fatal("fixture policy wave recreated a nested process-group deadline")
	}
	assertLiteralDispatchLabels(t, text)
}

func decodePolicyDecision(t *testing.T, data []byte) policyDecision {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decision policyDecision
	if err := decoder.Decode(&decision); err != nil {
		t.Fatal("policy CLI output is invalid")
	}
	return decision
}

func runPolicyCommand(safetyRoot string, environment []string, arguments ...string) ([]byte, []byte, error) {
	command := exec.Command("go", append([]string{"run", "./cmd/yamc-safety"}, arguments...)...)
	command.Dir = safetyRoot
	command.Env = environment
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func withoutPolicyEnvironment(environment []string) []string {
	blocked := map[string]struct{}{
		"HTTP_PROXY": {}, "HTTPS_PROXY": {}, "ALL_PROXY": {}, "NO_PROXY": {},
		"AWS_ACCESS_KEY_ID": {}, "AWS_SECRET_ACCESS_KEY": {}, "GITHUB_TOKEN": {}, "SSH_AUTH_SOCK": {},
	}
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if _, denied := blocked[key]; !denied {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func isGoRunExit(stderr []byte, exitCode int) bool {
	return strings.TrimSpace(string(stderr)) == "exit status "+strconv.Itoa(exitCode)
}
