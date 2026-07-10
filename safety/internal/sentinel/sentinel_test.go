package sentinel

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSentinelManifest(t *testing.T) {
	t.Run("accepts exact five-domain public scope", testProtectedManifestContract)
	t.Run("rejects invalid scope before resolution", testProtectedManifestNegatives)
	t.Run("freezes scope before observation", testProtectedManifestFreeze)
	t.Run("takes bounded opaque synthetic snapshots", testProtectedSnapshots)
	t.Run("returns typed incomplete snapshots", testIncompleteSnapshots)
	t.Run("renders only synthetic scoped CLI status", testSentinelManifestCLI)
	t.Run("binds the exact runner route", testSentinelManifestRunner)
}

func testProtectedManifestContract(t *testing.T) {
	manifest := loadProtectedManifest(t)
	if len(manifest.Surfaces) != 6 || manifest.Digest == "" {
		t.Fatal("protected manifest did not bind the exact scope")
	}
	wantRefs := map[string]struct{}{}
	for logicalRef := range protectedSurfaceContract {
		wantRefs[logicalRef] = struct{}{}
	}
	domains := map[string]int{}
	for _, surface := range manifest.Surfaces {
		if _, ok := wantRefs[surface.LogicalRef]; !ok || surface.Policy != PolicyRequired || surface.Bounds.MaxFiles < 1 || surface.Bounds.MaxBytes < 1 || surface.Bounds.Timeout < 1 {
			t.Fatal("protected manifest contains an unknown or unbounded surface")
		}
		delete(wantRefs, surface.LogicalRef)
		domains[string(surface.SurfaceDomain)]++
	}
	if len(wantRefs) != 0 || domains["worktree"] != 2 || domains["named-home"] != 1 || domains["manager-root"] != 1 || domains["service"] != 1 || domains["named-target"] != 1 {
		t.Fatal("protected manifest domain coverage changed")
	}
	raw := readProtectedManifest(t)
	for _, forbidden := range []string{"/Users/", "physical_path", "resolver_mapping", "host_identity", "service_output", "hmac_key"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("protected manifest persisted forbidden machine data: %s", forbidden)
		}
	}
}

func testProtectedManifestNegatives(t *testing.T) {
	base := loadProtectedManifest(t)
	mutations := []func(*ProtectedManifest){
		func(value *ProtectedManifest) { value.Surfaces[0].LogicalRef = "home:sentinel/worktree/tracked" },
		func(value *ProtectedManifest) { value.Surfaces[0].LogicalRef = "legacy:sentinel/worktree/tracked" },
		func(value *ProtectedManifest) { value.Surfaces[0].SurfaceDomain = "unknown" },
		func(value *ProtectedManifest) { value.Surfaces[0].LogicalRef = "repo:sentinel/../tracked" },
		func(value *ProtectedManifest) { value.Surfaces[0].LogicalRef = "repo:/sentinel/worktree/tracked" },
		func(value *ProtectedManifest) { value.Surfaces[0].LogicalRef = "repo:*" },
		func(value *ProtectedManifest) { value.Surfaces[0].AdapterID = "unsupported-required-v1" },
		func(value *ProtectedManifest) { value.Surfaces[0].Bounds.MaxBytes = 0 },
		func(value *ProtectedManifest) { value.Surfaces[1].SurfaceID = value.Surfaces[0].SurfaceID },
		func(value *ProtectedManifest) { value.Surfaces[1].LogicalRef = value.Surfaces[0].LogicalRef },
	}
	for index, mutate := range mutations {
		candidate := base
		candidate.Surfaces = append([]ProtectedSurface(nil), base.Surfaces...)
		mutate(&candidate)
		encoded, err := json.Marshal(candidate)
		if err != nil {
			t.Fatal("negative manifest setup failed")
		}
		if _, err := ParseProtectedManifest(encoded); err == nil {
			t.Fatalf("invalid manifest mutation %d reached resolution", index)
		}
	}
	duplicateKey := bytes.Replace(readProtectedManifest(t), []byte(`"schema_version": "1.0.0"`), []byte(`"schema_version": "1.0.0", "schema_version": "1.0.0"`), 1)
	if _, err := ParseProtectedManifest(duplicateKey); err == nil {
		t.Fatal("duplicate manifest key was accepted")
	}

	root := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "outside"), filepath.Join(root, "protected")); err != nil {
		t.Fatal("resolver escape setup failed")
	}
	resolver, err := NewSyntheticResolver(root)
	if err != nil {
		t.Fatal("synthetic resolver setup failed")
	}
	if _, err := resolver.resolve("home:.zshrc"); err == nil {
		t.Fatal("resolver escape was accepted before snapshot")
	}
}

