package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"example.invalid/yamc/safety/internal/artifact"
	"example.invalid/yamc/safety/internal/privacy"
)

const PolicySchemaVersion = "1.0.0"

type PolicyIntent string

const (
	IntentReportOnly       PolicyIntent = "report-only"
	IntentSyntheticFixture PolicyIntent = "synthetic-fixture"
)

type PolicyStatus string

const (
	StatusExtra            PolicyStatus = "extra"
	StatusUnmanagedPresent PolicyStatus = "unmanaged-present"
	StatusSyntheticFixture PolicyStatus = "synthetic-fixture"
)

type OperationKind string

const OperationFixtureFakeWrite OperationKind = "fixture-fake-write"

type Operation struct {
	Kind   OperationKind `json:"kind"`
	Target string        `json:"target"`
	Mode   string        `json:"mode"`
}

type PolicyRequest struct {
	SchemaVersion string       `json:"schema_version"`
	Provenance    string       `json:"provenance"`
	Intent        PolicyIntent `json:"intent"`
	Status        PolicyStatus `json:"status"`
	Operations    []Operation  `json:"operations"`
}

type PolicyDecision struct {
	Status     PolicyStatus `json:"status"`
	Operations []Operation  `json:"operations"`
}

type Policy struct{}

func Phase1Policy() Policy {
	return Policy{}
}

func ParsePolicy(data []byte) (PolicyRequest, error) {
	canonical, err := artifact.Canonicalize(data)
	if err != nil {
		return PolicyRequest{}, errors.New("phase-1 policy rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var request PolicyRequest
	if err := decoder.Decode(&request); err != nil {
		return PolicyRequest{}, errors.New("phase-1 policy rejected")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return PolicyRequest{}, errors.New("phase-1 policy rejected")
	}
	return request, nil
}

func (Policy) Evaluate(request PolicyRequest) (PolicyDecision, error) {
	if request.SchemaVersion != PolicySchemaVersion || request.Provenance != "synthetic" || request.Operations == nil {
		return PolicyDecision{}, errors.New("phase-1 policy rejected")
	}
	switch request.Intent {
	case IntentReportOnly:
		if (request.Status != StatusExtra && request.Status != StatusUnmanagedPresent) || len(request.Operations) != 0 {
			return PolicyDecision{}, errors.New("phase-1 report policy rejected")
		}
		return PolicyDecision{Status: request.Status, Operations: make([]Operation, 0)}, nil
	case IntentSyntheticFixture:
		if request.Status != StatusSyntheticFixture || len(request.Operations) != 1 || !validFixtureOperation(request.Operations[0]) {
			return PolicyDecision{}, errors.New("phase-1 fixture policy rejected")
		}
		return PolicyDecision{Status: request.Status, Operations: append([]Operation(nil), request.Operations...)}, nil
	default:
		return PolicyDecision{}, errors.New("phase-1 policy rejected")
	}
}

func validFixtureOperation(operation Operation) bool {
	if operation.Kind != OperationFixtureFakeWrite || operation.Mode != "synthetic" {
		return false
	}
	reference, err := privacy.ParseLogicalRef(operation.Target)
	return err == nil && reference.Namespace == privacy.NamespaceFixture
}
