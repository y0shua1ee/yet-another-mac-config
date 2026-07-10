package privacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type Namespace string

const (
	NamespaceRepo       Namespace = "repo"
	NamespaceHome       Namespace = "home"
	NamespaceFixture    Namespace = "fixture"
	NamespaceLocalState Namespace = "local-state"
	NamespaceNixOutput  Namespace = "nix-output"
	NamespaceProfile    Namespace = "profile"
)

type SurfaceDomain string

const (
	SurfaceWorktree    SurfaceDomain = "worktree"
	SurfaceNamedHome   SurfaceDomain = "named-home"
	SurfaceManagerRoot SurfaceDomain = "manager-root"
	SurfaceService     SurfaceDomain = "service"
	SurfaceNamedTarget SurfaceDomain = "named-target"
)

type ArtifactKind string

const (
	KindDesiredState         ArtifactKind = "desired-state"
	KindObservedState        ArtifactKind = "observed-state"
	KindGeneratedPlan        ArtifactKind = "generated-plan"
	KindAppliedReceipt       ArtifactKind = "applied-receipt"
	KindVerificationEvidence ArtifactKind = "verification-evidence"
	KindReadinessReport      ArtifactKind = "readiness-report"
	KindCommandResult        ArtifactKind = "command-result"
	KindStoreTransition      ArtifactKind = "store-transition"
)

type AdapterID string

const (
	AdapterArtifactStore AdapterID = "artifact-store-v1"
	AdapterCLIRenderer   AdapterID = "cli-renderer-v1"
	AdapterSynthetic     AdapterID = "synthetic-adapter-v1"
	AdapterFixtureFake   AdapterID = "fixture-fake-adapter-v1"
	AdapterPrivacyTest   AdapterID = "privacy-test-v1"
)

type ErrorCode string

const (
	CodeLogicalRefRejected ErrorCode = "PRIVACY_LOGICAL_REF_REJECTED"
	CodeSurfaceRejected    ErrorCode = "PRIVACY_SURFACE_REJECTED"
	CodeResolverRejected   ErrorCode = "PRIVACY_RESOLVER_REJECTED"
	CodeDataRejected       ErrorCode = "PRIVACY_DATA_REJECTED"
	CodeCanonicalRejected  ErrorCode = "PRIVACY_CANONICAL_REJECTED"
	CodeCommandRejected    ErrorCode = "PRIVACY_COMMAND_REJECTED"
	CodeOperationRejected  ErrorCode = "PRIVACY_OPERATION_REJECTED"
	CodeOutputRejected     ErrorCode = "PRIVACY_OUTPUT_REJECTED"
)

type Category string

const (
	CategoryInvalidLogicalRef Category = "invalid-logical-reference"
	CategoryUnknownNamespace  Category = "unknown-namespace"
	CategoryAbsoluteRef       Category = "unknown-absolute-reference"
	CategorySurfaceMismatch   Category = "surface-domain-mismatch"
	CategoryResolverEscape    Category = "resolver-escape"
	CategoryForbiddenData     Category = "forbidden-category"
	CategoryUnclassified      Category = "unclassified-data"
	CategoryCanonical         Category = "canonical-rejected"
	CategoryUnsupported       Category = "unsupported-command"
	CategoryOperation         Category = "operation-rejected"
)

type Remediation string

const (
	RemediationLogicalRef    Remediation = "use-registered-logical-reference"
	RemediationSurface       Remediation = "use-compatible-surface-reference"
	RemediationResolver      Remediation = "keep-resolution-process-local"
	RemediationPrivateData   Remediation = "remove-private-data"
	RemediationNormalization Remediation = "use-registered-normalization"
	RemediationCommand       Remediation = "use-supported-command"
	RemediationReview        Remediation = "review-safe-input"
)

type ErrorEnvelope struct {
	ErrorCode    ErrorCode    `json:"error_code"`
	ArtifactKind ArtifactKind `json:"artifact_kind"`
	AdapterID    AdapterID    `json:"adapter_id"`
	Pointer      string       `json:"pointer"`
	Category     Category     `json:"category"`
	Remediation  Remediation  `json:"remediation"`
}

func (envelope *ErrorEnvelope) Error() string {
	if envelope == nil {
		return string(CodeOperationRejected)
	}
	return fmt.Sprintf("%s:%s:%s", envelope.ErrorCode, envelope.Category, envelope.Pointer)
}

