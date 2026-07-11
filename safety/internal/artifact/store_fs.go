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

const storeClaimFileName = ".yamc-store-capability"

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
	name     string
	path     string
	root     *os.Root
	identity os.FileInfo
}

type storeFilesystem struct {
	parentPath     string
	base           string
	parentRoot     *os.Root
	parentIdentity os.FileInfo
	rootHandle     *os.Root
	rootIdentity   os.FileInfo
	claimFile      *os.File
	claimIdentity  os.FileInfo
}

func initializeStoreFilesystem(root string, mutable bool) (*storeFilesystem, *storeDirectory, *storeDirectory, error) {
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
	failed := true
	defer func() {
		if failed {
			_ = parentRoot.Close()
		}
	}()
	openedParent, err := parentRoot.Stat(".")
	if err != nil || !os.SameFile(parentIdentity, openedParent) {
		return nil, nil, nil, errors.New("store parent replaced")
	}
	if mutable {
		// mutable store 必须由当前调用独占创建；existing pathname 只能走只读 reopen。
		if err := parentRoot.Mkdir(base, 0o700); err != nil {
			return nil, nil, nil, errors.New("store root is not fresh")
		}
	}
	rootIdentity, err := parentRoot.Lstat(base)
	if err != nil || !rootIdentity.IsDir() || rootIdentity.Mode()&os.ModeSymlink != 0 {
		return nil, nil, nil, errors.New("store root rejected")
	}
	rootHandle, err := parentRoot.OpenRoot(base)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if failed {
			_ = rootHandle.Close()
		}
	}()
	openedRoot, err := rootHandle.Stat(".")
	if err != nil || !os.SameFile(rootIdentity, openedRoot) {
		return nil, nil, nil, errors.New("store root replaced")
	}

	claimFile, claimIdentity, err := openStoreClaim(rootHandle, mutable)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if failed {
			_ = claimFile.Close()
		}
	}()
	objects, err := openExactStoreDirectory(root, rootHandle, "sha256", mutable)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if failed {
			_ = objects.root.Close()
		}
	}()
	transitions, err := openExactStoreDirectory(root, rootHandle, "transitions", mutable)
	if err != nil {
		return nil, nil, nil, err
	}
	filesystem := &storeFilesystem{
		parentPath: parentPath, base: base, parentRoot: parentRoot, parentIdentity: parentIdentity,
		rootHandle: rootHandle, rootIdentity: rootIdentity, claimFile: claimFile, claimIdentity: claimIdentity,
	}
	if err := verifyStoreFilesystem(filesystem, objects, transitions); err != nil {
		_ = transitions.root.Close()
		return nil, nil, nil, err
	}
	failed = false
	return filesystem, objects, transitions, nil
}

func openStoreClaim(root *os.Root, create bool) (*os.File, os.FileInfo, error) {
	if create {
		name, err := storeTemporaryName()
		if err != nil {
			return nil, nil, err
		}
		claim, err := root.OpenFile(storeClaimFileName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, nil, errors.New("store capability unavailable")
		}
		failed := true
		defer func() {
			if failed {
				_ = claim.Close()
			}
		}()
		if _, err := io.WriteString(claim, strings.TrimPrefix(name, ".pending-")+"\n"); err != nil || claim.Sync() != nil {
			return nil, nil, errors.New("store capability unavailable")
		}
		identity, err := claim.Stat()
		if err != nil || !identity.Mode().IsRegular() || identity.Size() != 33 {
			return nil, nil, errors.New("store capability rejected")
		}
		failed = false
		return claim, identity, nil
	}
	identity, err := root.Lstat(storeClaimFileName)
	if err != nil || !identity.Mode().IsRegular() || identity.Mode()&os.ModeSymlink != 0 || identity.Size() != 33 {
		return nil, nil, errors.New("store capability rejected")
	}
	claim, err := root.OpenFile(storeClaimFileName, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, errors.New("store capability rejected")
	}
	opened, err := claim.Stat()
	if err != nil || !os.SameFile(identity, opened) {
		_ = claim.Close()
		return nil, nil, errors.New("store capability replaced")
	}
	data, err := io.ReadAll(io.LimitReader(claim, 34))
	if err != nil || len(data) != 33 || data[32] != '\n' {
		_ = claim.Close()
		return nil, nil, errors.New("store capability rejected")
	}
	if _, err := hex.DecodeString(string(data[:32])); err != nil {
		_ = claim.Close()
		return nil, nil, errors.New("store capability rejected")
	}
	return claim, identity, nil
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

func openExactStoreDirectory(root string, rootHandle *os.Root, name string, create bool) (*storeDirectory, error) {
	path := filepath.Join(root, name)
	if create {
		if err := rootHandle.Mkdir(name, 0o700); err != nil {
			return nil, errors.New("store directory unavailable")
		}
	}
	identity, err := rootHandle.Lstat(name)
	if err != nil || !identity.IsDir() || identity.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("store directory rejected")
	}
	inside, err := isWithin(root, path)
	if err != nil || !inside || filepath.Dir(path) != root {
		return nil, errors.New("store directory rejected")
	}
	rooted, err := rootHandle.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, err := rooted.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(identity, opened) {
		_ = rooted.Close()
		return nil, errors.New("store directory rejected")
	}
	namedAgain, err := rootHandle.Lstat(name)
	if err != nil || !os.SameFile(identity, namedAgain) {
		_ = rooted.Close()
		return nil, errors.New("store directory rejected")
	}
	return &storeDirectory{name: name, path: path, root: rooted, identity: identity}, nil
}

