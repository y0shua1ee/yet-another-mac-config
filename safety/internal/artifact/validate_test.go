package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

type kindCaseFile struct {
	SchemaVersion string     `json:"schema_version"`
	Cases         []kindCase `json:"cases"`
}

type kindCase struct {
	Kind    Kind            `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

func TestArtifactKinds(t *testing.T) {
	cases := loadKindCases(t)
	assertClosedKindRegistry(t, cases)
	assertValidPairsAndCrossKindRejection(t, cases)
	assertRestrictedCanonicalJSON(t)
	assertEnvelopeRejectionsAndNoPersistence(t, cases[0])
	assertClosedLifecyclePolicy(t, cases)
	assertRunnerRouteContract(t)
}

func assertClosedKindRegistry(t *testing.T, cases []kindCase) {
	t.Helper()
	want := []Kind{DesiredState, ObservedState, GeneratedPlan, AppliedReceipt, VerificationEvidence, ReadinessReport}
	got := RegisteredKinds()
	sort.Slice(want, func(left, right int) bool { return want[left] < want[right] })
	if len(got) != 6 || len(cases) != 6 {
		t.Fatalf("EXPECTED_RED: artifact-kind-behavior-missing")
	}
	for index := range want {
		if got[index] != want[index] || cases[index].Kind == "" {
			t.Fatalf("EXPECTED_RED: artifact-kind-behavior-missing")
		}
	}
}

func assertValidPairsAndCrossKindRejection(t *testing.T, cases []kindCase) {
	t.Helper()
	run := syntheticRun()
	provenance := syntheticProvenance()
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)

	for _, testCase := range cases {
		for _, storageClass := range []StorageClass{SyntheticGolden, ExternalLocalState} {
			t.Run(string(testCase.Kind)+"/"+string(storageClass), func(t *testing.T) {
				options := optionsFor(t, testCase.Kind, storageClass, now)
				canonical, envelope, err := NewWithOptions(testCase.Kind, run, provenance, testCase.Payload, options)
				if err != nil {
					t.Fatalf("valid class-kind pair rejected")
				}
				validated, err := Validate(testCase.Kind, canonical)
				if err != nil || validated.ContentDigest != envelope.ContentDigest {
					t.Fatalf("valid class-kind pair failed validation")
				}
				if storageClass == SyntheticGolden {
					_, err = DecodeAndValidate(canonical)
					assertContractError(t, err, CodeStorageReadOnly, "/storage_class")
				}
			})
		}

		for _, target := range cases {
			if target.Kind == testCase.Kind {
				continue
			}
			t.Run(string(testCase.Kind)+"-as-"+string(target.Kind), func(t *testing.T) {
				options := optionsFor(t, target.Kind, ExternalLocalState, now)
				if _, _, err := NewWithOptions(target.Kind, run, provenance, testCase.Payload, options); err == nil {
					t.Fatalf("EXPECTED_RED: artifact-kind-behavior-missing")
				}
			})
		}

		canonical, _, err := NewWithOptions(testCase.Kind, run, provenance, testCase.Payload, optionsFor(t, testCase.Kind, ExternalLocalState, now))
		if err != nil {
			t.Fatalf("valid artifact setup failed")
		}
		wrongExpected := cases[(indexOfKind(cases, testCase.Kind)+1)%len(cases)].Kind
		_, err = Validate(wrongExpected, canonical)
		assertContractError(t, err, CodeExpectedKindMismatch, "/kind")
	}
}

func assertRestrictedCanonicalJSON(t *testing.T) {
	t.Helper()
	left, err := Canonicalize([]byte(`{"b":1,"a":{"z":3,"y":[1,2]}}`))
	if err != nil {
		t.Fatalf("canonicalization rejected valid JSON")
	}
	right, err := Canonicalize([]byte(`{ "a" : { "y" : [1,2], "z" : 3 }, "b" : 1 }`))
	if err != nil || !bytes.Equal(left, right) {
		t.Fatalf("object key ordering changed canonical bytes")
	}
	ordered, _ := Canonicalize([]byte(`{"items":[1,2]}`))
	reordered, _ := Canonicalize([]byte(`{"items":[2,1]}`))
	if bytes.Equal(ordered, reordered) {
		t.Fatalf("array order lost integrity")
	}
	leftDigest, _ := DigestValue(json.RawMessage(left))
	rightDigest, _ := DigestValue(json.RawMessage(right))
	orderedDigest, _ := DigestValue(json.RawMessage(ordered))
	reorderedDigest, _ := DigestValue(json.RawMessage(reordered))
	if leftDigest != rightDigest || orderedDigest == reorderedDigest {
		t.Fatalf("canonical digest semantics rejected")
	}

	invalidCases := []struct {
		name    string
		data    []byte
		code    ErrorCode
		pointer string
	}{
		{"duplicate", []byte(`{"a":1,"a":2}`), CodeJSONDuplicateKey, "/a"},
		{"float", []byte(`{"a":1.0}`), CodeJSONInvalidNumber, "/a"},
		{"exponent", []byte(`{"a":1e2}`), CodeJSONInvalidNumber, "/a"},
		{"nan-like", []byte(`{"a":NaN}`), CodeJSONInvalid, "/a"},
		{"trailing", []byte(`{"a":1}{"b":2}`), CodeJSONTrailingValue, "/"},
		{"utf8", []byte{'{', '"', 'a', '"', ':', '"', 0xff, '"', '}'}, CodeJSONInvalidUTF8, "/"},
	}
	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Canonicalize(testCase.data)
			assertContractError(t, err, testCase.code, testCase.pointer)
		})
	}
}

func assertEnvelopeRejectionsAndNoPersistence(t *testing.T, validCase kindCase) {
	t.Helper()
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	canonical, envelope, err := NewWithOptions(validCase.Kind, syntheticRun(), syntheticProvenance(), validCase.Payload, optionsFor(t, validCase.Kind, ExternalLocalState, now))
	if err != nil {
		t.Fatalf("valid artifact setup failed")
	}
	if digest, err := digestCore(envelope); err != nil || digest != envelope.ContentDigest {
		t.Fatalf("digest does not cover the envelope core")
	}
	changedDigestOnly := envelope
	changedDigestOnly.ContentDigest = strings.Repeat("f", len(envelope.ContentDigest))
	if digest, _ := digestCore(changedDigestOnly); digest != envelope.ContentDigest {
		t.Fatalf("content_digest was included in its own digest domain")
	}
	changedCore := envelope
	changedCore.Run.RunID = "synthetic-run-kind-contracts-changed"
	if digest, _ := digestCore(changedCore); digest == envelope.ContentDigest {
		t.Fatalf("envelope field omitted from digest domain")
	}

	invalid := []struct {
		name    string
		data    []byte
		code    ErrorCode
		pointer string
	}{
		{"unknown-kind", mutateEnvelope(t, canonical, func(value map[string]any) { value["kind"] = "unknown-kind" }), CodeKindRejected, "/kind"},
		{"unknown-version", mutateEnvelope(t, canonical, func(value map[string]any) { value["schema_version"] = "9.9.9" }), CodeSchemaRejected, "/schema_version"},
		{"unknown-envelope-field", mutateEnvelope(t, canonical, func(value map[string]any) { value["unexpected"] = "synthetic" }), CodeEnvelopeRejected, "/"},
		{"missing-provenance", mutateEnvelope(t, canonical, func(value map[string]any) { delete(value, "provenance") }), CodeProvenanceRejected, "/provenance/mode"},
		{"unknown-payload-field", mutateEnvelope(t, canonical, func(value map[string]any) { value["payload"].(map[string]any)["unexpected"] = "synthetic" }), CodePayloadRejected, "/payload"},
		{"digest-mismatch", mutateEnvelope(t, canonical, func(value map[string]any) { value["content_digest"] = "sha256:" + strings.Repeat("f", 64) }), CodeDigestRejected, "/content_digest"},
		{"duplicate-before-canonical", []byte(`{"kind":"desired-state","kind":"observed-state"}`), CodeJSONDuplicateKey, "/kind"},
	}

	repositoryRoot := repositoryRoot(t)
	storeRoot := filepath.Join(t.TempDir(), "invalid-store")
	store, err := NewStore(storeRoot, repositoryRoot)
	if err != nil {
		t.Fatalf("external invalid-store setup failed")
	}
	for _, testCase := range invalid {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Validate(validCase.Kind, testCase.data)
			assertContractError(t, err, testCase.code, testCase.pointer)
			if _, err := store.Write(testCase.data); err == nil {
				t.Fatalf("invalid artifact reached persistence")
			}
			assertStoreEmpty(t, storeRoot)
		})
	}
}

func assertClosedLifecyclePolicy(t *testing.T, cases []kindCase) {
	t.Helper()
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	baseRun := syntheticRun()
	baseProvenance := syntheticProvenance()

	for _, testCase := range cases {
		t.Run(string(testCase.Kind)+"/runtime-external", func(t *testing.T) {
			provenance := Provenance{Mode: "runtime", InputDigests: []string{}}
			if _, _, err := NewWithOptions(testCase.Kind, baseRun, provenance, testCase.Payload, optionsFor(t, testCase.Kind, ExternalLocalState, now)); err != nil {
				t.Fatalf("runtime external-local-state pair rejected")
			}
		})
	}

	wrongClass := optionsFor(t, DesiredState, ExternalLocalState, now)
	wrongClass.StorageClass = StorageClass("caller-defined")
	_, _, err := NewWithOptions(DesiredState, baseRun, baseProvenance, cases[indexOfKind(cases, DesiredState)].Payload, wrongClass)
	assertContractError(t, err, CodeStorageRejected, "/storage_class")

	wrongRetention := optionsFor(t, DesiredState, ExternalLocalState, now)
	wrongRetention.Lifecycle.Retention = RetentionPolicy("caller-defined")
	_, _, err = NewWithOptions(DesiredState, baseRun, baseProvenance, cases[indexOfKind(cases, DesiredState)].Payload, wrongRetention)
	assertContractError(t, err, CodeLifecycleRejected, "/lifecycle/retention")

	wrongTTL := optionsFor(t, ObservedState, ExternalLocalState, now)
	wrongTTL.Lifecycle.ExpiresAt = now.Add(23 * time.Hour).Format(time.RFC3339)
	_, _, err = NewWithOptions(ObservedState, baseRun, baseProvenance, cases[indexOfKind(cases, ObservedState)].Payload, wrongTTL)
	assertContractError(t, err, CodeLifecycleRejected, "/lifecycle/expires_at")

	wrongTerminal := optionsFor(t, GeneratedPlan, ExternalLocalState, now)
	wrongTerminal.Lifecycle.TerminalState = TerminalState("caller-defined")
	_, _, err = NewWithOptions(GeneratedPlan, baseRun, baseProvenance, cases[indexOfKind(cases, GeneratedPlan)].Payload, wrongTerminal)
	assertContractError(t, err, CodeLifecycleRejected, "/lifecycle/terminal_state")

	missingReceipt := optionsFor(t, GeneratedPlan, ExternalLocalState, now)
	missingReceipt.Lifecycle.TerminalState = TerminalApplied
	_, _, err = NewWithOptions(GeneratedPlan, baseRun, baseProvenance, cases[indexOfKind(cases, GeneratedPlan)].Payload, missingReceipt)
	assertContractError(t, err, CodeLifecycleRejected, "/lifecycle/terminal_receipt_digest")

	nonSyntheticGolden := optionsFor(t, ReadinessReport, SyntheticGolden, now)
	_, _, err = NewWithOptions(ReadinessReport, baseRun, Provenance{Mode: "real-run", InputDigests: []string{}}, cases[indexOfKind(cases, ReadinessReport)].Payload, nonSyntheticGolden)
	assertContractError(t, err, CodeStorageRejected, "/storage_class")
}

func assertRunnerRouteContract(t *testing.T) {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), "safety", "scripts", "test.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("runner unavailable")
	}
	text := string(data)
	for _, literal := range []string{"task:artifact-kinds", "'./internal/artifact'", "'^TestArtifactKinds$'", "test-selection-not-exact", "unsupported-suite"} {
		if strings.Count(text, literal) != 1 {
			t.Fatalf("artifact-kinds runner route is not exact")
		}
	}
	for _, future := range []string{"task:privacy-boundary", "task:bounded-capture", "wave:privacy"} {
		if strings.Contains(text, future) {
			t.Fatalf("future runner route registered early")
		}
	}
}

func loadKindCases(t *testing.T) []kindCase {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), "safety", "testdata", "artifacts", "kind-cases.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("synthetic kind cases unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var fixture kindCaseFile
	if err := decoder.Decode(&fixture); err != nil || fixture.SchemaVersion != SchemaVersion || len(fixture.Cases) != 6 {
		t.Fatalf("synthetic kind cases rejected")
	}
	if err := requireEOF(decoder); err != nil {
		t.Fatalf("synthetic kind cases contain trailing data")
	}
	return fixture.Cases
}

func syntheticRun() RunMetadata {
	return RunMetadata{RunID: "synthetic-run-kind-contracts", Tier: "offline-static", SuiteID: "artifact-kinds"}
}

func syntheticProvenance() Provenance {
	return Provenance{Mode: "synthetic", InputDigests: []string{}}
}

func optionsFor(t *testing.T, kind Kind, storageClass StorageClass, now time.Time) BuildOptions {
	t.Helper()
	options, err := DefaultBuildOptions(kind, now)
	if err != nil {
		t.Fatalf("default lifecycle unavailable")
	}
	options.StorageClass = storageClass
	return options
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("source location unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
}

func indexOfKind(cases []kindCase, kind Kind) int {
	for index, testCase := range cases {
		if testCase.Kind == kind {
			return index
		}
	}
	return -1
}

func mutateEnvelope(t *testing.T, canonical []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("artifact mutation setup failed")
	}
	mutate(value)
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("artifact mutation encoding failed")
	}
	result, err := Canonicalize(encoded)
	if err != nil {
		t.Fatalf("artifact mutation canonicalization failed")
	}
	return result
}

func assertContractError(t *testing.T, err error, code ErrorCode, pointer string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected contract rejection")
	}
	var contractErr *ContractError
	if !errors.As(err, &contractErr) || contractErr.Code != code || contractErr.Pointer != pointer {
		t.Fatalf("unexpected contract rejection")
	}
	if strings.Contains(contractErr.Error(), "synthetic-developer") {
		t.Fatalf("contract error exposed input material")
	}
}

func assertStoreEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatalf("invalid-store inspection failed")
	}
	if len(entries) != 0 {
		t.Fatalf("invalid artifact persisted store state")
	}
}
