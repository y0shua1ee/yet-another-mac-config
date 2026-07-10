package privacy_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"example.invalid/yamc/safety/internal/artifact"
	"example.invalid/yamc/safety/internal/privacy"
)

func TestPrivacyBoundary(t *testing.T) {
	t.Run("six registered logical namespaces", testRegisteredLogicalReferences)
	t.Run("invalid and legacy logical references", testInvalidLogicalReferences)
	t.Run("closed surface compatibility matrix", testSurfaceCompatibility)
	t.Run("process local resolver containment", testResolverContainment)
	t.Run("canaries never reach output", testCanariesNeverReachOutput)
	t.Run("artifact store gates before write", testStoreGateBeforeWrite)
	t.Run("allowed string fields use closed value contracts", testClosedAllowedFieldContracts)
	t.Run("writers share the privacy gate", testWriterStructure)
}

func testRegisteredLogicalReferences(t *testing.T) {
	t.Helper()
	valid := map[privacy.Namespace]string{
		privacy.NamespaceRepo:       "repo:synthetic/config",
		privacy.NamespaceHome:       "home:.zshrc",
		privacy.NamespaceFixture:    "fixture:path/bin",
		privacy.NamespaceLocalState: "local-state:artifacts/sha256/public-id",
		privacy.NamespaceNixOutput:  "nix-output:synthetic-system",
		privacy.NamespaceProfile:    "profile:synthetic-developer",
	}
	if len(valid) != 6 {
		t.Fatal("registered namespace test is incomplete")
	}
	for expected, raw := range valid {
		reference, err := privacy.ParseLogicalRef(raw)
		if err != nil || reference.Namespace != expected || reference.String() != raw {
			t.Fatalf("registered logical reference rejected for namespace %q", expected)
		}
	}
	first, _ := privacy.ParseLogicalRef("repo:synthetic/config")
	second, _ := privacy.ParseLogicalRef("repo:synthetic/config")
	if first != second {
		t.Fatal("logical reference parsing is not deterministic")
	}
}

func testInvalidLogicalReferences(t *testing.T) {
	t.Helper()
	invalid := []string{
		"manager:synthetic/state",
		"service:synthetic/state",
		"external:synthetic/state",
		"unknown:synthetic/state",
		"repo:/synthetic/absolute",
		"repo:../synthetic/escape",
		"repo:synthetic/../escape",
		"repo:synthetic//ambiguous",
		`repo:synthetic\ambiguous`,
		"repo:synthetic:ambiguous",
		"repo:",
		"repo:synthetic\x00tail",
	}
	invalid[len(invalid)-1] = "repo:synthetic" + string([]byte{0}) + "tail"
	for _, raw := range invalid {
		if _, err := privacy.ParseLogicalRef(raw); err == nil {
			t.Fatal("invalid logical reference accepted")
		}
	}
}

func testSurfaceCompatibility(t *testing.T) {
	t.Helper()
	valid := map[privacy.SurfaceDomain][]string{
		privacy.SurfaceWorktree:    {"repo:sentinel/worktree/tracked", "repo:sentinel/worktree/index"},
		privacy.SurfaceNamedHome:   {"home:.zshrc"},
		privacy.SurfaceManagerRoot: {"home:sentinel/manager/synthetic-manager"},
		privacy.SurfaceService:     {"profile:sentinel/service/synthetic-service"},
		privacy.SurfaceNamedTarget: {"profile:sentinel/named-target/synthetic-target"},
	}
	representative := map[privacy.SurfaceDomain]string{}
	for domain, references := range valid {
		representative[domain] = references[0]
		for _, reference := range references {
			if err := privacy.ValidateSurface(domain, reference); err != nil {
				t.Fatalf("valid surface mapping rejected for %q", domain)
			}
		}
	}
	for domain := range valid {
		for otherDomain, reference := range representative {
			if domain == otherDomain {
				continue
			}
			if err := privacy.ValidateSurface(domain, reference); err == nil {
				t.Fatalf("wrong-domain reference accepted for %q", domain)
			}
		}
	}
	for _, invalid := range []struct {
		domain    privacy.SurfaceDomain
		reference string
	}{
		{privacy.SurfaceDomain("unknown"), "repo:sentinel/worktree/tracked"},
		{privacy.SurfaceWorktree, "repo:sentinel/worktree/../tracked"},
		{privacy.SurfaceNamedHome, "home:/synthetic/absolute"},
		{privacy.SurfaceManagerRoot, "manager:synthetic-manager"},
		{privacy.SurfaceService, "profile:sentinel/service/synthetic/service"},
		{privacy.SurfaceNamedTarget, "profile:sentinel/named-target/UPPER"},
	} {
		if err := privacy.ValidateSurface(invalid.domain, invalid.reference); err == nil {
			t.Fatal("invalid surface mapping accepted")
		}
	}
}

