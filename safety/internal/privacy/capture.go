package privacy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultCaptureTimeout    = 5 * time.Second
	MaximumCaptureTimeout    = 30 * time.Second
	DefaultCaptureStreamSize = int64(64 << 10)
	MaximumCaptureStreamSize = int64(256 << 10)

	fixtureChildArgument = "__yamc_fixture_adapter_v1"
	fixtureChildMarker   = "YAMC_FIXTURE_ADAPTER_MODE"
	fixtureChildSample   = "YAMC_FIXTURE_ADAPTER_SAMPLE"
)

type CommandID string

const (
	CommandFixtureFake           CommandID = "fixture-fake-v1"
	CommandFixtureTimeout        CommandID = "fixture-fake-timeout-v1"
	CommandFixtureStdoutOverflow CommandID = "fixture-fake-stdout-overflow-v1"
	CommandFixtureStderrOverflow CommandID = "fixture-fake-stderr-overflow-v1"
	CommandFixtureInvalidUTF8    CommandID = "fixture-fake-invalid-utf8-v1"
	CommandFixtureParseFailure   CommandID = "fixture-fake-parse-failure-v1"
	CommandFixtureUnknownField   CommandID = "fixture-fake-unknown-field-v1"
	CommandFixtureProcessFailure CommandID = "fixture-fake-process-failure-v1"
)

type Limits struct {
	Timeout     time.Duration
	StdoutBytes int64
	StderrBytes int64
}

type NormalizedFact struct {
	Ref   string `json:"ref"`
	State string `json:"state"`
}

type Observation struct {
	Status string           `json:"status"`
	Facts  []NormalizedFact `json:"facts,omitempty"`
}

type registryEntry struct {
	executable  string
	arguments   []string
	environment []string
}

type Registry struct {
	entries map[CommandID]registryEntry
}

type adapterSample struct {
	SchemaVersion string           `json:"schema_version"`
	Transport     string           `json:"transport"`
	Facts         []NormalizedFact `json:"facts"`
}

type streamResult struct {
	stream   string
	data     []byte
	overflow bool
	err      error
}

func init() {
	if len(os.Args) != 3 || os.Args[1] != fixtureChildArgument || os.Getenv(fixtureChildMarker) != "synthetic-v1" {
		return
	}
	runFixtureChild(os.Args[2])
}

func NormalizeLimits(requested Limits) (Limits, error) {
	result := requested
	if result.Timeout == 0 {
		result.Timeout = DefaultCaptureTimeout
	}
	if result.StdoutBytes == 0 {
		result.StdoutBytes = DefaultCaptureStreamSize
	}
	if result.StderrBytes == 0 {
		result.StderrBytes = DefaultCaptureStreamSize
	}
	if result.Timeout < 0 || result.Timeout > MaximumCaptureTimeout ||
		result.StdoutBytes < 0 || result.StdoutBytes > MaximumCaptureStreamSize ||
		result.StderrBytes < 0 || result.StderrBytes > MaximumCaptureStreamSize {
		return Limits{}, errors.New("capture limits rejected")
	}
	return result, nil
}

func MaterializeFixtureAdapter(fixtureRoot string, rawSample []byte) (*Registry, error) {
	if fixtureRoot == "" || !filepath.IsAbs(fixtureRoot) {
		return nil, errors.New("fixture adapter root rejected")
	}
	if len(rawSample) == 0 || len(rawSample) > int(DefaultCaptureStreamSize) {
		return nil, errors.New("fixture adapter sample rejected")
	}
	if _, err := parseAdapterSample(rawSample); err != nil {
		return nil, errors.New("fixture adapter sample rejected")
	}
	fixtureInfo, err := os.Lstat(fixtureRoot)
	if err != nil || !fixtureInfo.IsDir() || fixtureInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("fixture adapter root rejected")
	}
	binRoot := filepath.Join(fixtureRoot, "path", "bin")
	if err := os.MkdirAll(binRoot, 0o700); err != nil {
		return nil, errors.New("fixture adapter root rejected")
	}
	info, err := os.Lstat(binRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("fixture adapter root rejected")
	}
	sourcePath, err := os.Executable()
	if err != nil {
		return nil, errors.New("fixture adapter executable unavailable")
	}
	sourcePath, err = filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return nil, errors.New("fixture adapter executable unavailable")
	}
	destination := filepath.Join(binRoot, "yamc-fixture-adapter-v1")
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("fixture adapter executable already exists")
	}
	if err := copyExecutable(sourcePath, destination); err != nil {
		return nil, err
	}

	encodedSample := base64.StdEncoding.EncodeToString(rawSample)
	entries := make(map[CommandID]registryEntry)
	for commandID, mode := range map[CommandID]string{
		CommandFixtureFake:           "emit",
		CommandFixtureTimeout:        "timeout",
		CommandFixtureStdoutOverflow: "stdout-overflow",
		CommandFixtureStderrOverflow: "stderr-overflow",
		CommandFixtureInvalidUTF8:    "invalid-utf8",
		CommandFixtureParseFailure:   "parse-failure",
		CommandFixtureUnknownField:   "unknown-field",
		CommandFixtureProcessFailure: "process-failure",
	} {
		entries[commandID] = registryEntry{
			executable:  destination,
			arguments:   []string{fixtureChildArgument, mode},
			environment: []string{fixtureChildMarker + "=synthetic-v1", fixtureChildSample + "=" + encodedSample},
		}
	}
	return &Registry{entries: entries}, nil
}

