package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.invalid/yamc/safety/internal/artifact"
)

const freshObservedKey = "fresh-observed-state"

type lineageCaseFile struct {
	SchemaVersion string        `json:"schema_version"`
	Cases         []lineageCase `json:"cases"`
}

type lineageCase struct {
	Name  string               `json:"name"`
	Mode  artifact.LineageMode `json:"mode"`
	Valid bool                 `json:"valid"`
}

type graphBundle struct {
	graph                  artifact.LineageGraph
	canonical              map[artifact.Kind][]byte
	envelopes              map[artifact.Kind]artifact.Envelope
	freshObservedCanonical []byte
	freshObservedEnvelope  artifact.Envelope
	createdAt              time.Time
}

func TestArtifactLineage(t *testing.T) {
	apply := buildApplyBundle(t, "synthetic-run-apply", []string{"fixture.operation.first"}, []string{"fixture.operation.first"})
	readOnly := buildReadOnlyBundle(t, "synthetic-run-read-only")
	assertLineageCases(t, apply, readOnly)
	assertStoreLifecycle(t, apply)
	assertArtifactCLI(t, apply, readOnly)
	assertLineageRunnerContract(t)
}

func assertLineageCases(t *testing.T, apply, readOnly graphBundle) {
	t.Helper()
	cases := loadLineageCases(t)
	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			graph := cloneGraph(apply.graph)
			mode := testCase.Mode
			switch testCase.Name {
			case "valid-apply":
			case "valid-read-only":
				graph = cloneGraph(readOnly.graph)
			case "invalid-wrong-kind":
				graph.Desired = bytes.Clone(graph.Observed)
			case "invalid-substituted-digest":
				substitute := buildDesiredOnly(t, "synthetic-run-substitute", apply.createdAt, "repo:synthetic/substitute")
				graph.Desired = substitute
			case "invalid-stale-edge":
				stale := buildDesiredOnly(t, "synthetic-run-apply", apply.createdAt.Add(time.Minute), "repo:synthetic/config")
				graph.Desired = stale
			case "invalid-missing-edge":
				graph.Receipt = nil
			case "invalid-missing-fresh-observation":
				graph.FreshObserved = nil
			case "invalid-extra-edge":
				graph = cloneGraph(readOnly.graph)
				graph.Receipt = bytes.Clone(apply.graph.Receipt)
			case "invalid-substituted-fresh-observation":
				graph.FreshObserved = bytes.Clone(readOnly.graph.Observed)
			case "invalid-reordered-edge":
				reordered := buildApplyBundle(t, "synthetic-run-reordered", []string{"fixture.operation.first", "fixture.operation.second"}, []string{"fixture.operation.second", "fixture.operation.first"})
				graph = reordered.graph
			case "invalid-reused-pre-apply-observation":
				graph.Evidence = buildReusedObservationEvidence(t, apply)
			case "invalid-missing-postconditions":
				graph.Evidence = mutateCanonical(t, graph.Evidence, func(value map[string]any) {
					delete(value["payload"].(map[string]any), "expected_postconditions_digest")
				})
			case "invalid-latest-selection":
				graph.Plan = mutateCanonical(t, graph.Plan, func(value map[string]any) {
					value["payload"].(map[string]any)["desired_digest"] = "latest"
				})
			case "invalid-run-id-only-repair":
				graph.Desired = buildDesiredOnly(t, "synthetic-run-changed-only", apply.createdAt, "repo:synthetic/config")
			case "invalid-report-extra-evidence":
				graph.Report = buildExtraEvidenceReport(t, apply)
			default:
				t.Fatalf("unknown synthetic lineage case")
			}
			err := artifact.ValidateLineage(mode, graph)
			if testCase.Valid && err != nil {
				t.Fatalf("valid lineage rejected")
			}
			if !testCase.Valid && err == nil {
				t.Fatalf("invalid lineage accepted")
			}
		})
	}
}

