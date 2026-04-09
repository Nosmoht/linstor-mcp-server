package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreJobIdempotency(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	plan := PlanRecord{
		PlanID:         "plan_1",
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
		KubeContext:    "ctx",
		LinstorCluster: "cluster",
		Kind:           KindCluster,
		Name:           "homelab",
		Operation:      "reconcile",
		DesiredSpec:    map[string]any{"nodeSelector": map[string]any{"a": "b"}},
		Summary:        "summary",
		Diff:           "diff",
		Preconditions:  PlanPreconditions{KubeContext: "ctx", Absent: false},
		State:          "planned",
	}
	if err := store.SavePlan(ctx, plan); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}
	job1, existing, err := store.GetOrCreateJob(ctx, plan.PlanID, "idem")
	if err != nil {
		t.Fatalf("GetOrCreateJob() error = %v", err)
	}
	if existing {
		t.Fatal("first GetOrCreateJob() returned existing=true")
	}
	job2, existing, err := store.GetOrCreateJob(ctx, plan.PlanID, "idem")
	if err != nil {
		t.Fatalf("GetOrCreateJob() second error = %v", err)
	}
	if !existing {
		t.Fatal("second GetOrCreateJob() returned existing=false")
	}
	if job1.JobID != job2.JobID {
		t.Fatalf("job IDs differ: %s != %s", job1.JobID, job2.JobID)
	}
}

func TestStorePlanRoundTrip(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	plan := PlanRecord{
		PlanID:         "plan_roundtrip",
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
		KubeContext:    "ctx",
		LinstorCluster: "cluster",
		Kind:           KindSatelliteConfiguration,
		Name:           "object",
		Operation:      "create",
		DesiredSpec:    map[string]any{"storagePools": []any{}},
		Summary:        "summary",
		Diff:           "diff",
		Destructive:    false,
		Preconditions:  PlanPreconditions{Absent: true, KubeContext: "ctx"},
		State:          "planned",
	}
	if err := store.SavePlan(ctx, plan); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}
	got, err := store.GetPlan(ctx, plan.PlanID)
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetPlan() = nil, want plan")
	}
	if got.Kind != plan.Kind || got.Name != plan.Name || got.Operation != plan.Operation {
		t.Fatalf("GetPlan() roundtrip mismatch: %+v", got)
	}
}

func TestStoreUpdateJob(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	plan := PlanRecord{
		PlanID:         "plan_update_job",
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
		KubeContext:    "ctx",
		LinstorCluster: "cluster",
		Kind:           KindCluster,
		Name:           "homelab",
		Operation:      "reconcile",
		DesiredSpec:    map[string]any{"nodeSelector": map[string]any{"a": "b"}},
		Summary:        "summary",
		Diff:           "diff",
		Preconditions:  PlanPreconditions{KubeContext: "ctx", Absent: false},
		State:          "planned",
	}
	if err := store.SavePlan(ctx, plan); err != nil {
		t.Fatalf("SavePlan() error = %v", err)
	}
	job, _, err := store.GetOrCreateJob(ctx, plan.PlanID, "idem")
	if err != nil {
		t.Fatalf("GetOrCreateJob() error = %v", err)
	}
	job.Phase = "succeeded"
	job.ResultRef = "linstor://jobs/" + job.JobID
	if err := store.UpdateJob(ctx, *job); err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}
	got, err := store.GetJob(ctx, job.JobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if got == nil || got.Phase != "succeeded" {
		t.Fatalf("GetJob() = %+v, want succeeded job", got)
	}
}
