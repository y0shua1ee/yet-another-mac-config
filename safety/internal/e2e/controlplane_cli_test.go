package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"example.invalid/yamc/safety/internal/contract"
	"example.invalid/yamc/safety/internal/privacy"
)

type controlPlaneCLIOutput struct {
	Status string                       `json:"status"`
	Layers []contract.ControlPlaneLayer `json:"layers"`
	Facts  []contract.ControlPlaneFact  `json:"facts"`
}

func TestControlPlaneCLI(t *testing.T) {
	t.Run("renders synthetic logical ownership facts", testControlPlaneValidationCLI)
	t.Run("never invokes Nix Homebrew or delegated managers", testControlPlaneInvocationCanaries)
	t.Run("rejects duplicate owner and inspection options", testControlPlaneCLIRejections)
	t.Run("binds exact fixed runner pairs", testControlPlaneRunnerContract)
}

func testControlPlaneValidationCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	contractPath := writeControlPlaneContract(t, loadValidControlPlaneContract(t, safetyRoot))
	stdout, stderr, err := runCLI(safetyRoot, "validate", "controlplane", "--contract", contractPath)
	if err != nil || len(stderr) != 0 {
		t.Fatal("control-plane validation CLI failed")
	}
	output := decodeControlPlaneCLIOutput(t, stdout)
	if output.Status != "valid" || len(output.Layers) != 3 || len(output.Facts) != 6 {
		t.Fatal("control-plane CLI output changed")
	}
	for _, fact := range output.Facts {
		reference, parseErr := privacy.ParseLogicalRef(fact.Scope)
		if parseErr != nil || (reference.Namespace != privacy.NamespaceProfile && reference.Namespace != privacy.NamespaceRepo && reference.Namespace != privacy.NamespaceFixture) || fact.Executable == "" {
			t.Fatal("control-plane CLI rendered a physical or invalid identifier")
		}
	}
	combined := string(stdout) + string(stderr)
	for _, forbidden := range []string{contractPath, filepath.Dir(contractPath), "/Users/", "current-host", "fresh-install", "covered-surfaces-unchanged-for-run"} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("control-plane CLI leaked or overclaimed: %s", forbidden)
		}
	}
}

func testControlPlaneInvocationCanaries(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	contractPath := writeControlPlaneContract(t, loadValidControlPlaneContract(t, safetyRoot))
	canaryRoot := t.TempDir()
	canaryBin := filepath.Join(canaryRoot, "bin")
	markerRoot := filepath.Join(canaryRoot, "markers")
	if err := os.MkdirAll(canaryBin, 0o700); err != nil {
		t.Fatal("canary bin unavailable")
	}
	if err := os.MkdirAll(markerRoot, 0o700); err != nil {
		t.Fatal("canary marker root unavailable")
	}
	for _, name := range []string{"nix", "darwin-rebuild", "home-manager", "brew", "mise", "uv", "rustup"} {
		script := []byte("#!/bin/sh\n/usr/bin/touch \"$YAMC_CONTROLPLANE_CANARY/" + name + "\"\nexit 99\n")
		if err := os.WriteFile(filepath.Join(canaryBin, name), script, 0o700); err != nil {
			t.Fatal("invocation canary unavailable")
		}
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Fatal("local Go unavailable")
	}
	command := exec.Command(goBin, "run", "./cmd/yamc-safety", "validate", "controlplane", "--contract", contractPath)
	command.Dir = safetyRoot
	command.Env = append(replaceEnvironment(os.Environ(), "PATH", canaryBin+":"+filepath.Dir(goBin)+":/usr/bin:/bin"), "YAMC_CONTROLPLANE_CANARY="+markerRoot)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stderr.Len() != 0 {
		t.Fatal("control-plane canary validation failed")
	}
	markers, err := os.ReadDir(markerRoot)
	if err != nil || len(markers) != 0 {
		t.Fatal("control-plane validation invoked a machine manager")
	}
}

func testControlPlaneCLIRejections(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	valid := loadValidControlPlaneContract(t, safetyRoot)
	var candidate contract.ControlPlaneContract
	if err := json.Unmarshal(valid, &candidate); err != nil {
		t.Fatal("valid control-plane fixture rejected")
	}
	candidate.Facts = append(candidate.Facts, candidate.Facts[0])
	duplicate, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal("duplicate control-plane fixture unavailable")
	}
	contractPath := writeControlPlaneContract(t, duplicate)
	stdout, stderr, err := runCLI(safetyRoot, "validate", "controlplane", "--contract", contractPath)
	if err == nil || len(stdout) != 0 || len(stderr) == 0 || len(stderr) > maxCLIOutput || bytes.Contains(stderr, []byte(contractPath)) {
		t.Fatal("duplicate ownership did not fail closed")
	}
	stdout, stderr, err = runCLI(safetyRoot, "validate", "controlplane", "--contract", writeControlPlaneContract(t, valid), "--inspect-live")
	if err == nil || len(stdout) != 0 || len(stderr) == 0 || len(stderr) > maxCLIOutput {
		t.Fatal("control-plane validation accepted live inspection")
	}
}

func testControlPlaneRunnerContract(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	text := string(data)
	for _, required := range []string{
		"'./internal/contract'", "'^TestControlPlaneContract$'", "'TestControlPlaneContract'",
		"'./internal/e2e'", "'^TestControlPlaneCLI$'", "'TestControlPlaneCLI'",
		"task:controlplane-contract)",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("control-plane fixed runner pair missing: %s", required)
		}
	}
	if strings.Count(text, "task:controlplane-contract)") != 1 {
		t.Fatal("control-plane task label is not unique")
	}
}

func loadValidControlPlaneContract(t *testing.T, safetyRoot string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(safetyRoot, "testdata", "controlplane", "cases.json"))
	if err != nil {
		t.Fatal("control-plane cases unavailable")
	}
	var cases struct {
		SchemaVersion string          `json:"schema_version"`
		ValidContract json.RawMessage `json:"valid_contract"`
		Invalid       []string        `json:"invalid_mutations"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cases); err != nil || cases.SchemaVersion != contract.ControlPlaneSchemaVersion || len(cases.ValidContract) == 0 || len(cases.Invalid) == 0 {
		t.Fatal("control-plane cases rejected")
	}
	return append([]byte(nil), cases.ValidContract...)
}

func writeControlPlaneContract(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "controlplane.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal("control-plane contract write failed")
	}
	return path
}

func decodeControlPlaneCLIOutput(t *testing.T, data []byte) controlPlaneCLIOutput {
	t.Helper()
	var output controlPlaneCLIOutput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatal("control-plane CLI output invalid")
	}
	return output
}

func replaceEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}