func assertStoreLifecycle(t *testing.T, apply graphBundle) {
	t.Helper()
	_, repositoryRoot := projectRoots(t)
	now := apply.createdAt
	clock := func() time.Time { return now }
	storeRoot := filepath.Join(t.TempDir(), "store")
	store, err := artifact.NewStoreWithClock(storeRoot, repositoryRoot, clock)
	if err != nil {
		t.Fatalf("external store setup failed")
	}
	digests, err := store.WriteGraph(artifact.LineageApply, apply.graph)
	if err != nil || len(digests) != 7 {
		t.Fatalf("valid apply graph store failed")
	}
	expectedKinds := map[string]artifact.Kind{
		string(artifact.DesiredState):         artifact.DesiredState,
		string(artifact.ObservedState):        artifact.ObservedState,
		freshObservedKey:                      artifact.ObservedState,
		string(artifact.GeneratedPlan):        artifact.GeneratedPlan,
		string(artifact.AppliedReceipt):       artifact.AppliedReceipt,
		string(artifact.VerificationEvidence): artifact.VerificationEvidence,
		string(artifact.ReadinessReport):      artifact.ReadinessReport,
	}
	for label, digest := range digests {
		canonical, envelope, readErr := store.Read(digest)
		if readErr != nil || envelope.Kind != expectedKinds[label] || len(canonical) == 0 {
			t.Fatalf("exact store read failed")
		}
	}
	if digests[freshObservedKey] != apply.freshObservedEnvelope.ContentDigest {
		t.Fatalf("fresh observed artifact was not stored under its explicit graph key")
	}
	if _, fresh, err := store.Read(apply.freshObservedEnvelope.ContentDigest); err != nil || !exactStrings(fresh.Provenance.InputDigests, apply.envelopes[artifact.AppliedReceipt].ContentDigest) {
		t.Fatalf("fresh observed artifact did not retain exact receipt provenance")
	}

	planDigest := apply.envelopes[artifact.GeneratedPlan].ContentDigest
	planPath := objectPath(storeRoot, planDigest)
	markerTime := time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(planPath, markerTime, markerTime); err != nil {
		t.Fatalf("duplicate no-op setup failed")
	}
	if digest, err := store.Write(apply.canonical[artifact.GeneratedPlan]); err != nil || digest != planDigest {
		t.Fatalf("exact duplicate write rejected")
	}
	if info, err := os.Stat(planPath); err != nil || !info.ModTime().Equal(markerTime) {
		t.Fatalf("exact duplicate rewrote immutable bytes")
	}
	if err := store.Delete(planDigest); err == nil {
		t.Fatalf("nonterminal plan deletion accepted")
	}

	receiptDigest := apply.envelopes[artifact.AppliedReceipt].ContentDigest
	if err := store.TransitionPlan(planDigest, artifact.TerminalApplied, strings.Repeat("f", 71)); err == nil {
		t.Fatalf("substituted applied transition accepted")
	}
	if err := store.TransitionPlan(planDigest, artifact.TerminalApplied, receiptDigest); err != nil {
		t.Fatalf("exact applied transition rejected")
	}
	if err := store.Delete(planDigest); err == nil {
		t.Fatalf("evidence-pinned plan deletion accepted")
	}
	for _, kind := range []artifact.Kind{artifact.AppliedReceipt, artifact.VerificationEvidence, artifact.ReadinessReport} {
		if err := store.Delete(apply.envelopes[kind].ContentDigest); err == nil {
			t.Fatalf("evidence-bundle deletion accepted")
		}
	}
	now = now.Add(25 * time.Hour)
	snapshotDigests := []string{
		apply.envelopes[artifact.DesiredState].ContentDigest,
		apply.envelopes[artifact.ObservedState].ContentDigest,
		apply.freshObservedEnvelope.ContentDigest,
	}
	for _, digest := range snapshotDigests {
		if _, _, err := store.Read(digest); err != nil {
			t.Fatalf("transitively pinned snapshot expired")
		}
		if err := store.Delete(digest); err == nil {
			t.Fatalf("pinned snapshot deletion accepted")
		}
	}

	reopened, err := artifact.NewStoreWithClock(storeRoot, repositoryRoot, clock)
	if err != nil {
		t.Fatalf("persisted store reopen failed")
	}
	for _, digest := range snapshotDigests {
		if _, _, err := reopened.Read(digest); err != nil {
			t.Fatalf("reopened store lost transitive snapshot pin")
		}
		if err := reopened.Delete(digest); err == nil {
			t.Fatalf("reopened store accepted deletion of a pinned snapshot")
		}
	}
	if err := reopened.TransitionPlan(planDigest, artifact.TerminalAbandoned, abandonmentRecordDigest(t, planDigest)); err == nil {
		t.Fatalf("reopened store lost the applied terminal transition")
	}

	latestPath := filepath.Join(storeRoot, "sha256", "latest")
	if err := os.WriteFile(latestPath, apply.canonical[artifact.DesiredState], 0o600); err != nil {
		t.Fatalf("latest decoy setup failed")
	}
	if _, _, err := reopened.Read("latest"); err == nil {
		t.Fatalf("latest alias influenced selection")
	}
	if _, _, err := reopened.Read(apply.envelopes[artifact.DesiredState].ContentDigest); err != nil {
		t.Fatalf("exact digest read was displaced by decoy")
	}

	assertExpiredAndReleasedPins(t, apply, repositoryRoot)
	assertPreTerminalPlanWriteRejected(t, apply, repositoryRoot)
	assertTransitionAppliedValidatesReceiptLineage(t, apply, repositoryRoot)
	assertLateGraphCollisionIsAtomic(t, apply, repositoryRoot)
	assertStoreTamperAndCollision(t, apply, repositoryRoot)
	assertWrongPolicyWrites(t, apply, repositoryRoot)
}

