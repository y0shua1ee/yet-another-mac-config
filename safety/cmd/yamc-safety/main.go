package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"example.invalid/yamc/safety/internal/workflow"
)

type fixtureRunFlags struct {
	blueprintPath  string
	surfacesPath   string
	fixtureRoot    string
	storeRoot      string
	repositoryRoot string
	mode           string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) < 2 || arguments[0] != "fixture" || arguments[1] != "run" {
		writeSafeError(stderr, "UNSUPPORTED_COMMAND")
		return 64
	}
	parsed, err := parseFixtureRunFlags(arguments[2:])
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
	if parsed.blueprintPath == "" || parsed.surfacesPath == "" || parsed.fixtureRoot == "" ||
		parsed.storeRoot == "" || parsed.repositoryRoot == "" || parsed.mode != "synthetic" {
		return fixtureRunFlags{}, errors.New("arguments rejected")
	}
	return parsed, nil
}

func writeSafeError(writer io.Writer, code string) {
	_ = json.NewEncoder(writer).Encode(struct {
		ErrorCode string `json:"error_code"`
	}{ErrorCode: code})
}
