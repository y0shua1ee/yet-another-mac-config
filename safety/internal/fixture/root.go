package fixture

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.invalid/yamc/safety/internal/privacy"
)

const (
	markerSchemaVersion = "1.0.0"
	markerFileName      = ".yamc-fixture-marker.json"
	defaultRetentionTTL = 24 * time.Hour
	maximumRetentionTTL = 24 * time.Hour
)

type CreateOptions struct {
	Base           string
	RepositoryRoot string
	ProtectedRoots []string
	LogicalID      string
	KeepFixture    bool
	TTL            time.Duration
	Clock          func() time.Time
	Random         io.Reader
	EffectiveUID   func() int
}

type Paths struct {
	Root              string
	Home              string
	XDGConfig         string
	XDGData           string
	XDGCache          string
	XDGState          string
	XDGRuntime        string
	Temporary         string
	FakeBin           string
	NixManager        string
	HomebrewManager   string
	MiseManager       string
	UVManager         string
	RustupManager     string
	CargoManager      string
	GoManager         string
	NodeManager       string
	Trust             string
	NetworkCache      string
	ArtifactStore     string
	BlueprintWorktree string
	SentinelScratch   string
}

type Root struct {
	paths     Paths
	retention *Retention
}

type ownershipMarker struct {
	SchemaVersion string `json:"schema_version"`
	LogicalID     string `json:"logical_fixture_id"`
	CreatedAt     string `json:"created_at"`
	ExpiresAt     string `json:"expires_at"`
	EffectiveUID  int    `json:"effective_uid"`
	Nonce         string `json:"ownership_nonce"`
}

func Create(options CreateOptions) (*Root, error) {
	clock := options.Clock
	if clock == nil {
		clock = time.Now
	}
	random := options.Random
	if random == nil {
		random = rand.Reader
	}
	effectiveUID := options.EffectiveUID
	if effectiveUID == nil {
		effectiveUID = os.Geteuid
	}
	ttl := options.TTL
	if ttl == 0 {
		ttl = defaultRetentionTTL
	}
	if ttl <= 0 || ttl > maximumRetentionTTL {
		return nil, errors.New("fixture ttl rejected")
	}
	logicalID, err := privacy.ParseLogicalRef(options.LogicalID)
	if err != nil || logicalID.Namespace != privacy.NamespaceFixture {
		return nil, errors.New("fixture logical id rejected")
	}
	base, err := canonicalExistingDirectory(options.Base)
	if err != nil || containsParentReference(options.Base) {
		return nil, errors.New("fixture base rejected")
	}
	repository, err := canonicalExistingDirectory(options.RepositoryRoot)
	if err != nil {
		return nil, errors.New("repository root rejected")
	}
	protected := make([]string, 0, len(options.ProtectedRoots)+1)
	protected = append(protected, repository)
	for _, root := range options.ProtectedRoots {
		canonical, canonicalErr := canonicalExistingDirectory(root)
		if canonicalErr != nil {
			return nil, errors.New("protected root rejected")
		}
		protected = append(protected, canonical)
	}
	for _, root := range protected {
		inside, relationErr := isWithin(root, base)
		if relationErr != nil || inside {
			return nil, errors.New("fixture base overlaps protected root")
		}
	}

	nonce, err := newNonce(random)
	if err != nil {
		return nil, errors.New("fixture nonce unavailable")
	}
	physicalRoot := filepath.Join(base, "fixture-"+nonce)
	for _, root := range protected {
		inside, relationErr := isWithin(root, physicalRoot)
		if relationErr != nil || inside {
			return nil, errors.New("fixture root overlaps protected root")
		}
	}
	if err := os.Mkdir(physicalRoot, 0o700); err != nil {
		return nil, errors.New("fixture root unavailable")
	}

	createdAt := clock().UTC()
	marker := ownershipMarker{
		SchemaVersion: markerSchemaVersion,
		LogicalID:     logicalID.String(),
		CreatedAt:     createdAt.Format(time.RFC3339Nano),
		ExpiresAt:     createdAt.Add(ttl).Format(time.RFC3339Nano),
		EffectiveUID:  effectiveUID(),
		Nonce:         nonce,
	}
	if err := writeMarker(physicalRoot, marker); err != nil {
		return nil, err
	}

	paths := fixturePaths(physicalRoot)
	if err := createFixtureDirectories(paths); err != nil {
		return nil, err
	}
	retention := &Retention{
		base:         base,
		root:         physicalRoot,
		expected:     marker,
		keep:         options.KeepFixture,
		clock:        clock,
		effectiveUID: effectiveUID,
	}
	return &Root{paths: paths, retention: retention}, nil
}