func assertExpiredAndReleasedPins(t *testing.T, apply graphBundle, repositoryRoot string) {
	t.Helper()
	now := apply.createdAt
	storeRoot := filepath.Join(t.TempDir(), "release-store")
	store, err := artifact.NewStoreWithClock(storeRoot, repositoryRoot, func() time.Time { return now })
	if err != nil {
		t.Fatalf("release-store setup failed")
	}
	for _, kind := range []artifact.Kind{artifact.DesiredState, artifact.ObservedState, artifact.GeneratedPlan} {
		if _, err := store.Write(apply.canonical[kind]); err != nil {
			t.Fatalf("plan ancestry write failed")
		}
	}
	now = now.Add(25 * time.Hour)
	desiredDigest := apply.envelopes[artifact.DesiredState].ContentDigest
	if _, _, err := store.Read(desiredDigest); err != nil {
		t.Fatalf("nonterminal plan did not pin ancestor")
	}
	planDigest := apply.envelopes[artifact.GeneratedPlan].ContentDigest
	randomRecord := "sha256:" + strings.Repeat("a", 64)
	if randomRecord == abandonmentRecordDigest(t, planDigest) {
		randomRecord = "sha256:" + strings.Repeat("b", 64)
	}
	if err := store.TransitionPlan(planDigest, artifact.TerminalAbandoned, randomRecord); err == nil {
		t.Fatalf("arbitrary abandonment record digest accepted")
	}
	if err := store.TransitionPlan(planDigest, artifact.TerminalAbandoned, abandonmentRecordDigest(t, planDigest)); err != nil {
		t.Fatalf("explicit abandoned transition rejected")
	}
	reopened, err := artifact.NewStoreWithClock(storeRoot, repositoryRoot, func() time.Time { return now })
	if err != nil {
		t.Fatalf("abandoned store reopen failed")
	}
	if err := reopened.Delete(planDigest); err != nil {
		t.Fatalf("terminal unpinned plan deletion rejected")
	}
	if _, _, err := reopened.Read(desiredDigest); err == nil {
		t.Fatalf("expired unpinned snapshot remained usable")
	}
	if err := reopened.Delete(desiredDigest); err != nil {
		t.Fatalf("expired unpinned snapshot deletion rejected")
	}
}

func assertPreTerminalPlanWriteRejected(t *testing.T, apply graphBundle, repositoryRoot string) {
	t.Helper()
	storeRoot := filepath.Join(t.TempDir(), "pre-terminal-store")
	store, err := artifact.NewStoreWithClock(storeRoot, repositoryRoot, func() time.Time { return apply.createdAt })
	if err != nil {
		t.Fatalf("pre-terminal store setup failed")
	}
	for _, kind := range []artifact.Kind{artifact.DesiredState, artifact.ObservedState} {
		if _, err := store.Write(apply.canonical[kind]); err != nil {
			t.Fatalf("pre-terminal ancestry setup failed")
		}
	}
	var payload artifact.GeneratedPlanPayload
	if err := json.Unmarshal(apply.envelopes[artifact.GeneratedPlan].Payload, &payload); err != nil {
		t.Fatalf("pre-terminal plan payload setup failed")
	}
	options, err := artifact.DefaultBuildOptions(artifact.GeneratedPlan, apply.createdAt)
	if err != nil {
		t.Fatalf("pre-terminal lifecycle setup failed")
	}
	options.Lifecycle.TerminalState = artifact.TerminalAbandoned
	options.Lifecycle.AbandonmentRecordDigest = "sha256:" + strings.Repeat("c", 64)
	canonical, envelope, err := artifact.NewWithOptions(
		artifact.GeneratedPlan,
		apply.envelopes[artifact.GeneratedPlan].Run,
		artifact.Provenance{Mode: "synthetic", InputDigests: []string{payload.DesiredDigest, payload.ObservedDigest}},
		payload,
		options,
	)
	if err != nil {
		t.Fatalf("pre-terminal plan construction failed")
	}
	if _, err := store.Write(canonical); err == nil {
		t.Fatalf("plan carrying a caller-selected terminal state was accepted")
	}
	if _, err := os.Stat(objectPath(storeRoot, envelope.ContentDigest)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected pre-terminal plan reached the object store")
	}
}

func assertTransitionAppliedValidatesReceiptLineage(t *testing.T, apply graphBundle, repositoryRoot string) {
	t.Helper()
	storeRoot := filepath.Join(t.TempDir(), "transition-store")
	store, err := artifact.NewStoreWithClock(storeRoot, repositoryRoot, func() time.Time { return apply.createdAt })
	if err != nil {
		t.Fatalf("transition store setup failed")
	}
	for _, kind := range []artifact.Kind{artifact.DesiredState, artifact.ObservedState, artifact.GeneratedPlan} {
		if _, err := store.Write(apply.canonical[kind]); err != nil {
			t.Fatalf("transition ancestry setup failed")
		}
	}
	planDigest := apply.envelopes[artifact.GeneratedPlan].ContentDigest
	invalidCanonical, invalidReceipt := makeArtifact(
		t,
		artifact.AppliedReceipt,
		apply.envelopes[artifact.AppliedReceipt].Run,
		[]string{planDigest},
		artifact.AppliedReceiptPayload{
			PlanDigest:   planDigest,
			Mode:         "synthetic",
			OperationIDs: []string{"fixture.operation.not-in-plan"},
			Outcome:      "fixture:outcome/completed",
		},
		apply.createdAt,
	)
	if err := os.WriteFile(objectPath(storeRoot, invalidReceipt.ContentDigest), invalidCanonical, 0o600); err != nil {
		t.Fatalf("invalid receipt injection setup failed")
	}
	if err := store.TransitionPlan(planDigest, artifact.TerminalApplied, invalidReceipt.ContentDigest); err == nil {
		t.Fatalf("applied transition accepted a receipt without complete plan lineage")
	}
	if _, err := store.Write(apply.canonical[artifact.AppliedReceipt]); err != nil {
		t.Fatalf("valid transition receipt setup failed")
	}
	if err := store.TransitionPlan(planDigest, artifact.TerminalApplied, apply.envelopes[artifact.AppliedReceipt].ContentDigest); err != nil {
		t.Fatalf("valid applied transition rejected after negative lineage check")
	}
}

