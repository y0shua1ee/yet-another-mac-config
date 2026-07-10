package artifact

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	root string
}

func NewStore(root, repositoryRoot string) (*Store, error) {
	validated, err := ValidateExternalRoot(root, repositoryRoot)
	if err != nil {
		return nil, err
	}
	return &Store{root: validated}, nil
}

func ValidateExternalRoot(root, repositoryRoot string) (string, error) {
	if root == "" || repositoryRoot == "" || !filepath.IsAbs(root) || containsParentReference(root) {
		return "", errors.New("external root rejected")
	}
	repository, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return "", errors.New("repository root rejected")
	}
	repository, err = filepath.Abs(repository)
	if err != nil {
		return "", errors.New("repository root rejected")
	}
	candidate, err := canonicalForCreation(root)
	if err != nil {
		return "", err
	}
	inside, err := isWithin(repository, candidate)
	if err != nil || inside {
		return "", errors.New("external root overlaps repository")
	}
	return candidate, nil
}

func (store *Store) Write(canonical []byte) (string, error) {
	envelope, err := DecodeAndValidate(canonical)
	if err != nil {
		return "", err
	}
	digestName := strings.TrimPrefix(envelope.ContentDigest, "sha256:")
	objectDirectory := filepath.Join(store.root, "sha256")
	if err := os.MkdirAll(objectDirectory, 0o700); err != nil {
		return "", errors.New("artifact store unavailable")
	}
	objectPath := filepath.Join(objectDirectory, digestName)
	if existing, err := os.ReadFile(objectPath); err == nil {
		if !bytes.Equal(existing, canonical) {
			return "", errors.New("artifact store collision")
		}
		return envelope.ContentDigest, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("artifact store unavailable")
	}

	temporary, err := os.CreateTemp(objectDirectory, ".pending-")
	if err != nil {
		return "", errors.New("artifact store unavailable")
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return "", errors.New("artifact store unavailable")
	}
	if _, err := io.Copy(temporary, bytes.NewReader(canonical)); err != nil {
		return "", errors.New("artifact store unavailable")
	}
	if err := temporary.Sync(); err != nil {
		return "", errors.New("artifact store unavailable")
	}
	if err := temporary.Close(); err != nil {
		return "", errors.New("artifact store unavailable")
	}
	if err := os.Rename(temporaryPath, objectPath); err != nil {
		return "", errors.New("artifact store unavailable")
	}
	committed = true
	return envelope.ContentDigest, nil
}

func canonicalForCreation(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("external root rejected")
	}
	current := filepath.Clean(absolute)
	var missing []string
	for {
		_, err := os.Lstat(current)
		if err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(current)
			if resolveErr != nil {
				return "", errors.New("external root rejected")
			}
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", errors.New("external root rejected")
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("external root rejected")
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func containsParentReference(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func isWithin(parent, child string) (bool, error) {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false, err
	}
	if relative == "." {
		return true, nil
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
}
