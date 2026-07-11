package artifact

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreStabilizationContract(t *testing.T) {
	t.Run("fresh single writer capability", testFreshSingleWriterStore)
	t.Run("delete is append only", testStoreDeleteIsAppendOnly)
	t.Run("publish failure preserves replacement", testPublishFailurePreservesReplacement)
	t.Run("source has no pathname rollback", testStoreSourceHasNoPathnameRollback)
}

func testFreshSingleWriterStore(t *testing.T) {
	repository := repositoryRoot(t)
	root := filepath.Join(t.TempDir(), "store")
	first, err := NewStore(root, repository)
	if err != nil {
		t.Fatal("fresh store capability setup failed")
	}
	defer first.closeStoreFilesystem()
	if second, err := NewStore(root, repository); err == nil {
		second.closeStoreFilesystem()
		t.Fatal("mutable store accepted a second writer capability")
	}

	preexisting := filepath.Join(t.TempDir(), "preexisting-store")
	if err := os.Mkdir(preexisting, 0o700); err != nil {
		t.Fatal("preexisting store setup failed")
	}
	if _, err := NewStore(preexisting, repository); err == nil {
		t.Fatal("mutable store claimed a caller-created existing directory")
	}
}

func testStoreDeleteIsAppendOnly(t *testing.T) {
	repository := repositoryRoot(t)
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	clock := now
	root := filepath.Join(t.TempDir(), "store")
	store, err := NewStoreWithClock(root, repository, func() time.Time { return clock })
	if err != nil {
		t.Fatal("append-only store setup failed")
	}
	defer store.closeStoreFilesystem()
	canonical, envelope := stabilizationDesiredArtifact(t, now)
	if _, err := store.Write(canonical); err != nil {
		t.Fatal("append-only object setup failed")
	}
	path := filepath.Join(root, "sha256", strings.TrimPrefix(envelope.ContentDigest, "sha256:"))
	before, err := os.Lstat(path)
	if err != nil {
		t.Fatal("append-only object identity unavailable")
	}
	clock = now.Add(25 * time.Hour)
	if err := store.Delete(envelope.ContentDigest); err == nil {
		t.Fatal("store exposed physical object deletion")
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("delete request removed or replaced immutable object bytes")
	}
}

func testPublishFailurePreservesReplacement(t *testing.T) {
	repository := repositoryRoot(t)
	root := filepath.Join(t.TempDir(), "store")
	store, err := NewStore(root, repository)
	if err != nil {
		t.Fatal("publish replacement store setup failed")
	}
	defer store.closeStoreFilesystem()
	canonical, envelope := stabilizationDesiredArtifact(t, time.Now().UTC().Add(-time.Minute).Truncate(time.Second))
	replacement := []byte("replacement-must-survive")
	storeFilesystemTestHook = func(point string, directory *storeDirectory, name string) error {
		if point != "publish-after-link" {
			return nil
		}
		if err := directory.root.Rename(name, name+".owned"); err != nil {
			t.Fatal("published object quarantine setup failed")
		}
		file, err := directory.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			t.Fatal("replacement object setup failed")
		}
		if _, err := file.Write(replacement); err != nil || file.Close() != nil {
			t.Fatal("replacement object write failed")
		}
		return errors.New("injected post-link failure")
	}
	t.Cleanup(func() { storeFilesystemTestHook = nil })
	if _, err := store.Write(canonical); err == nil {
		t.Fatal("injected post-link failure unexpectedly passed")
	}
	path := filepath.Join(root, "sha256", strings.TrimPrefix(envelope.ContentDigest, "sha256:"))
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, replacement) {
		t.Fatal("publish rollback deleted or changed a replacement object")
	}
}

func testStoreSourceHasNoPathnameRollback(t *testing.T) {
	root := repositoryRoot(t)
	filesystemSource, err := os.ReadFile(filepath.Join(root, "safety", "internal", "artifact", "store_fs.go"))
	if err != nil {
		t.Fatal("store filesystem source unavailable")
	}
	storeSource, err := os.ReadFile(filepath.Join(root, "safety", "internal", "artifact", "store.go"))
	if err != nil {
		t.Fatal("store source unavailable")
	}
	for _, forbidden := range [][]byte{
		[]byte("directory.root.Remove("),
		[]byte("removeCreatedObjectFile"),
		[]byte("removeObjectFile("),
		[]byte("removeTransitionFile("),
	} {
		if bytes.Contains(filesystemSource, forbidden) || bytes.Contains(storeSource, forbidden) {
			t.Fatalf("store retains a pathname rollback/delete primitive: %s", forbidden)
		}
	}
}

func stabilizationDesiredArtifact(t *testing.T, createdAt time.Time) ([]byte, Envelope) {
	t.Helper()
	run, err := NewRunMetadata([]byte("store-stabilization"), "offline-static", "artifact-kinds")
	if err != nil {
		t.Fatal("stabilization run metadata unavailable")
	}
	options, err := DefaultBuildOptions(DesiredState, createdAt)
	if err != nil {
		t.Fatal("stabilization lifecycle unavailable")
	}
	canonical, envelope, err := NewWithOptions(
		DesiredState,
		run,
		Provenance{Mode: "synthetic", InputDigests: []string{}},
		DesiredPayload{Profile: "profile:synthetic-developer", Declarations: []Fact{{Ref: "repo:synthetic/config", State: "fixture:state/declared"}}},
		options,
	)
	if err != nil {
		t.Fatal("stabilization artifact unavailable")
	}
	return canonical, envelope
}