type LogicalRef struct {
	Namespace Namespace
	ID        string
	raw       string
}

func (reference LogicalRef) String() string {
	return reference.raw
}

type Candidate struct {
	ArtifactKind  ArtifactKind
	AdapterID     AdapterID
	Canonical     []byte
	Value         any
	LogicalRef    string
	SurfaceDomain SurfaceDomain
	Resolver      *Resolver
}

type Resolver struct {
	roots map[Namespace]string
}

type logicalRefError struct {
	category Category
}

func (err *logicalRefError) Error() string {
	return string(err.category)
}

type violation struct {
	code        ErrorCode
	pointer     string
	category    Category
	remediation Remediation
}

var (
	registeredNamespaces = map[Namespace]struct{}{
		NamespaceRepo: {}, NamespaceHome: {}, NamespaceFixture: {},
		NamespaceLocalState: {}, NamespaceNixOutput: {}, NamespaceProfile: {},
	}
	registeredKinds = map[ArtifactKind]struct{}{
		KindDesiredState: {}, KindObservedState: {}, KindGeneratedPlan: {},
		KindAppliedReceipt: {}, KindVerificationEvidence: {}, KindReadinessReport: {},
		KindCommandResult: {}, KindStoreTransition: {},
	}
	registeredAdapters = map[AdapterID]struct{}{
		AdapterArtifactStore: {}, AdapterCLIRenderer: {}, AdapterSynthetic: {},
		AdapterFixtureFake: {}, AdapterPrivacyTest: {},
	}
	registeredCodes = map[ErrorCode]struct{}{
		CodeLogicalRefRejected: {}, CodeSurfaceRejected: {}, CodeResolverRejected: {},
		CodeDataRejected: {}, CodeCanonicalRejected: {}, CodeCommandRejected: {},
		CodeOperationRejected: {}, CodeOutputRejected: {},
	}
	registeredCategories = map[Category]struct{}{
		CategoryInvalidLogicalRef: {}, CategoryUnknownNamespace: {}, CategoryAbsoluteRef: {},
		CategorySurfaceMismatch: {}, CategoryResolverEscape: {}, CategoryForbiddenData: {},
		CategoryUnclassified: {}, CategoryCanonical: {}, CategoryUnsupported: {}, CategoryOperation: {},
	}
	registeredRemediations = map[Remediation]struct{}{
		RemediationLogicalRef: {}, RemediationSurface: {}, RemediationResolver: {},
		RemediationPrivateData: {}, RemediationNormalization: {}, RemediationCommand: {},
		RemediationReview: {},
	}
	registeredPointers = map[string]struct{}{
		"/": {}, "/logical_ref": {}, "/surface_domain": {}, "/resolver": {},
		"/candidate": {}, "/payload/private-data": {}, "/payload/unclassified": {},
		"/canonical": {}, "/command": {}, "/artifact": {},
	}
	publicIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	hex64Pattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	integerPattern  = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
	windowsPath     = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
)

func ParseLogicalRef(raw string) (LogicalRef, error) {
	if raw == "" || strings.ContainsRune(raw, '\x00') || strings.Count(raw, ":") != 1 {
		return LogicalRef{}, &logicalRefError{category: CategoryInvalidLogicalRef}
	}
	namespaceText, identifier, _ := strings.Cut(raw, ":")
	namespace := Namespace(namespaceText)
	if _, ok := registeredNamespaces[namespace]; !ok {
		return LogicalRef{}, &logicalRefError{category: CategoryUnknownNamespace}
	}
	if identifier == "" || strings.HasPrefix(identifier, "/") || strings.HasPrefix(identifier, "~/") || windowsPath.MatchString(identifier) {
		return LogicalRef{}, &logicalRefError{category: CategoryAbsoluteRef}
	}
	if strings.Contains(identifier, "\\") || strings.Contains(identifier, "//") || strings.Contains(identifier, ":") {
		return LogicalRef{}, &logicalRefError{category: CategoryInvalidLogicalRef}
	}
	segments := strings.Split(identifier, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return LogicalRef{}, &logicalRefError{category: CategoryInvalidLogicalRef}
		}
	}
	return LogicalRef{Namespace: namespace, ID: identifier, raw: raw}, nil
}

