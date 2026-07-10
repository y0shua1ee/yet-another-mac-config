package sentinel

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	EvidenceSchemaVersion = "1.0.0"
	ScopedUnchangedClaim  = "covered-surfaces-unchanged-for-run"
	ChangeDetectedCode    = "change-detected-during-window"
)

type Verdict string

const (
	VerdictPassed        Verdict = "passed"
	VerdictViolation     Verdict = "violation"
	VerdictIndeterminate Verdict = "indeterminate"
	VerdictHarnessError  Verdict = "harness-error"
)

const (
	ExitPassed        = 0
	ExitViolation     = 20
	ExitIndeterminate = 21
	ExitHarnessError  = 22
)

type ObservationWindow struct {
	WindowID string `json:"window_id"`
	OpenedAt string `json:"opened_at"`
	ClosedAt string `json:"closed_at"`
	State    string `json:"state"`
}

type SurfaceEvidence struct {
	SurfaceID     string            `json:"surface_id"`
	SurfaceDomain string            `json:"surface_domain"`
	LogicalRef    string            `json:"logical_ref"`
	Policy        SurfacePolicy     `json:"policy"`
	BeforeStatus  ObservationStatus `json:"before_status"`
	AfterStatus   ObservationStatus `json:"after_status"`
	BeforeToken   string            `json:"before_token,omitempty"`
	AfterToken    string            `json:"after_token,omitempty"`
	BeforeReason  IncompleteReason  `json:"before_reason,omitempty"`
	AfterReason   IncompleteReason  `json:"after_reason,omitempty"`
}

type Evidence struct {
	SchemaVersion  string            `json:"schema_version"`
	SuiteID        string            `json:"suite_id"`
	SuiteDigest    string            `json:"suite_digest"`
	Tier           string            `json:"tier"`
	ManifestDigest string            `json:"manifest_digest"`
	Window         ObservationWindow `json:"window"`
	WindowDigest   string            `json:"window_digest"`
	Optional       []string          `json:"optional"`
	Excluded       []string          `json:"excluded"`
	Surfaces       []SurfaceEvidence `json:"surfaces"`
	Provenance     string            `json:"provenance"`
	realBinding    *realEvidenceBinding
}

type EvidenceOptions struct {
	SuiteID    string
	Tier       string
	WindowID   string
	OpenedAt   time.Time
	ClosedAt   time.Time
	Provenance string
}

