package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.invalid/yamc/safety/internal/privacy"
)

const (
	CodeStoreUnavailable   ErrorCode = "ARTIFACT_STORE_UNAVAILABLE"
	CodeStoreNotFound      ErrorCode = "ARTIFACT_STORE_NOT_FOUND"
	CodeStoreKeyMismatch   ErrorCode = "ARTIFACT_STORE_KEY_MISMATCH"
	CodeStoreCollision     ErrorCode = "ARTIFACT_STORE_COLLISION"
	CodeStoreExpired       ErrorCode = "ARTIFACT_STORE_EXPIRED"
	CodeStorePinned        ErrorCode = "ARTIFACT_STORE_PINNED"
	CodeStoreDeleteDenied  ErrorCode = "ARTIFACT_STORE_DELETE_DENIED"
	CodeStoreClockRejected ErrorCode = "ARTIFACT_STORE_CLOCK_REJECTED"
	CodePlanTransition     ErrorCode = "ARTIFACT_PLAN_TRANSITION_REJECTED"
	FreshObservedKey                 = "fresh-observed-state"
	maxStoredArtifactBytes           = 1 << 20
	snapshotClockSkew                = 2 * time.Minute
)

type storedMetadata struct {
	envelope   Envelope
	references []string
}

type planTransition struct {
	state        TerminalState
	recordDigest string
}

type persistedTransition struct {
	SchemaVersion string        `json:"schema_version"`
	PlanDigest    string        `json:"plan_digest"`
	State         TerminalState `json:"state"`
	RecordDigest  string        `json:"record_digest"`
}

type Store struct {
	root                string
	rootIdentity        os.FileInfo
	objectDirectory     *storeDirectory
	transitionDirectory *storeDirectory
	now                 func() time.Time
	objects             map[string]storedMetadata
	planTransitions     map[string]planTransition
}

func NewStore(root, repositoryRoot string) (*Store, error) {
	return NewStoreWithClock(root, repositoryRoot, time.Now)
}

