package sentinel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxSyntheticSurfaceBytes = 64 << 10

type Surface struct {
	ID         string `json:"id"`
	LogicalRef string `json:"logical_ref"`
	Seed       string `json:"seed"`
}

type Manifest struct {
	SchemaVersion string    `json:"schema_version"`
	ManifestID    string    `json:"manifest_id"`
	Surfaces      []Surface `json:"surfaces"`
	Digest        string    `json:"-"`
}

type Snapshot struct {
	ManifestDigest string `json:"manifest_digest"`
	StateDigest    string `json:"state_digest"`
}

type surfaceFact struct {
	LogicalRef string `json:"logical_ref"`
	Digest     string `json:"digest"`
}

func ParseManifest(data []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, errors.New("synthetic manifest rejected")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Manifest{}, errors.New("synthetic manifest rejected")
	}
	if manifest.SchemaVersion != "1.0.0" || !validFixtureReference(manifest.ManifestID) || len(manifest.Surfaces) == 0 {
		return Manifest{}, errors.New("synthetic manifest rejected")
	}
	seen := make(map[string]struct{}, len(manifest.Surfaces))
	for _, surface := range manifest.Surfaces {
		if !validFixtureReference(surface.ID) || !validFixtureReference(surface.LogicalRef) || surface.Seed == "" {
			return Manifest{}, errors.New("synthetic manifest rejected")
		}
		if _, exists := seen[surface.LogicalRef]; exists {
			return Manifest{}, errors.New("synthetic manifest rejected")
		}
		seen[surface.LogicalRef] = struct{}{}
	}
	canonical, err := json.Marshal(struct {
		SchemaVersion string    `json:"schema_version"`
		ManifestID    string    `json:"manifest_id"`
		Surfaces      []Surface `json:"surfaces"`
	}{manifest.SchemaVersion, manifest.ManifestID, manifest.Surfaces})
	if err != nil {
		return Manifest{}, errors.New("synthetic manifest rejected")
	}
	manifest.Digest = digestBytes(canonical)
	return manifest, nil
}

func PrepareSynthetic(manifest Manifest, fixtureRoot string) error {
	for _, surface := range manifest.Surfaces {
		path, err := syntheticPath(fixtureRoot, surface.LogicalRef)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return errors.New("synthetic surface unavailable")
		}
		if err := os.WriteFile(path, []byte(surface.Seed), 0o600); err != nil {
			return errors.New("synthetic surface unavailable")
		}
	}
	return nil
}

func ObserveSynthetic(manifest Manifest, fixtureRoot string) (Snapshot, error) {
	facts := make([]surfaceFact, 0, len(manifest.Surfaces))
	for _, surface := range manifest.Surfaces {
		path, err := syntheticPath(fixtureRoot, surface.LogicalRef)
		if err != nil {
			return Snapshot{}, err
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > maxSyntheticSurfaceBytes {
			return Snapshot{}, errors.New("synthetic surface observation failed")
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxSyntheticSurfaceBytes {
			return Snapshot{}, errors.New("synthetic surface observation failed")
		}
		facts = append(facts, surfaceFact{LogicalRef: surface.LogicalRef, Digest: digestBytes(data)})
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].LogicalRef < facts[j].LogicalRef })
	canonical, err := json.Marshal(facts)
	if err != nil {
		return Snapshot{}, errors.New("synthetic surface observation failed")
	}
	return Snapshot{ManifestDigest: manifest.Digest, StateDigest: digestBytes(canonical)}, nil
}

func Equal(before, after Snapshot) bool {
	return before.ManifestDigest != "" &&
		before.ManifestDigest == after.ManifestDigest &&
		before.StateDigest != "" &&
		before.StateDigest == after.StateDigest
}

func syntheticPath(fixtureRoot, logicalRef string) (string, error) {
	if !validFixtureReference(logicalRef) {
		return "", errors.New("synthetic reference rejected")
	}
	relative := strings.TrimPrefix(logicalRef, "fixture:")
	parts := strings.Split(relative, "/")
	return filepath.Join(append([]string{fixtureRoot, "protected"}, parts...)...), nil
}

func validFixtureReference(value string) bool {
	if !strings.HasPrefix(value, "fixture:") {
		return false
	}
	relative := strings.TrimPrefix(value, "fixture:")
	if relative == "" || strings.HasPrefix(relative, "/") || strings.ContainsRune(relative, '\x00') || strings.Contains(relative, "\\") {
		return false
	}
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
