package contract

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNoDestructiveDefaults(t *testing.T) {
	policy := Phase1Policy()
	t.Run("round trips extra state as report only", func(t *testing.T) {
		for _, status := range []PolicyStatus{StatusExtra, StatusUnmanagedPresent} {
			decision, err := policy.Evaluate(PolicyRequest{
				SchemaVersion: PolicySchemaVersion,
				Provenance:    "synthetic",
				Intent:        IntentReportOnly,
				Status:        status,
				Operations:    []Operation{},
			})
			if err != nil || decision.Status != status || decision.Operations == nil || len(decision.Operations) != 0 {
				t.Fatal("EXPECTED_RED: destructive-policy-behavior-missing")
			}
		}
	})
	t.Run("allows only one synthetic fixture write", func(t *testing.T) {
		operation := Operation{Kind: OperationFixtureFakeWrite, Target: "fixture:operation/synthetic-write", Mode: "synthetic"}
		request := PolicyRequest{
			SchemaVersion: PolicySchemaVersion,
			Provenance:    "synthetic",
			Intent:        IntentSyntheticFixture,
			Status:        StatusSyntheticFixture,
			Operations:    []Operation{operation},
		}
		decision, err := policy.Evaluate(request)
		if err != nil || len(decision.Operations) != 1 || decision.Operations[0] != operation {
			t.Fatal("synthetic fixture write rejected")
		}
		request.Operations[0].Target = "fixture:operation/substituted"
		if decision.Operations[0] != operation {
			t.Fatal("policy decision retained caller-owned mutable state")
		}
		for _, mutation := range []Operation{
			{Kind: OperationFixtureFakeWrite, Target: "repo:operation/synthetic-write", Mode: "synthetic"},
			{Kind: OperationFixtureFakeWrite, Target: "fixture:operation/synthetic-write", Mode: "live"},
		} {
			candidate := request
			candidate.Operations = []Operation{mutation}
			if _, err := policy.Evaluate(candidate); err == nil {
				t.Fatal("non-fixture or live receipt source accepted")
			}
		}
	})
	t.Run("rejects every mutable and destructive boundary", func(t *testing.T) {
		for _, forbidden := range []OperationKind{
			"cleanup", "uninstall", "zap", "runtime-delete", "delete", "prune", "trust",
			"download", "upgrade", "switch", "service-mutation", "defaults-mutation",
			"link-mutation", "destructive-convergence", "shell", "command", "apply",
		} {
			request := PolicyRequest{
				SchemaVersion: PolicySchemaVersion,
				Provenance:    "synthetic",
				Intent:        IntentSyntheticFixture,
				Status:        StatusSyntheticFixture,
				Operations:    []Operation{{Kind: forbidden, Target: "fixture:operation/rejected", Mode: "synthetic"}},
			}
			if _, err := policy.Evaluate(request); err == nil {
				t.Fatalf("forbidden operation accepted: %s", forbidden)
			}
		}
		reportWithOperation := PolicyRequest{
			SchemaVersion: PolicySchemaVersion,
			Provenance:    "synthetic",
			Intent:        IntentReportOnly,
			Status:        StatusUnmanagedPresent,
			Operations:    []Operation{{Kind: OperationFixtureFakeWrite, Target: "fixture:operation/rejected", Mode: "synthetic"}},
		}
		if _, err := policy.Evaluate(reportWithOperation); err == nil {
			t.Fatal("unmanaged report status converted into an operation")
		}
	})
	t.Run("rejects command fields and ownership authority injection", testClosedPolicySchema)
	t.Run("contains data only and no executor callback", testDataOnlyPolicyTypes)
}

func testClosedPolicySchema(t *testing.T) {
	t.Helper()
	base := map[string]any{
		"schema_version": PolicySchemaVersion,
		"provenance":     "synthetic",
		"intent":         string(IntentReportOnly),
		"status":         string(StatusUnmanagedPresent),
		"operations":     []any{},
	}
	for _, field := range []string{"command", "argv", "shell", "executor", "declaration_owner", "module_owner"} {
		candidate := make(map[string]any, len(base)+1)
		for key, value := range base {
			candidate[key] = value
		}
		candidate[field] = "synthetic"
		data, err := json.Marshal(candidate)
		if err != nil {
			t.Fatal("policy negative fixture unavailable")
		}
		if _, err := ParsePolicy(data); err == nil {
			t.Fatalf("policy accepted authority or command field: %s", field)
		}
	}
	missingOperations := make(map[string]any, len(base))
	for key, value := range base {
		if key != "operations" {
			missingOperations[key] = value
		}
	}
	data, _ := json.Marshal(missingOperations)
	request, err := ParsePolicy(data)
	if err != nil {
		t.Fatal("missing-operation-list fixture did not parse structurally")
	}
	if _, err := Phase1Policy().Evaluate(request); err == nil {
		t.Fatal("policy accepted an implicit null operation list")
	}
}

func testDataOnlyPolicyTypes(t *testing.T) {
	t.Helper()
	types := []reflect.Type{
		reflect.TypeOf(Policy{}),
		reflect.TypeOf(PolicyRequest{}),
		reflect.TypeOf(PolicyDecision{}),
		reflect.TypeOf(Operation{}),
	}
	for _, current := range types {
		for index := 0; index < current.NumField(); index++ {
			kind := current.Field(index).Type.Kind()
			if kind == reflect.Func || kind == reflect.Interface || kind == reflect.Chan || kind == reflect.UnsafePointer {
				t.Fatal("policy type exposes an executable callback")
			}
		}
	}
}
