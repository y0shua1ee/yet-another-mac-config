package sentinel

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type ObservationStatus string

const (
	ObservationComplete   ObservationStatus = "complete"
	ObservationIncomplete ObservationStatus = "incomplete"
)

type IncompleteReason string

const (
	ReasonUnreadable    IncompleteReason = "unreadable"
	ReasonRace          IncompleteReason = "race-detected"
	ReasonOverflow      IncompleteReason = "bound-exceeded"
	ReasonSymlinkEscape IncompleteReason = "symlink-escape"
	ReasonWindow        IncompleteReason = "window-exceeded"
)

type SurfaceSnapshot struct {
	SurfaceID     string            `json:"surface_id"`
	SurfaceDomain string            `json:"surface_domain"`
	LogicalRef    string            `json:"logical_ref"`
	Status        ObservationStatus `json:"status"`
	OpaqueState   string            `json:"opaque_snapshot,omitempty"`
	Reason        IncompleteReason  `json:"reason,omitempty"`
}

type ProtectedSnapshot struct {
	ManifestDigest string            `json:"manifest_digest"`
	WindowState    string            `json:"window_state"`
	Surfaces       []SurfaceSnapshot `json:"surfaces"`
}

type SnapshotOptions struct {
	Clock         func() time.Time
	BetweenPasses func(logicalRef string)
	DuringRead    func(path string)
}

type SyntheticResolver struct {
	root    string
	targets map[string]string
}

type nodeFact struct {
	RelativeID string `json:"relative_id"`
	Kind       string `json:"kind"`
	Mode       uint32 `json:"mode"`
	Size       int64  `json:"size"`
	Content    string `json:"content,omitempty"`
	Link       string `json:"link,omitempty"`
}

type fingerprintLimits struct {
	files int
	bytes int
}

var syntheticTargets = map[string]string{
	"repo:sentinel/worktree/tracked":               "protected/worktree/tracked",
	"repo:sentinel/worktree/index":                 "protected/worktree/index",
	"home:.zshrc":                                  "protected/home/zshrc",
	"home:sentinel/manager/mise-data":              "protected/manager/mise-data",
	"profile:sentinel/service/homebrew-mxcl-nginx": "protected/service/homebrew-mxcl-nginx",
	"profile:sentinel/named-target/system-shells":  "protected/named-target/system-shells",
}

func NewSyntheticResolver(root string) (*SyntheticResolver, error) {
	if root == "" || !filepath.IsAbs(root) {
		return nil, errors.New("synthetic resolver rejected")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("synthetic resolver rejected")
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, errors.New("synthetic resolver rejected")
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return nil, errors.New("synthetic resolver rejected")
	}
	targets := make(map[string]string, len(syntheticTargets))
	for logicalRef, relative := range syntheticTargets {
		targets[logicalRef] = relative
	}
	return &SyntheticResolver{root: canonical, targets: targets}, nil
}

func PrepareProtectedSynthetic(root string) (*SyntheticResolver, error) {
	resolver, err := NewSyntheticResolver(root)
	if err != nil {
		return nil, err
	}
	contents := map[string]string{
		"repo:sentinel/worktree/tracked":               "tracked-state-v1",
		"repo:sentinel/worktree/index":                 "index-state-v1",
		"home:.zshrc":                                  "shell-entry-v1",
		"profile:sentinel/service/homebrew-mxcl-nginx": "service-absent-v1",
		"profile:sentinel/named-target/system-shells":  "system-shells-v1",
	}
	for logicalRef, content := range contents {
		path, resolveErr := resolver.resolve(logicalRef)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, errors.New("synthetic protected surface unavailable")
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return nil, errors.New("synthetic protected surface unavailable")
		}
	}
	managerPath, err := resolver.resolve("home:sentinel/manager/mise-data")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(managerPath, "versions"), 0o700); err != nil {
		return nil, errors.New("synthetic protected surface unavailable")
	}
	if err := os.WriteFile(filepath.Join(managerPath, "versions", "state"), []byte("manager-state-v1"), 0o600); err != nil {
		return nil, errors.New("synthetic protected surface unavailable")
	}
	return resolver, nil
}

