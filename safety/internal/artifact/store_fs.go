package artifact

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var errStoreFileCollision = errors.New("store file collision")

// storeFilesystemTestHook 只为同包的受控竞态测试提供确定性调度点。
// production 默认值恒为 nil，调用方不能通过公开 API 或环境变量设置它。
var storeFilesystemTestHook func(point string, directory *storeDirectory, name string) error

func runStoreFilesystemTestHook(point string, directory *storeDirectory, name string) error {
	if storeFilesystemTestHook == nil {
		return nil
	}
	return storeFilesystemTestHook(point, directory, name)
}

type storeDirectory struct {
	path     string
	root     *os.Root
	identity os.FileInfo
}

func initializeStoreFilesystem(root string) (os.FileInfo, *storeDirectory, *storeDirectory, error) {
	parentPath := filepath.Dir(root)
	base := filepath.Base(root)
	if !validStoreFileName(base) {
		return nil, nil, nil, errors.New("store root rejected")
	}
	parentIdentity, err := exactDirectoryIdentity(parentPath)
	if err != nil {
		return nil, nil, nil, err
	}
	parentRoot, err := os.OpenRoot(parentPath)
	if err != nil {
		return nil, nil, nil, err
	}
	defer parentRoot.Close()
	openedParent, err := parentRoot.Stat(".")
	if err != nil || !os.SameFile(parentIdentity, openedParent) {
		return nil, nil, nil, errors.New("store parent replaced")
	}
	createdRoot := false
	if err := parentRoot.Mkdir(base, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, nil, nil, err
		}
	} else {
		createdRoot = true
	}
	rootIdentity, err := parentRoot.Lstat(base)
	if err != nil || !rootIdentity.IsDir() || rootIdentity.Mode()&os.ModeSymlink != 0 {
		if createdRoot {
			_ = parentRoot.Remove(base)
		}
		return nil, nil, nil, errors.New("store root rejected")
	}
	rootHandle, err := parentRoot.OpenRoot(base)
	if err != nil {
		if createdRoot {
			_ = parentRoot.Remove(base)
		}
		return nil, nil, nil, err
	}
	defer rootHandle.Close()
	openedRoot, err := rootHandle.Stat(".")
	if err != nil || !os.SameFile(rootIdentity, openedRoot) {
		if createdRoot {
			_ = parentRoot.Remove(base)
		}
		return nil, nil, nil, errors.New("store root replaced")
	}

	objects, objectsCreated, err := openExactStoreDirectory(root, rootHandle, "sha256")
	if err != nil {
		if createdRoot {
			_ = parentRoot.Remove(base)
		}
		return nil, nil, nil, err
	}
	transitions, transitionsCreated, err := openExactStoreDirectory(root, rootHandle, "transitions")
	if err != nil {
		_ = objects.root.Close()
		if objectsCreated {
			_ = rootHandle.Remove("sha256")
		}
		if createdRoot {
			_ = parentRoot.Remove(base)
		}
		return nil, nil, nil, err
	}
	rollback := func() {
		_ = transitions.root.Close()
		_ = objects.root.Close()
		if transitionsCreated {
			_ = rootHandle.Remove("transitions")
		}
		if objectsCreated {
			_ = rootHandle.Remove("sha256")
		}
		if createdRoot {
			_ = parentRoot.Remove(base)
		}
	}
	rootNamedAgain, rootErr := exactDirectoryIdentity(root)
	objectsNamedAgain, objectsErr := exactDirectoryIdentity(objects.path)
	transitionsNamedAgain, transitionsErr := exactDirectoryIdentity(transitions.path)
	parentNamedAgain, parentErr := exactDirectoryIdentity(parentPath)
	if rootErr != nil || objectsErr != nil || transitionsErr != nil || parentErr != nil || !os.SameFile(rootIdentity, rootNamedAgain) || !os.SameFile(objects.identity, objectsNamedAgain) || !os.SameFile(transitions.identity, transitionsNamedAgain) || !os.SameFile(parentIdentity, parentNamedAgain) {
		rollback()
		return nil, nil, nil, errors.New("store filesystem replaced")
	}
	return rootIdentity, objects, transitions, nil
}

func exactDirectoryIdentity(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("store directory rejected")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, errors.New("store directory rejected")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(path) {
		return nil, errors.New("store directory rejected")
	}
	return info, nil
}

