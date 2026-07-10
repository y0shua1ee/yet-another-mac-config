package sentinel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"time"

	"example.invalid/yamc/safety/internal/privacy"
)

const (
	protectedManifestSchema = "1.0.0"
	maximumSurfaceFiles     = 4096
	maximumSurfaceBytes     = 16 << 20
	maximumSurfaceTimeout   = 5 * time.Second
)

type SurfacePolicy string

const (
	PolicyRequired SurfacePolicy = "required"
	PolicyOptional SurfacePolicy = "optional"
	PolicyExcluded SurfacePolicy = "excluded"
)

type SurfaceBounds struct {
	MaxFiles int `json:"max_files"`
	MaxBytes int `json:"max_bytes"`
	Timeout  int `json:"timeout_ms"`
}

type ProtectedSurface struct {
	SurfaceID     string                `json:"surface_id"`
	SurfaceDomain privacy.SurfaceDomain `json:"surface_domain"`
	LogicalRef    string                `json:"logical_ref"`
	Policy        SurfacePolicy         `json:"policy"`
	AdapterID     string                `json:"adapter_id"`
	Bounds        SurfaceBounds         `json:"bounds"`
}

type ProtectedManifest struct {
	SchemaVersion string             `json:"schema_version"`
	SuiteID       string             `json:"suite_id"`
	TestID        string             `json:"test_id"`
	Surfaces      []ProtectedSurface `json:"surfaces"`
	Digest        string             `json:"-"`
}

type FrozenManifest struct {
	manifest ProtectedManifest
	digest   string
	started  bool
}

var publicManifestID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

var protectedSurfaceContract = map[string]struct {
	domain  privacy.SurfaceDomain
	adapter string
}{
	"repo:sentinel/worktree/tracked":               {privacy.SurfaceWorktree, "git-worktree-readonly-v1"},
	"repo:sentinel/worktree/index":                 {privacy.SurfaceWorktree, "git-index-readonly-v1"},
	"home:.zshrc":                                  {privacy.SurfaceNamedHome, "go-lstat-file-v1"},
	"home:sentinel/manager/mise-data":              {privacy.SurfaceManagerRoot, "go-bounded-tree-v1"},
	"profile:sentinel/service/homebrew-mxcl-nginx": {privacy.SurfaceService, "launchctl-print-service-v1"},
	"profile:sentinel/named-target/system-shells":  {privacy.SurfaceNamedTarget, "go-system-shells-file-v1"},
}

