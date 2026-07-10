package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"sort"

	"example.invalid/yamc/safety/internal/artifact"
	"example.invalid/yamc/safety/internal/privacy"
)

const ControlPlaneSchemaVersion = "1.0.0"

type Owner string

const (
	OwnerDeterminateNix       Owner = "determinate-nix"
	OwnerNixDarwin            Owner = "nix-darwin"
	OwnerHomeManager          Owner = "home-manager"
	OwnerNixStoreHomeManager  Owner = "nix-store-via-home-manager"
	OwnerHomebrew             Owner = "homebrew"
	OwnerMise                 Owner = "mise"
	OwnerUV                   Owner = "uv"
	OwnerRustup               Owner = "rustup"
	OwnerProjectWrapper       Owner = "project-wrapper"
	OwnerExclusiveNixDevShell Owner = "exclusive-nix-devshell"
)

type OwnerRole string

const (
	RoleNixDistribution           OwnerRole = "nix-distribution"
	RoleNixDaemon                 OwnerRole = "nix-daemon"
	RoleNixSupportBoundary        OwnerRole = "nix-support-boundary"
	RoleMachineComposition        OwnerRole = "machine-composition"
	RoleMachineActivation         OwnerRole = "machine-activation"
	RoleUserConfiguration         OwnerRole = "user-configuration"
	RoleNixBuiltManagerEntrypoint OwnerRole = "nix-built-manager-entrypoint"
	RoleConfigFile                OwnerRole = "config-file"
	RoleShellIntegration          OwnerRole = "shell-integration"
)

type ActivationContext string

const (
	ActivationSystemComposition ActivationContext = "system-composition"
	ActivationProjectSelection  ActivationContext = "project-selection"
	ActivationProjectWrapper    ActivationContext = "project-wrapper"
	ActivationExclusiveDevShell ActivationContext = "exclusive-devshell"
)

type ControlPlaneLayer struct {
	Owner Owner       `json:"owner"`
	Roles []OwnerRole `json:"roles"`
}

type ControlPlaneFact struct {
	Scope                   string            `json:"scope"`
	Executable              string            `json:"executable"`
	DeclarationOwner        Owner             `json:"declaration_owner"`
	ManagerBinaryOwner      Owner             `json:"manager_binary_owner"`
	ManagedPayloadOwner     Owner             `json:"managed_payload_owner"`
	SelectedExecutableOwner Owner             `json:"selected_executable_owner"`
	ActivationContext       ActivationContext `json:"activation_context"`
}

type ControlPlaneContract struct {
	SchemaVersion string              `json:"schema_version"`
	Provenance    string              `json:"provenance"`
	Layers        []ControlPlaneLayer `json:"layers"`
	Facts         []ControlPlaneFact  `json:"facts"`
}

type ownerPattern struct {
	declaration   Owner
	managerBinary Owner
	payload       Owner
	activation    ActivationContext
}

var executablePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

var requiredLayerRoles = map[Owner][]OwnerRole{
	OwnerDeterminateNix: {
		RoleNixDistribution,
		RoleNixDaemon,
		RoleNixSupportBoundary,
	},
	OwnerNixDarwin: {
		RoleMachineComposition,
		RoleMachineActivation,
	},
	OwnerHomeManager: {
		RoleUserConfiguration,
		RoleNixBuiltManagerEntrypoint,
		RoleConfigFile,
		RoleShellIntegration,
	},
}