func testProtectedManifestFreeze(t *testing.T) {
	manifest := loadProtectedManifest(t)
	frozen, err := FreezeProtectedManifest(manifest)
	if err != nil {
		t.Fatal("protected manifest did not freeze")
	}
	manifest.Surfaces[0].Policy = PolicyOptional
	if err := frozen.ValidateCurrent(manifest); err == nil {
		t.Fatal("post-start policy mutation was accepted")
	}
	manifest = frozen.Manifest()
	manifest.Surfaces[0].Bounds.MaxFiles++
	if err := frozen.ValidateCurrent(manifest); err == nil {
		t.Fatal("post-start bounds mutation was accepted")
	}
}

func testProtectedSnapshots(t *testing.T) {
	root := t.TempDir()
	resolver, err := PrepareProtectedSynthetic(root)
	if err != nil {
		t.Fatal("synthetic protected surfaces unavailable")
	}
	manifest := loadProtectedManifest(t)
	frozen, err := FreezeProtectedManifest(manifest)
	if err != nil {
		t.Fatal("protected manifest did not freeze")
	}
	key := bytes.Repeat([]byte{0x5a}, 32)
	before, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{})
	if err != nil || !allComplete(before) {
		t.Fatal("bounded protected snapshot was incomplete")
	}
	after, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{})
	if err != nil || !sameOpaqueState(before, after) {
		t.Fatal("unchanged protected snapshots differed")
	}
	for _, surface := range before.Surfaces {
		if !strings.HasPrefix(surface.OpaqueState, "hmac-sha256:") || len(surface.OpaqueState) != len("hmac-sha256:")+64 || strings.Contains(surface.OpaqueState, root) {
			t.Fatal("snapshot was not an opaque per-run HMAC")
		}
	}

	homePath, err := resolver.resolve("home:.zshrc")
	if err != nil {
		t.Fatal("synthetic home target unavailable")
	}
	info, err := os.Stat(homePath)
	if err != nil {
		t.Fatal("synthetic home target unavailable")
	}
	originalTime := info.ModTime()
	if err := os.WriteFile(homePath, []byte("shell-entry-v2"), 0o600); err != nil {
		t.Fatal("equal-size replacement setup failed")
	}
	if err := os.Chtimes(homePath, originalTime, originalTime); err != nil {
		t.Fatal("equal-mtime replacement setup failed")
	}
	changed, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{})
	if err != nil || sameOpaqueState(before, changed) {
		t.Fatal("equal-mtime equal-count content replacement was missed")
	}
}

func testIncompleteSnapshots(t *testing.T) {
	manifest := loadProtectedManifest(t)
	key := bytes.Repeat([]byte{0x33}, 32)

	t.Run("unreadable", func(t *testing.T) {
		root := t.TempDir()
		resolver, _ := PrepareProtectedSynthetic(root)
		path, _ := resolver.resolve("home:.zshrc")
		if err := os.Chmod(path, 0); err != nil {
			t.Fatal("unreadable setup failed")
		}
		defer os.Chmod(path, 0o600)
		frozen, _ := FreezeProtectedManifest(manifest)
		snapshot, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{})
		if err != nil || reasonFor(snapshot, "home:.zshrc") != ReasonUnreadable {
			t.Fatal("unreadable surface did not become incomplete")
		}
	})

	t.Run("race", func(t *testing.T) {
		root := t.TempDir()
		resolver, _ := PrepareProtectedSynthetic(root)
		frozen, _ := FreezeProtectedManifest(manifest)
		changed := false
		snapshot, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{BetweenPasses: func(logicalRef string) {
			if logicalRef == "home:.zshrc" && !changed {
				changed = true
				path, _ := resolver.resolve(logicalRef)
				_ = os.WriteFile(path, []byte("shell-entry-v2"), 0o600)
			}
		}})
		if err != nil || reasonFor(snapshot, "home:.zshrc") != ReasonRace {
			t.Fatal("in-window race did not become incomplete")
		}
	})

	t.Run("overflow", func(t *testing.T) {
		root := t.TempDir()
		resolver, _ := PrepareProtectedSynthetic(root)
		bounded := manifest
		bounded.Surfaces = append([]ProtectedSurface(nil), manifest.Surfaces...)
		for index := range bounded.Surfaces {
			if bounded.Surfaces[index].LogicalRef == "home:.zshrc" {
				bounded.Surfaces[index].Bounds.MaxBytes = 1
			}
		}
		bounded = reparseManifest(t, bounded)
		frozen, _ := FreezeProtectedManifest(bounded)
		snapshot, err := SnapshotProtected(frozen, bounded, resolver, key, SnapshotOptions{})
		if err != nil || reasonFor(snapshot, "home:.zshrc") != ReasonOverflow {
			t.Fatal("byte overflow did not become incomplete")
		}
	})

	t.Run("symlink escape", func(t *testing.T) {
		root := t.TempDir()
		resolver, _ := PrepareProtectedSynthetic(root)
		path, _ := resolver.resolve("home:.zshrc")
		if err := os.Remove(path); err != nil {
			t.Fatal("symlink escape setup failed")
		}
		outside := filepath.Join(filepath.Dir(root), "outside-synthetic-surface")
		if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
			t.Fatal("symlink escape setup failed")
		}
		defer os.Remove(outside)
		if err := os.Symlink(outside, path); err != nil {
			t.Fatal("symlink escape setup failed")
		}
		frozen, _ := FreezeProtectedManifest(manifest)
		snapshot, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{})
		if err != nil || reasonFor(snapshot, "home:.zshrc") != ReasonSymlinkEscape {
			t.Fatal("symlink escape did not become incomplete")
		}
	})

	t.Run("window", func(t *testing.T) {
		root := t.TempDir()
		resolver, _ := PrepareProtectedSynthetic(root)
		frozen, _ := FreezeProtectedManifest(manifest)
		current := time.Unix(0, 0)
		clock := func() time.Time {
			current = current.Add(10 * time.Second)
			return current
		}
		snapshot, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{Clock: clock})
		if err != nil || reasonFor(snapshot, "home:.zshrc") != ReasonWindow {
			t.Fatal("window overflow did not become incomplete")
		}
	})
}

