package artifact

import (
	"encoding/json"
)

type LineageMode string

const (
	LineageApply    LineageMode = "apply"
	LineageReadOnly LineageMode = "read-only"

	CodeLineageModeRejected   ErrorCode = "ARTIFACT_LINEAGE_MODE_REJECTED"
	CodeLineageMissingEdge    ErrorCode = "ARTIFACT_LINEAGE_MISSING_EDGE"
	CodeLineageExtraEdge      ErrorCode = "ARTIFACT_LINEAGE_EXTRA_EDGE"
	CodeLineageDigestMismatch ErrorCode = "ARTIFACT_LINEAGE_DIGEST_MISMATCH"
	CodeLineageOperationOrder ErrorCode = "ARTIFACT_LINEAGE_OPERATION_ORDER"
	CodeLineageFreshness      ErrorCode = "ARTIFACT_LINEAGE_FRESHNESS_REJECTED"
	CodeLineageProvenance     ErrorCode = "ARTIFACT_LINEAGE_PROVENANCE_REJECTED"
)

type LineageGraph struct {
	Desired                      []byte
	Observed                     []byte
	FreshObserved                []byte
	Plan                         []byte
	Receipt                      []byte
	Evidence                     []byte
	Report                       []byte
	ExpectedPostconditionsDigest string
}

type lineageNodes struct {
	desired  Envelope
	observed Envelope
	fresh    Envelope
	plan     Envelope
	receipt  Envelope
	evidence Envelope
	report   Envelope
}

func ValidateLineage(mode LineageMode, graph LineageGraph) error {
	_, err := validateLineage(mode, graph)
	return err
}

func validateLineage(mode LineageMode, graph LineageGraph) (lineageNodes, error) {
	if mode != LineageApply && mode != LineageReadOnly {
		return lineageNodes{}, contractError(CodeLineageModeRejected, "/lineage_mode")
	}
	if !IsDigest(graph.ExpectedPostconditionsDigest) {
		return lineageNodes{}, contractError(CodeLineageMissingEdge, "/expected_postconditions_digest")
	}
	desired, err := validateLineageNode(DesiredState, graph.Desired, "/desired")
	if err != nil {
		return lineageNodes{}, err
	}
	observed, err := validateLineageNode(ObservedState, graph.Observed, "/observed")
	if err != nil {
		return lineageNodes{}, err
	}
	evidence, err := validateLineageNode(VerificationEvidence, graph.Evidence, "/evidence")
	if err != nil {
		return lineageNodes{}, err
	}
	report, err := validateLineageNode(ReadinessReport, graph.Report, "/report")
	if err != nil {
		return lineageNodes{}, err
	}
	nodes := lineageNodes{desired: desired, observed: observed, evidence: evidence, report: report}

	switch mode {
	case LineageApply:
		if len(graph.Plan) == 0 || len(graph.Receipt) == 0 || len(graph.FreshObserved) == 0 {
			return lineageNodes{}, contractError(CodeLineageMissingEdge, "/receipt")
		}
		plan, planErr := validateLineageNode(GeneratedPlan, graph.Plan, "/plan")
		if planErr != nil {
			return lineageNodes{}, planErr
		}
		receipt, receiptErr := validateLineageNode(AppliedReceipt, graph.Receipt, "/receipt")
		if receiptErr != nil {
			return lineageNodes{}, receiptErr
		}
		nodes.plan = plan
		nodes.receipt = receipt
		fresh, freshErr := validateLineageNode(ObservedState, graph.FreshObserved, "/fresh_observed")
		if freshErr != nil {
			return lineageNodes{}, freshErr
		}
		nodes.fresh = fresh
		if err := validateApplyEdges(nodes, graph.ExpectedPostconditionsDigest); err != nil {
			return lineageNodes{}, err
		}
	case LineageReadOnly:
		if len(graph.Plan) != 0 || len(graph.Receipt) != 0 || len(graph.FreshObserved) != 0 {
			return lineageNodes{}, contractError(CodeLineageExtraEdge, "/receipt")
		}
		if err := validateReadOnlyEdges(nodes, graph.ExpectedPostconditionsDigest); err != nil {
			return lineageNodes{}, err
		}
	}
	if err := validateReportEdge(nodes.report, nodes.evidence); err != nil {
		return lineageNodes{}, err
	}
	return nodes, nil
}

