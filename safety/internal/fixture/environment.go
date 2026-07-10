package fixture

import "path/filepath"

func (root *Root) ChildEnvironment() []string {
	paths := root.paths
	return []string{
		"PATH=" + paths.FakeBin,
		"HOME=" + paths.Home,
		"XDG_CONFIG_HOME=" + paths.XDGConfig,
		"XDG_DATA_HOME=" + paths.XDGData,
		"XDG_CACHE_HOME=" + paths.XDGCache,
		"XDG_STATE_HOME=" + paths.XDGState,
		"XDG_RUNTIME_DIR=" + paths.XDGRuntime,
		"TMPDIR=" + paths.Temporary,
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOENV=off",
		"GOWORK=off",
		"CGO_ENABLED=0",
		"GOCACHE=" + filepath.Join(paths.GoManager, "build-cache"),
		"GOMODCACHE=" + filepath.Join(paths.GoManager, "module-cache"),
		"GOPATH=" + filepath.Join(paths.GoManager, "path"),
		"MISE_CONFIG_DIR=" + filepath.Join(paths.MiseManager, "config"),
		"MISE_DATA_DIR=" + filepath.Join(paths.MiseManager, "data"),
		"MISE_CACHE_DIR=" + filepath.Join(paths.MiseManager, "cache"),
		"UV_CACHE_DIR=" + filepath.Join(paths.UVManager, "cache"),
		"UV_PYTHON_INSTALL_DIR=" + filepath.Join(paths.UVManager, "python"),
		"RUSTUP_HOME=" + paths.RustupManager,
		"CARGO_HOME=" + paths.CargoManager,
		"NIX_STATE_DIR=" + paths.NixManager,
		"HOMEBREW_CACHE=" + filepath.Join(paths.HomebrewManager, "cache"),
		"HOMEBREW_LOGS=" + filepath.Join(paths.HomebrewManager, "logs"),
		"npm_config_cache=" + filepath.Join(paths.NodeManager, "cache"),
		"YAMC_ARTIFACT_STORE=" + paths.ArtifactStore,
		"YAMC_TRUST_ROOT=" + paths.Trust,
		"YAMC_NETWORK_CACHE=" + paths.NetworkCache,
		"YAMC_TEST_TIER=offline-static",
		"YAMC_NETWORK=deny",
	}
}
