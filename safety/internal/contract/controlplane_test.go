package contract

import (
	"bytes"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

type controlPlaneCases struct {
	SchemaVersion    string          `json:"schema_version"`
	ValidContract    json.RawMessage `json:"valid_contract"`
	InvalidMutations []string        `json:"invalid_mutations"`
}

func TestControlPlaneContract(t *testing.T) {
	cases, valid := loadControlPlaneCases(t)
	contract, err := ParseControlPlane(valid)
	if err != nil {
		t.Fatal("EXPECTED_RED: controlplane-ownership-behavior-missing")
	}
	t.Run("encodes exact primary control-plane layers", func(t *testing.T) {
		assertPrimaryLayers(t, contract)
	})
	t.Run("keeps declaration manager payload and selected owners separate", func(t *testing.T) {
		assertLayeredOwnership(t, contract)
	})
	t.Run("rejects duplicate primary owner keys", func(t *testing.T) {
		duplicate := append([]ControlPlaneFact(nil), contract.Facts...)
		duplicate = append(duplicate, contract.Facts[0])
		if ValidateOwnership(duplicate) == nil {
			t.Fatal("duplicate scope and executable acquired a second primary owner")
		}
	})
	t.Run("rejects module role collapse and unknown fields", func(t *testing.T) {
		assertInvalidControlPlaneMutations(t, cases, contract)
	})
	t.Run("keeps fixture public and synthetic", func(t *testing.T) {
		assertSyntheticControlPlaneFixture(t, valid)
	})
}

func loadControlPlaneCases(t *testing.T) (controlPlaneCases, []byte) {
	t.Helper()
	data, err := os.ReadFile("../../testdata/controlplane/cases.json")
	if err != nil {
		t.Fatal("control-plane cases unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var cases controlPlaneCases
	if err := decoder.Decode(&cases); err != nil || cases.SchemaVersion != ControlPlaneSchemaVersion || len(cases.InvalidMutations) != 5 {
		t.Fatal("control-plane cases rejected")
	}
	return cases, append([]byte(nil), cases.ValidContract...)
}

func assertPrimaryLayers(t *testing.T, value ControlPlaneContract) {
	t.Helper()
	want := map[Owner][]OwnerRole{
		OwnerDeterminateNix: {RoleNixDistribution, RoleNixDaemon, RoleNixSupportBoundary},
		OwnerNixDarwin:      {RoleMachineComposition, RoleMachineActivation},
		OwnerHomeManager: {
			RoleUserConfiguration,
			RoleNixBuiltManagerEntrypoint,
			RoleConfigFile,
			RoleShellIntegration,
		},
	}
	if len(value.Layers) != len(want) {
		t.Fatal("primary control-plane layer count changed")
	}
	for _, layer := range value.Layers {
		required, ok := want[layer.Owner]
		if !ok || !sameRoles(required, layer.Roles) {
			t.Fatal("primary control-plane role boundary collapsed")
		}
	}
}

func assertLayeredOwnership(t *testing.T, value ControlPlaneContract) {
	t.Helper()
	wantOwners := []Owner{OwnerHomebrew, OwnerMise, OwnerUV, OwnerRustup, OwnerProjectWrapper, OwnerExclusiveNixDevShell}
	gotOwners := make([]Owner, 0, len(value.Facts))
	for _, fact := range value.Facts {
		gotOwners = append(gotOwners, fact.SelectedExecutableOwner)
		if fact.Scope == "" || fact.Executable == "" || fact.DeclarationOwner == "" || fact.ManagerBinaryOwner == "" || fact.ManagedPayloadOwner == "" || fact.SelectedExecutableOwner == "" || fact.ActivationContext == "" {
			t.Fatal("ownership field was omitted")
		}
		if fact.SelectedExecutableOwner != fact.ManagedPayloadOwner {
			t.Fatal("selected executable did not retain its explicit payload owner")
		}
		if fact.SelectedExecutableOwner == OwnerMise || fact.SelectedExecutableOwner == OwnerUV || fact.SelectedExecutableOwner == OwnerRustup {
			if fact.DeclarationOwner != OwnerHomeManager || fact.ManagerBinaryOwner != OwnerNixStoreHomeManager || fact.ManagedPayloadOwner == fact.ManagerBinaryOwner {
				t.Fatal("Home Manager manager-entrypoint role collapsed into payload ownership")
			}
		}
		if fact.SelectedExecutableOwner == OwnerHomebrew && (fact.DeclarationOwner != OwnerNixDarwin || fact.ManagedPayloadOwner == fact.DeclarationOwner) {
			t.Fatal("nix-darwin declaration role collapsed into Homebrew payload ownership")
		}
	}
	sort.Slice(wantOwners, func(i, j int) bool { return wantOwners[i] < wantOwners[j] })
	sort.Slice(gotOwners, func(i, j int) bool { return gotOwners[i] < gotOwners[j] })
	if len(gotOwners) != len(wantOwners) {
		t.Fatal("delegated owner fixture coverage changed")
	}
	for index := range wantOwners {
		if gotOwners[index] != wantOwners[index] {
			t.Fatal("delegated owner fixture coverage changed")
		}
	}
}

func assertInvalidControlPlaneMutations(t *testing.T, cases controlPlaneCases, valid ControlPlaneContract) {
	t.Helper()
	wantMutations := map[string]struct{}{
		"duplicate-primary-owner":         {},
		"module-collapses-payload-owner":  {},
		"module-collapses-selected-owner": {},
		"role-set-collapse":               {},
		"unknown-activation-context":      {},
	}
	for _, name := range cases.InvalidMutations {
		if _, ok := wantMutations[name]; !ok {
			t.Fatal("unknown negative mutation fixture")
		}
		candidate := cloneControlPlane(t, valid)
		switch name {
		case "duplicate-primary-owner":
			candidate.Facts = append(candidate.Facts, candidate.Facts[0])
		case "module-collapses-payload-owner":
			candidate.Facts[1].ManagedPayloadOwner = OwnerNixStoreHomeManager
		case "module-collapses-selected-owner":
			candidate.Facts[1].SelectedExecutableOwner = OwnerHomeManager
		case "role-set-collapse":
			candidate.Layers[2].Roles = []OwnerRole{RoleUserConfiguration}
		case "unknown-activation-context":
			candidate.Facts[2].ActivationContext = ActivationContext("caller-defined")
		}
		if ValidateControlPlane(candidate) == nil {
			t.Fatalf("unsafe ownership mutation accepted: %s", name)
		}
	}

	var unknownValue map[string]any
	if err := json.Unmarshal(cases.ValidContract, &unknownValue); err != nil {
		t.Fatal("control-plane fixture decode failed")
	}
	unknownValue["unexpected"] = "synthetic"
	unknown, err := json.Marshal(unknownValue)
	if err != nil {
		t.Fatal("control-plane unknown-field fixture failed")
	}
	if _, err := ParseControlPlane(unknown); err == nil {
		t.Fatal("unknown control-plane field accepted")
	}
	duplicateKey := bytes.Replace(cases.ValidContract, []byte(`"schema_version": "1.0.0"`), []byte(`"schema_version": "1.0.0", "schema_version": "1.0.0"`), 1)
	if _, err := ParseControlPlane(duplicateKey); err == nil {
		t.Fatal("duplicate control-plane key accepted")
	}
	nonSynthetic := bytes.Replace(cases.ValidContract, []byte(`"provenance": "synthetic"`), []byte(`"provenance": "runtime"`), 1)
	if _, err := ParseControlPlane(nonSynthetic); err == nil {
		t.Fatal("non-synthetic control-plane contract accepted")
	}
}

func assertSyntheticControlPlaneFixture(t *testing.T, data []byte) {
	t.Helper()
	text := string(data)
	for _, forbidden := range []string{"/Users/", "/home/", "file:/", "localhost", "127.0.0.1", "current-host", "fresh-install"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("control-plane fixture contains host-derived data: %s", forbidden)
		}
	}
}

func cloneControlPlane(t *testing.T, value ControlPlaneContract) ControlPlaneContract {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal("control-plane clone failed")
	}
	var clone ControlPlaneContract
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatal("control-plane clone failed")
	}
	return clone
}
