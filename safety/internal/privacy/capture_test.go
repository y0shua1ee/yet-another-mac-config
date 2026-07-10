package privacy_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"example.invalid/yamc/safety/internal/privacy"
)

const rawTransportMarker = "synthetic-raw-boundary-canary"

func TestBoundedCapture(t *testing.T) {
	rawSample := readRawSample(t)
	fixtureRoot := t.TempDir()
	registry, err := privacy.MaterializeFixtureAdapter(fixtureRoot, rawSample)
	if err != nil {
		t.Fatal("fixture fake adapter materialization failed")
	}
	before := regularFiles(t, fixtureRoot)

	t.Run("defaults and hard maxima", testCaptureLimits)
	t.Run("fixed registry success", func(t *testing.T) { testCaptureSuccess(t, registry) })
	t.Run("closed command IDs", func(t *testing.T) { testClosedCommandIDs(t, registry) })
	t.Run("bounded failure paths", func(t *testing.T) { testCaptureFailures(t, registry) })
	t.Run("no arbitrary argv or shell surface", testCaptureStructure)

	after := regularFiles(t, fixtureRoot)
	if !reflect.DeepEqual(before, after) || len(after) != 1 {
		t.Fatal("capture created or retained a raw file")
	}
}

func testCaptureLimits(t *testing.T) {
	t.Helper()
	defaults, err := privacy.NormalizeLimits(privacy.Limits{})
	if err != nil || defaults.Timeout != 5*time.Second || defaults.StdoutBytes != 64<<10 || defaults.StderrBytes != 64<<10 {
		t.Fatal("capture defaults changed")
	}
	maximum, err := privacy.NormalizeLimits(privacy.Limits{Timeout: 30 * time.Second, StdoutBytes: 256 << 10, StderrBytes: 256 << 10})
	if err != nil || maximum.Timeout != 30*time.Second {
		t.Fatal("capture maximum limits rejected")
	}
	for _, invalid := range []privacy.Limits{
		{Timeout: 30*time.Second + time.Nanosecond},
		{StdoutBytes: (256 << 10) + 1},
		{StderrBytes: (256 << 10) + 1},
		{Timeout: -1},
		{StdoutBytes: -1},
		{StderrBytes: -1},
	} {
		if _, err := privacy.NormalizeLimits(invalid); err == nil {
			t.Fatal("capture limit override exceeded a hard maximum")
		}
	}
}

func testCaptureSuccess(t *testing.T, registry *privacy.Registry) {
	t.Helper()
	observation, rejection := privacy.Capture(context.Background(), registry, privacy.CommandFixtureFake, privacy.Limits{})
	if rejection != nil || observation.Status != "normalized" || len(observation.Facts) != 1 {
		t.Fatal("registered fixture adapter did not normalize its output")
	}
	if observation.Facts[0].Ref != "fixture:observed/shell-policy" || observation.Facts[0].State != "declared" {
		t.Fatal("normalized fixture fact changed")
	}
	encoded, err := json.Marshal(observation)
	if err != nil || strings.Contains(string(encoded), rawTransportMarker) {
		t.Fatal("successful capture retained raw transport data")
	}
}

func testClosedCommandIDs(t *testing.T, registry *privacy.Registry) {
	t.Helper()
	for _, commandID := range []privacy.CommandID{
		"unknown-command-v1",
		"fixture-fake-v1;synthetic-extra",
		"fixture-fake-v1 --synthetic-extra",
		"/bin/sh",
	} {
		observation, rejection := privacy.Capture(context.Background(), registry, commandID, privacy.Limits{})
		if rejection == nil || observation.Status != "unknown" || len(observation.Facts) != 0 {
			t.Fatal("caller-controlled command ID reached process creation")
		}
		assertCaptureErrorEnvelope(t, rejection)
	}
}

func testCaptureFailures(t *testing.T, registry *privacy.Registry) {
	t.Helper()
	tests := []struct {
		name      string
		commandID privacy.CommandID
		limits    privacy.Limits
	}{
		{"timeout", privacy.CommandFixtureTimeout, privacy.Limits{Timeout: 25 * time.Millisecond}},
		{"stdout overflow", privacy.CommandFixtureStdoutOverflow, privacy.Limits{}},
		{"stderr overflow", privacy.CommandFixtureStderrOverflow, privacy.Limits{}},
		{"invalid utf8", privacy.CommandFixtureInvalidUTF8, privacy.Limits{}},
		{"parse failure", privacy.CommandFixtureParseFailure, privacy.Limits{}},
		{"unknown field", privacy.CommandFixtureUnknownField, privacy.Limits{}},
		{"process failure", privacy.CommandFixtureProcessFailure, privacy.Limits{}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			observation, rejection := privacy.Capture(context.Background(), registry, testCase.commandID, testCase.limits)
			if rejection == nil || observation.Status != "unknown" || len(observation.Facts) != 0 {
				t.Fatal("capture failure exposed a partial observation")
			}
			assertCaptureErrorEnvelope(t, rejection)
			encoded, err := json.Marshal(struct {
				Observation privacy.Observation    `json:"observation"`
				Rejection   *privacy.ErrorEnvelope `json:"rejection"`
			}{observation, rejection})
			if err != nil || strings.Contains(string(encoded), rawTransportMarker) || strings.Contains(string(encoded), "synthetic-overflow") {
				t.Fatal("capture failure exposed raw bytes")
			}
		})
	}
}

func testCaptureStructure(t *testing.T) {
	t.Helper()
	if captureType := reflect.TypeOf(privacy.Capture); captureType.NumIn() != 4 || captureType.NumOut() != 2 {
		t.Fatal("capture API gained caller-controlled executable or argv parameters")
	}
	source, err := os.ReadFile("capture.go")
	if err != nil {
		t.Fatal("capture source unavailable")
	}
	text := string(source)
	for _, required := range []string{"exec.CommandContext", "io.LimitedReader", "StdoutPipe", "StderrPipe", "command.Env =", "clearBytes"} {
		if !strings.Contains(text, required) {
			t.Fatal("bounded capture structural control is missing")
		}
	}
	for _, forbidden := range []string{`exec.Command("sh"`, `exec.Command("bash"`, `"-c"`, "CombinedOutput", "cmd.Stdout = os.Stdout", "cmd.Stderr = os.Stderr"} {
		if strings.Contains(text, forbidden) {
			t.Fatal("capture contains a shell or inherited-terminal path")
		}
	}
}

func assertCaptureErrorEnvelope(t *testing.T, envelope *privacy.ErrorEnvelope) {
	t.Helper()
	if envelope == nil || privacy.ValidateErrorEnvelope(*envelope) != nil {
		t.Fatal("capture returned an unregistered error envelope")
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal("capture error envelope is not JSON")
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil || len(fields) != 6 {
		t.Fatal("capture error envelope is not the exact six-field schema")
	}
	for _, forbidden := range []string{rawTransportMarker, "synthetic-overflow", "invalid-utf8", "parse-failure", "process-failure"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatal("capture error contains process-derived data")
		}
	}
}

func readRawSample(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "raw", "fake-adapter.json"))
	if err != nil || !strings.Contains(string(data), rawTransportMarker) {
		t.Fatal("tracked synthetic raw sample unavailable")
	}
	return data
}

func regularFiles(t *testing.T, root string) []string {
	t.Helper()
	files := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, relative)
		}
		return nil
	}); err != nil {
		t.Fatal("fixture adapter file inventory failed")
	}
	sort.Strings(files)
	return files
}