func SnapshotProtected(frozen *FrozenManifest, current ProtectedManifest, resolver *SyntheticResolver, key []byte, options SnapshotOptions) (ProtectedSnapshot, error) {
	if err := frozen.ValidateCurrent(current); err != nil {
		return ProtectedSnapshot{}, err
	}
	if resolver == nil || len(key) < 32 {
		return ProtectedSnapshot{}, errors.New("protected snapshot input rejected")
	}
	clock := options.Clock
	if clock == nil {
		clock = time.Now
	}
	result := ProtectedSnapshot{ManifestDigest: frozen.digest, WindowState: "closed", Surfaces: make([]SurfaceSnapshot, 0, len(frozen.manifest.Surfaces))}
	for _, surface := range frozen.manifest.Surfaces {
		snapshot := SurfaceSnapshot{SurfaceID: surface.SurfaceID, SurfaceDomain: string(surface.SurfaceDomain), LogicalRef: surface.LogicalRef, Status: ObservationIncomplete}
		if surface.Policy == PolicyExcluded {
			snapshot.Reason = ReasonUnreadable
			result.Surfaces = append(result.Surfaces, snapshot)
			continue
		}
		path, err := resolver.resolve(surface.LogicalRef)
		if err != nil {
			return ProtectedSnapshot{}, errors.New("protected resolver escaped")
		}
		start := clock()
		first, reason := fingerprintSurface(resolver.root, path, surface.Bounds, start, clock, options.DuringRead)
		if reason != "" {
			snapshot.Reason = reason
			result.Surfaces = append(result.Surfaces, snapshot)
			continue
		}
		if options.BetweenPasses != nil {
			options.BetweenPasses(surface.LogicalRef)
		}
		second, reason := fingerprintSurface(resolver.root, path, surface.Bounds, start, clock, options.DuringRead)
		if reason != "" {
			snapshot.Reason = reason
			result.Surfaces = append(result.Surfaces, snapshot)
			continue
		}
		if !bytes.Equal(first, second) {
			snapshot.Reason = ReasonRace
			result.Surfaces = append(result.Surfaces, snapshot)
			continue
		}
		mac := hmac.New(sha256.New, key)
		_, _ = mac.Write([]byte(frozen.digest))
		_, _ = mac.Write([]byte{0})
		_, _ = mac.Write([]byte(surface.SurfaceID))
		_, _ = mac.Write([]byte{0})
		_, _ = mac.Write(first)
		snapshot.Status = ObservationComplete
		snapshot.OpaqueState = "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
		result.Surfaces = append(result.Surfaces, snapshot)
	}
	return result, nil
}

func (resolver *SyntheticResolver) resolve(logicalRef string) (string, error) {
	relative, ok := resolver.targets[logicalRef]
	if !ok {
		return "", errors.New("synthetic target rejected")
	}
	candidate := filepath.Join(resolver.root, filepath.FromSlash(relative))
	parent := filepath.Dir(candidate)
	for {
		info, err := os.Lstat(parent)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return "", errors.New("synthetic resolver escaped")
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", errors.New("synthetic resolver escaped")
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", errors.New("synthetic resolver escaped")
		}
		parent = next
	}
	inside, err := withinRoot(resolver.root, candidate)
	if err != nil || !inside {
		return "", errors.New("synthetic resolver escaped")
	}
	return candidate, nil
}

