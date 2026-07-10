package fixture

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"example.invalid/yamc/safety/internal/privacy"
)

const (
	networkManifestSchema = "1.0.0"
	maxNetworkManifest    = 64 << 10
	maxNetworkBytes       = 1 << 20
	maxNetworkTimeoutMS   = 10_000
)

type Tier string

const (
	TierOfflineStatic       Tier = "offline-static"
	TierIsolatedIntegration Tier = "isolated-integration"
	TierLiveCheck           Tier = "live-check"
)

type PolicyStatus string

const (
	PolicyReady          PolicyStatus = "ready"
	PolicyManualRequired PolicyStatus = "manual-required"
	PolicyUnknown        PolicyStatus = "unknown"
)

type PolicyDecision struct {
	Status            PolicyStatus `json:"status"`
	Tier              Tier         `json:"tier"`
	NetworkPolicy     string       `json:"network_policy"`
	Reason            string       `json:"reason"`
	TestID            string       `json:"test_id,omitempty"`
	ProbeID           string       `json:"probe_id,omitempty"`
	ContractValidated bool         `json:"contract_validated"`
}

type networkManifest struct {
	SchemaVersion string             `json:"schema_version"`
	Tests         []networkTestEntry `json:"tests"`
}

type networkTestEntry struct {
	TestID           string               `json:"test_id"`
	AdapterID        string               `json:"adapter_id"`
	Purpose          string               `json:"purpose"`
	Request          networkRequest       `json:"request"`
	Integrity        networkIntegrity     `json:"integrity"`
	Limits           networkLimits        `json:"limits"`
	CacheRef         string               `json:"cache_ref"`
	Credentials      string               `json:"credentials"`
	ProxyEnvironment string               `json:"proxy_environment"`
	EgressPolicy     string               `json:"egress_policy"`
	Authorization    networkAuthorization `json:"authorization"`
}

