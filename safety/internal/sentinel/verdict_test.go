package sentinel

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSentinelVerdicts(t *testing.T) {
	t.Run("returns exactly four verdicts and exits", testFourVerdicts)
	t.Run("permits only predeclared optional warnings", testOptionalWarnings)
	t.Run("binds every suite manifest window and token field", testEvidenceSubstitution)
	t.Run("keeps synthetic evidence structurally non-claiming", testSyntheticClaimCeiling)
	t.Run("combines after failures monotonically", testMonotonicVerdict)
}

func testFourVerdicts(t *testing.T) {
	manifest, evidence := validSyntheticEvidence(t)
	passed := Evaluate(manifest, evidence)
	if passed.Verdict != VerdictPassed || passed.ExitCode != ExitPassed || passed.Claim != "" || passed.EvidenceDigest == "" {
		t.Fatal("complete equal synthetic evidence did not pass without claim")
	}

	violation := evidence
	violation.Surfaces = append([]SurfaceEvidence(nil), evidence.Surfaces...)
	violation.Surfaces[0].AfterToken = opaqueToken(0x71)
	changed := Evaluate(manifest, violation)
	encoded, _ := json.Marshal(changed)
	if changed.Verdict != VerdictViolation || changed.ExitCode != ExitViolation || changed.ChangeCode != ChangeDetectedCode || changed.Reason != "" || bytes.Contains(encoded, []byte("restore")) || bytes.Contains(encoded, []byte("retry")) || bytes.Contains(encoded, []byte("attribution")) {
		t.Fatal("required difference did not produce the bounded violation")
	}

	incomplete := evidence
	incomplete.Surfaces = append([]SurfaceEvidence(nil), evidence.Surfaces...)
	incomplete.Surfaces[0].AfterStatus = ObservationIncomplete
	incomplete.Surfaces[0].AfterToken = ""
	incomplete.Surfaces[0].AfterReason = ReasonUnreadable
	unknown := Evaluate(manifest, incomplete)
	if unknown.Verdict != VerdictIndeterminate || unknown.ExitCode != ExitIndeterminate || unknown.Claim != "" {
		t.Fatal("missing required observation returned pass")
	}

	invalid := evidence
	invalid.ManifestDigest = sha256Digest([]byte("substituted"))
	harness := Evaluate(manifest, invalid)
	if harness.Verdict != VerdictHarnessError || harness.ExitCode != ExitHarnessError || harness.Claim != "" {
		t.Fatal("binding failure did not produce harness-error")
	}
}

func testOptionalWarnings(t *testing.T) {
	manifest, before, after := verdictSnapshots(t)
	manifest.Surfaces = append([]ProtectedSurface(nil), manifest.Surfaces...)
	manifest.Surfaces[0].Policy = PolicyOptional
	manifest = reparseManifest(t, manifest)
	for index := range before.Surfaces {
		if before.Surfaces[index].LogicalRef == manifest.Surfaces[0].LogicalRef {
			after.Surfaces[index].OpaqueState = opaqueToken(0x44)
		}
	}
	evidence, err := BuildEvidence(manifest, beforeWithDigest(before, manifest.Digest), beforeWithDigest(after, manifest.Digest), evidenceOptions(manifest, "synthetic"))
	if err != nil {
		t.Fatal("optional evidence setup failed")
	}
	result := Evaluate(manifest, evidence)
	if result.Verdict != VerdictPassed || result.ExitCode != 0 || len(result.Warnings) != 1 || result.Warnings[0] != "optional-change-detected" {
		t.Fatal("predeclared optional difference blocked required pass")
	}
	evidence.Optional = nil
	if result := Evaluate(manifest, evidence); result.Verdict != VerdictHarnessError {
		t.Fatal("post-start optional downgrade was accepted")
	}
}