func ValidateSurface(domain SurfaceDomain, raw string) error {
	reference, err := ParseLogicalRef(raw)
	if err != nil {
		return err
	}
	valid := false
	switch domain {
	case SurfaceWorktree:
		valid = reference.Namespace == NamespaceRepo && (reference.ID == "sentinel/worktree/tracked" || reference.ID == "sentinel/worktree/index")
	case SurfaceNamedHome:
		valid = reference.Namespace == NamespaceHome && reference.ID == ".zshrc"
	case SurfaceManagerRoot:
		valid = hasPublicIDSuffix(reference, NamespaceHome, "sentinel/manager/")
	case SurfaceService:
		valid = hasPublicIDSuffix(reference, NamespaceProfile, "sentinel/service/")
	case SurfaceNamedTarget:
		valid = hasPublicIDSuffix(reference, NamespaceProfile, "sentinel/named-target/")
	default:
		return &logicalRefError{category: CategorySurfaceMismatch}
	}
	if !valid {
		return &logicalRefError{category: CategorySurfaceMismatch}
	}
	return nil
}

func hasPublicIDSuffix(reference LogicalRef, namespace Namespace, prefix string) bool {
	if reference.Namespace != namespace || !strings.HasPrefix(reference.ID, prefix) {
		return false
	}
	identifier := strings.TrimPrefix(reference.ID, prefix)
	return !strings.Contains(identifier, "/") && publicIDPattern.MatchString(identifier)
}

func NewResolver(roots map[Namespace]string) (*Resolver, error) {
	if len(roots) == 0 {
		return nil, &logicalRefError{category: CategoryResolverEscape}
	}
	validated := make(map[Namespace]string, len(roots))
	for namespace, root := range roots {
		if _, ok := registeredNamespaces[namespace]; !ok || root == "" || !filepath.IsAbs(root) {
			return nil, &logicalRefError{category: CategoryResolverEscape}
		}
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil {
			return nil, &logicalRefError{category: CategoryResolverEscape}
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			return nil, &logicalRefError{category: CategoryResolverEscape}
		}
		validated[namespace] = filepath.Clean(resolved)
	}
	return &Resolver{roots: validated}, nil
}

func (resolver *Resolver) Resolve(raw string) (string, error) {
	if resolver == nil {
		return "", &logicalRefError{category: CategoryResolverEscape}
	}
	reference, err := ParseLogicalRef(raw)
	if err != nil {
		return "", err
	}
	root, ok := resolver.roots[reference.Namespace]
	if !ok {
		return "", &logicalRefError{category: CategoryResolverEscape}
	}
	candidate, err := canonicalForCreation(filepath.Join(root, filepath.FromSlash(reference.ID)))
	if err != nil {
		return "", &logicalRefError{category: CategoryResolverEscape}
	}
	inside, err := isWithin(root, candidate)
	if err != nil || !inside {
		return "", &logicalRefError{category: CategoryResolverEscape}
	}
	return candidate, nil
}

