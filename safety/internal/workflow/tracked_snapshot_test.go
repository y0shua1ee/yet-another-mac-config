package workflow

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTrackedRepositorySnapshot(t *testing.T) {
	t.Run("one frozen head and index", testFrozenRepositoryView)
	t.Run("intermediate symlink swap", testIntermediateSymlinkSwap)
}

func testFrozenRepositoryView(t *testing.T) {
	root := newTrackedRepositoryFixture(t, map[string]string{"a.txt": "commit-a\n", "b.txt": "commit-a\n"})
	commitA := fixtureGitOutput(t, root, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("commit-b\n"), 0o600); err != nil || os.WriteFile(filepath.Join(root, "b.txt"), []byte("commit-b\n"), 0o600) != nil {
		t.Fatal("second commit fixture unavailable")
	}
	fixtureGit(t, root, "add", "--", "a.txt", "b.txt")
	fixtureGit(t, root, "-c", "user.name=synthetic-fixture", "-c", "user.email=synthetic@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "commit b")
	commitB := fixtureGitOutput(t, root, "rev-parse", "HEAD")
	fixtureGit(t, root, "checkout", "-q", commitA)

	repository, err := openTrackedRepository(root)
	if err != nil {
		t.Fatal("tracked repository snapshot unavailable")
	}
	if _, err := validateTrackedInput(filepath.Join(root, "a.txt"), repository); err != nil {
		t.Fatal("initial frozen input rejected")
	}
	fixtureGit(t, root, "checkout", "-q", commitB)
	if _, err := validateTrackedInput(filepath.Join(root, "b.txt"), repository); err == nil {
		t.Fatal("one workflow mixed inputs from two HEAD/index snapshots")
	}
}

func testIntermediateSymlinkSwap(t *testing.T) {
	root := newTrackedRepositoryFixture(t, map[string]string{"nested/input.txt": "same-bytes\n"})
	external := filepath.Join(t.TempDir(), "external")
	if err := os.Mkdir(external, 0o700); err != nil || os.WriteFile(filepath.Join(external, "input.txt"), []byte("same-bytes\n"), 0o600) != nil {
		t.Fatal("external substitution fixture unavailable")
	}
	repository, err := openTrackedRepository(root)
	if err != nil {
		t.Fatal("tracked repository reader unavailable")
	}
	called := false
	trackedInputTestHook = func(point, relative string) {
		if called || point != "after-path-check" || relative != "nested/input.txt" {
			return
		}
		called = true
		if err := os.Rename(filepath.Join(root, "nested"), filepath.Join(root, "nested-owned")); err != nil {
			t.Fatal("intermediate directory move failed")
		}
		if err := os.Symlink(external, filepath.Join(root, "nested")); err != nil {
			t.Fatal("intermediate symlink substitution failed")
		}
	}
	t.Cleanup(func() { trackedInputTestHook = nil })
	input, err := validateTrackedInput(filepath.Join(root, "nested", "input.txt"), repository)
	if !called {
		t.Fatal("intermediate swap seam was not reached")
	}
	if err == nil || bytes.Equal(input.data, []byte("same-bytes\n")) {
		t.Fatal("tracked reader consumed byte-identical data through an intermediate symlink")
	}
}

func newTrackedRepositoryFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repository")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal("tracked repository fixture unavailable")
	}
	fixtureGit(t, root, "init", "-q")
	for relative, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil || os.WriteFile(path, []byte(content), 0o600) != nil {
			t.Fatal("tracked repository file unavailable")
		}
	}
	fixtureGit(t, root, "add", "--", ".")
	fixtureGit(t, root, "-c", "user.name=synthetic-fixture", "-c", "user.email=synthetic@example.invalid", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "commit a")
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal("tracked repository canonical path unavailable")
	}
	return canonical
}

func fixtureGit(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command := exec.Command("/usr/bin/git", append([]string{"--no-lazy-fetch", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-c", "protocol.allow=never", "-C", root}, arguments...)...)
	command.Env = []string{"HOME=/var/empty", "XDG_CONFIG_HOME=/var/empty", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "LANG=C", "PATH=/usr/bin:/bin"}
	if err := command.Run(); err != nil {
		t.Fatalf("isolated Git fixture command failed: %v", err)
	}
}

func fixtureGitOutput(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("/usr/bin/git", append([]string{"--no-lazy-fetch", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-c", "protocol.allow=never", "-C", root}, arguments...)...)
	command.Env = []string{"HOME=/var/empty", "XDG_CONFIG_HOME=/var/empty", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_OPTIONAL_LOCKS=0", "GIT_NO_LAZY_FETCH=1", "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "LANG=C", "PATH=/usr/bin:/bin"}
	output, err := command.Output()
	if err != nil {
		t.Fatalf("isolated Git fixture output failed: %v", err)
	}
	return string(bytes.TrimSpace(output))
}
