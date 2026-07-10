package fixture

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

const maxMarkerBytes = 4 << 10

type PrimaryVerdict string

const (
	VerdictPassed        PrimaryVerdict = "passed"
	VerdictViolation     PrimaryVerdict = "violation"
	VerdictIndeterminate PrimaryVerdict = "indeterminate"
	VerdictHarnessError  PrimaryVerdict = "harness-error"
)

type FrozenPrimary struct {
	verdict PrimaryVerdict
	frozen  bool
}

type TeardownStatus string

const (
	TeardownRemoved  TeardownStatus = "removed"
	TeardownRetained TeardownStatus = "retained"
	TeardownFailed   TeardownStatus = "teardown-failed"
)

type TeardownOutcome struct {
	Status         TeardownStatus `json:"status"`
	LogicalID      string         `json:"logical_fixture_id"`
	ExpiryCategory string         `json:"expiry_category"`
	Reason         string         `json:"reason,omitempty"`
}

type FinalResult struct {
	Verdict  PrimaryVerdict  `json:"verdict"`
	Teardown TeardownOutcome `json:"teardown"`
}

type Retention struct {
	base         string
	root         string
	expected     ownershipMarker
	keep         bool
	clock        func() time.Time
	effectiveUID func() int
}

func FreezePrimary(verdict PrimaryVerdict) (FrozenPrimary, error) {
	switch verdict {
	case VerdictPassed, VerdictViolation, VerdictIndeterminate, VerdictHarnessError:
		return FrozenPrimary{verdict: verdict, frozen: true}, nil
	default:
		return FrozenPrimary{}, errors.New("primary verdict rejected")
	}
}

func (retention *Retention) Finalize(frozen FrozenPrimary) FinalResult {
	if !frozen.frozen {
		return combineTeardown(FrozenPrimary{verdict: VerdictHarnessError, frozen: true}, retention.failure("primary-verdict-not-frozen"))
	}
	if retention.keep {
		if _, err := retention.validateOwnedFixture(); err != nil {
			return combineTeardown(frozen, retention.failure("ownership-validation-failed"))
		}
		return combineTeardown(frozen, TeardownOutcome{
			Status:         TeardownRetained,
			LogicalID:      retention.expected.LogicalID,
			ExpiryCategory: retention.expiryCategory(),
		})
	}
	return combineTeardown(frozen, retention.teardownOwnedFixture())
}

func (retention *Retention) TeardownExpiredOwnedFixture(frozen FrozenPrimary) FinalResult {
	if !frozen.frozen {
		return combineTeardown(FrozenPrimary{verdict: VerdictHarnessError, frozen: true}, retention.failure("primary-verdict-not-frozen"))
	}
	if !retention.keep {
		return combineTeardown(frozen, retention.failure("fixture-not-retained"))
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, retention.expected.ExpiresAt)
	if err != nil || retention.clock().UTC().Before(expiresAt) {
		return combineTeardown(frozen, retention.failure("fixture-not-expired"))
	}
	return combineTeardown(frozen, retention.teardownOwnedFixture())
}

func (retention *Retention) teardownOwnedFixture() TeardownOutcome {
	root, err := retention.validateOwnedFixture()
	if err != nil {
		return retention.failure("ownership-validation-failed")
	}
	if err := os.RemoveAll(root); err != nil {
		return retention.failure("owned-child-remove-failed")
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		return retention.failure("owned-child-remove-incomplete")
	}
	return TeardownOutcome{
		Status:         TeardownRemoved,
		LogicalID:      retention.expected.LogicalID,
		ExpiryCategory: retention.expiryCategory(),
	}
}

func (retention *Retention) validateOwnedFixture() (string, error) {
	base, err := canonicalExistingDirectory(retention.base)
	if err != nil || base != retention.base {
		return "", errors.New("retention base rejected")
	}
	rootInfo, err := os.Lstat(retention.root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("fixture root rejected")
	}
	root, err := filepath.EvalSymlinks(retention.root)
	if err != nil {
		return "", errors.New("fixture root rejected")
	}
	root, err = filepath.Abs(root)
	if err != nil || filepath.Dir(root) != base || root == base || filepath.Base(root) != "fixture-"+retention.expected.Nonce {
		return "", errors.New("fixture containment rejected")
	}
	marker, err := readMarker(root)
	if err != nil || marker != retention.expected {
		return "", errors.New("fixture marker rejected")
	}
	if marker.SchemaVersion != markerSchemaVersion || marker.EffectiveUID != retention.effectiveUID() || marker.Nonce == "" {
		return "", errors.New("fixture ownership rejected")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, marker.CreatedAt)
	if err != nil {
		return "", errors.New("fixture ttl rejected")
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, marker.ExpiresAt)
	if err != nil {
		return "", errors.New("fixture ttl rejected")
	}
	ttl := expiresAt.Sub(createdAt)
	if ttl <= 0 || ttl > maximumRetentionTTL {
		return "", errors.New("fixture ttl rejected")
	}
	return root, nil
}

func readMarker(root string) (ownershipMarker, error) {
	path := filepath.Join(root, markerFileName)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > maxMarkerBytes {
		return ownershipMarker{}, errors.New("fixture marker rejected")
	}
	file, err := os.Open(path)
	if err != nil {
		return ownershipMarker{}, errors.New("fixture marker rejected")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxMarkerBytes+1))
	if err != nil || len(data) > maxMarkerBytes {
		return ownershipMarker{}, errors.New("fixture marker rejected")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var marker ownershipMarker
	if err := decoder.Decode(&marker); err != nil {
		return ownershipMarker{}, errors.New("fixture marker rejected")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ownershipMarker{}, errors.New("fixture marker rejected")
	}
	return marker, nil
}

func combineTeardown(frozen FrozenPrimary, outcome TeardownOutcome) FinalResult {
	verdict := frozen.verdict
	if outcome.Status == TeardownFailed && verdict == VerdictPassed {
		verdict = VerdictHarnessError
	}
	return FinalResult{Verdict: verdict, Teardown: outcome}
}

func (retention *Retention) failure(reason string) TeardownOutcome {
	return TeardownOutcome{
		Status:         TeardownFailed,
		LogicalID:      retention.expected.LogicalID,
		ExpiryCategory: retention.expiryCategory(),
		Reason:         reason,
	}
}

func (retention *Retention) expiryCategory() string {
	expiresAt, err := time.Parse(time.RFC3339Nano, retention.expected.ExpiresAt)
	if err != nil {
		return "invalid"
	}
	if !retention.clock().UTC().Before(expiresAt) {
		return "expired"
	}
	return "within-24-hours"
}
