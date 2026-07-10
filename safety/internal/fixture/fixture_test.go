package fixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestFixtureLifecycle(t *testing.T) {
	t.Run("creates every isolated root below one fresh external child", testFixtureRoots)
	t.Run("constructs a blank allowlisted child environment", testFixtureEnvironment)
	t.Run("rejects protected traversal and symlink bases before creation", testFixtureCreationRejections)
	t.Run("removes passed and failed workloads only after verdict freeze", testDefaultTeardown)
	t.Run("retains only by pre-run choice and expires one owned child", testRetentionAndExpiry)
	t.Run("rejects ambiguous ownership with zero fixture deletion", testTeardownRejections)
	t.Run("combines teardown outcome monotonically", testMonotonicVerdict)
	t.Run("managed CLI renders logical retention identity only", testManagedFixtureCLI)
	t.Run("keeps teardown API narrow", testNarrowTeardownSurface)
}

func testFixtureRoots(t *testing.T) {
	root, base := createTestFixture(t, false, nil)
	paths := root.Paths()
	if filepath.Dir(paths.Root) != base {
		t.Fatal("fixture was not created as one direct child of its external base")
	}
	all := map[string]string{
		"home": paths.Home, "xdg-config": paths.XDGConfig, "xdg-data": paths.XDGData,
		"xdg-cache": paths.XDGCache, "xdg-state": paths.XDGState, "xdg-runtime": paths.XDGRuntime,
		"temporary": paths.Temporary, "fake-bin": paths.FakeBin, "nix": paths.NixManager,
		"homebrew": paths.HomebrewManager, "mise": paths.MiseManager, "uv": paths.UVManager,
		"rustup": paths.RustupManager, "cargo": paths.CargoManager, "go": paths.GoManager,
		"node": paths.NodeManager, "trust": paths.Trust, "network-cache": paths.NetworkCache,
		"artifact-store": paths.ArtifactStore, "blueprint-worktree": paths.BlueprintWorktree,
		"sentinel-scratch": paths.SentinelScratch,
	}
	for name, path := range all {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("isolated %s root is unavailable", name)
		}
		inside, err := isWithin(paths.Root, path)
		if err != nil || !inside {
			t.Fatalf("isolated %s root escaped the fixture", name)
		}
	}
	markerInfo, err := os.Lstat(filepath.Join(paths.Root, markerFileName))
	if err != nil || !markerInfo.Mode().IsRegular() || markerInfo.Mode().Perm() != 0o600 {
		t.Fatal("ownership marker is missing or has unsafe permissions")
	}
	frozen, _ := FreezePrimary(VerdictPassed)
	if final := root.Retention().Finalize(frozen); final.Teardown.Status != TeardownRemoved {
		t.Fatal("valid fixture was not removed by exact teardown")
	}
}

func testFixtureEnvironment(t *testing.T) {
	t.Setenv("HOME", "/ambient/home")
	t.Setenv("HTTPS_PROXY", "http://ambient.invalid")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "synthetic-ambient-secret")
	t.Setenv("ZDOTDIR", "/ambient/shell")
	root, _ := createTestFixture(t, false, nil)
	environment := environmentMap(t, root.ChildEnvironment())
	for _, forbidden := range []string{
		"HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "NO_PROXY", "AWS_SECRET_ACCESS_KEY",
		"GITHUB_TOKEN", "SSH_AUTH_SOCK", "ZDOTDIR", "BASH_ENV", "ENV",
	} {
		if _, ok := environment[forbidden]; ok {
			t.Fatalf("ambient variable %s survived the allowlist", forbidden)
		}
	}
	for key, want := range map[string]string{
		"HOME": root.paths.Home, "XDG_CONFIG_HOME": root.paths.XDGConfig,
		"XDG_DATA_HOME": root.paths.XDGData, "XDG_CACHE_HOME": root.paths.XDGCache,
		"XDG_STATE_HOME": root.paths.XDGState, "XDG_RUNTIME_DIR": root.paths.XDGRuntime,
		"TMPDIR": root.paths.Temporary, "PATH": root.paths.FakeBin,
		"GOTOOLCHAIN": "local", "GOPROXY": "off", "GOSUMDB": "off",
		"GOENV": "off", "GOWORK": "off", "CGO_ENABLED": "0",
		"YAMC_TEST_TIER": "offline-static", "YAMC_NETWORK": "deny",
	} {
		if environment[key] != want {
			t.Fatalf("allowlisted environment %s changed", key)
		}
	}
	for key, value := range environment {
		if strings.Contains(key, "HOME") || strings.Contains(key, "CACHE") || strings.Contains(key, "DIR") || key == "PATH" || key == "TMPDIR" || key == "GOPATH" || key == "GOCACHE" || key == "GOMODCACHE" || key == "NIX_STATE_DIR" || key == "RUSTUP_HOME" || key == "CARGO_HOME" || key == "YAMC_ARTIFACT_STORE" || key == "YAMC_TRUST_ROOT" || key == "YAMC_NETWORK_CACHE" {
			inside, err := isWithin(root.paths.Root, value)
			if err != nil || !inside {
				t.Fatalf("environment root %s escaped the fixture", key)
			}
		}
	}
	frozen, _ := FreezePrimary(VerdictPassed)
	root.Retention().Finalize(frozen)
}