func testSentinelManifestCLI(t *testing.T) {
	safetyRoot := safetyRoot(t)
	root := t.TempDir()
	if _, err := PrepareProtectedSynthetic(root); err != nil {
		t.Fatal("synthetic CLI surface setup failed")
	}
	command := exec.Command("go", "run", "./cmd/yamc-safety", "sentinel", "verify", "--mode", "synthetic", "--manifest", filepath.Join(safetyRoot, "manifests", "protected-surfaces.v1.json"), "--fixture-root", root)
	command.Dir = safetyRoot
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil || stderr.Len() != 0 {
		t.Fatal("synthetic sentinel CLI failed")
	}
	if bytes.Contains(stdout.Bytes(), []byte(root)) || bytes.Contains(stdout.Bytes(), []byte("shell-entry")) || !bytes.Contains(stdout.Bytes(), []byte(`"status":"synthetic-sentinel-passed"`)) || bytes.Contains(stdout.Bytes(), []byte("covered-surfaces-unchanged-for-run")) {
		t.Fatal("synthetic sentinel CLI leaked or overclaimed")
	}
	var output struct {
		Status   string            `json:"status"`
		Snapshot ProtectedSnapshot `json:"snapshot"`
	}
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil || output.Status != "synthetic-sentinel-passed" || !allComplete(output.Snapshot) {
		t.Fatal("synthetic sentinel CLI output changed")
	}
}

func testSentinelManifestRunner(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(safetyRoot(t), "scripts", "test.sh"))
	if err != nil {
		t.Fatal("runner source unavailable")
	}
	text := string(data)
	for _, literal := range []string{"'./internal/sentinel'", "'^TestSentinelManifest$'", "'TestSentinelManifest'", "task:sentinel-manifest)"} {
		if !strings.Contains(text, literal) {
			t.Fatalf("sentinel manifest runner literal missing: %s", literal)
		}
	}
	if strings.Count(text, "task:sentinel-manifest)") != 1 {
		t.Fatal("sentinel manifest runner label is not unique")
	}
}

func loadProtectedManifest(t *testing.T) ProtectedManifest {
	t.Helper()
	manifest, err := ParseProtectedManifest(readProtectedManifest(t))
	if err != nil {
		t.Fatal("tracked protected manifest rejected")
	}
	return manifest
}

func readProtectedManifest(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(safetyRoot(t), "manifests", "protected-surfaces.v1.json"))
	if err != nil {
		t.Fatal("tracked protected manifest unavailable")
	}
	return data
}

func reparseManifest(t *testing.T, manifest ProtectedManifest) ProtectedManifest {
	t.Helper()
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal("manifest reparse setup failed")
	}
	parsed, err := ParseProtectedManifest(encoded)
	if err != nil {
		t.Fatal("manifest reparse failed")
	}
	return parsed
}

func safetyRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal("safety root unavailable")
	}
	return root
}

func allComplete(snapshot ProtectedSnapshot) bool {
	if len(snapshot.Surfaces) != 6 || snapshot.WindowState != "closed" {
		return false
	}
	for _, surface := range snapshot.Surfaces {
		if surface.Status != ObservationComplete || surface.OpaqueState == "" || surface.Reason != "" {
			return false
		}
	}
	return true
}

func sameOpaqueState(left, right ProtectedSnapshot) bool {
	if left.ManifestDigest != right.ManifestDigest || len(left.Surfaces) != len(right.Surfaces) {
		return false
	}
	for index := range left.Surfaces {
		if left.Surfaces[index].LogicalRef != right.Surfaces[index].LogicalRef || left.Surfaces[index].OpaqueState != right.Surfaces[index].OpaqueState || left.Surfaces[index].Status != right.Surfaces[index].Status {
			return false
		}
	}
	return true
}

func reasonFor(snapshot ProtectedSnapshot, logicalRef string) IncompleteReason {
	for _, surface := range snapshot.Surfaces {
		if surface.LogicalRef == logicalRef {
			return surface.Reason
		}
	}
	return ""
}