func testEvidenceSubstitution(t *testing.T) {
	manifest, evidence := validSyntheticEvidence(t)
	mutations := []func(*Evidence){
		func(value *Evidence) { value.SuiteID = "substituted-suite" },
		func(value *Evidence) { value.SuiteDigest = sha256Digest([]byte("substituted")) },
		func(value *Evidence) { value.Tier = "live-check" },
		func(value *Evidence) { value.ManifestDigest = sha256Digest([]byte("substituted")) },
		func(value *Evidence) { value.Window.WindowID = "substituted-window" },
		func(value *Evidence) { value.Window.ClosedAt = value.Window.OpenedAt },
		func(value *Evidence) { value.Surfaces[0].BeforeToken = "sha256:not-opaque" },
		func(value *Evidence) { value.Surfaces[0].LogicalRef = value.Surfaces[1].LogicalRef },
		func(value *Evidence) { value.Surfaces = value.Surfaces[:len(value.Surfaces)-1] },
	}
	for index, mutate := range mutations {
		candidate := evidence
		candidate.Surfaces = append([]SurfaceEvidence(nil), evidence.Surfaces...)
		candidate.Optional = append([]string(nil), evidence.Optional...)
		candidate.Excluded = append([]string(nil), evidence.Excluded...)
		mutate(&candidate)
		if result := Evaluate(manifest, candidate); result.Verdict != VerdictHarnessError || result.ExitCode != ExitHarnessError {
			t.Fatalf("evidence substitution %d did not fail closed", index)
		}
	}
	encoded, _ := json.Marshal(evidence)
	encoded = bytes.Replace(encoded, []byte(`"suite_id":"phase-1-default"`), []byte(`"suite_id":"phase-1-default","suite_id":"phase-1-default"`), 1)
	if _, err := ParseEvidence(encoded); err == nil {
		t.Fatal("duplicate evidence key was accepted")
	}
}

func testSyntheticClaimCeiling(t *testing.T) {
	manifest, evidence := validSyntheticEvidence(t)
	result := Evaluate(manifest, evidence)
	for _, claim := range []string{ScopedUnchangedClaim, "whole-Mac-unchanged", "recovery-ready-on-current-host", "multi-host-verified", "fresh-install-verified"} {
		if rendered, err := RequestClaim(&evidence, result, claim); err == nil || rendered != "" {
			t.Fatalf("synthetic evidence rendered forbidden claim: %s", claim)
		}
	}
	crafted := evidence
	crafted.Provenance = "real"
	craftedResult := Evaluate(manifest, crafted)
	if craftedResult.Verdict != VerdictIndeterminate || craftedResult.ExitCode != ExitIndeterminate || craftedResult.Claim != "" || craftedResult.Reason != "real-envelope-binding-missing" {
		t.Fatal("crafted real provenance acquired claim capability")
	}
}

func testMonotonicVerdict(t *testing.T) {
	for _, primary := range []Verdict{VerdictViolation, VerdictIndeterminate, VerdictHarnessError} {
		if MonotonicCombine(primary, VerdictPassed) != primary {
			t.Fatal("after attempt improved a frozen non-pass")
		}
	}
	if MonotonicCombine(VerdictPassed, VerdictIndeterminate) != VerdictIndeterminate || MonotonicCombine(VerdictPassed, VerdictHarnessError) != VerdictHarnessError || MonotonicCombine(VerdictPassed, "unknown") != VerdictHarnessError {
		t.Fatal("after failure was masked")
	}
}

func validSyntheticEvidence(t *testing.T) (ProtectedManifest, Evidence) {
	t.Helper()
	manifest, before, after := verdictSnapshots(t)
	evidence, err := BuildEvidence(manifest, before, after, evidenceOptions(manifest, "synthetic"))
	if err != nil {
		t.Fatal("valid synthetic evidence setup failed")
	}
	return manifest, evidence
}

func verdictSnapshots(t *testing.T) (ProtectedManifest, ProtectedSnapshot, ProtectedSnapshot) {
	t.Helper()
	root := t.TempDir()
	resolver, err := PrepareProtectedSynthetic(root)
	if err != nil {
		t.Fatal("verdict fixture unavailable")
	}
	manifest := loadProtectedManifest(t)
	frozen, err := FreezeProtectedManifest(manifest)
	if err != nil {
		t.Fatal("verdict manifest did not freeze")
	}
	key := bytes.Repeat([]byte{0x19}, 32)
	before, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{})
	if err != nil || !allComplete(before) {
		t.Fatal("verdict before snapshot failed")
	}
	after, err := SnapshotProtected(frozen, manifest, resolver, key, SnapshotOptions{})
	if err != nil || !allComplete(after) {
		t.Fatal("verdict after snapshot failed")
	}
	return manifest, before, after
}

func evidenceOptions(manifest ProtectedManifest, provenance string) EvidenceOptions {
	opened := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	return EvidenceOptions{SuiteID: manifest.SuiteID, Tier: "offline-static", WindowID: "synthetic-window-01", OpenedAt: opened, ClosedAt: opened.Add(time.Second), Provenance: provenance}
}

func beforeWithDigest(snapshot ProtectedSnapshot, digest string) ProtectedSnapshot {
	snapshot.ManifestDigest = digest
	return snapshot
}

func opaqueToken(fill byte) string {
	return "hmac-sha256:" + strings.Repeat(hex.EncodeToString([]byte{fill}), 32)
}