func assertLateGraphCollisionIsAtomic(t *testing.T, apply graphBundle, repositoryRoot string) {
	t.Helper()
	storeRoot := filepath.Join(t.TempDir(), "late-collision-store")
	store, err := artifact.NewStoreWithClock(storeRoot, repositoryRoot, func() time.Time { return apply.createdAt })
	if err != nil {
		t.Fatalf("late-collision store setup failed")
	}
	objectRoot := filepath.Join(storeRoot, "sha256")
	if err := os.MkdirAll(objectRoot, 0o700); err != nil {
		t.Fatalf("late-collision object root setup failed")
	}
	reportDigest := apply.envelopes[artifact.ReadinessReport].ContentDigest
	decoy := apply.canonical[artifact.DesiredState]
	if err := os.WriteFile(objectPath(storeRoot, reportDigest), decoy, 0o600); err != nil {
		t.Fatalf("late-collision decoy setup failed")
	}
	if _, err := store.WriteGraph(artifact.LineageApply, apply.graph); err == nil {
		t.Fatalf("late graph collision unexpectedly succeeded")
	}
	entries, err := os.ReadDir(objectRoot)
	if err != nil || len(entries) != 1 || entries[0].Name() != strings.TrimPrefix(reportDigest, "sha256:") {
		t.Fatalf("failed graph left partially written objects")
	}
	storedDecoy, err := os.ReadFile(objectPath(storeRoot, reportDigest))
	if err != nil || !bytes.Equal(storedDecoy, decoy) {
		t.Fatalf("collision preflight modified the existing object")
	}
}

func assertStoreTamperAndCollision(t *testing.T, apply graphBundle, repositoryRoot string) {
	t.Helper()
	clock := func() time.Time { return apply.createdAt }
	tamperRoot := filepath.Join(t.TempDir(), "tamper-store")
	tamperStore, _ := artifact.NewStoreWithClock(tamperRoot, repositoryRoot, clock)
	if _, err := tamperStore.WriteGraph(artifact.LineageApply, apply.graph); err != nil {
		t.Fatalf("tamper-store setup failed")
	}
	desiredDigest := apply.envelopes[artifact.DesiredState].ContentDigest
	if err := os.WriteFile(objectPath(tamperRoot, desiredDigest), []byte(`{"tampered":true}`), 0o600); err != nil {
		t.Fatalf("tamper setup failed")
	}
	if _, _, err := tamperStore.Read(desiredDigest); err == nil {
		t.Fatalf("store key/content mismatch accepted")
	}

	collisionRoot := filepath.Join(t.TempDir(), "collision-store")
	collisionStore, _ := artifact.NewStoreWithClock(collisionRoot, repositoryRoot, clock)
	for _, kind := range []artifact.Kind{artifact.DesiredState, artifact.ObservedState} {
		if _, err := collisionStore.Write(apply.canonical[kind]); err != nil {
			t.Fatalf("collision ancestry setup failed")
		}
	}
	planDigest := apply.envelopes[artifact.GeneratedPlan].ContentDigest
	planPath := objectPath(collisionRoot, planDigest)
	if err := os.WriteFile(planPath, []byte(`{"different":true}`), 0o600); err != nil {
		t.Fatalf("collision setup failed")
	}
	if _, err := collisionStore.Write(apply.canonical[artifact.GeneratedPlan]); err == nil {
		t.Fatalf("different-byte overwrite accepted")
	}
}

func assertWrongPolicyWrites(t *testing.T, apply graphBundle, repositoryRoot string) {
	t.Helper()
	storeRoot := filepath.Join(t.TempDir(), "policy-store")
	store, _ := artifact.NewStoreWithClock(storeRoot, repositoryRoot, func() time.Time { return apply.createdAt })
	options, _ := artifact.DefaultBuildOptions(artifact.DesiredState, apply.createdAt)
	options.StorageClass = artifact.SyntheticGolden
	golden, _, err := artifact.NewWithOptions(
		artifact.DesiredState,
		apply.envelopes[artifact.DesiredState].Run,
		artifact.Provenance{Mode: "synthetic", InputDigests: []string{}},
		artifact.DesiredPayload{Profile: "profile:synthetic-developer", Declarations: []artifact.Fact{{Ref: "repo:synthetic/config", State: "fixture:state/declared"}}},
		options,
	)
	if err != nil {
		t.Fatalf("synthetic golden setup failed")
	}
	if _, err := store.Write(golden); err == nil {
		t.Fatalf("read-only synthetic golden write accepted")
	}
	wrongRetention := mutateCanonical(t, apply.canonical[artifact.DesiredState], func(value map[string]any) {
		value["provenance"].(map[string]any)["lifecycle"].(map[string]any)["retention"] = "caller-defined"
	})
	if _, err := store.Write(wrongRetention); err == nil {
		t.Fatalf("unsupported retention write accepted")
	}
}