var selectedOwnerPatterns = map[Owner]ownerPattern{
	OwnerHomebrew: {
		declaration: OwnerNixDarwin, managerBinary: OwnerHomebrew,
		payload: OwnerHomebrew, activation: ActivationSystemComposition,
	},
	OwnerMise: {
		declaration: OwnerHomeManager, managerBinary: OwnerNixStoreHomeManager,
		payload: OwnerMise, activation: ActivationProjectSelection,
	},
	OwnerUV: {
		declaration: OwnerHomeManager, managerBinary: OwnerNixStoreHomeManager,
		payload: OwnerUV, activation: ActivationProjectSelection,
	},
	OwnerRustup: {
		declaration: OwnerHomeManager, managerBinary: OwnerNixStoreHomeManager,
		payload: OwnerRustup, activation: ActivationProjectSelection,
	},
	OwnerProjectWrapper: {
		declaration: OwnerProjectWrapper, managerBinary: OwnerProjectWrapper,
		payload: OwnerProjectWrapper, activation: ActivationProjectWrapper,
	},
	OwnerExclusiveNixDevShell: {
		declaration: OwnerExclusiveNixDevShell, managerBinary: OwnerExclusiveNixDevShell,
		payload: OwnerExclusiveNixDevShell, activation: ActivationExclusiveDevShell,
	},
}

func ParseControlPlane(data []byte) (ControlPlaneContract, error) {
	canonical, err := artifact.Canonicalize(data)
	if err != nil {
		return ControlPlaneContract{}, errors.New("control-plane contract rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var parsed ControlPlaneContract
	if err := decoder.Decode(&parsed); err != nil {
		return ControlPlaneContract{}, errors.New("control-plane contract rejected")
	}
	if err := ValidateControlPlane(parsed); err != nil {
		return ControlPlaneContract{}, err
	}
	return parsed, nil
}

func ValidateControlPlane(value ControlPlaneContract) error {
	if value.SchemaVersion != ControlPlaneSchemaVersion || value.Provenance != "synthetic" {
		return errors.New("control-plane contract rejected")
	}
	if err := validateLayers(value.Layers); err != nil {
		return err
	}
	return ValidateOwnership(value.Facts)
}

func ValidateOwnership(facts []ControlPlaneFact) error {
	if len(facts) == 0 {
		return errors.New("control-plane ownership rejected")
	}
	seen := make(map[string]struct{}, len(facts))
	for _, fact := range facts {
		reference, err := privacy.ParseLogicalRef(fact.Scope)
		if err != nil || (reference.Namespace != privacy.NamespaceProfile && reference.Namespace != privacy.NamespaceRepo && reference.Namespace != privacy.NamespaceFixture) || !executablePattern.MatchString(fact.Executable) {
			return errors.New("control-plane ownership rejected")
		}
		key := fact.Scope + "\x00" + fact.Executable
		if _, exists := seen[key]; exists {
			return errors.New("control-plane ownership rejected")
		}
		seen[key] = struct{}{}

		pattern, ok := selectedOwnerPatterns[fact.SelectedExecutableOwner]
		if !ok || fact.DeclarationOwner != pattern.declaration || fact.ManagerBinaryOwner != pattern.managerBinary || fact.ManagedPayloadOwner != pattern.payload || fact.ActivationContext != pattern.activation {
			return errors.New("control-plane ownership rejected")
		}
	}
	return nil
}

func validateLayers(layers []ControlPlaneLayer) error {
	if len(layers) != len(requiredLayerRoles) {
		return errors.New("control-plane layers rejected")
	}
	seenOwners := make(map[Owner]struct{}, len(layers))
	for _, layer := range layers {
		required, ok := requiredLayerRoles[layer.Owner]
		if !ok {
			return errors.New("control-plane layers rejected")
		}
		if _, duplicate := seenOwners[layer.Owner]; duplicate {
			return errors.New("control-plane layers rejected")
		}
		seenOwners[layer.Owner] = struct{}{}
		if !sameRoles(required, layer.Roles) {
			return errors.New("control-plane layers rejected")
		}
	}
	return nil
}

func sameRoles(left, right []OwnerRole) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]OwnerRole(nil), left...)
	rightCopy := append([]OwnerRole(nil), right...)
	sort.Slice(leftCopy, func(i, j int) bool { return leftCopy[i] < leftCopy[j] })
	sort.Slice(rightCopy, func(i, j int) bool { return rightCopy[i] < rightCopy[j] })
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] || (index > 0 && rightCopy[index] == rightCopy[index-1]) {
			return false
		}
	}
	return true
}