type Evaluation struct {
	Verdict        Verdict  `json:"verdict"`
	ExitCode       int      `json:"exit_code"`
	Reason         string   `json:"reason,omitempty"`
	ChangeCode     string   `json:"change_code,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	Claim          string   `json:"claim,omitempty"`
	EvidenceDigest string   `json:"evidence_digest,omitempty"`
}

type ClaimMaterial struct {
	Claim          string
	EvidenceDigest string
	SuiteDigest    string
	ManifestDigest string
	Window         ObservationWindow
	WindowDigest   string
	Surfaces       []SurfaceEvidence
}

type realEvidenceBinding struct {
	ManifestDigest string
	SuiteDigest    string
	WindowID       string
	EvidenceDigest string
}

func BuildEvidence(manifest ProtectedManifest, before, after ProtectedSnapshot, options EvidenceOptions) (Evidence, error) {
	if err := validateProtectedManifest(manifest); err != nil || manifest.Digest == "" || before.ManifestDigest != manifest.Digest || after.ManifestDigest != manifest.Digest || before.WindowState != "closed" || after.WindowState != "closed" {
		return Evidence{}, errors.New("sentinel evidence input rejected")
	}
	if options.SuiteID != manifest.SuiteID || !validEvidenceTier(options.Tier) || !publicManifestID.MatchString(options.WindowID) || !options.ClosedAt.After(options.OpenedAt) {
		return Evidence{}, errors.New("sentinel evidence window rejected")
	}
	if options.Provenance != "synthetic" && options.Provenance != "real" {
		return Evidence{}, errors.New("sentinel evidence provenance rejected")
	}
	beforeByRef, err := snapshotsByRef(before)
	if err != nil {
		return Evidence{}, err
	}
	afterByRef, err := snapshotsByRef(after)
	if err != nil {
		return Evidence{}, err
	}
	window := ObservationWindow{
		WindowID: options.WindowID,
		OpenedAt: options.OpenedAt.UTC().Format(time.RFC3339Nano),
		ClosedAt: options.ClosedAt.UTC().Format(time.RFC3339Nano),
		State:    "closed",
	}
	evidence := Evidence{
		SchemaVersion:  EvidenceSchemaVersion,
		SuiteID:        options.SuiteID,
		SuiteDigest:    SuiteDigest(options.SuiteID, options.Tier),
		Tier:           options.Tier,
		ManifestDigest: manifest.Digest,
		Window:         window,
		WindowDigest:   observationWindowDigest(window),
		Optional:       make([]string, 0),
		Excluded:       make([]string, 0),
		Surfaces:       make([]SurfaceEvidence, 0, len(manifest.Surfaces)),
		Provenance:     options.Provenance,
	}
	for _, surface := range manifest.Surfaces {
		switch surface.Policy {
		case PolicyOptional:
			evidence.Optional = append(evidence.Optional, surface.LogicalRef)
		case PolicyExcluded:
			evidence.Excluded = append(evidence.Excluded, surface.LogicalRef)
			continue
		}
		beforeSnapshot, beforeOK := beforeByRef[surface.LogicalRef]
		afterSnapshot, afterOK := afterByRef[surface.LogicalRef]
		if !beforeOK || !afterOK {
			return Evidence{}, errors.New("sentinel evidence surface missing")
		}
		evidence.Surfaces = append(evidence.Surfaces, SurfaceEvidence{
			SurfaceID:     surface.SurfaceID,
			SurfaceDomain: string(surface.SurfaceDomain),
			LogicalRef:    surface.LogicalRef,
			Policy:        surface.Policy,
			BeforeStatus:  beforeSnapshot.Status,
			AfterStatus:   afterSnapshot.Status,
			BeforeToken:   beforeSnapshot.OpaqueState,
			AfterToken:    afterSnapshot.OpaqueState,
			BeforeReason:  beforeSnapshot.Reason,
			AfterReason:   afterSnapshot.Reason,
		})
	}
	sort.Strings(evidence.Optional)
	sort.Strings(evidence.Excluded)
	sort.Slice(evidence.Surfaces, func(i, j int) bool { return evidence.Surfaces[i].LogicalRef < evidence.Surfaces[j].LogicalRef })
	return evidence, nil
}

func ParseEvidence(data []byte) (Evidence, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return Evidence{}, errors.New("sentinel evidence rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var evidence Evidence
	if err := decoder.Decode(&evidence); err != nil {
		return Evidence{}, errors.New("sentinel evidence rejected")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Evidence{}, errors.New("sentinel evidence rejected")
	}
	return evidence, nil
}

func Evaluate(manifest ProtectedManifest, evidence Evidence) Evaluation {
	digest, digestErr := EvidenceDigest(evidence)
	harness := func(reason string) Evaluation {
		return Evaluation{Verdict: VerdictHarnessError, ExitCode: ExitHarnessError, Reason: reason, EvidenceDigest: digest}
	}
	if digestErr != nil || manifest.Digest == "" {
		return harness("evidence-binding-rejected")
	}
	canonicalManifest, err := canonicalProtectedManifest(manifest)
	if err != nil || sha256Digest(canonicalManifest) != manifest.Digest || validateProtectedManifest(manifest) != nil {
		return harness("manifest-binding-rejected")
	}
	if evidence.SchemaVersion != EvidenceSchemaVersion || evidence.SuiteID != manifest.SuiteID || evidence.SuiteDigest != SuiteDigest(evidence.SuiteID, evidence.Tier) || evidence.ManifestDigest != manifest.Digest || !validEvidenceTier(evidence.Tier) {
		return harness("evidence-binding-rejected")
	}
	if evidence.Provenance != "synthetic" && evidence.Provenance != "real" {
		return harness("evidence-provenance-rejected")
	}
	opened, openedErr := time.Parse(time.RFC3339Nano, evidence.Window.OpenedAt)
	closed, closedErr := time.Parse(time.RFC3339Nano, evidence.Window.ClosedAt)
	if openedErr != nil || closedErr != nil || !closed.After(opened) || evidence.Window.State != "closed" || !publicManifestID.MatchString(evidence.Window.WindowID) {
		return harness("observation-window-rejected")
	}
	if evidence.WindowDigest != observationWindowDigest(evidence.Window) {
		return harness("observation-window-binding-rejected")
	}
	wantOptional, wantExcluded := manifestPolicyLists(manifest)
	if !equalStrings(evidence.Optional, wantOptional) || !equalStrings(evidence.Excluded, wantExcluded) {
		return harness("surface-policy-substitution-rejected")
	}
	expected := make(map[string]ProtectedSurface)
	for _, surface := range manifest.Surfaces {
		if surface.Policy != PolicyExcluded {
			expected[surface.LogicalRef] = surface
		}
	}
	if len(evidence.Surfaces) != len(expected) {
		return harness("surface-evidence-count-rejected")
	}
	seen := make(map[string]struct{}, len(evidence.Surfaces))
	hasViolation := false
	hasIndeterminate := false
	warnings := make([]string, 0)
	for _, observed := range evidence.Surfaces {
		surface, ok := expected[observed.LogicalRef]
		if !ok {
			return harness("surface-evidence-substitution-rejected")
		}
		if _, duplicate := seen[observed.LogicalRef]; duplicate {
			return harness("surface-evidence-duplicate-rejected")
		}
		seen[observed.LogicalRef] = struct{}{}
		if observed.SurfaceID != surface.SurfaceID || observed.SurfaceDomain != string(surface.SurfaceDomain) || observed.Policy != surface.Policy {
			return harness("surface-evidence-substitution-rejected")
		}
		beforeComplete, beforeValid := validateObservation(observed.BeforeStatus, observed.BeforeToken, observed.BeforeReason)
		afterComplete, afterValid := validateObservation(observed.AfterStatus, observed.AfterToken, observed.AfterReason)
		if !beforeValid || !afterValid {
			return harness("surface-observation-rejected")
		}
		if surface.Policy == PolicyOptional {
			if !beforeComplete || !afterComplete {
				warnings = append(warnings, "optional-observation-incomplete")
			} else if observed.BeforeToken != observed.AfterToken {
				warnings = append(warnings, "optional-change-detected")
			}
			continue
		}
		if !beforeComplete || !afterComplete {
			hasIndeterminate = true
			continue
		}
		if observed.BeforeToken != observed.AfterToken {
			hasViolation = true
		}
	}
	if len(seen) != len(expected) {
		return harness("surface-evidence-missing")
	}
	sort.Strings(warnings)
	if hasViolation {
		return Evaluation{Verdict: VerdictViolation, ExitCode: ExitViolation, ChangeCode: ChangeDetectedCode, EvidenceDigest: digest}
	}
	if hasIndeterminate {
		return Evaluation{Verdict: VerdictIndeterminate, ExitCode: ExitIndeterminate, Reason: "required-observation-incomplete", EvidenceDigest: digest}
	}
	result := Evaluation{Verdict: VerdictPassed, ExitCode: ExitPassed, Warnings: warnings, EvidenceDigest: digest}
	if evidence.Provenance == "real" {
		if !validRealBinding(evidence, digest) {
			return Evaluation{Verdict: VerdictIndeterminate, ExitCode: ExitIndeterminate, Reason: "real-envelope-binding-missing", EvidenceDigest: digest}
		}
	}
	return result
}

func EvidenceDigest(evidence Evidence) (string, error) {
	copyEvidence := evidence
	copyEvidence.realBinding = nil
	copyEvidence.Optional = append([]string(nil), evidence.Optional...)
	copyEvidence.Excluded = append([]string(nil), evidence.Excluded...)
	copyEvidence.Surfaces = append([]SurfaceEvidence(nil), evidence.Surfaces...)
	sort.Strings(copyEvidence.Optional)
	sort.Strings(copyEvidence.Excluded)
	sort.Slice(copyEvidence.Surfaces, func(i, j int) bool { return copyEvidence.Surfaces[i].LogicalRef < copyEvidence.Surfaces[j].LogicalRef })
	canonical, err := json.Marshal(copyEvidence)
	if err != nil {
		return "", errors.New("sentinel evidence rejected")
	}
	return sha256Digest(canonical), nil
}

func SuiteDigest(suiteID, tier string) string {
	canonical, _ := json.Marshal(struct {
		SuiteID string `json:"suite_id"`
		Tier    string `json:"tier"`
	}{suiteID, tier})
	return sha256Digest(canonical)
}

func observationWindowDigest(window ObservationWindow) string {
	canonical, _ := json.Marshal(window)
	return sha256Digest(canonical)
}

func RequestClaim(evidence *Evidence, evaluation Evaluation, requested string) (string, error) {
	if evidence == nil || requested != ScopedUnchangedClaim || evaluation.Verdict != VerdictPassed || evaluation.ExitCode != ExitPassed {
		return "", errors.New("sentinel claim rejected")
	}
	digest, err := EvidenceDigest(*evidence)
	if err != nil || evaluation.EvidenceDigest != digest || !validRealBinding(*evidence, digest) {
		return "", errors.New("sentinel claim rejected")
	}
	evidence.realBinding = nil
	return ScopedUnchangedClaim, nil
}

func ConsumeClaim(evidence *Evidence, evaluation Evaluation, requested string) (ClaimMaterial, error) {
	claim, err := RequestClaim(evidence, evaluation, requested)
	if err != nil {
		return ClaimMaterial{}, err
	}
	return ClaimMaterial{
		Claim:          claim,
		EvidenceDigest: evaluation.EvidenceDigest,
		SuiteDigest:    evidence.SuiteDigest,
		ManifestDigest: evidence.ManifestDigest,
		Window:         evidence.Window,
		WindowDigest:   evidence.WindowDigest,
		Surfaces:       append([]SurfaceEvidence(nil), evidence.Surfaces...),
	}, nil
}

func MonotonicCombine(primary, after Verdict) Verdict {
	if primary != VerdictPassed {
		return primary
	}
	switch after {
	case VerdictPassed, VerdictViolation, VerdictIndeterminate, VerdictHarnessError:
		return after
	default:
		return VerdictHarnessError
	}
}

func bindRealEvidence(evidence *Evidence) error {
	if evidence == nil || evidence.Provenance != "real" {
		return errors.New("real evidence binding rejected")
	}
	digest, err := EvidenceDigest(*evidence)
	if err != nil {
		return err
	}
	evidence.realBinding = &realEvidenceBinding{
		ManifestDigest: evidence.ManifestDigest,
		SuiteDigest:    evidence.SuiteDigest,
		WindowID:       evidence.Window.WindowID,
		EvidenceDigest: digest,
	}
	return nil
}

func validRealBinding(evidence Evidence, digest string) bool {
	return evidence.realBinding != nil && evidence.realBinding.ManifestDigest == evidence.ManifestDigest && evidence.realBinding.SuiteDigest == evidence.SuiteDigest && evidence.realBinding.WindowID == evidence.Window.WindowID && evidence.realBinding.EvidenceDigest == digest
}

func snapshotsByRef(snapshot ProtectedSnapshot) (map[string]SurfaceSnapshot, error) {
	result := make(map[string]SurfaceSnapshot, len(snapshot.Surfaces))
	for _, surface := range snapshot.Surfaces {
		if _, duplicate := result[surface.LogicalRef]; duplicate {
			return nil, errors.New("duplicate surface snapshot rejected")
		}
		result[surface.LogicalRef] = surface
	}
	return result, nil
}

func validateObservation(status ObservationStatus, token string, reason IncompleteReason) (bool, bool) {
	switch status {
	case ObservationComplete:
		return true, validOpaqueToken(token) && reason == ""
	case ObservationIncomplete:
		return false, token == "" && validIncompleteReason(reason)
	default:
		return false, false
	}
}

func validOpaqueToken(token string) bool {
	if !strings.HasPrefix(token, "hmac-sha256:") || len(token) != len("hmac-sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(token, "hmac-sha256:"))
	return err == nil
}

func validIncompleteReason(reason IncompleteReason) bool {
	switch reason {
	case ReasonUnreadable, ReasonRace, ReasonOverflow, ReasonSymlinkEscape, ReasonWindow:
		return true
	default:
		return false
	}
}

func validEvidenceTier(tier string) bool {
	switch tier {
	case "offline-static", "isolated-integration", "real-sentinel-envelope":
		return true
	default:
		return false
	}
}

func manifestPolicyLists(manifest ProtectedManifest) ([]string, []string) {
	optional := make([]string, 0)
	excluded := make([]string, 0)
	for _, surface := range manifest.Surfaces {
		switch surface.Policy {
		case PolicyOptional:
			optional = append(optional, surface.LogicalRef)
		case PolicyExcluded:
			excluded = append(excluded, surface.LogicalRef)
		}
	}
	sort.Strings(optional)
	sort.Strings(excluded)
	return optional, excluded
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