func Gate(candidate Candidate) ([]byte, *ErrorEnvelope) {
	kind := candidate.ArtifactKind
	if _, ok := registeredKinds[kind]; !ok {
		kind = KindCommandResult
		return nil, newError(CodeDataRejected, kind, AdapterCLIRenderer, "/candidate", CategoryUnclassified, RemediationNormalization)
	}
	if _, ok := registeredAdapters[candidate.AdapterID]; !ok {
		return nil, newError(CodeDataRejected, kind, AdapterCLIRenderer, "/candidate", CategoryUnclassified, RemediationNormalization)
	}
	if (candidate.Canonical == nil) == (candidate.Value == nil) {
		return nil, newError(CodeDataRejected, kind, candidate.AdapterID, "/candidate", CategoryUnclassified, RemediationNormalization)
	}
	if candidate.LogicalRef != "" {
		if _, err := ParseLogicalRef(candidate.LogicalRef); err != nil {
			category := logicalCategory(err)
			return nil, newError(CodeLogicalRefRejected, kind, candidate.AdapterID, "/logical_ref", category, RemediationLogicalRef)
		}
		if candidate.SurfaceDomain != "" {
			if err := ValidateSurface(candidate.SurfaceDomain, candidate.LogicalRef); err != nil {
				return nil, newError(CodeSurfaceRejected, kind, candidate.AdapterID, "/surface_domain", CategorySurfaceMismatch, RemediationSurface)
			}
		}
		if candidate.Resolver != nil {
			if _, err := candidate.Resolver.Resolve(candidate.LogicalRef); err != nil {
				return nil, newError(CodeResolverRejected, kind, candidate.AdapterID, "/resolver", CategoryResolverEscape, RemediationResolver)
			}
		}
	} else if candidate.SurfaceDomain != "" || candidate.Resolver != nil {
		return nil, newError(CodeLogicalRefRejected, kind, candidate.AdapterID, "/logical_ref", CategoryInvalidLogicalRef, RemediationLogicalRef)
	}

	data := candidate.Canonical
	if candidate.Value != nil {
		encoded, err := json.Marshal(candidate.Value)
		if err != nil {
			return nil, newError(CodeCanonicalRejected, kind, candidate.AdapterID, "/canonical", CategoryCanonical, RemediationNormalization)
		}
		data = encoded
	}
	canonical, value, err := canonicalJSON(data)
	if err != nil || (candidate.Canonical != nil && !bytes.Equal(canonical, candidate.Canonical)) {
		return nil, newError(CodeCanonicalRejected, kind, candidate.AdapterID, "/canonical", CategoryCanonical, RemediationNormalization)
	}
	if found := scanValue(value, ""); found != nil {
		return nil, newError(found.code, kind, candidate.AdapterID, found.pointer, found.category, found.remediation)
	}
	return canonical, nil
}

func Render(writer io.Writer, candidate Candidate) *ErrorEnvelope {
	approved, rejected := Gate(candidate)
	if rejected != nil {
		return rejected
	}
	approved = append(approved, '\n')
	if _, err := writer.Write(approved); err != nil {
		return newError(CodeOutputRejected, candidate.ArtifactKind, candidate.AdapterID, "/artifact", CategoryOperation, RemediationReview)
	}
	return nil
}

func RenderError(writer io.Writer, envelope ErrorEnvelope) error {
	if err := ValidateErrorEnvelope(envelope); err != nil {
		return err
	}
	approved, rejected := Gate(Candidate{ArtifactKind: KindCommandResult, AdapterID: AdapterCLIRenderer, Value: envelope})
	if rejected != nil {
		return rejected
	}
	approved = append(approved, '\n')
	_, err := writer.Write(approved)
	return err
}

func ValidateErrorEnvelope(envelope ErrorEnvelope) error {
	if _, ok := registeredCodes[envelope.ErrorCode]; !ok {
		return errors.New("privacy error envelope rejected")
	}
	if _, ok := registeredKinds[envelope.ArtifactKind]; !ok {
		return errors.New("privacy error envelope rejected")
	}
	if _, ok := registeredAdapters[envelope.AdapterID]; !ok {
		return errors.New("privacy error envelope rejected")
	}
	if _, ok := registeredPointers[envelope.Pointer]; !ok {
		return errors.New("privacy error envelope rejected")
	}
	if _, ok := registeredCategories[envelope.Category]; !ok {
		return errors.New("privacy error envelope rejected")
	}
	if _, ok := registeredRemediations[envelope.Remediation]; !ok {
		return errors.New("privacy error envelope rejected")
	}
	return nil
}

func SafeOperationError(code ErrorCode, category Category, remediation Remediation) ErrorEnvelope {
	if _, ok := registeredCodes[code]; !ok {
		code = CodeOperationRejected
	}
	if _, ok := registeredCategories[category]; !ok {
		category = CategoryOperation
	}
	if _, ok := registeredRemediations[remediation]; !ok {
		remediation = RemediationReview
	}
	return *newError(code, KindCommandResult, AdapterCLIRenderer, "/command", category, remediation)
}

func newError(code ErrorCode, kind ArtifactKind, adapter AdapterID, pointer string, category Category, remediation Remediation) *ErrorEnvelope {
	return &ErrorEnvelope{ErrorCode: code, ArtifactKind: kind, AdapterID: adapter, Pointer: pointer, Category: category, Remediation: remediation}
}

func logicalCategory(err error) Category {
	var refErr *logicalRefError
	if errors.As(err, &refErr) {
		return refErr.category
	}
	return CategoryInvalidLogicalRef
}