func Capture(parent context.Context, registry *Registry, commandID CommandID, requested Limits) (Observation, *ErrorEnvelope) {
	unknown := Observation{Status: "unknown"}
	if parent == nil || registry == nil {
		return unknown, captureFailure(CodeOperationRejected, CategoryOperation, RemediationReview)
	}
	limits, err := NormalizeLimits(requested)
	if err != nil {
		return unknown, captureFailure(CodeOperationRejected, CategoryOperation, RemediationReview)
	}
	entry, ok := registry.entries[commandID]
	if !ok || entry.executable == "" || len(entry.arguments) != 2 || entry.arguments[0] != fixtureChildArgument || hasShellMetacharacter(string(commandID)) {
		return unknown, captureFailure(CodeCommandRejected, CategoryUnsupported, RemediationCommand)
	}

	ctx, cancel := context.WithTimeout(parent, limits.Timeout)
	defer cancel()
	command := exec.CommandContext(ctx, entry.executable, entry.arguments...)
	command.Env = append([]string(nil), entry.environment...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return unknown, captureFailure(CodeOperationRejected, CategoryOperation, RemediationReview)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return unknown, captureFailure(CodeOperationRejected, CategoryOperation, RemediationReview)
	}
	if err := command.Start(); err != nil {
		return unknown, captureFailure(CodeOperationRejected, CategoryOperation, RemediationReview)
	}

	results := make(chan streamResult, 2)
	go readLimitedStream("stdout", stdout, limits.StdoutBytes, cancel, results)
	go readLimitedStream("stderr", stderr, limits.StderrBytes, cancel, results)
	first := <-results
	second := <-results
	waitErr := command.Wait()
	stdoutResult, stderrResult := orderStreamResults(first, second)
	defer clearBytes(stdoutResult.data)
	defer clearBytes(stderrResult.data)

	if stdoutResult.overflow || stderrResult.overflow || stdoutResult.err != nil || stderrResult.err != nil {
		return unknown, captureFailure(CodeOperationRejected, CategoryOperation, RemediationReview)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || waitErr != nil || len(stderrResult.data) != 0 {
		return unknown, captureFailure(CodeOperationRejected, CategoryOperation, RemediationReview)
	}
	if !utf8.Valid(stdoutResult.data) || !utf8.Valid(stderrResult.data) {
		return unknown, captureFailure(CodeDataRejected, CategoryUnclassified, RemediationNormalization)
	}
	observation, err := parseAdapterSample(stdoutResult.data)
	if err != nil {
		return unknown, captureFailure(CodeDataRejected, CategoryUnclassified, RemediationNormalization)
	}
	approved, rejection := Gate(Candidate{ArtifactKind: KindCommandResult, AdapterID: AdapterFixtureFake, Value: observation})
	if rejection != nil || len(approved) == 0 {
		if rejection != nil {
			return unknown, rejection
		}
		return unknown, captureFailure(CodeDataRejected, CategoryUnclassified, RemediationNormalization)
	}
	return observation, nil
}

func parseAdapterSample(raw []byte) (Observation, error) {
	if len(raw) == 0 || len(raw) > int(MaximumCaptureStreamSize) || !utf8.Valid(raw) {
		return Observation{}, errors.New("adapter sample rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var sample adapterSample
	if err := decoder.Decode(&sample); err != nil {
		return Observation{}, errors.New("adapter sample rejected")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Observation{}, errors.New("adapter sample rejected")
	}
	if sample.SchemaVersion != "1.0.0" || sample.Transport != "synthetic-raw-boundary-canary" || len(sample.Facts) == 0 {
		return Observation{}, errors.New("adapter sample rejected")
	}
	for _, fact := range sample.Facts {
		if _, err := ParseLogicalRef(fact.Ref); err != nil || fact.State == "" {
			return Observation{}, errors.New("adapter sample rejected")
		}
	}
	return Observation{Status: "normalized", Facts: append([]NormalizedFact(nil), sample.Facts...)}, nil
}

func readLimitedStream(stream string, reader io.ReadCloser, limit int64, cancel context.CancelFunc, results chan<- streamResult) {
	defer reader.Close()
	limited := &io.LimitedReader{R: reader, N: limit + 1}
	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, limited)
	result := streamResult{stream: stream, data: bytes.Clone(buffer.Bytes()), err: err}
	buffer.Reset()
	if int64(len(result.data)) > limit {
		result.overflow = true
		cancel()
	}
	results <- result
}

func orderStreamResults(first, second streamResult) (streamResult, streamResult) {
	if first.stream == "stdout" && second.stream == "stderr" {
		return first, second
	}
	if first.stream == "stderr" && second.stream == "stdout" {
		return second, first
	}
	return streamResult{err: errors.New("capture stream rejected")}, streamResult{err: errors.New("capture stream rejected")}
}

func captureFailure(code ErrorCode, category Category, remediation Remediation) *ErrorEnvelope {
	return newError(code, KindCommandResult, AdapterFixtureFake, "/command", category, remediation)
}

func hasShellMetacharacter(value string) bool {
	return strings.ContainsAny(value, " \t\r\n;&|`$<>(){}[]*?!\\\"'")
}

func clearBytes(data []byte) {
	for index := range data {
		data[index] = 0
	}
}

func copyExecutable(sourcePath, destination string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return errors.New("fixture adapter executable unavailable")
	}
	defer source.Close()
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".adapter-")
	if err != nil {
		return errors.New("fixture adapter executable unavailable")
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o500); err != nil {
		return errors.New("fixture adapter executable unavailable")
	}
	if _, err := io.Copy(temporary, source); err != nil {
		return errors.New("fixture adapter executable unavailable")
	}
	if err := temporary.Sync(); err != nil {
		return errors.New("fixture adapter executable unavailable")
	}
	if err := temporary.Close(); err != nil {
		return errors.New("fixture adapter executable unavailable")
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return errors.New("fixture adapter executable unavailable")
	}
	return nil
}