func testFixtureCreationRejections(t *testing.T) {
	repository := t.TempDir()
	protected := t.TempDir()
	external := t.TempDir()
	insideRepository := filepath.Join(repository, "fixture-base")
	insideProtected := filepath.Join(protected, "fixture-base")
	if err := os.MkdirAll(insideRepository, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(insideProtected, 0o700); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	symlinkBase := filepath.Join(external, "base-link")
	if err := os.Symlink(target, symlinkBase); err != nil {
		t.Fatal(err)
	}
	traversalTarget := filepath.Join(external, "clean-base")
	if err := os.Mkdir(traversalTarget, 0o700); err != nil {
		t.Fatal(err)
	}
	traversalBase := external + string(filepath.Separator) + "segment" + string(filepath.Separator) + ".." + string(filepath.Separator) + "clean-base"

	cases := []struct {
		name      string
		base      string
		protected []string
	}{
		{name: "inside repository", base: insideRepository},
		{name: "inside protected root", base: insideProtected, protected: []string{protected}},
		{name: "symlink base", base: symlinkBase},
		{name: "parent traversal", base: traversalBase},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			before := directoryNames(t, filepath.Dir(testCase.base))
			_, err := Create(CreateOptions{
				Base:           testCase.base,
				RepositoryRoot: repository,
				ProtectedRoots: testCase.protected,
				LogicalID:      "fixture:rejected/root",
			})
			if err == nil {
				t.Fatal("unsafe fixture base was accepted")
			}
			after := directoryNames(t, filepath.Dir(testCase.base))
			if strings.Join(before, "\x00") != strings.Join(after, "\x00") {
				t.Fatal("rejected fixture creation wrote a child")
			}
		})
	}
}

func testDefaultTeardown(t *testing.T) {
	for _, verdict := range []PrimaryVerdict{VerdictPassed, VerdictViolation} {
		t.Run(string(verdict), func(t *testing.T) {
			root, base := createTestFixture(t, false, nil)
			physicalRoot := root.paths.Root
			frozen, err := FreezePrimary(verdict)
			if err != nil {
				t.Fatal("primary verdict did not freeze")
			}
			final := root.Retention().Finalize(frozen)
			if final.Verdict != verdict || final.Teardown.Status != TeardownRemoved {
				t.Fatal("default teardown changed the frozen verdict or did not remove the fixture")
			}
			if _, err := os.Lstat(physicalRoot); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("default teardown left the fixture behind")
			}
			if info, err := os.Lstat(base); err != nil || !info.IsDir() {
				t.Fatal("teardown removed the retention base")
			}
		})
	}
	root, _ := createTestFixture(t, false, nil)
	if final := root.Retention().Finalize(FrozenPrimary{}); final.Teardown.Status != TeardownFailed {
		t.Fatal("teardown ran before primary verdict freeze")
	}
	if _, err := os.Lstat(root.paths.Root); err != nil {
		t.Fatal("unfrozen teardown removed the fixture")
	}
}