func NewStoreWithClock(root, repositoryRoot string, clock func() time.Time) (*Store, error) {
	validated, err := ValidateExternalRoot(root, repositoryRoot)
	if err != nil {
		return nil, err
	}
	if clock == nil {
		return nil, contractError(CodeStoreUnavailable, "/clock")
	}
	rootIdentity, objectDirectory, transitionDirectory, err := initializeStoreFilesystem(validated)
	if err != nil {
		return nil, contractError(CodeStoreUnavailable, "/store")
	}
	store := &Store{
		root:                validated,
		rootIdentity:        rootIdentity,
		objectDirectory:     objectDirectory,
		transitionDirectory: transitionDirectory,
		now:                 clock,
		objects:             make(map[string]storedMetadata),
		planTransitions:     make(map[string]planTransition),
	}
	if err := store.rebuildMetadata(); err != nil {
		store.closeStoreFilesystem()
		return nil, err
	}
	return store, nil
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

func (store *Store) Validate(canonical []byte) (Envelope, error) {
	envelope, err := DecodeAndValidate(canonical)
	if err != nil {
		return Envelope{}, err
	}
	if err := validatePrivacy(canonical, envelope); err != nil {
		return Envelope{}, err
	}
	if err := store.validateSnapshotClock(envelope); err != nil {
		return Envelope{}, err
	}
	if store.snapshotExpired(envelope) && !store.isPinned(envelope.ContentDigest) {
		return Envelope{}, contractError(CodeStoreExpired, "/lifecycle/expires_at")
	}
	return envelope, nil
}

func (store *Store) Write(canonical []byte) (string, error) {
	digest, _, err := store.write(canonical)
	return digest, err
}

func (store *Store) write(canonical []byte) (digest string, created bool, resultErr error) {
	if len(canonical) == 0 || len(canonical) > maxStoredArtifactBytes {
		return "", false, contractError(CodeStoreUnavailable, "/artifact")
	}
	envelope, err := store.Validate(canonical)
	if err != nil {
		return "", false, err
	}
	if envelope.Kind == GeneratedPlan && envelope.Lifecycle.TerminalState != TerminalNonterminal {
		return "", false, contractError(CodePlanTransition, "/lifecycle/terminal_state")
	}
	metadataBefore := cloneMetadata(store.objects)
	success := false
	defer func() {
		if !success {
			store.objects = metadataBefore
		}
	}()
	if err := store.validateStoredReferences(envelope); err != nil {
		return "", false, err
	}
	if existing, err := store.readObjectFile(envelope.ContentDigest); err == nil {
		if !bytes.Equal(existing, canonical) {
			return "", false, contractError(CodeStoreCollision, "/content_digest")
		}
		validated, validateErr := DecodeAndValidate(existing)
		if validateErr != nil || validated.ContentDigest != envelope.ContentDigest {
			return "", false, contractError(CodeStoreKeyMismatch, "/content_digest")
		}
		store.recordMetadata(validated)
		success = true
		return envelope.ContentDigest, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, contractError(CodeStoreUnavailable, "/store")
	}
	created, err = store.publishObjectFile(envelope.ContentDigest, canonical)
	if err != nil {
		if errors.Is(err, errStoreFileCollision) {
			return "", false, contractError(CodeStoreCollision, "/content_digest")
		}
		return "", false, contractError(CodeStoreUnavailable, "/store")
	}
	store.recordMetadata(envelope)
	success = true
	return envelope.ContentDigest, created, nil
}

type graphWrite struct {
	label     string
	kind      Kind
	canonical []byte
}

func (store *Store) WriteGraph(mode LineageMode, graph LineageGraph) (map[string]string, error) {
	if err := ValidateLineage(mode, graph); err != nil {
		return nil, err
	}
	ordered := []graphWrite{
		{string(DesiredState), DesiredState, graph.Desired},
		{string(ObservedState), ObservedState, graph.Observed},
	}
	if mode == LineageApply {
		ordered = append(ordered,
			graphWrite{string(GeneratedPlan), GeneratedPlan, graph.Plan},
			graphWrite{string(AppliedReceipt), AppliedReceipt, graph.Receipt},
			graphWrite{FreshObservedKey, ObservedState, graph.FreshObserved},
		)
	}
	ordered = append(ordered,
		graphWrite{string(VerificationEvidence), VerificationEvidence, graph.Evidence},
		graphWrite{string(ReadinessReport), ReadinessReport, graph.Report},
	)
	if err := store.preflightGraph(ordered); err != nil {
		return nil, err
	}
	metadataBefore := cloneMetadata(store.objects)
	createdDigests := make([]string, 0, len(ordered))
	rollback := func() {
		store.objects = metadataBefore
		for _, digest := range createdDigests {
			store.removeCreatedObjectFile(digest)
		}
	}
	result := make(map[string]string, len(ordered))
	for _, item := range ordered {
		digest, created, err := store.write(item.canonical)
		if err != nil {
			rollback()
			return nil, err
		}
		if created {
			createdDigests = append(createdDigests, digest)
		}
		result[item.label] = digest
	}
	return result, nil
}

func (store *Store) preflightGraph(ordered []graphWrite) error {
	for _, item := range ordered {
		envelope, err := store.Validate(item.canonical)
		if err != nil || envelope.Kind != item.kind {
			if err != nil {
				return err
			}
			return contractError(CodeKindRejected, "/kind")
		}
		if envelope.Kind == GeneratedPlan && envelope.Lifecycle.TerminalState != TerminalNonterminal {
			return contractError(CodePlanTransition, "/lifecycle/terminal_state")
		}
		existing, readErr := store.readObjectFile(envelope.ContentDigest)
		if readErr == nil {
			if !bytes.Equal(existing, item.canonical) {
				return contractError(CodeStoreCollision, "/content_digest")
			}
			if validated, validateErr := DecodeAndValidate(existing); validateErr != nil || validated.ContentDigest != envelope.ContentDigest {
				return contractError(CodeStoreKeyMismatch, "/content_digest")
			}
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return contractError(CodeStoreUnavailable, "/store")
		}
	}
	return nil
}

func (store *Store) Read(digest string) (canonical []byte, envelope Envelope, resultErr error) {
	metadataBefore := cloneMetadata(store.objects)
	defer func() {
		if resultErr != nil {
			store.objects = metadataBefore
		}
	}()
	canonical, envelope, err := store.loadExact(digest, true)
	if err != nil {
		return nil, Envelope{}, err
	}
	if err := store.validateStoredReferences(envelope); err != nil {
		return nil, Envelope{}, err
	}
	return canonical, envelope, nil
}

func (store *Store) Delete(digest string) (resultErr error) {
	metadataBefore := cloneMetadata(store.objects)
	defer func() {
		if resultErr != nil {
			store.objects = metadataBefore
		}
	}()
	_, envelope, err := store.loadExact(digest, false)
	if err != nil {
		return err
	}
	// 已终结的计划允许在祖先快照过期后清理，因此不能再强制解析已释放的引用。
	terminalPlan := envelope.Kind == GeneratedPlan && store.planState(digest, envelope) != TerminalNonterminal
	if !terminalPlan {
		if err := store.validateStoredReferences(envelope); err != nil {
			return err
		}
	}
	if store.isPinned(digest) {
		return contractError(CodeStorePinned, "/content_digest")
	}
	switch envelope.Kind {
	case DesiredState, ObservedState:
		if !store.snapshotExpired(envelope) {
			return contractError(CodeStoreDeleteDenied, "/lifecycle/expires_at")
		}
	case GeneratedPlan:
		if store.planState(digest, envelope) == TerminalNonterminal {
			return contractError(CodeStoreDeleteDenied, "/lifecycle/terminal_state")
		}
	case AppliedReceipt, VerificationEvidence, ReadinessReport:
		return contractError(CodeStoreDeleteDenied, "/lifecycle/retention")
	default:
		return contractError(CodeStoreDeleteDenied, "/kind")
	}
	if err := store.removeObjectFile(digest); err != nil {
		return contractError(CodeStoreUnavailable, "/store")
	}
	if envelope.Kind == GeneratedPlan {
		_ = store.removeTransitionFile(digest)
	}
	delete(store.objects, digest)
	delete(store.planTransitions, digest)
	return nil
}

func (store *Store) TransitionPlan(planDigest string, state TerminalState, recordDigest string) (resultErr error) {
	metadataBefore := cloneMetadata(store.objects)
	defer func() {
		if resultErr != nil {
			store.objects = metadataBefore
		}
	}()
	_, plan, err := store.loadExact(planDigest, true)
	if err != nil || plan.Kind != GeneratedPlan || store.planState(planDigest, plan) != TerminalNonterminal {
		return contractError(CodePlanTransition, "/plan_digest")
	}
	switch state {
	case TerminalApplied:
		_, receipt, receiptErr := store.loadExact(recordDigest, true)
		if receiptErr != nil || receipt.Kind != AppliedReceipt {
			return contractError(CodePlanTransition, "/receipt_digest")
		}
		payload, decodeErr := decodePayload[AppliedReceiptPayload](receipt)
		if decodeErr != nil || payload.PlanDigest != planDigest {
			return contractError(CodePlanTransition, "/receipt_digest")
		}
		if referenceErr := store.validateStoredReferences(receipt); referenceErr != nil {
			return contractError(CodePlanTransition, "/receipt_digest")
		}
	case TerminalAbandoned:
		expectedRecord, digestErr := AbandonmentRecordDigest(planDigest)
		if digestErr != nil || recordDigest != expectedRecord {
			return contractError(CodePlanTransition, "/abandonment_record_digest")
		}
	default:
		return contractError(CodePlanTransition, "/terminal_state")
	}
	transition := planTransition{state: state, recordDigest: recordDigest}
	if err := store.writeTransition(planDigest, transition); err != nil {
		return err
	}
	store.planTransitions[planDigest] = transition
	return nil
}

func AbandonmentRecordDigest(planDigest string) (string, error) {
	if !IsDigest(planDigest) {
		return "", contractError(CodePlanTransition, "/plan_digest")
	}
	return DigestValue(struct {
		PlanDigest string        `json:"plan_digest"`
		State      TerminalState `json:"state"`
	}{PlanDigest: planDigest, State: TerminalAbandoned})
}

func (store *Store) validateStoredReferences(envelope Envelope) error {
	switch envelope.Kind {
	case DesiredState, ObservedState:
		return nil
	case GeneratedPlan:
		payload, err := decodePayload[GeneratedPlanPayload](envelope)
		if err != nil {
			return contractError(CodePayloadRejected, "/payload")
		}
		_, desired, desiredErr := store.loadExact(payload.DesiredDigest, true)
		_, observed, observedErr := store.loadExact(payload.ObservedDigest, true)
		if desiredErr != nil || observedErr != nil || desired.Kind != DesiredState || observed.Kind != ObservedState || !exactDigests(envelope.Provenance.InputDigests, desired.ContentDigest, observed.ContentDigest) {
			return contractError(CodeLineageDigestMismatch, "/payload")
		}
		return nil
	case AppliedReceipt:
		payload, err := decodePayload[AppliedReceiptPayload](envelope)
		if err != nil {
			return contractError(CodePayloadRejected, "/payload")
		}
		_, plan, planErr := store.loadExact(payload.PlanDigest, true)
		if planErr != nil || plan.Kind != GeneratedPlan {
			return contractError(CodeLineageDigestMismatch, "/payload/plan_digest")
		}
		planPayload, decodeErr := decodePayload[GeneratedPlanPayload](plan)
		if decodeErr != nil || !orderedSubset(planPayload.OperationIDs, payload.OperationIDs) || !exactDigests(envelope.Provenance.InputDigests, plan.ContentDigest) {
			return contractError(CodeLineageOperationOrder, "/payload/operation_ids")
		}
		return store.validateStoredReferences(plan)
	case VerificationEvidence:
		payload, err := decodePayload[VerificationEvidencePayload](envelope)
		if err != nil {
			return contractError(CodePayloadRejected, "/payload")
		}
		if payload.PlanDigest != "" || payload.ReceiptDigest != "" {
			graph, graphErr := store.graphThroughEvidence(envelope, payload)
			if graphErr != nil {
				return graphErr
			}
			nodes, nodeErr := store.nodesForGraph(graph)
			if nodeErr != nil {
				return nodeErr
			}
			nodes.evidence = envelope
			return validateApplyEdges(nodes, payload.ExpectedPostconditionsDigest)
		}
		graph, graphErr := store.graphThroughEvidence(envelope, payload)
		if graphErr != nil {
			return graphErr
		}
		nodes, nodeErr := store.nodesForGraph(graph)
		if nodeErr != nil {
			return nodeErr
		}
		nodes.evidence = envelope
		return validateReadOnlyEdges(nodes, payload.ExpectedPostconditionsDigest)
	case ReadinessReport:
		graph, mode, err := store.graphFromReport(envelope)
		if err != nil {
			return err
		}
		return ValidateLineage(mode, graph)
	default:
		return contractError(CodeKindRejected, "/kind")
	}
}

func (store *Store) graphFromReport(report Envelope) (LineageGraph, LineageMode, error) {
	payload, err := decodePayload[ReadinessReportPayload](report)
	if err != nil {
		return LineageGraph{}, "", contractError(CodePayloadRejected, "/report/payload")
	}
	evidenceDigests := payload.EvidenceDigests
	if payload.EvidenceDigest != "" {
		evidenceDigests = []string{payload.EvidenceDigest}
	}
	if len(evidenceDigests) != 1 {
		return LineageGraph{}, "", contractError(CodeLineageExtraEdge, "/report/payload/evidence_digests")
	}
	evidenceCanonical, evidence, evidenceErr := store.loadExact(evidenceDigests[0], true)
	if evidenceErr != nil || evidence.Kind != VerificationEvidence {
		return LineageGraph{}, "", contractError(CodeLineageDigestMismatch, "/report/payload/evidence_digests")
	}
	evidencePayload, decodeErr := decodePayload[VerificationEvidencePayload](evidence)
	if decodeErr != nil {
		return LineageGraph{}, "", contractError(CodePayloadRejected, "/evidence/payload")
	}
	graph, graphErr := store.graphThroughEvidence(evidence, evidencePayload)
	if graphErr != nil {
		return LineageGraph{}, "", graphErr
	}
	graph.Evidence = evidenceCanonical
	graph.Report, _ = Canonicalize(mustMarshalEnvelope(report))
	mode := LineageReadOnly
	if evidencePayload.PlanDigest != "" || evidencePayload.ReceiptDigest != "" {
		mode = LineageApply
	}
	return graph, mode, nil
}

func (store *Store) graphThroughEvidence(evidence Envelope, payload VerificationEvidencePayload) (LineageGraph, error) {
	graph := LineageGraph{ExpectedPostconditionsDigest: payload.ExpectedPostconditionsDigest}
	if payload.PlanDigest != "" || payload.ReceiptDigest != "" {
		planCanonical, plan, planErr := store.loadExact(payload.PlanDigest, true)
		receiptCanonical, receipt, receiptErr := store.loadExact(payload.ReceiptDigest, true)
		freshCanonical, fresh, freshErr := store.loadExact(payload.FreshObservedDigest, true)
		if planErr != nil || receiptErr != nil || freshErr != nil || plan.Kind != GeneratedPlan || receipt.Kind != AppliedReceipt || fresh.Kind != ObservedState {
			return LineageGraph{}, contractError(CodeLineageDigestMismatch, "/evidence/payload")
		}
		planPayload, decodeErr := decodePayload[GeneratedPlanPayload](plan)
		if decodeErr != nil {
			return LineageGraph{}, contractError(CodePayloadRejected, "/plan/payload")
		}
		desiredCanonical, _, desiredErr := store.loadExact(planPayload.DesiredDigest, true)
		observedCanonical, _, observedErr := store.loadExact(planPayload.ObservedDigest, true)
		if desiredErr != nil || observedErr != nil {
			return LineageGraph{}, contractError(CodeLineageMissingEdge, "/plan/payload")
		}
		graph.Desired = desiredCanonical
		graph.Observed = observedCanonical
		graph.Plan = planCanonical
		graph.Receipt = receiptCanonical
		graph.FreshObserved = freshCanonical
		return graph, nil
	}
	desiredCanonical, _, desiredErr := store.loadExact(payload.DesiredDigest, true)
	observedCanonical, _, observedErr := store.loadExact(payload.FreshObservedDigest, true)
	if desiredErr != nil || observedErr != nil {
		return LineageGraph{}, contractError(CodeLineageMissingEdge, "/evidence/payload")
	}
	graph.Desired = desiredCanonical
	graph.Observed = observedCanonical
	return graph, nil
}

func (store *Store) nodesForGraph(graph LineageGraph) (lineageNodes, error) {
	desired, err := Validate(DesiredState, graph.Desired)
	if err != nil {
		return lineageNodes{}, err
	}
	observed, err := Validate(ObservedState, graph.Observed)
	if err != nil {
		return lineageNodes{}, err
	}
	nodes := lineageNodes{desired: desired, observed: observed}
	if len(graph.Plan) != 0 {
		nodes.plan, err = Validate(GeneratedPlan, graph.Plan)
		if err != nil {
			return lineageNodes{}, err
		}
	}
	if len(graph.Receipt) != 0 {
		nodes.receipt, err = Validate(AppliedReceipt, graph.Receipt)
		if err != nil {
			return lineageNodes{}, err
		}
	}
	if len(graph.FreshObserved) != 0 {
		nodes.fresh, err = Validate(ObservedState, graph.FreshObserved)
		if err != nil {
			return lineageNodes{}, err
		}
	}
	return nodes, nil
}

func (store *Store) loadExact(digest string, enforceExpiry bool) ([]byte, Envelope, error) {
	if !IsDigest(digest) {
		return nil, Envelope{}, contractError(CodeStoreKeyMismatch, "/content_digest")
	}
	canonical, err := store.readObjectFile(digest)
	if errors.Is(err, os.ErrNotExist) {
		return nil, Envelope{}, contractError(CodeStoreNotFound, "/content_digest")
	}
	if err != nil {
		return nil, Envelope{}, contractError(CodeStoreUnavailable, "/store")
	}
	envelope, err := DecodeAndValidate(canonical)
	if err != nil {
		return nil, Envelope{}, err
	}
	if err := validatePrivacy(canonical, envelope); err != nil {
		return nil, Envelope{}, err
	}
	if envelope.ContentDigest != digest {
		return nil, Envelope{}, contractError(CodeStoreKeyMismatch, "/content_digest")
	}
	if err := store.validateSnapshotClock(envelope); err != nil {
		return nil, Envelope{}, err
	}
	store.recordMetadata(envelope)
	if enforceExpiry && store.snapshotExpired(envelope) && !store.isPinned(digest) {
		return nil, Envelope{}, contractError(CodeStoreExpired, "/lifecycle/expires_at")
	}
	return canonical, envelope, nil
}

func (store *Store) recordMetadata(envelope Envelope) {
	store.objects[envelope.ContentDigest] = storedMetadata{envelope: envelope, references: store.references(envelope)}
}

func (store *Store) references(envelope Envelope) []string {
	result := make([]string, 0, 4)
	switch envelope.Kind {
	case GeneratedPlan:
		payload, _ := decodePayload[GeneratedPlanPayload](envelope)
		result = append(result, payload.DesiredDigest, payload.ObservedDigest)
	case AppliedReceipt:
		payload, _ := decodePayload[AppliedReceiptPayload](envelope)
		result = append(result, payload.PlanDigest)
	case VerificationEvidence:
		payload, _ := decodePayload[VerificationEvidencePayload](envelope)
		for _, digest := range []string{payload.DesiredDigest, payload.PlanDigest, payload.ReceiptDigest, payload.FreshObservedDigest} {
			if IsDigest(digest) {
				result = append(result, digest)
			}
		}
	case ReadinessReport:
		payload, _ := decodePayload[ReadinessReportPayload](envelope)
		if payload.EvidenceDigest != "" {
			result = append(result, payload.EvidenceDigest)
		} else {
			result = append(result, payload.EvidenceDigests...)
		}
	}
	return result
}

func (store *Store) isPinned(target string) bool {
	for digest, metadata := range store.objects {
		if digest == target {
			continue
		}
		pins := metadata.envelope.Kind == AppliedReceipt || metadata.envelope.Kind == VerificationEvidence || metadata.envelope.Kind == ReadinessReport
		if metadata.envelope.Kind == GeneratedPlan && store.planState(digest, metadata.envelope) == TerminalNonterminal {
			pins = true
		}
		if pins && store.reaches(digest, target, make(map[string]bool)) {
			return true
		}
	}
	return false
}

func (store *Store) reaches(current, target string, seen map[string]bool) bool {
	if seen[current] {
		return false
	}
	seen[current] = true
	metadata, ok := store.objects[current]
	if !ok {
		return false
	}
	for _, reference := range metadata.references {
		if reference == target || store.reaches(reference, target, seen) {
			return true
		}
	}
	return false
}

func (store *Store) planState(digest string, envelope Envelope) TerminalState {
	if transition, ok := store.planTransitions[digest]; ok {
		return transition.state
	}
	return envelope.Lifecycle.TerminalState
}

func (store *Store) snapshotExpired(envelope Envelope) bool {
	if envelope.Lifecycle.Retention != Snapshot24Hours {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, envelope.Lifecycle.ExpiresAt)
	return err != nil || !store.now().UTC().Before(expiresAt)
}

func (store *Store) validateSnapshotClock(envelope Envelope) error {
	if envelope.Lifecycle.Retention != Snapshot24Hours {
		return nil
	}
	createdAt, createdErr := time.Parse(time.RFC3339, envelope.Lifecycle.CreatedAt)
	expiresAt, expiresErr := time.Parse(time.RFC3339, envelope.Lifecycle.ExpiresAt)
	now := store.now().UTC()
	if createdErr != nil || expiresErr != nil || createdAt.After(now.Add(snapshotClockSkew)) || expiresAt.After(now.Add(SnapshotLifetime+snapshotClockSkew)) {
		return contractError(CodeStoreClockRejected, "/lifecycle/created_at")
	}
	return nil
}

func (store *Store) rebuildMetadata() error {
	entries, err := store.objectDirectoryEntries()
	if err != nil {
		return contractError(CodeStoreUnavailable, "/store")
	}
	for _, entry := range entries {
		digest := "sha256:" + entry.Name()
		if !IsDigest(digest) {
			return contractError(CodeStoreKeyMismatch, "/content_digest")
		}
		canonical, readErr := store.readObjectFile(digest)
		if readErr != nil {
			return contractError(CodeStoreUnavailable, "/store")
		}
		envelope, validateErr := DecodeAndValidate(canonical)
		if validateErr != nil || envelope.ContentDigest != digest {
			return contractError(CodeStoreKeyMismatch, "/content_digest")
		}
		if privacyErr := validatePrivacy(canonical, envelope); privacyErr != nil {
			return privacyErr
		}
		if clockErr := store.validateSnapshotClock(envelope); clockErr != nil {
			return clockErr
		}
		if envelope.Kind == GeneratedPlan && envelope.Lifecycle.TerminalState != TerminalNonterminal {
			return contractError(CodePlanTransition, "/lifecycle/terminal_state")
		}
		store.recordMetadata(envelope)
	}
	transitionEntries, err := store.transitionDirectoryEntries()
	if err != nil {
		return contractError(CodeStoreUnavailable, "/transition")
	}
	for _, entry := range transitionEntries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			return contractError(CodePlanTransition, "/transition")
		}
		digest := "sha256:" + strings.TrimSuffix(entry.Name(), ".json")
		metadata, known := store.objects[digest]
		if !IsDigest(digest) || !known || metadata.envelope.Kind != GeneratedPlan {
			return contractError(CodePlanTransition, "/transition")
		}
		transition, exists, transitionErr := store.readTransition(digest)
		if transitionErr != nil || !exists {
			if transitionErr != nil {
				return transitionErr
			}
			return contractError(CodePlanTransition, "/transition")
		}
		store.planTransitions[digest] = transition
	}
	for digest, metadata := range store.objects {
		if metadata.envelope.Kind == GeneratedPlan && store.planState(digest, metadata.envelope) != TerminalNonterminal {
			continue
		}
		if err := store.validateStoredReferences(metadata.envelope); err != nil {
			return err
		}
	}
	return nil
}