func assertArtifactCLI(t *testing.T, apply, readOnly graphBundle) {
	t.Helper()
	safetyRoot, repositoryRoot := projectRoots(t)
	fixtureRoot := t.TempDir()
	applyFiles := writeBundleFiles(t, filepath.Join(fixtureRoot, "apply"), apply)
	readOnlyFiles := writeBundleFiles(t, filepath.Join(fixtureRoot, "read-only"), readOnly)

	stdout, stderr, err := runCLI(safetyRoot, "validate", "--expect-kind", string(artifact.DesiredState), "--artifact", applyFiles[string(artifact.DesiredState)])
	if err != nil || len(stderr) != 0 {
		t.Fatalf("valid CLI artifact validation failed")
	}
	assertBoundedAndPrivate(t, stdout, stderr, repositoryRoot, fixtureRoot)
	var validation struct {
		Status string        `json:"status"`
		Kind   artifact.Kind `json:"kind"`
		Digest string        `json:"digest"`
	}
	decodeStrict(t, stdout, &validation)
	if validation.Status != "valid" || validation.Kind != artifact.DesiredState || validation.Digest != apply.envelopes[artifact.DesiredState].ContentDigest {
		t.Fatalf("bounded validate output rejected")
	}
	if _, _, err := runCLI(safetyRoot, "validate", "--expect-kind", string(artifact.ObservedState), "--artifact", applyFiles[string(artifact.DesiredState)]); err == nil {
		t.Fatalf("CLI expected-kind substitution accepted")
	}

	applyStore := filepath.Join(fixtureRoot, "apply-store")
	stdout, stderr, err = runCLI(safetyRoot, storeCLIArguments(artifact.LineageApply, applyStore, repositoryRoot, applyFiles)...)
	if err != nil {
		t.Fatalf("valid apply CLI store failed")
	}
	assertBoundedAndPrivate(t, stdout, stderr, repositoryRoot, fixtureRoot)
	assertObjectCount(t, applyStore, 7)

	readOnlyStore := filepath.Join(fixtureRoot, "read-only-store")
	if _, _, err := runCLI(safetyRoot, storeCLIArguments(artifact.LineageReadOnly, readOnlyStore, repositoryRoot, readOnlyFiles)...); err != nil {
		t.Fatalf("valid read-only CLI store failed")
	}
	assertObjectCount(t, readOnlyStore, 4)

	invalidFiles := writeBundleFiles(t, filepath.Join(fixtureRoot, "invalid"), apply)
	substitute := buildDesiredOnly(t, "synthetic-run-cli-substitute", apply.createdAt, "repo:synthetic/substitute")
	if err := os.WriteFile(invalidFiles[string(artifact.DesiredState)], substitute, 0o600); err != nil {
		t.Fatalf("CLI substitution setup failed")
	}
	invalidStore := filepath.Join(fixtureRoot, "invalid-store")
	if _, _, err := runCLI(safetyRoot, storeCLIArguments(artifact.LineageApply, invalidStore, repositoryRoot, invalidFiles)...); err == nil {
		t.Fatalf("invalid CLI graph unexpectedly stored")
	}
	assertNoStoreObjects(t, invalidStore)

	missingFreshStore := filepath.Join(fixtureRoot, "missing-fresh-store")
	missingFreshArgs := storeCLIArguments(artifact.LineageApply, missingFreshStore, repositoryRoot, applyFiles)
	for index := range missingFreshArgs {
		if missingFreshArgs[index] == "--fresh-observed" {
			missingFreshArgs = append(missingFreshArgs[:index], missingFreshArgs[index+2:]...)
			break
		}
	}
	if _, _, err := runCLI(safetyRoot, missingFreshArgs...); err == nil {
		t.Fatalf("apply CLI accepted a graph without explicit fresh observation")
	}
	assertNoStoreObjects(t, missingFreshStore)

	latestDirectory := filepath.Join(fixtureRoot, "latest-directory")
	if err := os.MkdirAll(latestDirectory, 0o700); err != nil {
		t.Fatalf("latest directory setup failed")
	}
	latestArgs := storeCLIArguments(artifact.LineageApply, filepath.Join(fixtureRoot, "latest-store"), repositoryRoot, applyFiles)
	for index := range latestArgs {
		if latestArgs[index] == "--plan" {
			latestArgs[index+1] = latestDirectory
		}
	}
	if _, _, err := runCLI(safetyRoot, latestArgs...); err == nil {
		t.Fatalf("CLI directory/latest discovery fallback accepted")
	}
	assertNoStoreObjects(t, filepath.Join(fixtureRoot, "latest-store"))
	if _, _, err := runCLI(safetyRoot, "store", "--future-route"); err == nil {
		t.Fatalf("unknown CLI store route accepted")
	}
}