func testResolverContainment(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "logical-root")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(filepath.Join(root, "safe"), 0o700); err != nil {
		t.Fatal("synthetic resolver root unavailable")
	}
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal("synthetic outside root unavailable")
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal("synthetic resolver escape unavailable")
	}
	resolver, err := privacy.NewResolver(map[privacy.Namespace]string{privacy.NamespaceRepo: root})
	if err != nil {
		t.Fatal("process-local resolver rejected")
	}
	resolved, err := resolver.Resolve("repo:safe/child")
	canonicalRoot, canonicalErr := filepath.EvalSymlinks(root)
	relative, relativeErr := filepath.Rel(canonicalRoot, resolved)
	if err != nil || canonicalErr != nil || relativeErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("contained logical reference failed to resolve")
	}
	if _, err := resolver.Resolve("repo:escape/child"); err == nil {
		t.Fatal("resolver symlink escape accepted")
	}
	if _, rejection := privacy.Gate(privacy.Candidate{
		ArtifactKind: privacy.KindCommandResult,
		AdapterID:    privacy.AdapterPrivacyTest,
		Value:        map[string]any{"status": "synthetic"},
		LogicalRef:   "repo:escape/child",
		Resolver:     resolver,
	}); rejection == nil || rejection.Category != privacy.CategoryResolverEscape {
		t.Fatal("resolver escape did not fail before rendering")
	}
}