func testRetentionAndExpiry(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	root, base := createTestFixture(t, true, clock)
	physicalRoot := root.paths.Root
	frozen, _ := FreezePrimary(VerdictPassed)
	retained := root.Retention().Finalize(frozen)
	if retained.Verdict != VerdictPassed || retained.Teardown.Status != TeardownRetained || retained.Teardown.ExpiryCategory != "within-24-hours" {
		t.Fatal("pre-run keep choice did not retain the owned fixture")
	}
	if retained.Teardown.LogicalID != "fixture:lifecycle/test" || strings.Contains(retained.Teardown.LogicalID, base) {
		t.Fatal("retention outcome did not use logical identity")
	}
	if _, err := os.Lstat(physicalRoot); err != nil {
		t.Fatal("retained fixture is unavailable")
	}
	tooEarly := root.Retention().TeardownExpiredOwnedFixture(frozen)
	if tooEarly.Teardown.Status != TeardownFailed || tooEarly.Verdict != VerdictHarnessError {
		t.Fatal("unexpired retained fixture was eligible for expiry teardown")
	}
	if _, err := os.Lstat(physicalRoot); err != nil {
		t.Fatal("early expiry attempt removed the fixture")
	}
	now = now.Add(defaultRetentionTTL)
	expired := root.Retention().TeardownExpiredOwnedFixture(frozen)
	if expired.Teardown.Status != TeardownRemoved || expired.Teardown.ExpiryCategory != "expired" || expired.Verdict != VerdictPassed {
		t.Fatal("expired owned fixture did not tear down exactly")
	}
	if _, err := os.Lstat(physicalRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expired fixture still exists")
	}
	if info, err := os.Lstat(base); err != nil || !info.IsDir() {
		t.Fatal("expiry teardown removed the retention base")
	}
}