func scanValue(value any, field string) *violation {
	switch typed := value.(type) {
	case map[string]any:
		if domainValue, ok := typed["surface_domain"].(string); ok {
			referenceValue, hasReference := typed["ref"].(string)
			if !hasReference {
				referenceValue, hasReference = typed["logical_ref"].(string)
			}
			if !hasReference || ValidateSurface(SurfaceDomain(domainValue), referenceValue) != nil {
				return &violation{CodeSurfaceRejected, "/surface_domain", CategorySurfaceMismatch, RemediationSurface}
			}
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			normalized := normalizeField(key)
			if forbiddenField(normalized) {
				return &violation{CodeDataRejected, "/payload/private-data", CategoryForbiddenData, RemediationPrivateData}
			}
			if found := scanValue(typed[key], normalized); found != nil {
				return found
			}
		}
	case []any:
		for _, item := range typed {
			if found := scanValue(item, field); found != nil {
				return found
			}
		}
	case string:
		if strings.ContainsRune(typed, '\x00') {
			return &violation{CodeDataRejected, "/payload/unclassified", CategoryUnclassified, RemediationNormalization}
		}
		if field == "pointer" {
			if _, ok := registeredPointers[typed]; !ok {
				return &violation{CodeDataRejected, "/payload/unclassified", CategoryUnclassified, RemediationNormalization}
			}
			return nil
		}
		if looksAbsoluteReference(typed) || looksPrivateNetwork(typed) {
			return &violation{CodeLogicalRefRejected, "/logical_ref", CategoryAbsoluteRef, RemediationLogicalRef}
		}
		if violation := validateStringField(field, typed); violation != nil {
			return violation
		}
	}
	return nil
}

func validateStringField(field, value string) *violation {
	reject := func() *violation {
		return &violation{CodeDataRejected, "/payload/unclassified", CategoryUnclassified, RemediationNormalization}
	}
	logicalReject := func(err error) *violation {
		return &violation{CodeLogicalRefRejected, "/logical_ref", logicalCategory(err), RemediationLogicalRef}
	}
	if field == "pointer" {
		if _, ok := registeredPointers[value]; !ok {
			return reject()
		}
		return nil
	}
	if isLogicalField(field) {
		if _, err := ParseLogicalRef(value); err != nil {
			return logicalReject(err)
		}
		return nil
	}
	if hasRegisteredPrefix(value) {
		if _, err := ParseLogicalRef(value); err != nil {
			return logicalReject(err)
		}
		return nil
	}
	if isDigestField(field) || isSHA256Digest(value) {
		if !isSHA256Digest(value) {
			return reject()
		}
		return nil
	}
	if isOpaqueTokenField(field) {
		if !isOpaqueToken(value) {
			return reject()
		}
		return nil
	}
	if isTimestampField(field) {
		if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
			return reject()
		}
		return nil
	}
	switch field {
	case "run_id", "suite_id", "operation_id", "operation_ids", "surface_id", "manifest_id", "test_id", "probe_id", "window_id", "task_suite", "wave", "id", "executable", "relative_id":
		if !isSafePublicID(value) {
			return reject()
		}
	case "decision_id":
		if !regexp.MustCompile(`^D-[0-9]{2}$`).MatchString(value) {
			return reject()
		}
	case "schema_version", "version":
		if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(value) {
			return reject()
		}
	case "state", "window_state", "terminal_state":
		if !closedState(value) {
			return reject()
		}
	case "status", "before_status", "after_status", "retention_status":
		if !closedStatus(value) {
			return reject()
		}
	case "reason", "before_reason", "after_reason":
		if !closedReason(value) {
			return reject()
		}
	case "verdict":
		if !oneOf(value, "passed", "violation", "indeterminate", "harness-error") {
			return reject()
		}
	case "tier":
		if !oneOf(value, "offline-static", "isolated-integration", "live-check", "real-sentinel-envelope") {
			return reject()
		}
	case "mode":
		if !oneOf(value, "synthetic", "real-run", "real", "report-only", "apply", "read-only") {
			return reject()
		}
	case "claim", "expected_claim":
		if value != "" && value != "covered-surfaces-unchanged-for-run" {
			return reject()
		}
	case "surface_domain":
		if !oneOf(value, string(SurfaceWorktree), string(SurfaceNamedHome), string(SurfaceManagerRoot), string(SurfaceService), string(SurfaceNamedTarget)) {
			return reject()
		}
	case "policy":
		if !oneOf(value, "required", "optional", "excluded") {
			return reject()
		}
	case "proof_state":
		if !oneOf(value, "current", "missing") {
			return reject()
		}
	case "storage_class":
		if !oneOf(value, "synthetic-golden", "external-local-state") {
			return reject()
		}
	case "retention":
		if !oneOf(value, "snapshot-24h", "append-only-plan", "append-only-evidence-bundle") {
			return reject()
		}
	case "error_code":
		if _, ok := registeredCodes[ErrorCode(value)]; !ok {
			return reject()
		}
	case "artifact_kind":
		if _, ok := registeredKinds[ArtifactKind(value)]; !ok {
			return reject()
		}
	case "adapter_id":
		if !isSafePublicID(value) {
			return reject()
		}
	case "category":
		if _, ok := registeredCategories[Category(value)]; !ok {
			return reject()
		}
	case "remediation":
		if _, ok := registeredRemediations[Remediation(value)]; !ok {
			return reject()
		}
	default:
		if !isSafePublicID(value) {
			return reject()
		}
	}
	return nil
}

