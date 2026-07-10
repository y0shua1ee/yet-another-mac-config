package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"example.invalid/yamc/safety/internal/artifact"
	"example.invalid/yamc/safety/internal/workflow"
)

const maxArtifactBytes = 1 << 20

type fixtureRunFlags struct {
	blueprintPath  string
	surfacesPath   string
	fixtureRoot    string
	storeRoot      string
	repositoryRoot string
	mode           string
}

type validateFlags struct {
	expectedKind string
	artifactPath string
}

type storeFlags struct {
	mode              string
	storeRoot         string
	repositoryRoot    string
	desiredPath       string
	observedPath      string
	freshObservedPath string
	planPath          string
	receiptPath       string
	evidencePath      string
	reportPath        string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		writeSafeError(stderr, "UNSUPPORTED_COMMAND")
		return 64
	}
	switch arguments[0] {
	case "fixture":
		if len(arguments) < 2 || arguments[1] != "run" {
			writeSafeError(stderr, "UNSUPPORTED_COMMAND")
			return 64
		}
		return runFixture(arguments[2:], stdout, stderr)
	case "validate":
		return runValidate(arguments[1:], stdout, stderr)
	case "store":
		return runStore(arguments[1:], stdout, stderr)
	default:
		writeSafeError(stderr, "UNSUPPORTED_COMMAND")
		return 64
	}
}

func runFixture(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseFixtureRunFlags(arguments)
	if err != nil {
		writeSafeError(stderr, "FIXTURE_ARGUMENTS_REJECTED")
		return 64
	}
	summary, err := workflow.RunSynthetic(workflow.Options{
		BlueprintPath:  parsed.blueprintPath,
		SurfacesPath:   parsed.surfacesPath,
		FixtureRoot:    parsed.fixtureRoot,
		StoreRoot:      parsed.storeRoot,
		RepositoryRoot: parsed.repositoryRoot,
		Mode:           parsed.mode,
	})
	if err != nil {
		writeSafeError(stderr, "FIXTURE_RUN_REJECTED")
		return 2
	}
	if err := json.NewEncoder(stdout).Encode(summary); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return 0
}

func runValidate(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseValidateFlags(arguments)
	if err != nil || !knownKind(artifact.Kind(parsed.expectedKind)) {
		writeSafeError(stderr, "VALIDATE_ARGUMENTS_REJECTED")
		return 64
	}
	canonical, err := readBoundedArtifact(parsed.artifactPath)
	if err != nil {
		writeSafeError(stderr, "ARTIFACT_READ_REJECTED")
		return 2
	}
	envelope, err := artifact.Validate(artifact.Kind(parsed.expectedKind), canonical)
	if err != nil {
		writeSafeError(stderr, "ARTIFACT_VALIDATION_REJECTED")
		return 2
	}
	result := struct {
		Status string        `json:"status"`
		Kind   artifact.Kind `json:"kind"`
		Digest string        `json:"digest"`
	}{Status: "valid", Kind: envelope.Kind, Digest: envelope.ContentDigest}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return 0
}

func runStore(arguments []string, stdout, stderr io.Writer) int {
	parsed, err := parseStoreFlags(arguments)
	if err != nil {
		writeSafeError(stderr, "STORE_ARGUMENTS_REJECTED")
		return 64
	}
	graph, err := readLineageGraph(parsed)
	if err != nil {
		writeSafeError(stderr, "ARTIFACT_READ_REJECTED")
		return 2
	}
	mode := artifact.LineageMode(parsed.mode)
	store, err := artifact.NewStore(parsed.storeRoot, parsed.repositoryRoot)
	if err != nil {
		writeSafeError(stderr, "STORE_ROOT_REJECTED")
		return 2
	}
	digests, err := store.WriteGraph(mode, graph)
	if err != nil {
		writeSafeError(stderr, "ARTIFACT_STORE_REJECTED")
		return 2
	}
	result := struct {
		Status  string               `json:"status"`
		Mode    artifact.LineageMode `json:"mode"`
		Digests map[string]string    `json:"digests"`
	}{Status: "stored", Mode: mode, Digests: digests}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		writeSafeError(stderr, "OUTPUT_REJECTED")
		return 70
	}
	return 0
}