func ParseProtectedManifest(data []byte) (ProtectedManifest, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return ProtectedManifest{}, errors.New("protected manifest rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest ProtectedManifest
	if err := decoder.Decode(&manifest); err != nil {
		return ProtectedManifest{}, errors.New("protected manifest rejected")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ProtectedManifest{}, errors.New("protected manifest rejected")
	}
	if err := validateProtectedManifest(manifest); err != nil {
		return ProtectedManifest{}, err
	}
	canonical, err := canonicalProtectedManifest(manifest)
	if err != nil {
		return ProtectedManifest{}, errors.New("protected manifest rejected")
	}
	manifest.Digest = sha256Digest(canonical)
	return manifest, nil
}

func FreezeProtectedManifest(manifest ProtectedManifest) (*FrozenManifest, error) {
	if err := validateProtectedManifest(manifest); err != nil {
		return nil, err
	}
	canonical, err := canonicalProtectedManifest(manifest)
	if err != nil {
		return nil, errors.New("protected manifest rejected")
	}
	digest := sha256Digest(canonical)
	if manifest.Digest == "" || manifest.Digest != digest {
		return nil, errors.New("protected manifest digest rejected")
	}
	copyManifest := manifest
	copyManifest.Surfaces = append([]ProtectedSurface(nil), manifest.Surfaces...)
	return &FrozenManifest{manifest: copyManifest, digest: digest, started: true}, nil
}

func (frozen *FrozenManifest) Manifest() ProtectedManifest {
	if frozen == nil {
		return ProtectedManifest{}
	}
	manifest := frozen.manifest
	manifest.Surfaces = append([]ProtectedSurface(nil), frozen.manifest.Surfaces...)
	return manifest
}

func (frozen *FrozenManifest) ValidateCurrent(current ProtectedManifest) error {
	if frozen == nil || !frozen.started {
		return errors.New("observation window not started")
	}
	canonical, err := canonicalProtectedManifest(current)
	if err != nil || sha256Digest(canonical) != frozen.digest || current.Digest != frozen.digest {
		return errors.New("protected manifest changed after observation start")
	}
	return nil
}

func validateProtectedManifest(manifest ProtectedManifest) error {
	if manifest.SchemaVersion != protectedManifestSchema || !publicManifestID.MatchString(manifest.SuiteID) || !publicManifestID.MatchString(manifest.TestID) {
		return errors.New("protected manifest rejected")
	}
	if len(manifest.Surfaces) != len(protectedSurfaceContract) {
		return errors.New("protected manifest scope rejected")
	}
	seenIDs := make(map[string]struct{}, len(manifest.Surfaces))
	seenRefs := make(map[string]struct{}, len(manifest.Surfaces))
	domains := make(map[privacy.SurfaceDomain]int)
	for _, surface := range manifest.Surfaces {
		if !publicManifestID.MatchString(surface.SurfaceID) {
			return errors.New("protected surface id rejected")
		}
		if _, exists := seenIDs[surface.SurfaceID]; exists {
			return errors.New("duplicate protected surface rejected")
		}
		if _, exists := seenRefs[surface.LogicalRef]; exists {
			return errors.New("duplicate protected surface rejected")
		}
		seenIDs[surface.SurfaceID] = struct{}{}
		seenRefs[surface.LogicalRef] = struct{}{}
		contract, ok := protectedSurfaceContract[surface.LogicalRef]
		if !ok || contract.domain != surface.SurfaceDomain || contract.adapter != surface.AdapterID {
			return errors.New("protected surface contract rejected")
		}
		if err := privacy.ValidateSurface(surface.SurfaceDomain, surface.LogicalRef); err != nil {
			return errors.New("protected surface reference rejected")
		}
		switch surface.Policy {
		case PolicyRequired, PolicyOptional, PolicyExcluded:
		default:
			return errors.New("protected surface policy rejected")
		}
		if surface.Bounds.MaxFiles < 1 || surface.Bounds.MaxFiles > maximumSurfaceFiles || surface.Bounds.MaxBytes < 1 || surface.Bounds.MaxBytes > maximumSurfaceBytes || surface.Bounds.Timeout < 1 || time.Duration(surface.Bounds.Timeout)*time.Millisecond > maximumSurfaceTimeout {
			return errors.New("protected surface bounds rejected")
		}
		domains[surface.SurfaceDomain]++
	}
	if domains[privacy.SurfaceWorktree] != 2 || domains[privacy.SurfaceNamedHome] != 1 || domains[privacy.SurfaceManagerRoot] != 1 || domains[privacy.SurfaceService] != 1 || domains[privacy.SurfaceNamedTarget] != 1 {
		return errors.New("protected manifest domains rejected")
	}
	return nil
}

func canonicalProtectedManifest(manifest ProtectedManifest) ([]byte, error) {
	copyManifest := manifest
	copyManifest.Digest = ""
	copyManifest.Surfaces = append([]ProtectedSurface(nil), manifest.Surfaces...)
	sort.Slice(copyManifest.Surfaces, func(i, j int) bool {
		return copyManifest.Surfaces[i].LogicalRef < copyManifest.Surfaces[j].LogicalRef
	})
	return json.Marshal(copyManifest)
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("json trailing data rejected")
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("json object key rejected")
			}
			if _, exists := seen[key]; exists {
				return errors.New("duplicate json key rejected")
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("json object rejected")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("json array rejected")
		}
	default:
		return errors.New("json delimiter rejected")
	}
	return nil
}