func openExactStoreDirectory(root string, rootHandle *os.Root, name string) (*storeDirectory, bool, error) {
	path := filepath.Join(root, name)
	created := false
	if err := rootHandle.Mkdir(name, 0o700); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, false, err
		}
	} else {
		created = true
	}
	identity, err := rootHandle.Lstat(name)
	if err != nil || !identity.IsDir() || identity.Mode()&os.ModeSymlink != 0 {
		if created {
			_ = rootHandle.Remove(name)
		}
		return nil, false, errors.New("store directory rejected")
	}
	inside, err := isWithin(root, path)
	if err != nil || !inside || filepath.Dir(path) != root {
		if created {
			_ = rootHandle.Remove(name)
		}
		return nil, false, errors.New("store directory rejected")
	}
	rooted, err := rootHandle.OpenRoot(name)
	if err != nil {
		if created {
			_ = rootHandle.Remove(name)
		}
		return nil, false, err
	}
	opened, err := rooted.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(identity, opened) {
		_ = rooted.Close()
		if created {
			_ = rootHandle.Remove(name)
		}
		return nil, false, errors.New("store directory rejected")
	}
	namedAgain, err := rootHandle.Lstat(name)
	if err != nil || !os.SameFile(identity, namedAgain) {
		_ = rooted.Close()
		if created {
			_ = rootHandle.Remove(name)
		}
		return nil, false, errors.New("store directory rejected")
	}
	return &storeDirectory{path: path, root: rooted, identity: identity}, created, nil
}

func (store *Store) verifyStoreDirectory(directory *storeDirectory) error {
	if store == nil || directory == nil || directory.root == nil || store.rootIdentity == nil {
		return errors.New("store directory rejected")
	}
	rootInfo, err := exactDirectoryIdentity(store.root)
	if err != nil || !os.SameFile(store.rootIdentity, rootInfo) {
		return errors.New("store directory replaced")
	}
	named, err := exactDirectoryIdentity(directory.path)
	if err != nil || !os.SameFile(directory.identity, named) {
		return errors.New("store directory replaced")
	}
	opened, err := directory.root.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(directory.identity, opened) || !os.SameFile(named, opened) {
		return errors.New("store directory replaced")
	}
	inside, err := isWithin(store.root, directory.path)
	if err != nil || !inside || filepath.Dir(directory.path) != store.root {
		return errors.New("store directory escaped")
	}
	return nil
}

func (store *Store) readObjectFile(digest string) ([]byte, error) {
	return store.readStoreFile(store.objectDirectory, strings.TrimPrefix(digest, "sha256:"))
}

func (store *Store) readTransitionFile(planDigest string) ([]byte, error) {
	return store.readStoreFile(store.transitionDirectory, strings.TrimPrefix(planDigest, "sha256:")+".json")
}

func (store *Store) readStoreFile(directory *storeDirectory, name string) ([]byte, error) {
	if !validStoreFileName(name) {
		return nil, errors.New("store file rejected")
	}
	if err := store.verifyStoreDirectory(directory); err != nil {
		return nil, err
	}
	before, err := directory.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Size() < 0 || before.Size() > maxStoredArtifactBytes {
		return nil, errors.New("store file rejected")
	}
	file, err := directory.root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, errors.New("store file replaced")
	}
	namedAgain, err := directory.root.Lstat(name)
	if err != nil || !namedAgain.Mode().IsRegular() || namedAgain.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, namedAgain) {
		return nil, errors.New("store file replaced")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxStoredArtifactBytes+1))
	if err != nil || len(data) > maxStoredArtifactBytes {
		return nil, errors.New("store file rejected")
	}
	openedAfter, openedErr := file.Stat()
	after, afterErr := directory.root.Lstat(name)
	if openedErr != nil || afterErr != nil || !openedAfter.Mode().IsRegular() || !after.Mode().IsRegular() || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, openedAfter) || !os.SameFile(before, after) || before.Size() != openedAfter.Size() || before.Size() != after.Size() || before.Mode() != openedAfter.Mode() || before.Mode() != after.Mode() || !before.ModTime().Equal(openedAfter.ModTime()) || !before.ModTime().Equal(after.ModTime()) {
		return nil, errors.New("store file changed during read")
	}
	if err := store.verifyStoreDirectory(directory); err != nil {
		return nil, err
	}
	return data, nil
}

func (store *Store) publishObjectFile(digest string, data []byte) (bool, error) {
	return store.publishStoreFile(store.objectDirectory, strings.TrimPrefix(digest, "sha256:"), data)
}

func (store *Store) publishTransitionFile(planDigest string, data []byte) (bool, error) {
	return store.publishStoreFile(store.transitionDirectory, strings.TrimPrefix(planDigest, "sha256:")+".json", data)
}

