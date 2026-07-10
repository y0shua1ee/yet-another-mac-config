package e2e

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.invalid/yamc/safety/internal/contract"
	"example.invalid/yamc/safety/internal/privacy"
)

func TestNoCleanupCLI(t *testing.T) {
	t.Run("round trips unmanaged status with no operation", testReportOnlyPolicyCLI)
	t.Run("rejects destructive policy before output or state", testDestructivePolicyCLI)
	t.Run("keeps the only receipt synthetic and fixture scoped", testSyntheticFixtureReceipt)
	t.Run("has no mutable command or shell dispatch edge", testNoMutableProductionRoute)
	t.Run("allows only exact offline Git input plumbing", testExactTrackedGitPlumbing)
	t.Run("binds exact task pairs and fixed fresh-root wave", testNoCleanupRunnerContract)
}

func testReportOnlyPolicyCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	for _, status := range []contract.PolicyStatus{contract.StatusExtra, contract.StatusUnmanagedPresent} {
		request := contract.PolicyRequest{
			SchemaVersion: contract.PolicySchemaVersion,
			Provenance:    "synthetic",
			Intent:        contract.IntentReportOnly,
			Status:        status,
			Operations:    []contract.Operation{},
		}
		contractPath := writePolicyContract(t, request)
		stdout, stderr, err := runCLI(safetyRoot, "validate", "policy", "--contract", contractPath)
		if err != nil || len(stderr) != 0 {
			t.Fatal("report-only policy CLI failed")
		}
		var decision contract.PolicyDecision
		decodeStrict(t, stdout, &decision)
		if decision.Status != status || decision.Operations == nil || len(decision.Operations) != 0 {
			t.Fatal("unmanaged status gained an operation")
		}
		if bytes.Contains(stdout, []byte(contractPath)) || bytes.Contains(stderr, []byte(contractPath)) {
			t.Fatal("policy CLI rendered a physical input path")
		}
	}
}

func testDestructivePolicyCLI(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	request := contract.PolicyRequest{
		SchemaVersion: contract.PolicySchemaVersion,
		Provenance:    "synthetic",
		Intent:        contract.IntentSyntheticFixture,
		Status:        contract.StatusSyntheticFixture,
		Operations: []contract.Operation{{
			Kind:   contract.OperationKind("destructive-convergence"),
			Target: "fixture:operation/rejected",
			Mode:   "synthetic",
		}},
	}
	base := t.TempDir()
	contractPath := filepath.Join(base, "policy.json")
	data, err := json.Marshal(request)
	if err != nil || os.WriteFile(contractPath, data, 0o600) != nil {
		t.Fatal("destructive policy fixture unavailable")
	}
	before, err := os.ReadDir(base)
	if err != nil || len(before) != 1 {
		t.Fatal("destructive policy fixture root changed before validation")
	}
	stdout, stderr, err := runCLI(safetyRoot, "validate", "policy", "--contract", contractPath)
	if err == nil || len(stdout) != 0 || len(stderr) == 0 || len(stderr) > maxCLIOutput {
		t.Fatal("destructive policy did not fail closed")
	}
	after, err := os.ReadDir(base)
	if err != nil || len(after) != 1 || after[0].Name() != "policy.json" {
		t.Fatal("rejected policy created state before validation")
	}
	if bytes.Contains(stderr, []byte(contractPath)) || bytes.Contains(stderr, []byte("destructive-convergence")) {
		t.Fatal("policy rejection reflected caller input")
	}

	stdout, stderr, err = runCLI(safetyRoot, "apply")
	if err == nil || len(stdout) != 0 || len(stderr) == 0 || len(stderr) > maxCLIOutput {
		t.Fatal("CLI exposed an apply route")
	}
}