func runFixtureChild(mode string) {
	switch mode {
	case "emit":
		raw, err := base64.StdEncoding.DecodeString(os.Getenv(fixtureChildSample))
		if err != nil || len(raw) == 0 || len(raw) > int(DefaultCaptureStreamSize) {
			os.Exit(23)
		}
		_, _ = os.Stdout.Write(raw)
		clearBytes(raw)
		os.Exit(0)
	case "timeout":
		time.Sleep(time.Minute)
		os.Exit(0)
	case "stdout-overflow":
		writeSyntheticOverflow(os.Stdout)
		os.Exit(0)
	case "stderr-overflow":
		writeSyntheticOverflow(os.Stderr)
		os.Exit(0)
	case "invalid-utf8":
		_, _ = os.Stdout.Write([]byte{0xff, 0xfe, 0xfd})
		os.Exit(0)
	case "parse-failure":
		_, _ = os.Stdout.Write([]byte("{"))
		os.Exit(0)
	case "unknown-field":
		_, _ = os.Stdout.Write([]byte(`{"schema_version":"1.0.0","transport":"synthetic-raw-boundary-canary","facts":[],"unknown":"synthetic"}`))
		os.Exit(0)
	case "process-failure":
		os.Exit(19)
	default:
		os.Exit(23)
	}
}

func writeSyntheticOverflow(writer io.Writer) {
	chunk := bytes.Repeat([]byte("synthetic-overflow\n"), 256)
	for written := 0; written <= int(MaximumCaptureStreamSize); written += len(chunk) {
		if _, err := writer.Write(chunk); err != nil {
			return
		}
	}
}
