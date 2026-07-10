package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"example.invalid/yamc/safety/internal/sentinel"
)

func TestRealSentinelCLI(t *testing.T) {
	t.Run("stops at manual-required before any real adapter", testRealSentinelProofGateCLI)
	t.Run("rejects mixed or incomplete real-mode arguments", testRealSentinelArgumentDenials)
	t.Run("rejects malformed proof without reflecting inputs", testRealSentinelMalformedProof)
	t.Run("binds exact task pairs and three-handler wave", testRealSentinelRunnerContract)
}

func testRealSentinelProofGateCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	manifestPath := filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json")
	adapterManifestPath := filepath.Join(safetyRoot, "manifests", "real-adapters.v1.json")
	stdout, stderr, err := runCLI(safetyRoot,
		"sentinel", "verify",
		"--mode", "real",
		"--manifest", manifestPath,
		"--adapter-manifest", adapterManifestPath,
	)
	if err == nil || !isGoRunExit(stderr, 32) {
		t.Fatal("unavailable required proof did not return exact manual-required exit")
	}
	var assessment sentinel.RealProofAssessment
	decodeStrict(t, stdout, &assessment)
	if assessment.Status != "manual-required" || assessment.Verdict != sentinel.VerdictIndeterminate || assessment.ExitCode != 32 || assessment.Reason != "required-real-adapter-proof-unavailable" || assessment.ClaimEligible {
		t.Fatal("real CLI proof assessment changed or overclaimed")
	}
	allowedStops := map[string]struct{}{
		"worktree\x00repo:sentinel/worktree/tracked":                  {},
		"worktree\x00repo:sentinel/worktree/index":                    {},
		"named-home\x00home:.zshrc":                                   {},
		"manager-root\x00home:sentinel/manager/mise-data":             {},
		"service\x00profile:sentinel/service/homebrew-mxcl-nginx":     {},
		"named-target\x00profile:sentinel/named-target/system-shells": {},
	}
	if _, ok := allowedStops[string(assessment.SurfaceDomain)+"\x00"+assessment.LogicalRef]; !ok {
		t.Fatal("real CLI proof gate stopped outside the closed required scope")
	}
	combined := append(append([]byte{}, stdout...), stderr...)
	for _, forbidden := range []string{manifestPath, adapterManifestPath, safetyRoot, "/Users/", "/usr/bin/git", "/bin/launchctl", "effective_uid", "host_identity", "resolver_mapping", "service_output", "raw_output", "hmac_key", sentinel.ScopedUnchangedClaim, "whole-Mac-unchanged", "recovery-ready-on-current-host", "multi-host-verified", "fresh-install-verified"} {
		if forbidden != "" && bytes.Contains(combined, []byte(forbidden)) {
			t.Fatal("real proof gate leaked process data or a claim")
		}
	}
}

func testRealSentinelArgumentDenials(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	manifestPath := filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json")
	adapterManifestPath := filepath.Join(safetyRoot, "manifests", "real-adapters.v1.json")
	privateInput := filepath.Join(t.TempDir(), "caller-controlled")
	cases := [][]string{
		{"sentinel", "verify", "--mode", "real", "--manifest", manifestPath},
		{"sentinel", "verify", "--mode", "real", "--manifest", manifestPath, "--adapter-manifest", adapterManifestPath, "--fixture-root", privateInput},
		{"sentinel", "verify", "--mode", "synthetic", "--manifest", manifestPath, "--fixture-root", privateInput, "--adapter-manifest", adapterManifestPath},
	}
	for _, arguments := range cases {
		stdout, stderr, err := runCLI(safetyRoot, arguments...)
		if err == nil || !hasGoRunExit(stderr, 64) || len(stdout) != 0 || len(stderr) == 0 || len(stderr) > maxCLIOutput {
			t.Fatal("mixed or incomplete real-mode arguments were not bounded and rejected")
		}
		combined := append(append([]byte{}, stdout...), stderr...)
		for _, forbidden := range []string{manifestPath, adapterManifestPath, privateInput, safetyRoot, "/Users/", sentinel.ScopedUnchangedClaim} {
			if forbidden != "" && bytes.Contains(combined, []byte(forbidden)) {
				t.Fatal("argument rejection reflected caller input or a claim")
			}
		}
	}
}

