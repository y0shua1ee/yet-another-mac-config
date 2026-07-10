package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const SchemaVersion = "1.0.0"

type Kind string

const (
	DesiredState         Kind = "desired-state"
	ObservedState        Kind = "observed-state"
	GeneratedPlan        Kind = "generated-plan"
	AppliedReceipt       Kind = "applied-receipt"
	VerificationEvidence Kind = "verification-evidence"
	ReadinessReport      Kind = "readiness-report"
)

var closedKinds = map[Kind]struct{}{
	DesiredState:         {},
	ObservedState:        {},
	GeneratedPlan:        {},
	AppliedReceipt:       {},
	VerificationEvidence: {},
	ReadinessReport:      {},
}

type RunMetadata struct {
	RunID   string `json:"run_id"`
	Tier    string `json:"tier"`
	SuiteID string `json:"suite_id"`
}

type Producer struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Provenance struct {
	Mode         string   `json:"mode"`
	InputDigests []string `json:"input_digests"`
}

type Envelope struct {
	Kind          Kind            `json:"kind"`
	SchemaVersion string          `json:"schema_version"`
	Run           RunMetadata     `json:"run"`
	Producer      Producer        `json:"producer"`
	Provenance    Provenance      `json:"provenance"`
	Payload       json.RawMessage `json:"payload"`
	ContentDigest string          `json:"content_digest"`
}

type envelopeCore struct {
	Kind          Kind            `json:"kind"`
	SchemaVersion string          `json:"schema_version"`
	Run           RunMetadata     `json:"run"`
	Producer      Producer        `json:"producer"`
	Provenance    Provenance      `json:"provenance"`
	Payload       json.RawMessage `json:"payload"`
}

func New(kind Kind, run RunMetadata, provenance Provenance, payload any) ([]byte, Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil || len(payloadBytes) == 0 || payloadBytes[0] != '{' {
		return nil, Envelope{}, errors.New("artifact payload rejected")
	}

	envelope := Envelope{
		Kind:          kind,
		SchemaVersion: SchemaVersion,
		Run:           run,
		Producer: Producer{
			ID:      "yamc-safety",
			Version: "0.1.0",
		},
		Provenance: provenance,
		Payload:    payloadBytes,
	}
	if err := validateCommon(envelope); err != nil {
		return nil, Envelope{}, err
	}

	digest, err := digestCore(envelope)
	if err != nil {
		return nil, Envelope{}, err
	}
	envelope.ContentDigest = digest
	canonical, err := json.Marshal(envelope)
	if err != nil {
		return nil, Envelope{}, errors.New("artifact encoding failed")
	}
	return canonical, envelope, nil
}

func DecodeAndValidate(canonical []byte) (Envelope, error) {
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var envelope Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return Envelope{}, errors.New("artifact envelope rejected")
	}
	if err := requireEOF(decoder); err != nil {
		return Envelope{}, err
	}
	if err := validateCommon(envelope); err != nil {
		return Envelope{}, err
	}
	wantDigest, err := digestCore(envelope)
	if err != nil || envelope.ContentDigest != wantDigest {
		return Envelope{}, errors.New("artifact digest rejected")
	}
	reencoded, err := json.Marshal(envelope)
	if err != nil || !bytes.Equal(reencoded, canonical) {
		return Envelope{}, errors.New("artifact bytes are not canonical")
	}
	return envelope, nil
}

func DigestValue(value any) (string, error) {
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", errors.New("canonical value rejected")
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func IsDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	raw := strings.TrimPrefix(value, "sha256:")
	decoded, err := hex.DecodeString(raw)
	return err == nil && len(decoded) == sha256.Size
}

func validateCommon(envelope Envelope) error {
	if _, ok := closedKinds[envelope.Kind]; !ok {
		return errors.New("artifact kind rejected")
	}
	if envelope.SchemaVersion != SchemaVersion {
		return errors.New("artifact schema rejected")
	}
	if envelope.Run.RunID == "" || envelope.Run.Tier == "" || envelope.Run.SuiteID == "" {
		return errors.New("artifact run metadata rejected")
	}
	if envelope.Producer.ID != "yamc-safety" || envelope.Producer.Version == "" {
		return errors.New("artifact producer rejected")
	}
	if envelope.Provenance.Mode != "synthetic" {
		return errors.New("artifact provenance rejected")
	}
	for _, digest := range envelope.Provenance.InputDigests {
		if !IsDigest(digest) {
			return errors.New("artifact input digest rejected")
		}
	}
	if len(envelope.Payload) == 0 || envelope.Payload[0] != '{' || !json.Valid(envelope.Payload) {
		return errors.New("artifact payload rejected")
	}
	return nil
}

func digestCore(envelope Envelope) (string, error) {
	core := envelopeCore{
		Kind:          envelope.Kind,
		SchemaVersion: envelope.SchemaVersion,
		Run:           envelope.Run,
		Producer:      envelope.Producer,
		Provenance:    envelope.Provenance,
		Payload:       envelope.Payload,
	}
	return DigestValue(core)
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values rejected")
	}
	return nil
}