func testTeardownRejections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *Root)
	}{
		{name: "invalid marker", mutate: func(t *testing.T, root *Root) {
			if err := os.WriteFile(filepath.Join(root.paths.Root, markerFileName), []byte("{\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "wrong uid", mutate: func(t *testing.T, root *Root) {
			marker := root.retention.expected
			marker.EffectiveUID++
			overwriteMarker(t, root.paths.Root, marker)
		}},
		{name: "wrong nonce", mutate: func(t *testing.T, root *Root) {
			marker := root.retention.expected
			marker.Nonce = strings.Repeat("0", len(marker.Nonce))
			overwriteMarker(t, root.paths.Root, marker)
		}},
		{name: "wrong schema", mutate: func(t *testing.T, root *Root) {
			marker := root.retention.expected
			marker.SchemaVersion = "2.0.0"
			overwriteMarker(t, root.paths.Root, marker)
		}},
		{name: "invalid ttl", mutate: func(t *testing.T, root *Root) {
			marker := root.retention.expected
			created, _ := time.Parse(time.RFC3339Nano, marker.CreatedAt)
			marker.ExpiresAt = created.Add(maximumRetentionTTL + time.Second).Format(time.RFC3339Nano)
			overwriteMarker(t, root.paths.Root, marker)
		}},
		{name: "unmarked root", mutate: func(t *testing.T, root *Root) {
			if err := os.Remove(filepath.Join(root.paths.Root, markerFileName)); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "base escape", mutate: func(t *testing.T, root *Root) {
			outside := t.TempDir()
			if err := os.WriteFile(filepath.Join(outside, "witness"), []byte("synthetic"), 0o600); err != nil {
				t.Fatal(err)
			}
			root.retention.root = outside
		}},
		{name: "retention base target", mutate: func(t *testing.T, root *Root) {
			root.retention.root = root.retention.base
		}},
		{name: "symlink target", mutate: func(t *testing.T, root *Root) {
			target := t.TempDir()
			if err := os.WriteFile(filepath.Join(target, "witness"), []byte("synthetic"), 0o600); err != nil {
				t.Fatal(err)
			}
			frozen, _ := FreezePrimary(VerdictViolation)
			if final := root.Retention().Finalize(frozen); final.Teardown.Status != TeardownRemoved {
				t.Fatal("symlink negative setup could not use exact owned teardown")
			}
			if err := os.Symlink(target, root.paths.Root); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			root, base := createTestFixture(t, false, nil)
			originalRoot := root.paths.Root
			witness := filepath.Join(originalRoot, "synthetic-witness")
			if err := os.WriteFile(witness, []byte("synthetic"), 0o600); err != nil {
				t.Fatal(err)
			}
			testCase.mutate(t, root)
			frozen, _ := FreezePrimary(VerdictViolation)
			final := root.Retention().Finalize(frozen)
			if final.Teardown.Status != TeardownFailed || final.Verdict != VerdictViolation {
				t.Fatal("ambiguous teardown did not fail monotonically")
			}
			if info, err := os.Lstat(base); err != nil || !info.IsDir() {
				t.Fatal("ambiguous teardown deleted the retention base")
			}
			if testCase.name == "symlink target" {
				target, err := os.Readlink(originalRoot)
				if err != nil {
					t.Fatal("symlink was deleted")
				}
				if _, err := os.Lstat(filepath.Join(target, "witness")); err != nil {
					t.Fatal("symlink target content was deleted")
				}
			} else if testCase.name != "base escape" && testCase.name != "retention base target" {
				if _, err := os.Lstat(witness); err != nil {
					t.Fatal("ambiguous teardown deleted fixture content")
				}
			}
		})
	}
}

func testMonotonicVerdict(t *testing.T) {
	for _, verdict := range []PrimaryVerdict{VerdictPassed, VerdictViolation, VerdictIndeterminate, VerdictHarnessError} {
		root, _ := createTestFixture(t, false, nil)
		if err := os.Remove(filepath.Join(root.paths.Root, markerFileName)); err != nil {
			t.Fatal(err)
		}
		frozen, _ := FreezePrimary(verdict)
		final := root.Retention().Finalize(frozen)
		if final.Teardown.Status != TeardownFailed {
			t.Fatal("teardown failure was hidden")
		}
		if verdict == VerdictPassed && final.Verdict != VerdictHarnessError {
			t.Fatal("teardown error did not worsen a passed workload")
		}
		if verdict != VerdictPassed && final.Verdict != verdict {
			t.Fatal("teardown error rewrote a frozen non-pass verdict")
		}
		if final.Verdict == VerdictPassed {
			t.Fatal("teardown error produced pass")
		}
	}
}

func testManagedFixtureCLI(t *testing.T) {
	safetyRoot, repositoryRoot := fixtureProjectRoots(t)
	blueprintPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "input.json")
	surfacesPath := filepath.Join(safetyRoot, "testdata", "blueprints", "walking-skeleton", "protected-surfaces.json")

	defaultBase := t.TempDir()
	defaultOutput := runManagedFixtureCLI(t, safetyRoot, repositoryRoot, blueprintPath, surfacesPath, defaultBase, "fixture:lifecycle/cli-default", false)
	if defaultOutput.Retention != TeardownRemoved || defaultOutput.LogicalRef != "fixture:lifecycle/cli-default" {
		t.Fatal("managed CLI default did not remove its fixture")
	}
	if names := directoryNames(t, defaultBase); len(names) != 0 {
		t.Fatal("managed CLI default left fixture state behind")
	}

	keptBase := t.TempDir()
	keptOutput := runManagedFixtureCLI(t, safetyRoot, repositoryRoot, blueprintPath, surfacesPath, keptBase, "fixture:lifecycle/cli-kept", true)
	if keptOutput.Retention != TeardownRetained || keptOutput.LogicalRef != "fixture:lifecycle/cli-kept" || keptOutput.ExpiryCategory != "within-24-hours" {
		t.Fatal("managed CLI keep result is incomplete")
	}
	children := directoryNames(t, keptBase)
	if len(children) != 1 {
		t.Fatal("managed CLI retained more than one owned fixture child")
	}
	retainedRoot := filepath.Join(keptBase, children[0])
	marker, err := readMarker(retainedRoot)
	if err != nil {
		t.Fatal("managed CLI retained fixture has no valid marker")
	}
	canonicalBase, err := canonicalExistingDirectory(keptBase)
	if err != nil {
		t.Fatal("managed CLI retention base is unavailable")
	}
	canonicalRoot, err := filepath.EvalSymlinks(retainedRoot)
	if err != nil {
		t.Fatal("managed CLI retained root is unavailable")
	}
	retention := &Retention{base: canonicalBase, root: canonicalRoot, expected: marker, clock: time.Now, effectiveUID: os.Geteuid}
	frozen, _ := FreezePrimary(VerdictPassed)
	if final := retention.Finalize(frozen); final.Teardown.Status != TeardownRemoved {
		t.Fatal("managed CLI retained fixture could not be removed by exact ownership")
	}
}

func testNarrowTeardownSurface(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("fixture source unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(current), "retention.go"))
	if err != nil {
		t.Fatal("retention implementation unavailable")
	}
	text := string(data)
	for _, forbidden := range []string{
		"func Cleanup", "func RemovePath", "func DeletePath", "filepath.Walk", "filepath.WalkDir",
		"os.RemoveAll(retention.base)", "restore", "converge", "autoRetry",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("retention exposes broad or restorative behavior: %s", forbidden)
		}
	}
	if strings.Count(text, "os.RemoveAll(") != 1 || !strings.Contains(text, "teardownOwnedFixture") {
		t.Fatal("fixture teardown is not confined to one marker-owned operation")
	}
}

func createTestFixture(t *testing.T, keep bool, clock func() time.Time) (*Root, string) {
	t.Helper()
	repository := t.TempDir()
	base := t.TempDir()
	root, err := Create(CreateOptions{
		Base:           base,
		RepositoryRoot: repository,
		LogicalID:      "fixture:lifecycle/test",
		KeepFixture:    keep,
		Clock:          clock,
	})
	if err != nil {
		t.Fatalf("fixture creation failed: %v", err)
	}
	return root, root.retention.base
}

func environmentMap(t *testing.T, entries []string) map[string]string {
	t.Helper()
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		if !found || key == "" {
			t.Fatal("child environment entry is malformed")
		}
		if _, exists := result[key]; exists {
			t.Fatal("child environment contains duplicate keys")
		}
		result[key] = value
	}
	return result
}