func validateApplyEdges(nodes lineageNodes, expectedDigest string) error {
	var planPayload GeneratedPlanPayload
	if err := decodeClosed(nodes.plan.Payload, &planPayload, "/plan/payload"); err != nil {
		return err
	}
	if planPayload.DesiredDigest != nodes.desired.ContentDigest || planPayload.ObservedDigest != nodes.observed.ContentDigest || planPayload.ExpectedPostconditionsDigest != expectedDigest {
		return contractError(CodeLineageDigestMismatch, "/plan/payload")
	}
	if !exactDigests(nodes.plan.Provenance.InputDigests, nodes.desired.ContentDigest, nodes.observed.ContentDigest) {
		return contractError(CodeLineageProvenance, "/plan/provenance/input_digests")
	}

	var receiptPayload AppliedReceiptPayload
	if err := decodeClosed(nodes.receipt.Payload, &receiptPayload, "/receipt/payload"); err != nil {
		return err
	}
	if receiptPayload.PlanDigest != nodes.plan.ContentDigest {
		return contractError(CodeLineageDigestMismatch, "/receipt/payload/plan_digest")
	}
	if !orderedSubset(planPayload.OperationIDs, receiptPayload.OperationIDs) {
		return contractError(CodeLineageOperationOrder, "/receipt/payload/operation_ids")
	}
	if !exactDigests(nodes.receipt.Provenance.InputDigests, nodes.plan.ContentDigest) {
		return contractError(CodeLineageProvenance, "/receipt/provenance/input_digests")
	}

	var evidencePayload VerificationEvidencePayload
	if err := decodeClosed(nodes.evidence.Payload, &evidencePayload, "/evidence/payload"); err != nil {
		return err
	}
	if evidencePayload.PlanDigest != nodes.plan.ContentDigest || evidencePayload.ReceiptDigest != nodes.receipt.ContentDigest || evidencePayload.DesiredDigest != "" || evidencePayload.ExpectedPostconditionsDigest != expectedDigest {
		return contractError(CodeLineageDigestMismatch, "/evidence/payload")
	}
	var freshPayload ObservedPayload
	if err := decodeClosed(nodes.fresh.Payload, &freshPayload, "/fresh_observed/payload"); err != nil {
		return err
	}
	if evidencePayload.FreshObservedDigest != nodes.fresh.ContentDigest || evidencePayload.FreshObserved.ContentDigest != nodes.fresh.ContentDigest || evidencePayload.FreshObserved.SourceReceiptDigest != nodes.receipt.ContentDigest || nodes.fresh.ContentDigest == nodes.observed.ContentDigest || evidencePayload.FreshObserved.Scope != freshPayload.Scope || !observedContainsState(freshPayload, evidencePayload.FreshObserved.State) {
		return contractError(CodeLineageFreshness, "/evidence/payload/fresh_observed_digest")
	}
	if !exactDigests(nodes.fresh.Provenance.InputDigests, nodes.receipt.ContentDigest) {
		return contractError(CodeLineageProvenance, "/fresh_observed/provenance/input_digests")
	}
	if !exactDigests(nodes.evidence.Provenance.InputDigests, nodes.plan.ContentDigest, nodes.receipt.ContentDigest, expectedDigest, nodes.fresh.ContentDigest) {
		return contractError(CodeLineageProvenance, "/evidence/provenance/input_digests")
	}
	return nil
}

func observedContainsState(payload ObservedPayload, state string) bool {
	if state == "" {
		return false
	}
	for _, fact := range payload.Facts {
		if fact.State == state {
			return true
		}
	}
	return false
}

func validateReadOnlyEdges(nodes lineageNodes, expectedDigest string) error {
	var observedPayload ObservedPayload
	if err := decodeClosed(nodes.observed.Payload, &observedPayload, "/observed/payload"); err != nil {
		return err
	}
	var evidencePayload VerificationEvidencePayload
	if err := decodeClosed(nodes.evidence.Payload, &evidencePayload, "/evidence/payload"); err != nil {
		return err
	}
	if evidencePayload.PlanDigest != "" || evidencePayload.ReceiptDigest != "" || evidencePayload.DesiredDigest != nodes.desired.ContentDigest || evidencePayload.ExpectedPostconditionsDigest != expectedDigest {
		return contractError(CodeLineageDigestMismatch, "/evidence/payload")
	}
	if evidencePayload.FreshObservedDigest != nodes.observed.ContentDigest || evidencePayload.FreshObserved.ContentDigest != nodes.observed.ContentDigest || evidencePayload.FreshObserved.SourceReceiptDigest != "" || evidencePayload.FreshObserved.Scope != observedPayload.Scope {
		return contractError(CodeLineageFreshness, "/evidence/payload/fresh_observed_digest")
	}
	if !exactDigests(nodes.evidence.Provenance.InputDigests, nodes.desired.ContentDigest, nodes.observed.ContentDigest, expectedDigest) {
		return contractError(CodeLineageProvenance, "/evidence/provenance/input_digests")
	}
	return nil
}

func validateReportEdge(report, evidence Envelope) error {
	var payload ReadinessReportPayload
	if err := decodeClosed(report.Payload, &payload, "/report/payload"); err != nil {
		return err
	}
	evidenceDigests := payload.EvidenceDigests
	if payload.EvidenceDigest != "" {
		evidenceDigests = []string{payload.EvidenceDigest}
	}
	if !exactDigests(evidenceDigests, evidence.ContentDigest) {
		return contractError(CodeLineageDigestMismatch, "/report/payload/evidence_digests")
	}
	if !exactDigests(report.Provenance.InputDigests, evidence.ContentDigest) {
		return contractError(CodeLineageProvenance, "/report/provenance/input_digests")
	}
	return nil
}

func validateLineageNode(kind Kind, canonical []byte, pointer string) (Envelope, error) {
	if len(canonical) == 0 {
		return Envelope{}, contractError(CodeLineageMissingEdge, pointer)
	}
	envelope, err := Validate(kind, canonical)
	if err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

func exactDigests(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range expected {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func orderedSubset(planOperations, receiptOperations []string) bool {
	if len(receiptOperations) == 0 || len(receiptOperations) > len(planOperations) {
		return false
	}
	next := 0
	for _, operationID := range planOperations {
		if operationID == receiptOperations[next] {
			next++
			if next == len(receiptOperations) {
				return true
			}
		}
	}
	return false
}

func decodePayload[T any](envelope Envelope) (T, error) {
	var payload T
	err := json.Unmarshal(envelope.Payload, &payload)
	return payload, err
}