func testSyntheticFixtureReceipt(t *testing.T) {
	safetyRoot, repositoryRoot := projectRoots(t)
	externalBase := t.TempDir()
	blueprintPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "input.json")
	surfacesPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "protected-surfaces.json")
	stdout, stderr, err := runCLI(safetyRoot, managedFixtureArgs(blueprintPath, surfacesPath, externalBase, "fixture:no-cleanup/receipt", repositoryRoot, "synthetic", true)...)
	if err != nil || len(stderr) != 0 {
		t.Fatal("synthetic fixture receipt run failed")
	}
	var output managedRunOutput
	decodeStrict(t, stdout, &output)
	summary := output.Summary
	fixtureRoot := onlyFixtureChild(t, externalBase)
	storeRoot := filepath.Join(fixtureRoot, "artifact-store")
	artifacts := readStoredArtifacts(t, storeRoot)
	receipt := summaryArtifact(t, artifacts, summary, "applied-receipt")
	payload := payloadMap(t, receipt)
	operationIDs, ok := payload["operation_ids"].([]any)
	if payload["mode"] != "synthetic" || !ok || len(operationIDs) != 1 {
		t.Fatal("receipt escaped the synthetic fixture boundary")
	}
	operationID, ok := operationIDs[0].(string)
	if !ok || !privacy.IsRegisteredOperationID(operationID) {
		t.Fatal("receipt operation is not fixture scoped")
	}

	before, readErr := os.ReadDir(externalBase)
	if readErr != nil {
		t.Fatal("managed fixture base unavailable")
	}
	stdout, stderr, err = runCLI(safetyRoot, managedFixtureArgs(blueprintPath, surfacesPath, externalBase, "fixture:no-cleanup/live", repositoryRoot, "live", false)...)
	if err == nil || len(stdout) != 0 || len(stderr) == 0 {
		t.Fatal("live receipt mode did not fail closed")
	}
	after, readErr := os.ReadDir(externalBase)
	if readErr != nil || len(after) != len(before) {
		t.Fatal("live receipt mode wrote before rejection")
	}
}

