package app

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAllowedOperation(t *testing.T) {
	if !allowedOperation("create") {
		t.Fatal("allowedOperation(create) = false, want true")
	}
	if allowedOperation("delete") {
		t.Fatal("allowedOperation(delete) = true, want false")
	}
}

func TestValidatePreconditionsAbsent(t *testing.T) {
	pre := PlanPreconditions{
		Absent:      true,
		KubeContext: "ctx",
	}
	if err := validatePreconditions(pre, nil, "ctx"); err != nil {
		t.Fatalf("validatePreconditions() error = %v", err)
	}
}

func TestValidatePreconditionsChanged(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetUID("uid-2")
	obj.SetResourceVersion("2")
	obj.SetGeneration(2)
	pre := PlanPreconditions{
		Absent:          false,
		UID:             "uid-1",
		ResourceVersion: "1",
		Generation:      1,
		KubeContext:     "ctx",
	}
	if err := validatePreconditions(pre, obj, "ctx"); err == nil {
		t.Fatal("validatePreconditions() error = nil, want stale_plan")
	}
}

func TestPlanTarget(t *testing.T) {
	kind, _, _, err := planTarget("satellite_configuration")
	if err != nil {
		t.Fatalf("planTarget() error = %v", err)
	}
	if kind != KindSatelliteConfiguration {
		t.Fatalf("planTarget() kind = %q, want %q", kind, KindSatelliteConfiguration)
	}
}

func TestRenderDiff(t *testing.T) {
	diff, err := renderDiff(map[string]any{"a": "b"}, map[string]any{"a": "c"})
	if err != nil {
		t.Fatalf("renderDiff() error = %v", err)
	}
	if diff == "" {
		t.Fatal("renderDiff() = empty string")
	}
}
