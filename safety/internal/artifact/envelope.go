package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"example.invalid/yamc/safety/internal/privacy"
)

const SchemaVersion = "1.0.0"

var publicIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,95}$`)

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
	Mode         string       `json:"mode"`
	InputDigests []string     `json:"input_digests"`
	StorageClass StorageClass `json:"storage_class"`
	Lifecycle    Lifecycle    `json:"lifecycle"`
}

type Envelope struct {
	Kind          Kind            `json:"kind"`
	SchemaVersion string          `json:"schema_version"`
	Run           RunMetadata     `json:"run"`
	Producer      Producer        `json:"producer"`
	Provenance    Provenance      `json:"provenance"`
	Payload       json.RawMessage `json:"payload"`
	ContentDigest string          `json:"content_digest"`
	StorageClass  StorageClass    `json:"-"`
	Lifecycle     Lifecycle       `json:"-"`
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
	options, err := DefaultBuildOptions(kind, time.Now())
	if err != nil {
		return nil, Envelope{}, err
	}
	return NewWithOptions(kind, run, provenance, payload, options)
}

func NewWithOptions(kind Kind, run RunMetadata, provenance Provenance, payload any, options BuildOptions) ([]byte, Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, Envelope{}, contractError(CodePayloadRejected, "/payload")
	}
	payloadBytes, err = Canonicalize(payloadBytes)
	if err != nil || len(payloadBytes) == 0 || payloadBytes[0] != '{' {
		return nil, Envelope{}, contractError(CodePayloadRejected, "/payload")
	}

	provenance.StorageClass = options.StorageClass
	provenance.Lifecycle = options.Lifecycle
	envelope := Envelope{
		Kind:          kind,
		SchemaVersion: SchemaVersion,
		Run:           run,
		Producer: Producer{
			ID:      "yamc-safety",
			Version: "0.1.0",
		},
		Provenance:   provenance,
		Payload:      payloadBytes,
		StorageClass: options.StorageClass,
		Lifecycle:    options.Lifecycle,
	}
	if err := validateEnvelope(envelope); err != nil {
		return nil, Envelope{}, err
	}
	digest, err := digestCore(envelope)
	if err != nil {
		return nil, Envelope{}, err
	}
	envelope.ContentDigest = digest
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, Envelope{}, contractError(CodeCanonicalRejected, "/")
	}
	canonical, err := Canonicalize(encoded)
	if err != nil {
		return nil, Envelope{}, err
	}
	return canonical, envelope, nil
}

func Validate(expectedKind Kind, canonical []byte) (Envelope, error) {
	envelope, err := decodeValidated(canonical)
	if err != nil {
		return Envelope{}, err
	}
	if expectedKind != "" && envelope.Kind != expectedKind {
		return Envelope{}, contractError(CodeExpectedKindMismatch, "/kind")
	}
	return envelope, nil
}

func DecodeAndValidate(canonical []byte) (Envelope, error) {
	envelope, err := decodeValidated(canonical)
	if err != nil {
		return Envelope{}, err
	}
	if envelope.StorageClass == SyntheticGolden {
		return Envelope{}, contractError(CodeStorageReadOnly, "/storage_class")
	}
	return envelope, nil
}

func decodeValidated(data []byte) (Envelope, error) {
	canonical, err := Canonicalize(data)
	if err != nil {
		return Envelope{}, err
	}
	if !bytes.Equal(canonical, data) {
		return Envelope{}, contractError(CodeCanonicalRejected, "/")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return Envelope{}, contractError(CodeEnvelopeRejected, "/")
	}
	if err := requireEOF(decoder); err != nil {
		return Envelope{}, err
	}
	envelope.StorageClass = envelope.Provenance.StorageClass
	envelope.Lifecycle = envelope.Provenance.Lifecycle
	if err := validateEnvelope(envelope); err != nil {
		return Envelope{}, err
	}
	wantDigest, err := digestCore(envelope)
	if err != nil || envelope.ContentDigest != wantDigest {
		return Envelope{}, contractError(CodeDigestRejected, "/content_digest")
	}
	return envelope, nil
}

func DigestValue(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", contractError(CodeCanonicalRejected, "/")
	}
	canonical, err := Canonicalize(encoded)
	if err != nil {
		return "", err
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
	return err == nil && len(decoded) == sha256.Size && raw == strings.ToLower(raw)
}

// IsPublicID accepts only bounded, non-secret structural identifiers.
func IsPublicID(value string) bool {
	if !publicIDPattern.MatchString(value) {
		return false
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"api-key", "api_key", "private-key", "private_key"} {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	for _, marker := range []string{"secret", "token", "password", "credential", "private", "provider", "username", "hostname", "api-key", "apikey"} {
		for _, segment := range strings.FieldsFunc(lower, func(r rune) bool { return r == '.' || r == '_' || r == '-' }) {
			if segment == marker {
				return false
			}
		}
	}
	for _, prefix := range []string{"sk-", "ghp_", "gho_", "ghu_", "ghs_", "ghr_", "xox", "akia", "eyj"} {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	return true
}

// NewRunMetadata derives the public run identifier from trusted structural input.
// Callers provide a registered suite and cannot persist a human or machine identity as run metadata.
func NewRunMetadata(seed []byte, tier, suiteID string) (RunMetadata, error) {
	if len(seed) == 0 || !privacy.IsRegisteredSuiteID(suiteID) || (tier != "offline-static" && tier != "isolated-integration" && tier != "real-sentinel-envelope") {
		return RunMetadata{}, contractError(CodeEnvelopeRejected, "/run")
	}
	domain := "synthetic-run"
	if tier == "real-sentinel-envelope" {
		domain = "real-run"
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("yamc-safety-run-v1\x00"))
	_, _ = hash.Write([]byte(tier))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(suiteID))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(seed)
	return RunMetadata{RunID: domain + "-" + hex.EncodeToString(hash.Sum(nil)[:24]), Tier: tier, SuiteID: suiteID}, nil
}

func validateEnvelope(envelope Envelope) error {
	if _, ok := kindRegistry[envelope.Kind]; !ok {
		return contractError(CodeKindRejected, "/kind")
	}
	if envelope.SchemaVersion != SchemaVersion {
		return contractError(CodeSchemaRejected, "/schema_version")
	}
	if !privacy.IsTrustedRunID(envelope.Run.RunID) || !privacy.IsRegisteredSuiteID(envelope.Run.SuiteID) || (envelope.Run.Tier != "offline-static" && envelope.Run.Tier != "isolated-integration" && envelope.Run.Tier != "real-sentinel-envelope") {
		return contractError(CodeEnvelopeRejected, "/run")
	}
	if envelope.Producer.ID != "yamc-safety" || envelope.Producer.Version == "" {
		return contractError(CodeEnvelopeRejected, "/producer")
	}
	if envelope.Provenance.Mode != "synthetic" && envelope.Provenance.Mode != "runtime" && envelope.Provenance.Mode != "real-run" {
		return contractError(CodeProvenanceRejected, "/provenance/mode")
	}
	if envelope.Provenance.InputDigests == nil {
		return contractError(CodeProvenanceRejected, "/provenance/input_digests")
	}
	for index, digest := range envelope.Provenance.InputDigests {
		if !IsDigest(digest) {
			return contractError(CodeProvenanceRejected, joinPointer("/provenance/input_digests", strconv.Itoa(index)))
		}
	}
	if err := ValidatePayload(envelope.Kind, envelope.Payload); err != nil {
		return err
	}
	return validatePolicy(envelope)
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