func testNoMutableProductionRoute(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	fileSet := token.NewFileSet()
	allowedExecImports := map[string]struct{}{
		filepath.Join("internal", "privacy", "capture.go"):    {},
		filepath.Join("internal", "sentinel", "real.go"):      {},
		filepath.Join("internal", "workflow", "synthetic.go"): {},
	}
	forbiddenRoutes := map[string]struct{}{
		"apply": {}, "cleanup": {}, "uninstall": {}, "zap": {}, "runtime-delete": {},
		"prune": {}, "trust": {}, "download": {}, "upgrade": {}, "switch": {},
		"service": {}, "defaults": {}, "link": {}, "shell": {}, "command": {},
	}
	mainPath := filepath.Join(safetyRoot, "cmd", "yamc-safety", "main.go")
	err := filepath.WalkDir(safetyRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if strings.EqualFold(entry.Name(), "executor") {
				t.Fatal("production graph contains an executor package")
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || (!strings.Contains(path, string(filepath.Separator)+"internal"+string(filepath.Separator)) && path != mainPath) {
			return nil
		}
		relative, relErr := filepath.Rel(safetyRoot, path)
		if relErr != nil {
			return relErr
		}
		parsed, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		for _, imported := range parsed.Imports {
			value, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr != nil {
				return unquoteErr
			}
			if strings.Contains(value, "executor") {
				t.Fatalf("production import reaches an executor: %s", relative)
			}
			if value == "os/exec" {
				if _, ok := allowedExecImports[relative]; !ok {
					t.Fatalf("unexpected command-capable import: %s", relative)
				}
			}
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if ok && literal.Kind == token.STRING {
				value, unquoteErr := strconv.Unquote(literal.Value)
				if unquoteErr == nil {
					switch value {
					case "sh", "bash", "zsh", "/bin/sh", "/bin/bash", "/bin/zsh", "/usr/bin/env sh", "/usr/bin/env bash":
						t.Fatalf("production source exposes shell dispatch: %s", relative)
					}
				}
			}
			return true
		})
		if path == mainPath {
			ast.Inspect(parsed, func(node ast.Node) bool {
				clause, ok := node.(*ast.CaseClause)
				if !ok {
					return true
				}
				for _, expression := range clause.List {
					literal, ok := expression.(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						continue
					}
					value, unquoteErr := strconv.Unquote(literal.Value)
					if unquoteErr == nil {
						if _, forbidden := forbiddenRoutes[value]; forbidden {
							t.Fatalf("CLI dispatch exposes a mutable route: %s", value)
						}
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("production graph scan failed: %v", err)
	}
}

func testExactTrackedGitPlumbing(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	path := filepath.Join(safetyRoot, "internal", "workflow", "synthetic.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("tracked-input plumbing source unavailable")
	}
	text := string(data)
	for _, required := range []string{
		`"/usr/bin/git"`, `"--no-lazy-fetch"`, `"--literal-pathspecs"`, `"core.fsmonitor=false"`, `"core.hooksPath=/dev/null"`, `"protocol.allow=never"`,
		`"GIT_CONFIG_NOSYSTEM=1"`, `"GIT_CONFIG_GLOBAL=/dev/null"`, `"GIT_OPTIONAL_LOCKS=0"`, `"GIT_NO_LAZY_FETCH=1"`, `"GIT_NO_REPLACE_OBJECTS=1"`,
		`"GIT_LITERAL_PATHSPECS=1"`, `"GIT_TERMINAL_PROMPT=0"`, `"GIT_ASKPASS=/usr/bin/false"`, `"SSH_ASKPASS=/usr/bin/false"`,
		`type gitProofOperation uint8`, `switch operation`,
		`"rev-parse", "--show-toplevel"`, `"ls-files", "-z", "--stage", "--error-unmatch"`, `"rev-parse", "--verify", "HEAD^{commit}"`,
		`"ls-tree", "-z", "--full-tree"`, `"cat-file", "blob"`,
	} {
		if !strings.Contains(text, required) {
			t.Fatal("exact tracked-input Git contract is incomplete")
		}
	}
	if strings.Contains(text, "os.Environ(") || strings.Contains(text, "arguments ...string") {
		t.Fatal("tracked-input Git plumbing exposed ambient or arbitrary command input")
	}

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, data, 0)
	if err != nil {
		t.Fatal("tracked-input plumbing source did not parse")
	}
	commandCalls := 0
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != "exec" {
			return true
		}
		commandCalls++
		if selector.Sel.Name != "CommandContext" || len(call.Args) < 2 {
			t.Fatal("tracked-input plumbing exposed a non-context command edge")
		}
		executable, ok := call.Args[1].(*ast.BasicLit)
		if !ok || executable.Kind != token.STRING || executable.Value != `"/usr/bin/git"` {
			t.Fatal("tracked-input plumbing executable is not fixed Git")
		}
		return true
	})
	if commandCalls != 1 {
		t.Fatal("tracked-input plumbing command edge is not singular")
	}
}

func testNoCleanupRunnerContract(t *testing.T) {
	safetyRoot, _ := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	text := string(data)
	for _, required := range []string{
		"'./internal/contract'", "'^TestNoDestructiveDefaults$'", "'TestNoDestructiveDefaults'",
		"'./internal/e2e'", "'^TestNoCleanupCLI$'", "'TestNoCleanupCLI'",
		"run_wave_child controlplane-contract", "run_wave_child no-destructive-defaults",
		"task:no-destructive-defaults)", "wave:controlplane)",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("no-cleanup fixed runner contract missing: %s", required)
		}
	}
	if strings.Count(text, "task:no-destructive-defaults)") != 1 || strings.Count(text, "wave:controlplane)") != 1 {
		t.Fatal("no-cleanup task or control-plane wave label is not unique")
	}
	waveStart := strings.Index(text, "run_controlplane_wave()")
	if waveStart < 0 {
		t.Fatal("control-plane wave function is unavailable")
	}
	waveEnd := strings.Index(text[waveStart:], "\n}\n")
	if waveEnd < 0 || strings.Contains(text[waveStart:waveStart+waveEnd], "run_with_runner_deadline /bin/bash") {
		t.Fatal("control-plane wave added a nested process-group wrapper")
	}
	assertLiteralDispatchLabels(t, text)
}

func writePolicyContract(t *testing.T, request contract.PolicyRequest) string {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal("policy contract encode failed")
	}
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal("policy contract write failed")
	}
	return path
}