func (store *Store) writeTransition(planDigest string, transition planTransition) error {
	record := persistedTransition{SchemaVersion: SchemaVersion, PlanDigest: planDigest, State: transition.state, RecordDigest: transition.recordDigest}
	encoded, err := json.Marshal(record)
	if err != nil {
		return contractError(CodePlanTransition, "/transition")
	}
	canonical, err := Canonicalize(encoded)
	if err != nil {
		return contractError(CodePlanTransition, "/transition")
	}
	if _, rejection := privacy.Gate(privacy.Candidate{
		ArtifactKind: privacy.KindStoreTransition,
		AdapterID:    privacy.AdapterArtifactStore,
		Canonical:    canonical,
	}); rejection != nil {
		return rejection
	}
	if _, err := store.publishTransitionFile(planDigest, canonical); err != nil {
		return contractError(CodePlanTransition, "/transition")
	}
	return nil
}

func (store *Store) readTransition(planDigest string) (planTransition, bool, error) {
	canonical, err := store.readTransitionFile(planDigest)
	if errors.Is(err, os.ErrNotExist) {
		return planTransition{}, false, nil
	}
	if err != nil {
		return planTransition{}, false, contractError(CodeStoreUnavailable, "/transition")
	}
	recanonical, canonicalErr := Canonicalize(canonical)
	if canonicalErr != nil || !bytes.Equal(recanonical, canonical) {
		return planTransition{}, false, contractError(CodePlanTransition, "/transition")
	}
	var record persistedTransition
	if err := decodeClosed(canonical, &record, "/transition"); err != nil || record.SchemaVersion != SchemaVersion || record.PlanDigest != planDigest || !IsDigest(record.RecordDigest) {
		return planTransition{}, false, contractError(CodePlanTransition, "/transition")
	}
	transition := planTransition{state: record.State, recordDigest: record.RecordDigest}
	switch transition.state {
	case TerminalApplied:
		_, receipt, receiptErr := store.loadExact(transition.recordDigest, true)
		if receiptErr != nil || receipt.Kind != AppliedReceipt {
			return planTransition{}, false, contractError(CodePlanTransition, "/transition")
		}
		payload, payloadErr := decodePayload[AppliedReceiptPayload](receipt)
		if payloadErr != nil || payload.PlanDigest != planDigest || store.validateStoredReferences(receipt) != nil {
			return planTransition{}, false, contractError(CodePlanTransition, "/transition")
		}
	case TerminalAbandoned:
		expected, digestErr := AbandonmentRecordDigest(planDigest)
		if digestErr != nil || transition.recordDigest != expected {
			return planTransition{}, false, contractError(CodePlanTransition, "/transition")
		}
	default:
		return planTransition{}, false, contractError(CodePlanTransition, "/transition")
	}
	return transition, true, nil
}

func cloneMetadata(source map[string]storedMetadata) map[string]storedMetadata {
	result := make(map[string]storedMetadata, len(source))
	for digest, metadata := range source {
		result[digest] = storedMetadata{envelope: metadata.envelope, references: append([]string(nil), metadata.references...)}
	}
	return result
}

func mustMarshalEnvelope(envelope Envelope) []byte {
	canonical, _, err := NewWithOptions(envelope.Kind, envelope.Run, envelope.Provenance, envelope.Payload, BuildOptions{StorageClass: envelope.StorageClass, Lifecycle: envelope.Lifecycle})
	if err != nil {
		return nil
	}
	return canonical
}

func validatePrivacy(canonical []byte, envelope Envelope) error {
	_, rejection := privacy.Gate(privacy.Candidate{
		ArtifactKind: privacy.ArtifactKind(envelope.Kind),
		AdapterID:    privacy.AdapterArtifactStore,
		Canonical:    canonical,
	})
	if rejection != nil {
		return rejection
	}
	return nil
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