func fingerprintSurface(root, target string, bounds SurfaceBounds, start time.Time, clock func() time.Time, duringRead func(string)) ([]byte, IncompleteReason) {
	limits := &fingerprintLimits{}
	deadline := start.Add(time.Duration(bounds.Timeout) * time.Millisecond)
	facts := make([]nodeFact, 0)
	err := filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.New(string(ReasonUnreadable))
		}
		if clock().After(deadline) {
			return errors.New(string(ReasonWindow))
		}
		limits.files++
		if limits.files > bounds.MaxFiles {
			return errors.New(string(ReasonOverflow))
		}
		inside, err := withinRoot(root, path)
		if err != nil || !inside {
			return errors.New(string(ReasonSymlinkEscape))
		}
		info, err := os.Lstat(path)
		if err != nil {
			return errors.New(string(ReasonUnreadable))
		}
		relative, err := filepath.Rel(target, path)
		if err != nil {
			return errors.New(string(ReasonUnreadable))
		}
		fact := nodeFact{RelativeID: filepath.ToSlash(relative), Mode: uint32(info.Mode()), Size: info.Size()}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return errors.New(string(ReasonUnreadable))
			}
			resolved := link
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(filepath.Dir(path), resolved)
			}
			resolved = filepath.Clean(resolved)
			inside, err := withinRoot(root, resolved)
			if err != nil || !inside {
				return errors.New(string(ReasonSymlinkEscape))
			}
			fact.Kind = "symlink"
			fact.Link, _ = filepath.Rel(root, resolved)
		case info.IsDir():
			fact.Kind = "directory"
		case info.Mode().IsRegular():
			if info.Mode().Perm()&0o444 == 0 {
				return errors.New(string(ReasonUnreadable))
			}
			remaining := bounds.MaxBytes - limits.bytes
			content, readBytes, reason := readBoundedRegular(root, path, info, remaining, deadline, clock, duringRead)
			if reason != "" {
				return errors.New(string(reason))
			}
			limits.bytes += readBytes
			fact.Kind = "regular"
			fact.Content = content
		default:
			return errors.New(string(ReasonUnreadable))
		}
		facts = append(facts, fact)
		return nil
	})
	if err != nil {
		reason := IncompleteReason(err.Error())
		switch reason {
		case ReasonUnreadable, ReasonOverflow, ReasonSymlinkEscape, ReasonWindow:
			return nil, reason
		default:
			return nil, ReasonUnreadable
		}
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].RelativeID < facts[j].RelativeID })
	canonical, err := json.Marshal(facts)
	if err != nil {
		return nil, ReasonUnreadable
	}
	return canonical, ""
}

func readBoundedRegular(root, path string, expected os.FileInfo, remaining int, deadline time.Time, clock func() time.Time, duringRead func(string)) (string, int, IncompleteReason) {
	inside, err := withinRoot(root, path)
	if err != nil || !inside {
		return "", 0, ReasonSymlinkEscape
	}
	if remaining < 0 || expected.Size() > int64(remaining) {
		return "", 0, ReasonOverflow
	}
	if clock().After(deadline) {
		return "", 0, ReasonWindow
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", 0, ReasonUnreadable
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return "", 0, ReasonUnreadable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(expected, opened) || !sameFileIdentity(expected, opened) {
		return "", 0, ReasonRace
	}
	if opened.Size() > int64(remaining) {
		return "", 0, ReasonOverflow
	}
	if duringRead != nil {
		duringRead(path)
	}
	hash := sha256.New()
	readBytes, err := io.Copy(hash, io.LimitReader(file, int64(remaining)+1))
	if err != nil {
		return "", 0, ReasonUnreadable
	}
	if readBytes > int64(remaining) {
		return "", 0, ReasonOverflow
	}
	if clock().After(deadline) {
		return "", 0, ReasonWindow
	}
	closed, err := file.Stat()
	pathInfo, pathErr := os.Lstat(path)
	if err != nil || pathErr != nil || !os.SameFile(opened, closed) || !os.SameFile(opened, pathInfo) || !sameFileIdentity(opened, closed) || !sameFileIdentity(opened, pathInfo) || readBytes != closed.Size() {
		return "", 0, ReasonRace
	}
	return hex.EncodeToString(hash.Sum(nil)), int(readBytes), ""
}

func sameFileIdentity(left, right os.FileInfo) bool {
	return left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime())
}

func withinRoot(root, candidate string) (bool, error) {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false, err
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))), nil
}