type networkRequest struct {
	Protocol  string `json:"protocol"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Method    string `json:"method"`
	URL       string `json:"url"`
	Redirects int    `json:"redirects"`
}

type networkIntegrity struct {
	Algorithm string `json:"algorithm"`
	Digest    string `json:"digest"`
}

type networkLimits struct {
	MaxBytes  int64 `json:"max_bytes"`
	TimeoutMS int64 `json:"timeout_ms"`
}

type networkAuthorization struct {
	Mode             string `json:"mode"`
	RequiredArgument string `json:"required_argument"`
	Execution        string `json:"execution"`
}

type NetworkPolicy struct {
	entries map[string]networkTestEntry
}

type LiveProbeProof struct {
	ProbeID                          string
	OfficialReadOnlySemanticsCurrent bool
	OfficialReviewExpiresAt          time.Time
	IsolatedNegativeEvidenceDigest   string
	IsolatedNegativeEvidenceCurrent  bool
}

type LivePolicy struct{}

var (
	publicContractID = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	sha256Digest     = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

func ParseTier(raw string) (Tier, error) {
	if raw == "" {
		return TierOfflineStatic, nil
	}
	tier := Tier(raw)
	switch tier {
	case TierOfflineStatic, TierIsolatedIntegration, TierLiveCheck:
		return tier, nil
	default:
		return "", errors.New("test tier rejected")
	}
}

func DefaultPolicyDecision(tier Tier) PolicyDecision {
	switch tier {
	case TierOfflineStatic:
		return PolicyDecision{Status: PolicyReady, Tier: tier, NetworkPolicy: "denied", Reason: "offline-default"}
	case TierIsolatedIntegration:
		return PolicyDecision{Status: PolicyReady, Tier: tier, NetworkPolicy: "denied", Reason: "isolated-offline"}
	case TierLiveCheck:
		return (LivePolicy{}).Evaluate("")
	default:
		return PolicyDecision{Status: PolicyUnknown, Tier: tier, NetworkPolicy: "denied", Reason: "tier-unknown"}
	}
}

func LoadNetworkPolicy(manifestPath, repositoryRoot string) (*NetworkPolicy, error) {
	if manifestPath == "" || repositoryRoot == "" || !filepath.IsAbs(manifestPath) || containsParentReference(manifestPath) {
		return nil, errors.New("network manifest rejected")
	}
	repository, err := canonicalExistingDirectory(repositoryRoot)
	if err != nil {
		return nil, errors.New("repository root rejected")
	}
	info, err := os.Lstat(manifestPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > maxNetworkManifest {
		return nil, errors.New("network manifest rejected")
	}
	resolved, err := filepath.EvalSymlinks(manifestPath)
	if err != nil {
		return nil, errors.New("network manifest rejected")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return nil, errors.New("network manifest rejected")
	}
	relative, err := filepath.Rel(repository, resolved)
	if err != nil || filepath.ToSlash(relative) != "safety/manifests/network-tests.v1.json" {
		return nil, errors.New("network manifest rejected")
	}
	file, err := os.Open(resolved)
	if err != nil {
		return nil, errors.New("network manifest rejected")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxNetworkManifest+1))
	if err != nil || len(data) > maxNetworkManifest {
		return nil, errors.New("network manifest rejected")
	}
	return ParseNetworkPolicy(data)
}

func ParseNetworkPolicy(data []byte) (*NetworkPolicy, error) {
	if len(data) == 0 || len(data) > maxNetworkManifest {
		return nil, errors.New("network manifest rejected")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, errors.New("network manifest rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest networkManifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, errors.New("network manifest rejected")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("network manifest rejected")
	}
	if manifest.SchemaVersion != networkManifestSchema || len(manifest.Tests) == 0 {
		return nil, errors.New("network manifest rejected")
	}
	policy := &NetworkPolicy{entries: make(map[string]networkTestEntry, len(manifest.Tests))}
	adapters := make(map[string]struct{}, len(manifest.Tests))
	for _, entry := range manifest.Tests {
		if err := validateNetworkEntry(entry); err != nil {
			return nil, err
		}
		if _, exists := policy.entries[entry.TestID]; exists {
			return nil, errors.New("network test id duplicated")
		}
		if _, exists := adapters[entry.AdapterID]; exists {
			return nil, errors.New("network adapter id duplicated")
		}
		policy.entries[entry.TestID] = entry
		adapters[entry.AdapterID] = struct{}{}
	}
	return policy, nil
}

func (policy *NetworkPolicy) Authorize(exactTestID string, tier Tier, ambientKeys []string) PolicyDecision {
	decision := PolicyDecision{
		Status:        PolicyManualRequired,
		Tier:          tier,
		NetworkPolicy: "denied",
		Reason:        "exact-network-test-required",
	}
	if tier != TierIsolatedIntegration {
		decision.Reason = "tier-network-denied"
		return decision
	}
	if exactTestID == "" || !validContractID(exactTestID) {
		return decision
	}
	if hasForbiddenAmbientKey(ambientKeys) {
		decision.Reason = "ambient-state-forbidden"
		return decision
	}
	entry, exists := policy.entries[exactTestID]
	if !exists {
		decision.Reason = "network-test-unknown"
		return decision
	}
	decision.TestID = entry.TestID
	decision.ContractValidated = true
	decision.Reason = "network-execution-unavailable-phase-1"
	return decision
}

func ValidateLiveProbeProof(proof LiveProbeProof, now time.Time) error {
	if !validContractID(proof.ProbeID) || !proof.OfficialReadOnlySemanticsCurrent || !proof.IsolatedNegativeEvidenceCurrent {
		return errors.New("live probe proof rejected")
	}
	if proof.OfficialReviewExpiresAt.IsZero() || !now.UTC().Before(proof.OfficialReviewExpiresAt.UTC()) {
		return errors.New("live probe proof rejected")
	}
	if !sha256Digest.MatchString(proof.IsolatedNegativeEvidenceDigest) {
		return errors.New("live probe proof rejected")
	}
	return nil
}

func (LivePolicy) Evaluate(probeID string) PolicyDecision {
	decision := PolicyDecision{
		Status:        PolicyUnknown,
		Tier:          TierLiveCheck,
		NetworkPolicy: "denied",
		Reason:        "live-probe-unapproved",
	}
	if validContractID(probeID) {
		decision.ProbeID = probeID
	}
	return decision
}

func validateNetworkEntry(entry networkTestEntry) error {
	if !validContractID(entry.TestID) || !validContractID(entry.AdapterID) || entry.Purpose != "fetch-one-public-synthetic-fixture" {
		return errors.New("network identity rejected")
	}
	if entry.Request.Protocol != "https" || entry.Request.Host != "example.invalid" || entry.Request.Port != 443 || entry.Request.Method != "GET" || entry.Request.Redirects != 0 {
		return errors.New("network request rejected")
	}
	parsed, err := url.Parse(entry.Request.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "example.invalid" || parsed.Port() != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("network url rejected")
	}
	if !strings.HasPrefix(parsed.Path, "/fixtures/") || !strings.HasSuffix(parsed.Path, ".tar.gz") || strings.Contains(parsed.Path, "//") || containsParentReference(parsed.Path) || parsed.EscapedPath() != parsed.Path {
		return errors.New("network url rejected")
	}
	if entry.Integrity.Algorithm != "sha256" || !sha256Digest.MatchString(entry.Integrity.Digest) {
		return errors.New("network integrity rejected")
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(entry.Integrity.Digest, "sha256:")); err != nil {
		return errors.New("network integrity rejected")
	}
	if entry.Limits.MaxBytes <= 0 || entry.Limits.MaxBytes > maxNetworkBytes || entry.Limits.TimeoutMS <= 0 || entry.Limits.TimeoutMS > maxNetworkTimeoutMS {
		return errors.New("network limits rejected")
	}
	cache, err := privacy.ParseLogicalRef(entry.CacheRef)
	wantCache := "fixture:network-cache/" + entry.TestID
	if err != nil || cache.Namespace != privacy.NamespaceFixture || cache.String() != wantCache {
		return errors.New("network cache rejected")
	}
	if entry.Credentials != "forbidden" || entry.ProxyEnvironment != "forbidden" || entry.EgressPolicy != "exact-url-only" {
		return errors.New("network isolation rejected")
	}
	if entry.Authorization.Mode != "exact-test-id" || entry.Authorization.RequiredArgument != "allow-network-test" || entry.Authorization.Execution != "validation-only" {
		return errors.New("network authorization rejected")
	}
	return nil
}

func validContractID(value string) bool {
	return len(value) <= 96 && publicContractID.MatchString(value) && !strings.ContainsAny(value, "*?[]{}$();/\\")
}

func hasForbiddenAmbientKey(entries []string) bool {
	for _, entry := range entries {
		key, _, _ := strings.Cut(entry, "=")
		key = strings.ToUpper(key)
		if key == "HTTP_PROXY" || key == "HTTPS_PROXY" || key == "ALL_PROXY" || key == "NO_PROXY" ||
			strings.HasPrefix(key, "AWS_") || strings.HasPrefix(key, "GITHUB_") || strings.HasPrefix(key, "SSH_") ||
			strings.Contains(key, "TOKEN") || strings.Contains(key, "SECRET") || strings.Contains(key, "PASSWORD") ||
			strings.Contains(key, "CREDENTIAL") || strings.HasSuffix(key, "_KEY") {
			return true
		}
	}
	return false
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing json rejected")
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
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
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("json object rejected")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("json array rejected")
		}
	default:
		return errors.New("json delimiter rejected")
	}
	return nil
}
