package fixture

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const syntheticNetworkTestID = "fixture.download.synthetic-archive.v1"

func TestTierNetworkPolicy(t *testing.T) {
	t.Run("keeps three closed tiers and defaults offline", testClosedTiers)
	t.Run("validates one tracked exact network contract without executing it", testExactNetworkContract)
	t.Run("denies missing wildcard unknown ambient and higher-tier authorization", testNetworkAuthorizationDenials)
	t.Run("rejects every incomplete network manifest field", testNetworkManifestRejections)
	t.Run("requires both current official semantics and isolated negative evidence", testLiveProbeProof)
	t.Run("contains no network or live command executor", testNoNetworkExecutor)
}

func testClosedTiers(t *testing.T) {
	for _, testCase := range []struct {
		raw  string
		want Tier
	}{
		{raw: "", want: TierOfflineStatic},
		{raw: "offline-static", want: TierOfflineStatic},
		{raw: "isolated-integration", want: TierIsolatedIntegration},
		{raw: "live-check", want: TierLiveCheck},
	} {
		got, err := ParseTier(testCase.raw)
		if err != nil || got != testCase.want {
			t.Fatalf("tier %q did not remain closed", testCase.raw)
		}
	}
	for _, invalid := range []string{"offline", "integration", "live", "*", "isolated-integration;live-check"} {
		if _, err := ParseTier(invalid); err == nil {
			t.Fatalf("invalid tier %q was accepted", invalid)
		}
	}
	offline := DefaultPolicyDecision(TierOfflineStatic)
	isolated := DefaultPolicyDecision(TierIsolatedIntegration)
	live := DefaultPolicyDecision(TierLiveCheck)
	if offline.Status != PolicyReady || offline.NetworkPolicy != "denied" || offline.Tier != TierOfflineStatic {
		t.Fatal("default invocation is not offline and network-denied")
	}
	if isolated.Status != PolicyReady || isolated.NetworkPolicy != "denied" || isolated.Tier != TierIsolatedIntegration {
		t.Fatal("isolated integration did not remain offline")
	}
	if live.Status != PolicyUnknown || live.NetworkPolicy != "denied" || live.Tier != TierLiveCheck {
		t.Fatal("live-check did not remain a separate non-executing tier")
	}
}