func assertLineageRunnerContract(t *testing.T) {
	t.Helper()
	_, repositoryRoot := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(repositoryRoot, "safety", "scripts", "test.sh"))
	if err != nil {
		t.Fatalf("runner unavailable")
	}
	text := string(data)
	for _, literal := range []string{"task:artifact-lineage", "'./internal/e2e'", "'^TestArtifactLineage$'", "wave:artifact-contracts", "task artifact-kinds", "task artifact-lineage"} {
		if !strings.Contains(text, literal) {
			t.Fatalf("lineage runner route is incomplete")
		}
	}
	for _, future := range []string{"task:privacy-boundary", "task:bounded-capture", "wave:privacy"} {
		if strings.Contains(text, future) {
			t.Fatalf("Phase 3 runner route registered early")
		}
	}
}

func buildApplyBundle(t *testing.T, runID string, planOperations, receiptOperations []string) graphBundle {
	t.Helper()
	createdAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	run := artifact.RunMetadata{RunID: runID, Tier: "offline-static", SuiteID: "artifact-lineage"}
	bundle := newGraphBundle(createdAt)
	desiredPayload := artifact.DesiredPayload{Profile: "profile:synthetic-developer", Declarations: []artifact.Fact{{Ref: "repo:synthetic/config", State: "fixture:state/declared"}}}
	desiredCanonical, desired := makeArtifact(t, artifact.DesiredState, run, nil, desiredPayload, createdAt)
	observedPayload := artifact.ObservedPayload{Scope: "fixture:scope/synthetic", Facts: []artifact.Fact{{Ref: "fixture:fact/synthetic", State: "fixture:state/observed"}}}
	observedCanonical, observed := makeArtifact(t, artifact.ObservedState, run, nil, observedPayload, createdAt)
	expectedDigest, _ := artifact.DigestValue([]artifact.Fact{{Ref: "fixture:fact/synthetic", State: "fixture:state/applied"}})
	planPayload := artifact.GeneratedPlanPayload{DesiredDigest: desired.ContentDigest, ObservedDigest: observed.ContentDigest, ExpectedPostconditionsDigest: expectedDigest, OperationIDs: planOperations}
	planCanonical, plan := makeArtifact(t, artifact.GeneratedPlan, run, []string{desired.ContentDigest, observed.ContentDigest}, planPayload, createdAt)
	receiptPayload := artifact.AppliedReceiptPayload{PlanDigest: plan.ContentDigest, Mode: "synthetic", OperationIDs: receiptOperations, Outcome: "fixture:outcome/completed"}
	receiptCanonical, receipt := makeArtifact(t, artifact.AppliedReceipt, run, []string{plan.ContentDigest}, receiptPayload, createdAt)
	freshObservedPayload := artifact.ObservedPayload{
		Scope: "fixture:scope/synthetic",
		Facts: []artifact.Fact{{Ref: "fixture:fact/synthetic", State: "fixture:state/fresh"}},
	}
	freshObservedCanonical, freshObserved := makeArtifact(t, artifact.ObservedState, run, []string{receipt.ContentDigest}, freshObservedPayload, createdAt)
	evidencePayload := artifact.VerificationEvidencePayload{
		PlanDigest:                   plan.ContentDigest,
		ReceiptDigest:                receipt.ContentDigest,
		ExpectedPostconditionsDigest: expectedDigest,
		FreshObservedDigest:          freshObserved.ContentDigest,
		FreshObserved: artifact.FreshObserved{
			Scope:               freshObservedPayload.Scope,
			State:               "fixture:state/fresh",
			SourceReceiptDigest: receipt.ContentDigest,
			ContentDigest:       freshObserved.ContentDigest,
		},
	}
	evidenceCanonical, evidence := makeArtifact(t, artifact.VerificationEvidence, run, []string{plan.ContentDigest, receipt.ContentDigest, expectedDigest, freshObserved.ContentDigest}, evidencePayload, createdAt)
	reportPayload := artifact.ReadinessReportPayload{EvidenceDigest: evidence.ContentDigest, State: "synthetic-sentinel-passed"}
	reportCanonical, report := makeArtifact(t, artifact.ReadinessReport, run, []string{evidence.ContentDigest}, reportPayload, createdAt)
	bundle.add(artifact.DesiredState, desiredCanonical, desired)
	bundle.add(artifact.ObservedState, observedCanonical, observed)
	bundle.add(artifact.GeneratedPlan, planCanonical, plan)
	bundle.add(artifact.AppliedReceipt, receiptCanonical, receipt)
	bundle.addFreshObserved(freshObservedCanonical, freshObserved)
	bundle.add(artifact.VerificationEvidence, evidenceCanonical, evidence)
	bundle.add(artifact.ReadinessReport, reportCanonical, report)
	bundle.graph = artifact.LineageGraph{Desired: desiredCanonical, Observed: observedCanonical, Plan: planCanonical, Receipt: receiptCanonical, FreshObserved: freshObservedCanonical, Evidence: evidenceCanonical, Report: reportCanonical, ExpectedPostconditionsDigest: expectedDigest}
	return bundle
}