func isDigestField(field string) bool {
	return field == "digest" || strings.HasSuffix(field, "_digest") || strings.HasSuffix(field, "_digests")
}

func isOpaqueTokenField(field string) bool {
	return field == "opaque_snapshot" || strings.HasSuffix(field, "_token")
}

func isTimestampField(field string) bool {
	return strings.HasSuffix(field, "_at") || field == "valid_until"
}

func isSHA256Digest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && hex64Pattern.MatchString(strings.TrimPrefix(value, "sha256:"))
}

func isOpaqueToken(value string) bool {
	return strings.HasPrefix(value, "hmac-sha256:") && hex64Pattern.MatchString(strings.TrimPrefix(value, "hmac-sha256:"))
}

func isSafePublicID(value string) bool {
	if !publicIDPattern.MatchString(value) {
		return false
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"api-key", "api_key", "private-key", "private_key"} {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	for _, segment := range strings.FieldsFunc(lower, func(r rune) bool { return r == '.' || r == '_' || r == '-' }) {
		if oneOf(segment, "secret", "token", "password", "credential", "private", "provider", "username", "hostname", "apikey") {
			return false
		}
	}
	for _, prefix := range []string{"sk-", "ghp_", "gho_", "ghu_", "ghs_", "ghr_", "xox", "akia", "eyj"} {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	return true
}

func closedState(value string) bool {
	if _, err := ParseLogicalRef(value); err == nil {
		return true
	}
	return oneOf(value, "closed", "nonterminal", "applied", "abandoned", "synthetic-sentinel-passed", "manual-required", "indeterminate", "violation", "harness-error", "covered-surfaces-unchanged-for-run")
}

func closedStatus(value string) bool {
	return oneOf(value,
		"synthetic-sentinel-passed", "synthetic-report-claim-ineligible", "synthetic-fixture", "valid", "stored", "normalized", "unknown", "ready", "manual-required",
		"complete", "incomplete", "passed", "violation", "indeterminate", "harness-error", "extra", "unmanaged-present",
		"removed", "retained", "teardown-failed", "covered-surfaces-unchanged-for-run")
}

func closedReason(value string) bool {
	return oneOf(value,
		"unreadable", "race-detected", "bound-exceeded", "symlink-escape", "window-exceeded",
		"offline-default", "isolated-offline", "tier-unknown", "exact-network-test-required", "network-execution-unavailable-phase-1", "network-test-unknown", "tier-network-denied", "live-probe-unapproved", "network-manifest-rejected", "ambient-state-forbidden",
		"fixture-not-expired", "fixture-not-retained", "owned-child-remove-failed", "owned-child-remove-incomplete", "ownership-validation-failed", "primary-verdict-not-frozen",
		"controlled-real-envelope-runner-required", "all-required-real-adapters-proven", "real-adapter-registry-rejected", "required-real-adapter-proof-unavailable", "required-real-adapter-unavailable", "real-manifest-freeze-rejected", "real-envelope-input-rejected", "real-adapter-implementation-rejected", "real-evidence-build-rejected", "primary-run-indeterminate", "primary-run-harness-error",
		"evidence-binding-rejected", "manifest-binding-rejected", "evidence-provenance-rejected", "observation-window-rejected", "observation-window-binding-rejected", "surface-policy-substitution-rejected", "surface-evidence-count-rejected", "surface-evidence-substitution-rejected", "surface-evidence-duplicate-rejected", "surface-observation-rejected", "surface-evidence-missing", "required-observation-incomplete", "real-envelope-binding-missing", "claim-rejected")
}

func oneOf(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}

func normalizeField(field string) string {
	return strings.ToLower(strings.ReplaceAll(field, "-", "_"))
}

func forbiddenField(field string) bool {
	forbidden := map[string]struct{}{
		"secret": {}, "secret_value": {}, "token": {}, "api_key": {}, "password": {},
		"credential": {}, "credentials": {}, "private_key": {}, "username": {}, "hostname": {},
		"serial": {}, "serial_number": {}, "hardware_fingerprint": {}, "provider": {},
		"provider_item": {}, "provider_item_reference": {}, "provider_ref": {}, "private_network": {},
		"environment": {}, "env": {}, "stdout": {}, "stderr": {}, "raw": {}, "raw_output": {},
		"raw_bytes": {}, "physical_path": {}, "absolute_path": {}, "path": {}, "resolver_mapping": {},
		"uid": {}, "host_identity": {}, "service_output": {}, "query_bytes": {}, "hmac_key": {},
		"identity": {}, "value": {}, "effective_uid": {}, "ownership_nonce": {},
	}
	if _, ok := forbidden[field]; ok {
		return true
	}
	for _, marker := range []string{"password", "credential", "private_key", "api_key", "provider_item", "hardware_fingerprint", "raw_output"} {
		if strings.HasPrefix(field, marker+"_") || strings.HasSuffix(field, "_"+marker) || strings.Contains(field, "_"+marker+"_") {
			return true
		}
	}
	return false
}

func isLogicalField(field string) bool {
	switch field {
	case "ref", "logical_ref", "profile", "scope", "outcome", "cache_ref", "selected_executable":
		return true
	default:
		return false
	}
}

func hasRegisteredPrefix(value string) bool {
	before, _, found := strings.Cut(value, ":")
	if !found {
		return false
	}
	_, ok := registeredNamespaces[Namespace(before)]
	return ok
}

func looksAbsoluteReference(value string) bool {
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "file:") || windowsPath.MatchString(value) || strings.Contains(value, "://")
}

func looksPrivateNetwork(value string) bool {
	ip := net.ParseIP(value)
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast())
}