func testExactNetworkContract(t *testing.T) {
	safetyRoot, repositoryRoot := fixtureProjectRoots(t)
	manifestPath := filepath.Join(safetyRoot, "manifests", "network-tests.v1.json")
	policy, err := LoadNetworkPolicy(manifestPath, repositoryRoot)
	if err != nil {
		t.Fatalf("tracked network manifest was rejected: %v", err)
	}
	decision := policy.Authorize(syntheticNetworkTestID, TierIsolatedIntegration, nil)
	if decision.Status != PolicyManualRequired || decision.Tier != TierIsolatedIntegration || decision.NetworkPolicy != "denied" || !decision.ContractValidated || decision.TestID != syntheticNetworkTestID || decision.Reason != "network-execution-unavailable-phase-1" {
		t.Fatal("exact network contract was not validated and then denied safely")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal("tracked network manifest is unavailable")
	}
	policyFromBytes, err := ParseNetworkPolicy(data)
	if err != nil || len(policyFromBytes.entries) != 1 {
		t.Fatal("tracked manifest did not parse as one exact contract")
	}
	entry := policyFromBytes.entries[syntheticNetworkTestID]
	if entry.Request.Protocol != "https" || entry.Request.Host != "example.invalid" || entry.Request.Port != 443 || entry.Request.Method != "GET" || entry.Request.Redirects != 0 {
		t.Fatal("tracked request contract changed")
	}
	if entry.Authorization.Mode != "exact-test-id" || entry.Authorization.RequiredArgument != "allow-network-test" || entry.Authorization.Execution != "validation-only" {
		t.Fatal("tracked exact authorization contract changed")
	}
	if entry.Limits.MaxBytes != maxNetworkBytes || entry.Limits.TimeoutMS <= 0 || entry.Limits.TimeoutMS > maxNetworkTimeoutMS {
		t.Fatal("tracked byte or timeout bounds changed")
	}

	outsideCopy := filepath.Join(t.TempDir(), "network-tests.v1.json")
	if err := os.WriteFile(outsideCopy, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNetworkPolicy(outsideCopy, repositoryRoot); err == nil {
		t.Fatal("untracked network manifest location was accepted")
	}
	symlinkPath := filepath.Join(t.TempDir(), "network-tests.v1.json")
	if err := os.Symlink(manifestPath, symlinkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNetworkPolicy(symlinkPath, repositoryRoot); err == nil {
		t.Fatal("symlinked network manifest was accepted")
	}
}

func testNetworkAuthorizationDenials(t *testing.T) {
	data := trackedNetworkManifest(t)
	policy, err := ParseNetworkPolicy(data)
	if err != nil {
		t.Fatal("network policy setup failed")
	}
	tests := []struct {
		name    string
		id      string
		tier    Tier
		ambient []string
		reason  string
	}{
		{name: "missing id", tier: TierIsolatedIntegration, reason: "exact-network-test-required"},
		{name: "wildcard id", id: "*", tier: TierIsolatedIntegration, reason: "exact-network-test-required"},
		{name: "generic id", id: "network", tier: TierIsolatedIntegration, reason: "network-test-unknown"},
		{name: "unknown id", id: "fixture.download.unknown.v1", tier: TierIsolatedIntegration, reason: "network-test-unknown"},
		{name: "shell shape", id: "fixture.download.synthetic-archive.v1;printf", tier: TierIsolatedIntegration, reason: "exact-network-test-required"},
		{name: "ambient proxy", id: syntheticNetworkTestID, tier: TierIsolatedIntegration, ambient: []string{"HTTPS_PROXY=http://synthetic.invalid"}, reason: "ambient-state-forbidden"},
		{name: "ambient credential", id: syntheticNetworkTestID, tier: TierIsolatedIntegration, ambient: []string{"SYNTHETIC_TOKEN=not-a-real-value"}, reason: "ambient-state-forbidden"},
		{name: "offline tier", id: syntheticNetworkTestID, tier: TierOfflineStatic, reason: "tier-network-denied"},
		{name: "live tier", id: syntheticNetworkTestID, tier: TierLiveCheck, reason: "tier-network-denied"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			decision := policy.Authorize(testCase.id, testCase.tier, testCase.ambient)
			if decision.Status != PolicyManualRequired || decision.Tier != testCase.tier || decision.NetworkPolicy != "denied" || decision.Reason != testCase.reason || decision.ContractValidated || decision.TestID != "" {
				t.Fatal("network denial escalated tier, reflected input, or changed status")
			}
		})
	}
}

func testNetworkManifestRejections(t *testing.T) {
	base := decodeNetworkManifest(t, trackedNetworkManifest(t))
	tests := []struct {
		name   string
		mutate func(*networkManifest)
	}{
		{name: "schema", mutate: func(manifest *networkManifest) { manifest.SchemaVersion = "2.0.0" }},
		{name: "empty tests", mutate: func(manifest *networkManifest) { manifest.Tests = nil }},
		{name: "wildcard id", mutate: func(manifest *networkManifest) { manifest.Tests[0].TestID = "*" }},
		{name: "adapter", mutate: func(manifest *networkManifest) { manifest.Tests[0].AdapterID = "adapter;run" }},
		{name: "purpose", mutate: func(manifest *networkManifest) { manifest.Tests[0].Purpose = "download-anything" }},
		{name: "protocol", mutate: func(manifest *networkManifest) { manifest.Tests[0].Request.Protocol = "http" }},
		{name: "host", mutate: func(manifest *networkManifest) { manifest.Tests[0].Request.Host = "localhost" }},
		{name: "port", mutate: func(manifest *networkManifest) { manifest.Tests[0].Request.Port = 80 }},
		{name: "method", mutate: func(manifest *networkManifest) { manifest.Tests[0].Request.Method = "POST" }},
		{name: "url host", mutate: func(manifest *networkManifest) {
			manifest.Tests[0].Request.URL = "https://localhost/fixtures/synthetic.tar.gz"
		}},
		{name: "redirect", mutate: func(manifest *networkManifest) { manifest.Tests[0].Request.Redirects = 1 }},
		{name: "digest algorithm", mutate: func(manifest *networkManifest) { manifest.Tests[0].Integrity.Algorithm = "none" }},
		{name: "digest", mutate: func(manifest *networkManifest) { manifest.Tests[0].Integrity.Digest = "sha256:synthetic" }},
		{name: "bytes", mutate: func(manifest *networkManifest) { manifest.Tests[0].Limits.MaxBytes = maxNetworkBytes + 1 }},
		{name: "timeout", mutate: func(manifest *networkManifest) { manifest.Tests[0].Limits.TimeoutMS = maxNetworkTimeoutMS + 1 }},
		{name: "cache", mutate: func(manifest *networkManifest) { manifest.Tests[0].CacheRef = "local-state:shared-cache" }},
		{name: "credentials", mutate: func(manifest *networkManifest) { manifest.Tests[0].Credentials = "allowed" }},
		{name: "proxy", mutate: func(manifest *networkManifest) { manifest.Tests[0].ProxyEnvironment = "inherited" }},
		{name: "egress", mutate: func(manifest *networkManifest) { manifest.Tests[0].EgressPolicy = "host-only" }},
		{name: "authorization mode", mutate: func(manifest *networkManifest) { manifest.Tests[0].Authorization.Mode = "global" }},
		{name: "authorization argument", mutate: func(manifest *networkManifest) { manifest.Tests[0].Authorization.RequiredArgument = "network" }},
		{name: "authorization execution", mutate: func(manifest *networkManifest) { manifest.Tests[0].Authorization.Execution = "execute" }},
		{name: "duplicate test", mutate: func(manifest *networkManifest) { manifest.Tests = append(manifest.Tests, manifest.Tests[0]) }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			manifest := cloneNetworkManifest(t, base)
			testCase.mutate(&manifest)
			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseNetworkPolicy(data); err == nil {
				t.Fatal("invalid network manifest was accepted")
			}
		})
	}
	unknownField := bytes.Replace(trackedNetworkManifest(t), []byte(`"schema_version": "1.0.0"`), []byte(`"schema_version": "1.0.0", "unknown": true`), 1)
	if _, err := ParseNetworkPolicy(unknownField); err == nil {
		t.Fatal("unknown network manifest field was accepted")
	}
	duplicateKey := []byte(`{"schema_version":"1.0.0","schema_version":"1.0.0","tests":[]}`)
	if _, err := ParseNetworkPolicy(duplicateKey); err == nil {
		t.Fatal("duplicate network manifest key was accepted")
	}
}

