package artifact

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type Kind string

const (
	DesiredState         Kind = "desired-state"
	ObservedState        Kind = "observed-state"
	GeneratedPlan        Kind = "generated-plan"
	AppliedReceipt       Kind = "applied-receipt"
	VerificationEvidence Kind = "verification-evidence"
	ReadinessReport      Kind = "readiness-report"
)

type StorageClass string

const (
	SyntheticGolden    StorageClass = "synthetic-golden"
	ExternalLocalState StorageClass = "external-local-state"
)

type RetentionPolicy string

const (
	Snapshot24Hours    RetentionPolicy = "snapshot-24h"
	AppendOnlyPlan     RetentionPolicy = "append-only-plan"
	AppendOnlyEvidence RetentionPolicy = "append-only-evidence-bundle"
	SnapshotLifetime                   = 24 * time.Hour
)

type TerminalState string

const (
	TerminalNone        TerminalState = ""
	TerminalNonterminal TerminalState = "nonterminal"
	TerminalApplied     TerminalState = "applied"
	TerminalAbandoned   TerminalState = "abandoned"
)

type Lifecycle struct {
	Retention               RetentionPolicy `json:"retention"`
	CreatedAt               string          `json:"created_at"`
	ExpiresAt               string          `json:"expires_at,omitempty"`
	TerminalState           TerminalState   `json:"terminal_state,omitempty"`
	TerminalReceiptDigest   string          `json:"terminal_receipt_digest,omitempty"`
	AbandonmentRecordDigest string          `json:"abandonment_record_digest,omitempty"`
}

type BuildOptions struct {
	StorageClass StorageClass
	Lifecycle    Lifecycle
}

type Fact struct {
	Ref   string `json:"ref"`
	State string `json:"state"`
}

type DesiredPayload struct {
	Profile      string `json:"profile"`
	Declarations []Fact `json:"declarations"`
}

type ObservedPayload struct {
	Scope string `json:"scope"`
	Facts []Fact `json:"facts"`
}

type GeneratedPlanPayload struct {
	DesiredDigest                string   `json:"desired_digest"`
	ObservedDigest               string   `json:"observed_digest"`
	ExpectedPostconditionsDigest string   `json:"expected_postconditions_digest"`
	OperationIDs                 []string `json:"operation_ids"`
}

type AppliedReceiptPayload struct {
	PlanDigest   string   `json:"plan_digest"`
	Mode         string   `json:"mode"`
	OperationIDs []string `json:"operation_ids"`
	Outcome      string   `json:"outcome"`
}

type FreshObserved struct {
	Scope               string `json:"scope"`
	State               string `json:"state"`
	SourceReceiptDigest string `json:"source_receipt_digest,omitempty"`
	ContentDigest       string `json:"content_digest"`
}

type VerificationEvidencePayload struct {
	PlanDigest                   string        `json:"plan_digest,omitempty"`
	ReceiptDigest                string        `json:"receipt_digest,omitempty"`
	DesiredDigest                string        `json:"desired_digest,omitempty"`
	ExpectedPostconditionsDigest string        `json:"expected_postconditions_digest"`
	FreshObservedDigest          string        `json:"fresh_observed_digest"`
	FreshObserved                FreshObserved `json:"fresh_observed"`
	ManifestDigest               string        `json:"manifest_digest,omitempty"`
	SentinelBeforeDigest         string        `json:"sentinel_before_digest,omitempty"`
	SentinelAfterDigest          string        `json:"sentinel_after_digest,omitempty"`
}

type ReadinessReportPayload struct {
	EvidenceDigest  string   `json:"evidence_digest,omitempty"`
	EvidenceDigests []string `json:"evidence_digests,omitempty"`
	State           string   `json:"state"`
}

type kindContract struct {
	retention     RetentionPolicy
	storage       [2]StorageClass
	validate      func(json.RawMessage) error
	terminalState bool
}

var kindRegistry = map[Kind]kindContract{
	DesiredState:         {retention: Snapshot24Hours, storage: [2]StorageClass{SyntheticGolden, ExternalLocalState}, validate: validateDesiredPayload},
	ObservedState:        {retention: Snapshot24Hours, storage: [2]StorageClass{SyntheticGolden, ExternalLocalState}, validate: validateObservedPayload},
	GeneratedPlan:        {retention: AppendOnlyPlan, storage: [2]StorageClass{SyntheticGolden, ExternalLocalState}, validate: validateGeneratedPlanPayload, terminalState: true},
	AppliedReceipt:       {retention: AppendOnlyEvidence, storage: [2]StorageClass{SyntheticGolden, ExternalLocalState}, validate: validateAppliedReceiptPayload},
	VerificationEvidence: {retention: AppendOnlyEvidence, storage: [2]StorageClass{SyntheticGolden, ExternalLocalState}, validate: validateVerificationEvidencePayload},
	ReadinessReport:      {retention: AppendOnlyEvidence, storage: [2]StorageClass{SyntheticGolden, ExternalLocalState}, validate: validateReadinessReportPayload},
}