func testCanariesNeverReachOutput(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "canaries", "cases.json"))
	if err != nil {
		t.Fatal("synthetic canary cases unavailable")
	}
	var fixture struct {
		SchemaVersion string `json:"schema_version"`
		Cases         []struct {
			Name      string         `json:"name"`
			Candidate map[string]any `json:"candidate"`
		} `json:"cases"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil || fixture.SchemaVersion != "1.0.0" || len(fixture.Cases) < 16 {
		t.Fatal("synthetic canary cases rejected")
	}
	for _, item := range fixture.Cases {
		candidate := privacy.Candidate{ArtifactKind: privacy.KindCommandResult, AdapterID: privacy.AdapterPrivacyTest, Value: item.Candidate}
		approved, rejection := privacy.Gate(candidate)
		if rejection == nil || approved != nil {
			t.Fatalf("canary %q crossed the privacy boundary", item.Name)
		}
		var stdout bytes.Buffer
		if renderErr := privacy.Render(&stdout, candidate); renderErr == nil || stdout.Len() != 0 {
			t.Fatalf("canary %q reached stdout", item.Name)
		}
		var stderr bytes.Buffer
		if err := privacy.RenderError(&stderr, *rejection); err != nil {
			t.Fatalf("safe error envelope rejected for %q", item.Name)
		}
		assertExactErrorEnvelope(t, stderr.Bytes())
		for _, canary := range stringLeaves(item.Candidate) {
			if canary == "" {
				continue
			}
			sum := sha256.Sum256([]byte(canary))
			for _, forbidden := range []string{canary, filepath.Base(canary), hex.EncodeToString(sum[:])} {
				if forbidden != "." && forbidden != "/" && strings.Contains(stderr.String(), forbidden) {
					t.Fatalf("canary %q or a derived fingerprint reached stderr", item.Name)
				}
			}
		}
	}
	tampered := privacy.SafeOperationError(privacy.CodeOperationRejected, privacy.CategoryOperation, privacy.RemediationReview)
	tampered.Pointer = "/unregistered"
	var output bytes.Buffer
	if err := privacy.RenderError(&output, tampered); err == nil || output.Len() != 0 {
		t.Fatal("unregistered diagnostic field reached stderr")
	}
}

func testStoreGateBeforeWrite(t *testing.T) {
	t.Helper()
	repositoryRoot := t.TempDir()
	storeRoot := filepath.Join(t.TempDir(), "store")
	store, err := artifact.NewStore(storeRoot, repositoryRoot)
	if err != nil {
		t.Fatal("synthetic store setup failed")
	}
	run := artifact.RunMetadata{RunID: "synthetic-run-privacy", Tier: "offline-static", SuiteID: "privacy-boundary"}
	provenance := artifact.Provenance{Mode: "synthetic", InputDigests: []string{}}
	if _, _, err := artifact.New(artifact.DesiredState, run, provenance, artifact.DesiredPayload{
		Profile:      "profile:synthetic-developer",
		Declarations: []artifact.Fact{{Ref: "repo:/synthetic/private", State: "fixture:state/declared"}},
	}); err == nil {
		t.Fatal("privacy-negative artifact crossed the closed kind contract")
	}
	safe, _, err := artifact.New(artifact.DesiredState, run, provenance, artifact.DesiredPayload{
		Profile:      "profile:synthetic-developer",
		Declarations: []artifact.Fact{{Ref: "repo:synthetic/config", State: "fixture:state/declared"}},
	})
	if err != nil {
		t.Fatal("safe artifact construction failed")
	}
	unsafe := mutateArtifactCanonical(t, safe, func(value map[string]any) {
		value["payload"].(map[string]any)["declarations"].([]any)[0].(map[string]any)["ref"] = "repo:/synthetic/private"
	})
	if _, err := store.Write(unsafe); err == nil {
		t.Fatal("privacy-negative artifact reached the store")
	} else if strings.Contains(err.Error(), "repo:/synthetic/private") {
		t.Fatal("privacy-negative value reached the store diagnostic")
	}
	if entries, err := os.ReadDir(filepath.Join(storeRoot, "sha256")); err == nil && len(entries) != 0 {
		t.Fatal("privacy rejection left a stored object")
	}

	if _, err := store.Write(safe); err != nil {
		t.Fatal("safe artifact rejected by the privacy gate")
	}
}

func testClosedAllowedFieldContracts(t *testing.T) {
	t.Helper()
	repositoryRoot := t.TempDir()
	storeRoot := filepath.Join(t.TempDir(), "store")
	store, err := artifact.NewStore(storeRoot, repositoryRoot)
	if err != nil {
		t.Fatal("closed field store setup failed")
	}
	validRun := artifact.RunMetadata{RunID: "synthetic-run-closed-fields", Tier: "offline-static", SuiteID: "privacy-boundary"}
	provenance := artifact.Provenance{Mode: "synthetic", InputDigests: []string{}}
	valid, _, err := artifact.New(artifact.DesiredState, validRun, provenance, artifact.DesiredPayload{
		Profile:      "profile:synthetic-developer",
		Declarations: []artifact.Fact{{Ref: "repo:synthetic/config", State: "fixture:state/declared"}},
	})
	if err != nil {
		t.Fatal("closed field baseline unavailable")
	}
	mutations := []struct {
		name   string
		canary string
		apply  func(map[string]any)
	}{
		{"run_id", "synthetic-run-secret-canary", func(value map[string]any) { value["run"].(map[string]any)["run_id"] = "synthetic-run-secret-canary" }},
		{"suite_id", "synthetic-username-canary", func(value map[string]any) { value["run"].(map[string]any)["suite_id"] = "synthetic-username-canary" }},
		{"state", "provider-account-reference", func(value map[string]any) {
			value["payload"].(map[string]any)["declarations"].([]any)[0].(map[string]any)["state"] = "provider-account-reference"
		}},
	}
	var cliArtifact []byte
	for index, mutation := range mutations {
		candidate := mutateArtifactCanonical(t, valid, mutation.apply)
		if index == 0 {
			cliArtifact = candidate
		}
		if _, err := store.Write(candidate); err == nil || strings.Contains(err.Error(), mutation.canary) {
			t.Fatalf("closed %s canary crossed or echoed from Store.Write", mutation.name)
		}
	}
	dummy := "sha256:" + strings.Repeat("1", 64)
	if _, _, err := artifact.New(artifact.GeneratedPlan, validRun, provenance, artifact.GeneratedPlanPayload{
		DesiredDigest: dummy, ObservedDigest: dummy, ExpectedPostconditionsDigest: dummy,
		OperationIDs: []string{"fixture.operation.secret-token"},
	}); err == nil {
		t.Fatal("operation_ids secret crossed the artifact contract")
	}
	for _, candidate := range []map[string]any{
		{"status": "sk-synthetic-token-canary"},
		{"reason": "provider-account-reference"},
	} {
		var output bytes.Buffer
		if rejection := privacy.Render(&output, privacy.Candidate{ArtifactKind: privacy.KindCommandResult, AdapterID: privacy.AdapterPrivacyTest, Value: candidate}); rejection == nil || output.Len() != 0 {
			t.Fatal("closed command-result field crossed the renderer")
		}
	}

	safetyRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal("closed field CLI root unavailable")
	}
	artifactPath := filepath.Join(t.TempDir(), "candidate.json")
	if err := os.WriteFile(artifactPath, cliArtifact, 0o600); err != nil {
		t.Fatal("closed field CLI candidate unavailable")
	}
	command := exec.Command("go", "run", "./cmd/yamc-safety", "validate", "--kind", "desired-state", "--artifact", artifactPath)
	command.Dir = safetyRoot
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err == nil || stdout.Len() != 0 || strings.Contains(stderr.String(), "synthetic-run-secret-canary") {
		t.Fatal("closed run_id canary crossed or echoed from the CLI")
	}
	if entries, err := os.ReadDir(filepath.Join(storeRoot, "sha256")); err == nil && len(entries) != 0 {
		t.Fatal("closed field canary left a stored object")
	}
}

func mutateArtifactCanonical(t *testing.T, canonical []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(canonical, &value); err != nil {
		t.Fatal("artifact mutation setup failed")
	}
	delete(value, "content_digest")
	mutate(value)
	digest, err := artifact.DigestValue(value)
	if err != nil {
		t.Fatal("artifact mutation digest failed")
	}
	value["content_digest"] = digest
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal("artifact mutation encoding failed")
	}
	result, err := artifact.Canonicalize(encoded)
	if err != nil {
		t.Fatal("artifact mutation canonicalization failed")
	}
	return result
}

func testWriterStructure(t *testing.T) {
	t.Helper()
	checks := map[string][]string{
		filepath.Join("..", "artifact", "store.go"):                {"privacy.Gate(", "validatePrivacy("},
		filepath.Join("..", "..", "cmd", "yamc-safety", "main.go"): {"privacy.Gate(", "privacy.Render(", "privacy.RenderError("},
	}
	for path, required := range checks {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal("production writer source unavailable")
		}
		for _, literal := range required {
			if !bytes.Contains(data, []byte(literal)) {
				t.Fatalf("production writer %q bypasses the shared privacy gate", filepath.Base(path))
			}
		}
	}
	mainSource, _ := os.ReadFile(filepath.Join("..", "..", "cmd", "yamc-safety", "main.go"))
	for _, bypass := range []string{"json.NewEncoder(stdout)", "fmt.Print", "fmt.Fprint"} {
		if bytes.Contains(mainSource, []byte(bypass)) {
			t.Fatal("CLI retains a direct output bypass")
		}
	}
}

func assertExactErrorEnvelope(t *testing.T, encoded []byte) {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil || len(fields) != 6 {
		t.Fatal("privacy error envelope is not the exact six-field schema")
	}
	allowed := map[string]struct{}{
		"error_code": {}, "artifact_kind": {}, "adapter_id": {},
		"pointer": {}, "category": {}, "remediation": {},
	}
	for field := range fields {
		if _, ok := allowed[field]; !ok {
			t.Fatal("privacy error envelope contains an unregistered field")
		}
	}
}

func stringLeaves(value any) []string {
	result := make([]string, 0)
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for _, child := range typed {
				visit(child)
			}
		case []any:
			for _, child := range typed {
				visit(child)
			}
		case string:
			result = append(result, typed)
		}
	}
	visit(value)
	return result
}