func buildReadOnlyBundle(t *testing.T, runID string) graphBundle {
	t.Helper()
	createdAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	run := artifact.RunMetadata{RunID: runID, Tier: "offline-static", SuiteID: "artifact-lineage-read-only"}
	bundle := newGraphBundle(createdAt)
	desiredCanonical, desired := makeArtifact(t, artifact.DesiredState, run, nil, artifact.DesiredPayload{Profile: "profile:synthetic-developer", Declarations: []artifact.Fact{{Ref: "repo:synthetic/config", State: "fixture:state/declared"}}}, createdAt)
	observedPayload := artifact.ObservedPayload{Scope: "fixture:scope/read-only", Facts: []artifact.Fact{{Ref: "fixture:fact/read-only", State: "fixture:state/fresh"}}}
	observedCanonical, observed := makeArtifact(t, artifact.ObservedState, run, nil, observedPayload, createdAt)
	expectedDigest, _ := artifact.DigestValue([]artifact.Fact{{Ref: "fixture:fact/read-only", State: "fixture:state/expected"}})
	evidencePayload := artifact.VerificationEvidencePayload{
		DesiredDigest:                desired.ContentDigest,
		ExpectedPostconditionsDigest: expectedDigest,
		FreshObservedDigest:          observed.ContentDigest,
		FreshObserved:                artifact.FreshObserved{Scope: observedPayload.Scope, State: "fixture:state/fresh", ContentDigest: observed.ContentDigest},
	}
	evidenceCanonical, evidence := makeArtifact(t, artifact.VerificationEvidence, run, []string{desired.ContentDigest, observed.ContentDigest, expectedDigest}, evidencePayload, createdAt)
	reportCanonical, report := makeArtifact(t, artifact.ReadinessReport, run, []string{evidence.ContentDigest}, artifact.ReadinessReportPayload{EvidenceDigest: evidence.ContentDigest, State: "synthetic-sentinel-passed"}, createdAt)
	bundle.add(artifact.DesiredState, desiredCanonical, desired)
	bundle.add(artifact.ObservedState, observedCanonical, observed)
	bundle.add(artifact.VerificationEvidence, evidenceCanonical, evidence)
	bundle.add(artifact.ReadinessReport, reportCanonical, report)
	bundle.graph = artifact.LineageGraph{Desired: desiredCanonical, Observed: observedCanonical, Evidence: evidenceCanonical, Report: reportCanonical, ExpectedPostconditionsDigest: expectedDigest}
	return bundle
}

func buildDesiredOnly(t *testing.T, runID string, createdAt time.Time, ref string) []byte {
	t.Helper()
	run := artifact.RunMetadata{RunID: runID, Tier: "offline-static", SuiteID: "artifact-lineage"}
	canonical, _ := makeArtifact(t, artifact.DesiredState, run, nil, artifact.DesiredPayload{Profile: "profile:synthetic-developer", Declarations: []artifact.Fact{{Ref: ref, State: "fixture:state/declared"}}}, createdAt)
	return canonical
}

func buildReusedObservationEvidence(t *testing.T, apply graphBundle) []byte {
	t.Helper()
	run := apply.envelopes[artifact.VerificationEvidence].Run
	plan := apply.envelopes[artifact.GeneratedPlan]
	receipt := apply.envelopes[artifact.AppliedReceipt]
	observed := apply.envelopes[artifact.ObservedState]
	payload := artifact.VerificationEvidencePayload{
		PlanDigest:                   plan.ContentDigest,
		ReceiptDigest:                receipt.ContentDigest,
		ExpectedPostconditionsDigest: apply.graph.ExpectedPostconditionsDigest,
		FreshObservedDigest:          observed.ContentDigest,
		FreshObserved:                artifact.FreshObserved{Scope: "fixture:scope/synthetic", State: "fixture:state/observed", SourceReceiptDigest: receipt.ContentDigest, ContentDigest: observed.ContentDigest},
	}
	canonical, _ := makeArtifact(t, artifact.VerificationEvidence, run, []string{plan.ContentDigest, receipt.ContentDigest, apply.graph.ExpectedPostconditionsDigest, observed.ContentDigest}, payload, apply.createdAt)
	return canonical
}

func buildExtraEvidenceReport(t *testing.T, apply graphBundle) []byte {
	t.Helper()
	evidence := apply.envelopes[artifact.VerificationEvidence]
	extra := "sha256:" + strings.Repeat("9", 64)
	canonical, _ := makeArtifact(t, artifact.ReadinessReport, apply.envelopes[artifact.ReadinessReport].Run, []string{evidence.ContentDigest}, artifact.ReadinessReportPayload{EvidenceDigests: []string{evidence.ContentDigest, extra}, State: "synthetic-sentinel-passed"}, apply.createdAt)
	return canonical
}