func RegisteredKinds() []Kind {
	kinds := make([]Kind, 0, len(kindRegistry))
	for kind := range kindRegistry {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(left, right int) bool { return kinds[left] < kinds[right] })
	return kinds
}

func DefaultBuildOptions(kind Kind, now time.Time) (BuildOptions, error) {
	contract, ok := kindRegistry[kind]
	if !ok {
		return BuildOptions{}, contractError(CodeKindRejected, "/kind")
	}
	now = now.UTC().Truncate(time.Second)
	lifecycle := Lifecycle{Retention: contract.retention, CreatedAt: now.Format(time.RFC3339)}
	switch contract.retention {
	case Snapshot24Hours:
		lifecycle.ExpiresAt = now.Add(SnapshotLifetime).Format(time.RFC3339)
	case AppendOnlyPlan:
		lifecycle.TerminalState = TerminalNonterminal
	}
	return BuildOptions{StorageClass: ExternalLocalState, Lifecycle: lifecycle}, nil
}

func ValidatePayload(kind Kind, payload json.RawMessage) error {
	contract, ok := kindRegistry[kind]
	if !ok {
		return contractError(CodeKindRejected, "/kind")
	}
	return contract.validate(payload)
}

func validatePolicy(envelope Envelope) error {
	contract, ok := kindRegistry[envelope.Kind]
	if !ok {
		return contractError(CodeKindRejected, "/kind")
	}
	allowedClass := false
	for _, storageClass := range contract.storage {
		if envelope.StorageClass == storageClass {
			allowedClass = true
		}
	}
	if !allowedClass {
		return contractError(CodeStorageRejected, "/storage_class")
	}
	if envelope.StorageClass == SyntheticGolden && envelope.Provenance.Mode != "synthetic" {
		return contractError(CodeStorageRejected, "/storage_class")
	}
	if envelope.Lifecycle.Retention != contract.retention {
		return contractError(CodeLifecycleRejected, "/lifecycle/retention")
	}
	createdAt, err := time.Parse(time.RFC3339, envelope.Lifecycle.CreatedAt)
	if err != nil || envelope.Lifecycle.CreatedAt != createdAt.UTC().Format(time.RFC3339) {
		return contractError(CodeLifecycleRejected, "/lifecycle/created_at")
	}
	switch contract.retention {
	case Snapshot24Hours:
		expiresAt, parseErr := time.Parse(time.RFC3339, envelope.Lifecycle.ExpiresAt)
		if parseErr != nil || !expiresAt.Equal(createdAt.Add(SnapshotLifetime)) {
			return contractError(CodeLifecycleRejected, "/lifecycle/expires_at")
		}
		if envelope.Lifecycle.TerminalState != TerminalNone || envelope.Lifecycle.TerminalReceiptDigest != "" || envelope.Lifecycle.AbandonmentRecordDigest != "" {
			return contractError(CodeLifecycleRejected, "/lifecycle/terminal_state")
		}
	case AppendOnlyPlan:
		if envelope.Lifecycle.ExpiresAt != "" {
			return contractError(CodeLifecycleRejected, "/lifecycle/expires_at")
		}
		switch envelope.Lifecycle.TerminalState {
		case TerminalNonterminal:
			if envelope.Lifecycle.TerminalReceiptDigest != "" || envelope.Lifecycle.AbandonmentRecordDigest != "" {
				return contractError(CodeLifecycleRejected, "/lifecycle/terminal_state")
			}
		case TerminalApplied:
			if !IsDigest(envelope.Lifecycle.TerminalReceiptDigest) || envelope.Lifecycle.AbandonmentRecordDigest != "" {
				return contractError(CodeLifecycleRejected, "/lifecycle/terminal_receipt_digest")
			}
		case TerminalAbandoned:
			if !IsDigest(envelope.Lifecycle.AbandonmentRecordDigest) || envelope.Lifecycle.TerminalReceiptDigest != "" {
				return contractError(CodeLifecycleRejected, "/lifecycle/abandonment_record_digest")
			}
		default:
			return contractError(CodeLifecycleRejected, "/lifecycle/terminal_state")
		}
	case AppendOnlyEvidence:
		if envelope.Lifecycle.ExpiresAt != "" || envelope.Lifecycle.TerminalState != TerminalNone || envelope.Lifecycle.TerminalReceiptDigest != "" || envelope.Lifecycle.AbandonmentRecordDigest != "" {
			return contractError(CodeLifecycleRejected, "/lifecycle")
		}
	default:
		return contractError(CodeLifecycleRejected, "/lifecycle/retention")
	}
	return nil
}