func testRealSentinelMalformedProof(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	manifestPath := filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json")
	proofPath := filepath.Join(t.TempDir(), "malformed-proof.json")
	if err := os.WriteFile(proofPath, []byte(`{"schema_version":"1.0.0","adapters":[]}`), 0o600); err != nil {
		t.Fatal("malformed proof setup failed")
	}
	stdout, stderr, err := runCLI(safetyRoot,
		"sentinel", "verify",
		"--mode", "real",
		"--manifest", manifestPath,
		"--adapter-manifest", proofPath,
	)
	if err == nil || !hasGoRunExit(stderr, sentinel.ExitHarnessError) || len(stdout) != 0 || len(stderr) == 0 || len(stderr) > maxCLIOutput {
		t.Fatal("malformed real proof did not fail closed")
	}
	combined := append(append([]byte{}, stdout...), stderr...)
	for _, forbidden := range []string{manifestPath, proofPath, safetyRoot, "/Users/", "adapters", sentinel.ScopedUnchangedClaim} {
		if forbidden != "" && bytes.Contains(combined, []byte(forbidden)) {
			t.Fatal("malformed proof rejection reflected raw input")
		}
	}
	fifoPath := filepath.Join(t.TempDir(), "proof.fifo")
	if err := syscall.Mkfifo(fifoPath, 0o600); err != nil {
		t.Fatal("FIFO proof setup failed")
	}
	stdout, stderr, err = runCLI(safetyRoot,
		"sentinel", "verify",
		"--mode", "real",
		"--manifest", manifestPath,
		"--adapter-manifest", fifoPath,
	)
	if err == nil || !hasGoRunExit(stderr, sentinel.ExitHarnessError) || len(stdout) != 0 || bytes.Contains(stderr, []byte(fifoPath)) {
		t.Fatal("FIFO proof path blocked or escaped bounded rejection")
	}
}