func makeArtifact(t *testing.T, kind artifact.Kind, run artifact.RunMetadata, inputs []string, payload any, createdAt time.Time) ([]byte, artifact.Envelope) {
	t.Helper()
	if inputs == nil {
		inputs = []string{}
	}
	options, err := artifact.DefaultBuildOptions(kind, createdAt)
	if err != nil {
		t.Fatalf("artifact lifecycle setup failed")
	}
	canonical, envelope, err := artifact.NewWithOptions(kind, run, artifact.Provenance{Mode: "synthetic", InputDigests: inputs}, payload, options)
	if err != nil {
		t.Fatalf("synthetic artifact construction failed")
	}
	return canonical, envelope
}

func newGraphBundle(createdAt time.Time) graphBundle {
	return graphBundle{canonical: make(map[artifact.Kind][]byte), envelopes: make(map[artifact.Kind]artifact.Envelope), createdAt: createdAt}
}

func (bundle *graphBundle) add(kind artifact.Kind, canonical []byte, envelope artifact.Envelope) {
	bundle.canonical[kind] = canonical
	bundle.envelopes[kind] = envelope
}

func (bundle *graphBundle) addFreshObserved(canonical []byte, envelope artifact.Envelope) {
	bundle.freshObservedCanonical = canonical
	bundle.freshObservedEnvelope = envelope
}

func cloneGraph(graph artifact.LineageGraph) artifact.LineageGraph {
	return artifact.LineageGraph{
		Desired:                      bytes.Clone(graph.Desired),
		Observed:                     bytes.Clone(graph.Observed),
		Plan:                         bytes.Clone(graph.Plan),
		Receipt:                      bytes.Clone(graph.Receipt),
		FreshObserved:                bytes.Clone(graph.FreshObserved),
		Evidence:                     bytes.Clone(graph.Evidence),
		Report:                       bytes.Clone(graph.Report),
		ExpectedPostconditionsDigest: graph.ExpectedPostconditionsDigest,
	}
}

func mutateCanonical(t *testing.T, canonical []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("canonical mutation setup failed")
	}
	mutate(value)
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("canonical mutation encoding failed")
	}
	result, err := artifact.Canonicalize(encoded)
	if err != nil {
		t.Fatalf("canonical mutation rejected")
	}
	return result
}

func loadLineageCases(t *testing.T) []lineageCase {
	t.Helper()
	safetyRoot, _ := projectRoots(t)
	data, err := os.ReadFile(filepath.Join(safetyRoot, "testdata", "artifacts", "lineage-cases.json"))
	if err != nil {
		t.Fatalf("synthetic lineage cases unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var fixture lineageCaseFile
	if err := decoder.Decode(&fixture); err != nil || fixture.SchemaVersion != artifact.SchemaVersion || len(fixture.Cases) < 2 {
		t.Fatalf("synthetic lineage cases rejected")
	}
	if decoder.Decode(&struct{}{}) == nil {
		t.Fatalf("synthetic lineage cases contain trailing data")
	}
	return fixture.Cases
}

func objectPath(root, digest string) string {
	return filepath.Join(root, "sha256", strings.TrimPrefix(digest, "sha256:"))
}

func abandonmentRecordDigest(t *testing.T, planDigest string) string {
	t.Helper()
	digest, err := artifact.AbandonmentRecordDigest(planDigest)
	if err != nil {
		t.Fatalf("abandonment record digest setup failed")
	}
	return digest
}

func exactStrings(actual []string, expected ...string) bool {
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

func writeBundleFiles(t *testing.T, root string, bundle graphBundle) map[string]string {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("bundle file root unavailable")
	}
	result := make(map[string]string, len(bundle.canonical)+1)
	for kind, canonical := range bundle.canonical {
		path := filepath.Join(root, string(kind)+".json")
		if err := os.WriteFile(path, canonical, 0o600); err != nil {
			t.Fatalf("bundle artifact write failed")
		}
		result[string(kind)] = path
	}
	if len(bundle.freshObservedCanonical) != 0 {
		path := filepath.Join(root, freshObservedKey+".json")
		if err := os.WriteFile(path, bundle.freshObservedCanonical, 0o600); err != nil {
			t.Fatalf("fresh observed artifact write failed")
		}
		result[freshObservedKey] = path
	}
	return result
}

func storeCLIArguments(mode artifact.LineageMode, root, repositoryRoot string, files map[string]string) []string {
	arguments := []string{
		"store",
		"--mode", string(mode),
		"--root", root,
		"--repo-root", repositoryRoot,
		"--desired", files[string(artifact.DesiredState)],
		"--observed", files[string(artifact.ObservedState)],
		"--evidence", files[string(artifact.VerificationEvidence)],
		"--report", files[string(artifact.ReadinessReport)],
	}
	if mode == artifact.LineageApply {
		arguments = append(arguments,
			"--plan", files[string(artifact.GeneratedPlan)],
			"--receipt", files[string(artifact.AppliedReceipt)],
			"--fresh-observed", files[freshObservedKey],
		)
	}
	return arguments
}

func assertObjectCount(t *testing.T, root string, want int) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "sha256"))
	if err != nil || len(entries) != want {
		t.Fatalf("unexpected content-addressed object count")
	}
}

func assertNoStoreObjects(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "sha256"))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil || len(entries) != 0 {
		t.Fatalf("rejected graph wrote store objects")
	}
}