func validateDesiredPayload(raw json.RawMessage) error {
	var payload DesiredPayload
	if err := decodeClosed(raw, &payload, "/payload"); err != nil {
		return err
	}
	if payload.Profile == "" || len(payload.Declarations) == 0 || !validFacts(payload.Declarations) {
		return contractError(CodePayloadRejected, "/payload")
	}
	return nil
}

func validateObservedPayload(raw json.RawMessage) error {
	var payload ObservedPayload
	if err := decodeClosed(raw, &payload, "/payload"); err != nil {
		return err
	}
	if payload.Scope == "" || len(payload.Facts) == 0 || !validFacts(payload.Facts) {
		return contractError(CodePayloadRejected, "/payload")
	}
	return nil
}

func validateGeneratedPlanPayload(raw json.RawMessage) error {
	var payload GeneratedPlanPayload
	if err := decodeClosed(raw, &payload, "/payload"); err != nil {
		return err
	}
	if !IsDigest(payload.DesiredDigest) || !IsDigest(payload.ObservedDigest) || !IsDigest(payload.ExpectedPostconditionsDigest) || !validOperationIDs(payload.OperationIDs) {
		return contractError(CodePayloadRejected, "/payload")
	}
	return nil
}

func validateAppliedReceiptPayload(raw json.RawMessage) error {
	var payload AppliedReceiptPayload
	if err := decodeClosed(raw, &payload, "/payload"); err != nil {
		return err
	}
	if !IsDigest(payload.PlanDigest) || (payload.Mode != "synthetic" && payload.Mode != "real-run") || !validOperationIDs(payload.OperationIDs) || payload.Outcome == "" {
		return contractError(CodePayloadRejected, "/payload")
	}
	return nil
}

func validateVerificationEvidencePayload(raw json.RawMessage) error {
	var payload VerificationEvidencePayload
	if err := decodeClosed(raw, &payload, "/payload"); err != nil {
		return err
	}
	for _, digest := range []string{payload.ExpectedPostconditionsDigest, payload.FreshObservedDigest, payload.FreshObserved.ContentDigest} {
		if !IsDigest(digest) {
			return contractError(CodePayloadRejected, "/payload")
		}
	}
	for _, optionalDigest := range []string{payload.PlanDigest, payload.ReceiptDigest, payload.DesiredDigest, payload.FreshObserved.SourceReceiptDigest, payload.ManifestDigest, payload.SentinelBeforeDigest, payload.SentinelAfterDigest} {
		if optionalDigest != "" && !IsDigest(optionalDigest) {
			return contractError(CodePayloadRejected, "/payload")
		}
	}
	if payload.FreshObserved.Scope == "" || payload.FreshObserved.State == "" || payload.FreshObservedDigest != payload.FreshObserved.ContentDigest {
		return contractError(CodePayloadRejected, "/payload/fresh_observed")
	}
	return nil
}

func validateReadinessReportPayload(raw json.RawMessage) error {
	var payload ReadinessReportPayload
	if err := decodeClosed(raw, &payload, "/payload"); err != nil {
		return err
	}
	if payload.State == "" || (payload.EvidenceDigest == "") == (len(payload.EvidenceDigests) == 0) {
		return contractError(CodePayloadRejected, "/payload")
	}
	if payload.EvidenceDigest != "" && !IsDigest(payload.EvidenceDigest) {
		return contractError(CodePayloadRejected, "/payload/evidence_digest")
	}
	for _, digest := range payload.EvidenceDigests {
		if !IsDigest(digest) {
			return contractError(CodePayloadRejected, "/payload/evidence_digests")
		}
	}
	return nil
}

func validFacts(facts []Fact) bool {
	for _, fact := range facts {
		if fact.Ref == "" || fact.State == "" {
			return false
		}
	}
	return true
}

func validOperationIDs(operationIDs []string) bool {
	if len(operationIDs) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(operationIDs))
	for _, operationID := range operationIDs {
		if operationID == "" || strings.ContainsAny(operationID, " \t\r\n") {
			return false
		}
		if _, exists := seen[operationID]; exists {
			return false
		}
		seen[operationID] = struct{}{}
	}
	return true
}