func testLiveProbeProof(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	valid := LiveProbeProof{
		ProbeID:                          "fixture.live.synthetic.v1",
		OfficialReadOnlySemanticsCurrent: true,
		OfficialReviewExpiresAt:          now.Add(time.Hour),
		IsolatedNegativeEvidenceDigest:   "sha256:" + strings.Repeat("a", 64),
		IsolatedNegativeEvidenceCurrent:  true,
	}
	if err := ValidateLiveProbeProof(valid, now); err != nil {
		t.Fatal("complete synthetic proof contract was rejected")
	}
	for _, mutate := range []func(*LiveProbeProof){
		func(proof *LiveProbeProof) { proof.OfficialReadOnlySemanticsCurrent = false },
		func(proof *LiveProbeProof) { proof.IsolatedNegativeEvidenceCurrent = false },
		func(proof *LiveProbeProof) { proof.OfficialReviewExpiresAt = now },
		func(proof *LiveProbeProof) { proof.IsolatedNegativeEvidenceDigest = "sha256:missing" },
		func(proof *LiveProbeProof) { proof.ProbeID = "*" },
	} {
		proof := valid
		mutate(&proof)
		if err := ValidateLiveProbeProof(proof, now); err == nil {
			t.Fatal("incomplete live probe proof was accepted")
		}
	}
	decision := (LivePolicy{}).Evaluate(valid.ProbeID)
	if decision.Status != PolicyUnknown || decision.Tier != TierLiveCheck || decision.NetworkPolicy != "denied" || decision.ContractValidated || decision.Reason != "live-probe-unapproved" {
		t.Fatal("Phase 1 live policy executed or approved a proof")
	}
}

func testNoNetworkExecutor(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("network source unavailable")
	}
	directory := filepath.Dir(current)
	for _, file := range []string{filepath.Join(directory, "network.go"), filepath.Join(directory, "..", "..", "cmd", "yamc-safety", "main.go")} {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal("policy source unavailable")
		}
		text := string(data)
		for _, forbidden := range []string{"net/http", "http.Client", "http.NewRequest", "Dial(", "Listen(", "RoundTrip(", "exec.Command", "os/exec"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("policy source contains a network or command executor: %s", forbidden)
			}
		}
	}
	manifest := string(trackedNetworkManifest(t))
	if !strings.Contains(manifest, "example.invalid") || strings.Contains(manifest, "localhost") || strings.Contains(manifest, "127.0.0.1") || strings.Contains(manifest, "credentials-allowed") {
		t.Fatal("tracked manifest is not synthetic public metadata only")
	}
}

func trackedNetworkManifest(t *testing.T) []byte {
	t.Helper()
	safetyRoot, _ := fixtureProjectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "manifests", "network-tests.v1.json"))
	if err != nil {
		t.Fatal("tracked network manifest unavailable")
	}
	return data
}

func decodeNetworkManifest(t *testing.T, data []byte) networkManifest {
	t.Helper()
	var manifest networkManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func cloneNetworkManifest(t *testing.T, manifest networkManifest) networkManifest {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return decodeNetworkManifest(t, data)
}