func (root *Root) Paths() Paths {
	return root.paths
}

func (root *Root) LogicalID() string {
	return root.retention.expected.LogicalID
}

func (root *Root) Retention() *Retention {
	return root.retention
}

func fixturePaths(root string) Paths {
	managerRoot := filepath.Join(root, "managers")
	return Paths{
		Root:              root,
		Home:              filepath.Join(root, "home"),
		XDGConfig:         filepath.Join(root, "xdg", "config"),
		XDGData:           filepath.Join(root, "xdg", "data"),
		XDGCache:          filepath.Join(root, "xdg", "cache"),
		XDGState:          filepath.Join(root, "xdg", "state"),
		XDGRuntime:        filepath.Join(root, "xdg", "runtime"),
		Temporary:         filepath.Join(root, "tmp"),
		FakeBin:           filepath.Join(root, "path", "bin"),
		NixManager:        filepath.Join(managerRoot, "nix"),
		HomebrewManager:   filepath.Join(managerRoot, "homebrew"),
		MiseManager:       filepath.Join(managerRoot, "mise"),
		UVManager:         filepath.Join(managerRoot, "uv"),
		RustupManager:     filepath.Join(managerRoot, "rustup"),
		CargoManager:      filepath.Join(managerRoot, "cargo"),
		GoManager:         filepath.Join(managerRoot, "go"),
		NodeManager:       filepath.Join(managerRoot, "node"),
		Trust:             filepath.Join(root, "trust"),
		NetworkCache:      filepath.Join(root, "network-cache"),
		ArtifactStore:     filepath.Join(root, "artifact-store"),
		BlueprintWorktree: filepath.Join(root, "blueprint-worktree"),
		SentinelScratch:   filepath.Join(root, "sentinel-scratch"),
	}
}

func createFixtureDirectories(paths Paths) error {
	directories := []string{
		paths.Home,
		paths.XDGConfig,
		paths.XDGData,
		paths.XDGCache,
		paths.XDGState,
		paths.XDGRuntime,
		paths.Temporary,
		paths.FakeBin,
		paths.NixManager,
		filepath.Join(paths.HomebrewManager, "cache"),
		filepath.Join(paths.HomebrewManager, "logs"),
		filepath.Join(paths.MiseManager, "config"),
		filepath.Join(paths.MiseManager, "data"),
		filepath.Join(paths.MiseManager, "cache"),
		filepath.Join(paths.UVManager, "cache"),
		filepath.Join(paths.UVManager, "python"),
		paths.RustupManager,
		paths.CargoManager,
		filepath.Join(paths.GoManager, "build-cache"),
		filepath.Join(paths.GoManager, "module-cache"),
		filepath.Join(paths.GoManager, "path"),
		filepath.Join(paths.NodeManager, "cache"),
		paths.Trust,
		paths.NetworkCache,
		paths.ArtifactStore,
		paths.BlueprintWorktree,
		paths.SentinelScratch,
	}
	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return errors.New("fixture directory unavailable")
		}
		inside, err := isWithin(paths.Root, directory)
		if err != nil || !inside {
			return errors.New("fixture directory escaped root")
		}
	}
	return nil
}

func writeMarker(root string, marker ownershipMarker) error {
	encoded, err := json.Marshal(marker)
	if err != nil {
		return errors.New("fixture marker unavailable")
	}
	encoded = append(encoded, '\n')
	path := filepath.Join(root, markerFileName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("fixture marker unavailable")
	}
	if _, err := file.Write(encoded); err != nil {
		_ = file.Close()
		return errors.New("fixture marker unavailable")
	}
	if err := file.Close(); err != nil {
		return errors.New("fixture marker unavailable")
	}
	return nil
}

func newNonce(random io.Reader) (string, error) {
	buffer := make([]byte, 16)
	if _, err := io.ReadFull(random, buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func canonicalExistingDirectory(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) || containsParentReference(path) {
		return "", errors.New("path rejected")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("path rejected")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.New("path rejected")
	}
	return filepath.Abs(resolved)
}

func containsParentReference(path string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func isWithin(root, candidate string) (bool, error) {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false, err
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))), nil
}