func directoryNames(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func overwriteMarker(t *testing.T, root string, marker ownershipMarker) {
	t.Helper()
	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(root, markerFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

type managedFixtureOutput struct {
	Summary        json.RawMessage `json:"summary"`
	LogicalRef     string          `json:"logical_ref"`
	Retention      TeardownStatus  `json:"retention_status"`
	ExpiryCategory string          `json:"expiry_category"`
}

func runManagedFixtureCLI(t *testing.T, safetyRoot, repositoryRoot, blueprintPath, surfacesPath, base, logicalID string, keep bool) managedFixtureOutput {
	t.Helper()
	arguments := []string{
		"run", "./cmd/yamc-safety", "fixture", "run",
		"--blueprint", blueprintPath,
		"--surfaces", surfacesPath,
		"--fixture-base", base,
		"--fixture-id", logicalID,
		"--repo-root", repositoryRoot,
		"--mode", "synthetic",
	}
	if keep {
		arguments = append(arguments, "--keep-fixture")
	}
	command := exec.Command("go", arguments...)
	command.Dir = safetyRoot
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("managed fixture CLI failed: %s", strings.TrimSpace(stderr.String()))
	}
	if stderr.Len() != 0 || bytes.Contains(stdout.Bytes(), []byte(base)) || bytes.Contains(stderr.Bytes(), []byte(base)) {
		t.Fatal("managed fixture CLI exposed a physical root")
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	var output managedFixtureOutput
	if err := decoder.Decode(&output); err != nil {
		t.Fatal("managed fixture CLI output is invalid")
	}
	if len(output.Summary) == 0 {
		t.Fatal("managed fixture CLI omitted the synthetic run summary")
	}
	return output
}

func fixtureProjectRoots(t *testing.T) (string, string) {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("fixture test source path unavailable")
	}
	safetyRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	return safetyRoot, filepath.Dir(safetyRoot)
}