func (store *Store) publishStoreFile(directory *storeDirectory, name string, data []byte) (created bool, resultErr error) {
	published := false
	if !validStoreFileName(name) || len(data) == 0 || len(data) > maxStoredArtifactBytes {
		return false, errors.New("store file rejected")
	}
	if existing, err := store.readStoreFile(directory, name); err == nil {
		if bytes.Equal(existing, data) {
			return false, nil
		}
		return false, errStoreFileCollision
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := store.verifyStoreDirectory(directory); err != nil {
		return false, err
	}
	temporaryName, err := storeTemporaryName()
	if err != nil {
		return false, err
	}
	temporary, err := directory.root.OpenFile(temporaryName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = temporary.Close()
		_ = directory.root.Remove(temporaryName)
	}()
	if _, err := io.Copy(temporary, bytes.NewReader(data)); err != nil {
		return false, err
	}
	if err := temporary.Sync(); err != nil {
		return false, err
	}
	if err := temporary.Close(); err != nil {
		return false, err
	}
	if err := directory.root.Link(temporaryName, name); err != nil {
		if existing, readErr := store.readStoreFile(directory, name); readErr == nil && bytes.Equal(existing, data) {
			return false, nil
		}
		return false, errStoreFileCollision
	}
	created = true
	published = true
	defer func() {
		if resultErr != nil && published {
			_ = directory.root.Remove(name)
			_ = syncStoreDirectory(directory)
		}
	}()
	if err := runStoreFilesystemTestHook("publish-after-link", directory, name); err != nil {
		return false, err
	}
	if err := syncStoreDirectory(directory); err != nil {
		return false, err
	}
	if err := store.verifyStoreDirectory(directory); err != nil {
		return false, err
	}
	return true, nil
}

func (store *Store) removeObjectFile(digest string) error {
	return store.removeStoreFile(store.objectDirectory, strings.TrimPrefix(digest, "sha256:"))
}

func (store *Store) removeTransitionFile(planDigest string) error {
	return store.removeStoreFile(store.transitionDirectory, strings.TrimPrefix(planDigest, "sha256:")+".json")
}

func (store *Store) removeStoreFile(directory *storeDirectory, name string) error {
	if !validStoreFileName(name) {
		return errors.New("store file rejected")
	}
	if err := store.verifyStoreDirectory(directory); err != nil {
		return err
	}
	info, err := directory.root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("store file rejected")
	}
	if err := runStoreFilesystemTestHook("delete-before-remove", directory, name); err != nil {
		return err
	}
	if err := directory.root.Remove(name); err != nil {
		return err
	}
	if err := syncStoreDirectory(directory); err != nil {
		return err
	}
	return store.verifyStoreDirectory(directory)
}

func (store *Store) removeCreatedObjectFile(digest string) {
	name := strings.TrimPrefix(digest, "sha256:")
	if validStoreFileName(name) && store.objectDirectory != nil && store.objectDirectory.root != nil {
		_ = store.objectDirectory.root.Remove(name)
		_ = syncStoreDirectory(store.objectDirectory)
	}
}

func (store *Store) objectDirectoryEntries() ([]os.DirEntry, error) {
	return store.storeDirectoryEntries(store.objectDirectory)
}

func (store *Store) transitionDirectoryEntries() ([]os.DirEntry, error) {
	return store.storeDirectoryEntries(store.transitionDirectory)
}

func (store *Store) storeDirectoryEntries(target *storeDirectory) ([]os.DirEntry, error) {
	if err := store.verifyStoreDirectory(target); err != nil {
		return nil, err
	}
	directory, err := target.root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, readErr := directory.ReadDir(-1)
	closeErr := directory.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err := store.verifyStoreDirectory(target); err != nil {
		return nil, err
	}
	return entries, nil
}

func syncStoreDirectory(directory *storeDirectory) error {
	if directory == nil || directory.root == nil {
		return errors.New("store directory rejected")
	}
	file, err := directory.root.Open(".")
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func storeTemporaryName() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return ".pending-" + hex.EncodeToString(random[:]), nil
}

func validStoreFileName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\\x00\r\n\t")
}

func (store *Store) closeStoreFilesystem() {
	if store == nil {
		return
	}
	if store.objectDirectory != nil && store.objectDirectory.root != nil {
		_ = store.objectDirectory.root.Close()
	}
	if store.transitionDirectory != nil && store.transitionDirectory.root != nil {
		_ = store.transitionDirectory.root.Close()
	}
}