func testRealSentinelRunnerContract(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	text := string(data)
	for _, required := range []string{
		"'./internal/sentinel'",
		"'^TestRealSentinelEnvelope$'",
		"'TestRealSentinelEnvelope'",
		"'./internal/e2e'",
		"'^TestRealSentinelCLI$'",
		"'TestRealSentinelCLI'",
		"task:real-sentinel-envelope)",
		"wave:sentinels)",
		"run_with_runner_deadline",
		"run_wave_child",
		"RUNNER_BUDGET_SECONDS=15",
		"RUNNER_BUDGET_SECONDS=47",
		"RUNNER_BUDGET_SECONDS=305",
		"runner-deadline-exceeded",
		"exit 70 unless setpgrp(0, 0);",
		"$SIG{TERM} = sub",
		"kill \"TERM\", $pid;",
		"kill 0, -$pid",
		"$stop_descendants->();",
		"handle_test_signal 143",
		"test -count=1 -timeout=30s",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("real sentinel runner literal missing: %s", required)
		}
	}
	waitIndex := strings.Index(text, "waitpid($pid, 0);")
	descendantCleanupIndex := strings.Index(text, "$stop_descendants->();")
	if waitIndex < 0 || descendantCleanupIndex < 0 || descendantCleanupIndex < waitIndex {
		t.Fatal("normal child exit does not clean its surviving process group")
	}
	if strings.Count(text, "task:real-sentinel-envelope)") != 1 || strings.Count(text, "wave:sentinels)") != 1 {
		t.Fatal("real sentinel runner labels are not unique")
	}
	mainSource, err := os.ReadFile(filepath.Join(safetyRoot, "cmd", "yamc-safety", "main.go"))
	if err != nil || !bytes.Contains(mainSource, []byte("LoadRealAdapterRegistry(data, time.Now().UTC())")) {
		t.Fatal("real proof date is not normalized to UTC")
	}
	if !strings.Contains(text, "task:real-sentinel-envelope)\n    run_real_sentinel_envelope\n    ;;") || !strings.Contains(text, "wave:sentinels)\n    run_sentinels_wave\n    ;;") {
		t.Fatal("sentinel case labels are not bound to their exact handlers")
	}
	realHandlerStart := strings.Index(text, "run_real_sentinel_envelope()")
	if realHandlerStart < 0 {
		t.Fatal("real sentinel handler unavailable")
	}
	realHandlerEnd := strings.Index(text[realHandlerStart:], "\n}\n")
	if realHandlerEnd < 0 {
		t.Fatal("real sentinel handler unavailable")
	}
	realHandler := text[realHandlerStart : realHandlerStart+realHandlerEnd]
	for _, pair := range []string{
		"'./internal/sentinel' \\\n    '^TestRealSentinelEnvelope$' \\\n    'TestRealSentinelEnvelope'",
		"'./internal/e2e' \\\n    '^TestRealSentinelCLI$' \\\n    'TestRealSentinelCLI'",
	} {
		if strings.Count(realHandler, pair) != 1 {
			t.Fatal("real sentinel handler does not bind one exact package-pattern-name triple")
		}
	}
	if strings.Count(realHandler, "run_exact_go_suite") != 2 {
		t.Fatal("real sentinel handler selects an unexpected test pair")
	}
	waveStart := strings.Index(text, "run_sentinels_wave()")
	if waveStart < 0 {
		t.Fatal("sentinel wave body unavailable")
	}
	waveEnd := strings.Index(text[waveStart:], "\n}\n")
	if waveEnd < 0 {
		t.Fatal("sentinel wave body unavailable")
	}
	wave := text[waveStart : waveStart+waveEnd]
	wantHandlers := []string{"sentinel-manifest", "sentinel-verdicts", "real-sentinel-envelope"}
	for _, handler := range wantHandlers {
		if strings.Count(wave, "run_wave_child "+handler) != 1 {
			t.Fatalf("sentinel wave does not invoke exactly one %s handler", handler)
		}
	}
	if strings.Count(wave, "run_wave_child ") != len(wantHandlers) {
		t.Fatal("sentinel wave aggregates an unexpected handler")
	}
	if strings.Contains(wave, "run_with_runner_deadline /bin/bash") {
		t.Fatal("sentinel wave recreated a nested process-group deadline")
	}
	for _, forbidden := range []string{" launchctl ", " git ", " curl ", " eval ", " go test ", "fixture run"} {
		if strings.Contains(wave, forbidden) {
			t.Fatal("sentinel wave contains a command outside the three fixed child handlers")
		}
	}
	waveChildStart := strings.Index(text, "run_wave_child()")
	if waveChildStart < 0 {
		t.Fatal("wave child deadline helper unavailable")
	}
	waveChildEnd := strings.Index(text[waveChildStart:], "\n}\n")
	if waveChildEnd < 0 {
		t.Fatal("wave child deadline helper unavailable")
	}
	waveChild := text[waveChildStart : waveChildStart+waveChildEnd]
	for _, required := range []string{
		"/bin/bash \"${SCRIPT_DIR}/test.sh\" task \"${suite_name}\"",
		"child_status}\" -eq 124",
		"remaining}\" -lt 15",
		"print_runner_deadline \"${suite_name}\"",
		"return 124",
	} {
		if !strings.Contains(waveChild, required) {
			t.Fatalf("wave child deadline contract missing: %s", required)
		}
	}
	if strings.Contains(waveChild, "run_with_runner_deadline") {
		t.Fatal("wave child helper nests a process-group deadline")
	}
	waveDeadline := strings.Index(waveChild, "child_status}\" -eq 124")
	waveOutputCap := strings.Index(waveChild, "${#output} > 65536")
	if waveDeadline < 0 || waveOutputCap < 0 || waveDeadline > waveOutputCap {
		t.Fatal("wave child output cap can overwrite an observed deadline")
	}
	runGoStart := strings.Index(text, "run_go_suite()")
	if runGoStart < 0 {
		t.Fatal("go suite deadline helper unavailable")
	}
	runGoEnd := strings.Index(text[runGoStart:], "\n}\n")
	if runGoEnd < 0 {
		t.Fatal("go suite deadline helper unavailable")
	}
	runGoSuite := text[runGoStart : runGoStart+runGoEnd]
	goDeadline := strings.Index(runGoSuite, "status}\" -eq 124")
	goOutputCap := strings.Index(runGoSuite, "${#output} > 65536")
	if goDeadline < 0 || goOutputCap < 0 || goDeadline > goOutputCap {
		t.Fatal("go suite output cap can overwrite an observed deadline")
	}
	compiledStart := strings.Index(text, "run_compiled_go_suite()")
	if compiledStart < 0 {
		t.Fatal("compiled go suite deadline helper unavailable")
	}
	compiledEnd := strings.Index(text[compiledStart:], "\n}\n")
	if compiledEnd < 0 {
		t.Fatal("compiled go suite deadline helper unavailable")
	}
	compiledSuite := text[compiledStart : compiledStart+compiledEnd]
	compiledDeadline := strings.Index(compiledSuite, "status}\" -eq 124")
	compiledOutputCap := strings.Index(compiledSuite, "${#output} > 65536")
	if compiledDeadline < 0 || compiledOutputCap < 0 || compiledDeadline > compiledOutputCap {
		t.Fatal("compiled go suite output cap can overwrite an observed deadline")
	}
	exactSuiteStart := strings.Index(text, "run_exact_go_suite()")
	if exactSuiteStart < 0 {
		t.Fatal("exact suite helper unavailable")
	}
	exactSuiteEnd := strings.Index(text[exactSuiteStart:], "\n}\n")
	if exactSuiteEnd < 0 {
		t.Fatal("exact suite helper unavailable")
	}
	exactSuite := text[exactSuiteStart : exactSuiteStart+exactSuiteEnd]
	for _, required := range []string{"test -c -o \"${test_binary}\"", "-test.list \"${test_pattern}\"", "run_compiled_go_suite", "build_status}\" -eq 124", "listing_status}\" -eq 124", "TEST_STATUS:-0}\" -eq 124", "print_runner_deadline", "return 124"} {
		if !strings.Contains(exactSuite, required) {
			t.Fatalf("exact suite timeout mapping missing: %s", required)
		}
	}

	var proofManifest struct {
		Adapters []struct {
			AdapterID  string  `json:"adapter_id"`
			ProofState string  `json:"proof_state"`
			ReviewedAt string  `json:"reviewed_at"`
			ValidUntil *string `json:"valid_until"`
		} `json:"adapters"`
	}
	proofData, err := os.ReadFile(filepath.Join(safetyRoot, "manifests", "real-adapters.v1.json"))
	if err != nil || json.Unmarshal(proofData, &proofManifest) != nil {
		t.Fatal("tracked real proof manifest unavailable")
	}
	launchctlMissing := false
	for _, adapter := range proofManifest.Adapters {
		if adapter.ReviewedAt == "" || adapter.ReviewedAt > "2026-07-10" {
			t.Fatal("adapter proof review date is later than the authoritative project date")
		}
		if adapter.AdapterID == "launchctl-print-service-v1" {
			launchctlMissing = adapter.ProofState == "missing" && adapter.ValidUntil == nil
		}
	}
	if !launchctlMissing {
		t.Fatal("unproven launchctl adapter was promoted in the tracked registry")
	}
}

func hasGoRunExit(stderr []byte, exitCode int) bool {
	wanted := "exit status " + strconv.Itoa(exitCode)
	return strings.HasSuffix(strings.TrimSpace(string(stderr)), wanted)
}