func parseFixtureRunFlags(arguments []string) (fixtureRunFlags, error) {
	var parsed fixtureRunFlags
	flags := flag.NewFlagSet("fixture-run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.blueprintPath, "blueprint", "", "")
	flags.StringVar(&parsed.surfacesPath, "surfaces", "", "")
	flags.StringVar(&parsed.fixtureRoot, "fixture-root", "", "")
	flags.StringVar(&parsed.storeRoot, "store-root", "", "")
	flags.StringVar(&parsed.repositoryRoot, "repo-root", "", "")
	flags.StringVar(&parsed.mode, "mode", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return fixtureRunFlags{}, errors.New("arguments rejected")
	}
	if parsed.blueprintPath == "" || parsed.surfacesPath == "" || parsed.fixtureRoot == "" || parsed.storeRoot == "" || parsed.repositoryRoot == "" || parsed.mode != "synthetic" {
		return fixtureRunFlags{}, errors.New("arguments rejected")
	}
	return parsed, nil
}

func parseValidateFlags(arguments []string) (validateFlags, error) {
	var parsed validateFlags
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.expectedKind, "expect-kind", "", "")
	flags.StringVar(&parsed.artifactPath, "artifact", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || parsed.expectedKind == "" || parsed.artifactPath == "" {
		return validateFlags{}, errors.New("arguments rejected")
	}
	return parsed, nil
}

func parseStoreFlags(arguments []string) (storeFlags, error) {
	var parsed storeFlags
	flags := flag.NewFlagSet("store", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&parsed.mode, "mode", "", "")
	flags.StringVar(&parsed.storeRoot, "root", "", "")
	flags.StringVar(&parsed.repositoryRoot, "repo-root", "", "")
	flags.StringVar(&parsed.desiredPath, "desired", "", "")
	flags.StringVar(&parsed.observedPath, "observed", "", "")
	flags.StringVar(&parsed.freshObservedPath, "fresh-observed", "", "")
	flags.StringVar(&parsed.planPath, "plan", "", "")
	flags.StringVar(&parsed.receiptPath, "receipt", "", "")
	flags.StringVar(&parsed.evidencePath, "evidence", "", "")
	flags.StringVar(&parsed.reportPath, "report", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return storeFlags{}, errors.New("arguments rejected")
	}
	if parsed.storeRoot == "" || parsed.repositoryRoot == "" || parsed.desiredPath == "" || parsed.observedPath == "" || parsed.evidencePath == "" || parsed.reportPath == "" {
		return storeFlags{}, errors.New("arguments rejected")
	}
	switch artifact.LineageMode(parsed.mode) {
	case artifact.LineageApply:
		if parsed.planPath == "" || parsed.receiptPath == "" || parsed.freshObservedPath == "" {
			return storeFlags{}, errors.New("arguments rejected")
		}
	case artifact.LineageReadOnly:
		if parsed.planPath != "" || parsed.receiptPath != "" || parsed.freshObservedPath != "" {
			return storeFlags{}, errors.New("arguments rejected")
		}
	default:
		return storeFlags{}, errors.New("arguments rejected")
	}
	return parsed, nil
}

func readLineageGraph(parsed storeFlags) (artifact.LineageGraph, error) {
	desired, err := readBoundedArtifact(parsed.desiredPath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	observed, err := readBoundedArtifact(parsed.observedPath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	evidence, err := readBoundedArtifact(parsed.evidencePath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	report, err := readBoundedArtifact(parsed.reportPath)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	graph := artifact.LineageGraph{Desired: desired, Observed: observed, Evidence: evidence, Report: report}
	if parsed.mode == string(artifact.LineageApply) {
		graph.Plan, err = readBoundedArtifact(parsed.planPath)
		if err != nil {
			return artifact.LineageGraph{}, err
		}
		graph.Receipt, err = readBoundedArtifact(parsed.receiptPath)
		if err != nil {
			return artifact.LineageGraph{}, err
		}
		graph.FreshObserved, err = readBoundedArtifact(parsed.freshObservedPath)
		if err != nil {
			return artifact.LineageGraph{}, err
		}
	}
	var evidenceEnvelope artifact.Envelope
	evidenceEnvelope, err = artifact.Validate(artifact.VerificationEvidence, evidence)
	if err != nil {
		return artifact.LineageGraph{}, err
	}
	var evidencePayload struct {
		ExpectedPostconditionsDigest string `json:"expected_postconditions_digest"`
	}
	if err := json.Unmarshal(evidenceEnvelope.Payload, &evidencePayload); err != nil || evidencePayload.ExpectedPostconditionsDigest == "" {
		return artifact.LineageGraph{}, errors.New("evidence rejected")
	}
	graph.ExpectedPostconditionsDigest = evidencePayload.ExpectedPostconditionsDigest
	return graph, nil
}

func readBoundedArtifact(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("artifact rejected")
	}
	var buffer bytes.Buffer
	if _, err := io.CopyN(&buffer, file, maxArtifactBytes+1); err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.New("artifact rejected")
	}
	if buffer.Len() > maxArtifactBytes {
		return nil, errors.New("artifact rejected")
	}
	return buffer.Bytes(), nil
}

func knownKind(kind artifact.Kind) bool {
	for _, candidate := range artifact.RegisteredKinds() {
		if kind == candidate {
			return true
		}
	}
	return false
}

func writeSafeError(writer io.Writer, code string) {
	_ = json.NewEncoder(writer).Encode(struct {
		ErrorCode string `json:"error_code"`
	}{ErrorCode: code})
}