func (store *Store) verifyStoreDirectory(directory *storeDirectory) error {
	if store == nil || store.filesystem == nil || directory == nil || directory.root == nil {
		return errors.New("store directory rejected")
	}
	return verifyStoreFilesystem(store.filesystem, directory)
}

func verifyStoreFilesystem(filesystem *storeFilesystem, directories ...*storeDirectory) error {
	if filesystem == nil || filesystem.parentRoot == nil || filesystem.rootHandle == nil || filesystem.claimFile == nil {
		return errors.New("store filesystem rejected")
	}
	parentNamed, err := exactDirectoryIdentity(filesystem.parentPath)
	parentOpened, openedErr := filesystem.parentRoot.Stat(".")
	if err != nil || openedErr != nil || !os.SameFile(filesystem.parentIdentity, parentNamed) || !os.SameFile(filesystem.parentIdentity, parentOpened) {
		return errors.New("store parent replaced")
	}
	rootNamed, err := filesystem.parentRoot.Lstat(filesystem.base)
	rootOpened, openedErr := filesystem.rootHandle.Stat(".")
	if err != nil || openedErr != nil || !rootNamed.IsDir() || rootNamed.Mode()&os.ModeSymlink != 0 || !os.SameFile(filesystem.rootIdentity, rootNamed) || !os.SameFile(filesystem.rootIdentity, rootOpened) {
		return errors.New("store root replaced")
	}
	claimNamed, err := filesystem.rootHandle.Lstat(storeClaimFileName)
	claimOpened, openedErr := filesystem.claimFile.Stat()
	if err != nil || openedErr != nil || !claimNamed.Mode().IsRegular() || claimNamed.Mode()&os.ModeSymlink != 0 || !os.SameFile(filesystem.claimIdentity, claimNamed) || !os.SameFile(filesystem.claimIdentity, claimOpened) {
		return errors.New("store capability replaced")
	}
	for _, directory := range directories {
		if directory == nil || directory.root == nil {
			return errors.New("store directory rejected")
		}
		named, nameErr := filesystem.rootHandle.Lstat(directory.name)
		opened, openErr := directory.root.Stat(".")
		if nameErr != nil || openErr != nil || !named.IsDir() || named.Mode()&os.ModeSymlink != 0 || !os.SameFile(directory.identity, named) || !os.SameFile(directory.identity, opened) {
			return errors.New("store directory replaced")
		}
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

func (store *Store) publishStoreFile(directory *storeDirectory, name string, data []byte) (bool, error) {
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
	defer temporary.Close()
	if _, err := io.Copy(temporary, bytes.NewReader(data)); err != nil {
		return false, err
	}
	if err := temporary.Sync(); err != nil {
		return false, err
	}
	if err := temporary.Close(); err != nil {
		return false, err
	}
	if err := store.verifyStoreDirectory(directory); err != nil {
		return false, err
	}
	if err := directory.root.Link(temporaryName, name); err != nil {
		if existing, readErr := store.readStoreFile(directory, name); readErr == nil && bytes.Equal(existing, data) {
			return false, nil
		}
		return false, errStoreFileCollision
	}
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
	result := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".pending-") {
			info, infoErr := target.root.Lstat(entry.Name())
			if infoErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 0 || info.Size() > maxStoredArtifactBytes {
				return nil, errors.New("store staging entry rejected")
			}
			continue
		}
		result = append(result, entry)
	}
	return result, nil
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
	if store.filesystem != nil {
		if store.filesystem.claimFile != nil {
			_ = store.filesystem.claimFile.Close()
		}
		if store.filesystem.rootHandle != nil {
			_ = store.filesystem.rootHandle.Close()
		}
		if store.filesystem.parentRoot != nil {
			_ = store.filesystem.parentRoot.Close()
		}
	}
}