func canonicalJSON(data []byte) ([]byte, any, error) {
	if len(data) == 0 || !utf8.Valid(data) {
		return nil, nil, errors.New("canonical input rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := readJSONValue(decoder)
	if err != nil {
		return nil, nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, nil, errors.New("canonical input rejected")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, nil, errors.New("canonical input rejected")
	}
	return canonical, value, nil
}

func readJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch typed := token.(type) {
	case json.Delim:
		switch typed {
		case '{':
			result := make(map[string]any)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, errors.New("canonical input rejected")
				}
				if _, exists := result[key]; exists {
					return nil, errors.New("canonical input rejected")
				}
				child, err := readJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				result[key] = child
			}
			closing, err := decoder.Token()
			if err != nil || closing != json.Delim('}') {
				return nil, errors.New("canonical input rejected")
			}
			return result, nil
		case '[':
			result := make([]any, 0)
			for decoder.More() {
				child, err := readJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				result = append(result, child)
			}
			closing, err := decoder.Token()
			if err != nil || closing != json.Delim(']') {
				return nil, errors.New("canonical input rejected")
			}
			return result, nil
		default:
			return nil, errors.New("canonical input rejected")
		}
	case json.Number:
		if !integerPattern.MatchString(typed.String()) {
			return nil, errors.New("canonical input rejected")
		}
		return typed, nil
	case string, bool, nil:
		return typed, nil
	default:
		return nil, errors.New("canonical input rejected")
	}
}

func canonicalForCreation(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	current := filepath.Clean(absolute)
	missing := make([]string, 0)
	for {
		_, err := os.Lstat(current)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("resolver path rejected")
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func isWithin(parent, child string) (bool, error) {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false, err
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))), nil
}
