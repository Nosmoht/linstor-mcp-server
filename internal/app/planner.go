package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ntbc/linstor-mcp-server/internal/config"
	"github.com/ntbc/linstor-mcp-server/internal/kube"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Planner struct {
	cfg   config.Config
	kube  *kube.Client
	store *Store
}

type PlanClusterConfigInput struct {
	Kind      string         `json:"kind" jsonschema:"cluster config kind: cluster, satellite_configuration, or node_connection"`
	Name      string         `json:"name" jsonschema:"cluster-scoped object name"`
	Operation string         `json:"operation" jsonschema:"operation: create, update, or reconcile"`
	Spec      map[string]any `json:"spec" jsonschema:"desired spec object"`
}

type PlanClusterConfigOutput struct {
	Plan PlanRecord `json:"plan"`
}

type ApplyPlanInput struct {
	PlanID                 string `json:"plan_id" jsonschema:"plan ID from plan_cluster_config"`
	IdempotencyKey         string `json:"idempotency_key" jsonschema:"client-generated retry key"`
	AcknowledgeDestructive bool   `json:"acknowledge_destructive,omitempty" jsonschema:"must be true for destructive plans"`
}

type ApplyPlanOutput struct {
	Job JobRecord `json:"job"`
}

type JobInput struct {
	JobID string `json:"job_id" jsonschema:"job ID"`
}

type JobOutput struct {
	Job JobRecord `json:"job"`
}

func NewPlanner(cfg config.Config, kubeClient *kube.Client, store *Store) *Planner {
	return &Planner{cfg: cfg, kube: kubeClient, store: store}
}

func (p *Planner) PlanClusterConfig(ctx context.Context, in PlanClusterConfigInput) (PlanClusterConfigOutput, error) {
	kind, gvr, kindName, err := planTarget(strings.ToLower(strings.TrimSpace(in.Kind)))
	if err != nil {
		return PlanClusterConfigOutput{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return PlanClusterConfigOutput{}, fmt.Errorf("validation_error: name is required")
	}
	if len(in.Spec) == 0 {
		return PlanClusterConfigOutput{}, fmt.Errorf("validation_error: spec is required")
	}
	if !allowedOperation(in.Operation) {
		return PlanClusterConfigOutput{}, fmt.Errorf("validation_error: unsupported operation %q", in.Operation)
	}

	existing, err := p.kube.GetClusterScoped(ctx, gvr, in.Name)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return PlanClusterConfigOutput{}, err
	}
	pre := PlanPreconditions{
		Absent:      existing == nil || existing.GetName() == "",
		KubeContext: p.kube.CurrentCtx,
	}
	if existing != nil && existing.GetName() != "" {
		pre.UID = string(existing.GetUID())
		pre.ResourceVersion = existing.GetResourceVersion()
		pre.Generation = existing.GetGeneration()
	}

	currentSpec := map[string]any{}
	if existing != nil {
		currentSpec, _, _ = unstructured.NestedMap(existing.Object, "spec")
	}
	diff, err := renderDiff(currentSpec, in.Spec)
	if err != nil {
		return PlanClusterConfigOutput{}, err
	}

	plan := PlanRecord{
		PlanID:         newID("plan"),
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(5 * time.Minute),
		KubeContext:    p.kube.CurrentCtx,
		LinstorCluster: p.cfg.LinstorCluster,
		Kind:           kind,
		Name:           in.Name,
		Operation:      strings.ToLower(in.Operation),
		DesiredSpec:    in.Spec,
		Summary:        fmt.Sprintf("%s %s %q", strings.ToLower(in.Operation), kindName, in.Name),
		Diff:           diff,
		Destructive:    false,
		Preconditions:  pre,
		State:          "planned",
	}
	if err := p.store.SavePlan(ctx, plan); err != nil {
		return PlanClusterConfigOutput{}, err
	}
	return PlanClusterConfigOutput{Plan: plan}, nil
}

func (p *Planner) ApplyPlan(ctx context.Context, in ApplyPlanInput) (ApplyPlanOutput, error) {
	if strings.TrimSpace(in.PlanID) == "" {
		return ApplyPlanOutput{}, fmt.Errorf("validation_error: plan_id is required")
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return ApplyPlanOutput{}, fmt.Errorf("validation_error: idempotency_key is required")
	}
	plan, err := p.store.GetPlan(ctx, in.PlanID)
	if err != nil {
		return ApplyPlanOutput{}, err
	}
	if plan == nil {
		return ApplyPlanOutput{}, fmt.Errorf("validation_error: plan %q not found", in.PlanID)
	}
	if time.Now().UTC().After(plan.ExpiresAt) {
		return ApplyPlanOutput{}, fmt.Errorf("stale_plan: plan %q expired at %s", plan.PlanID, plan.ExpiresAt.Format(time.RFC3339))
	}
	if plan.Destructive && !in.AcknowledgeDestructive {
		return ApplyPlanOutput{}, fmt.Errorf("validation_error: destructive plans require acknowledge_destructive=true")
	}
	job, existing, err := p.store.GetOrCreateJob(ctx, plan.PlanID, in.IdempotencyKey)
	if err != nil {
		return ApplyPlanOutput{}, err
	}
	if existing {
		return ApplyPlanOutput{Job: *job}, nil
	}

	if err := p.apply(ctx, plan, job); err != nil {
		job.Phase = "failed"
		job.Error = err.Error()
		_ = p.store.UpdateJob(ctx, *job)
		return ApplyPlanOutput{}, err
	}
	job.Phase = "succeeded"
	job.ResultRef = fmt.Sprintf("linstor://jobs/%s", job.JobID)
	if err := p.store.UpdateJob(ctx, *job); err != nil {
		return ApplyPlanOutput{}, err
	}
	return ApplyPlanOutput{Job: *job}, nil
}

func (p *Planner) GetJob(ctx context.Context, in JobInput) (JobOutput, error) {
	job, err := p.store.GetJob(ctx, in.JobID)
	if err != nil {
		return JobOutput{}, err
	}
	if job == nil {
		return JobOutput{}, fmt.Errorf("validation_error: job %q not found", in.JobID)
	}
	return JobOutput{Job: *job}, nil
}

func (p *Planner) CancelJob(ctx context.Context, in JobInput) (JobOutput, error) {
	job, err := p.store.GetJob(ctx, in.JobID)
	if err != nil {
		return JobOutput{}, err
	}
	if job == nil {
		return JobOutput{}, fmt.Errorf("validation_error: job %q not found", in.JobID)
	}
	switch job.Phase {
	case "succeeded", "failed", "cancelled":
		return JobOutput{Job: *job}, nil
	default:
		job.Phase = "cancelled"
		if err := p.store.UpdateJob(ctx, *job); err != nil {
			return JobOutput{}, err
		}
		return JobOutput{Job: *job}, nil
	}
}

func (p *Planner) apply(ctx context.Context, plan *PlanRecord, job *JobRecord) error {
	kind, gvr, kindName, err := planTarget(string(plan.Kind))
	if err != nil {
		return err
	}
	current, err := p.kube.GetClusterScoped(ctx, gvr, plan.Name)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	if err := validatePreconditions(plan.Preconditions, current, p.kube.CurrentCtx); err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	if current != nil && current.GetName() != "" {
		obj = current.DeepCopy()
	} else {
		obj.SetAPIVersion("piraeus.io/v1")
		obj.SetKind(kindName)
		obj.SetName(plan.Name)
	}
	if err := unstructured.SetNestedMap(obj.Object, plan.DesiredSpec, "spec"); err != nil {
		return err
	}
	if _, err := p.kube.ApplyClusterScoped(ctx, gvr, obj); err != nil {
		return fmt.Errorf("backend_error: apply %s %q: %w", kind, plan.Name, err)
	}
	return nil
}

func validatePreconditions(pre PlanPreconditions, current *unstructured.Unstructured, kubeContext string) error {
	if pre.KubeContext != kubeContext {
		return fmt.Errorf("stale_plan: kube context changed from %q to %q", pre.KubeContext, kubeContext)
	}
	if pre.Absent {
		if current != nil && current.GetName() != "" {
			return fmt.Errorf("stale_plan: target now exists")
		}
		return nil
	}
	if current == nil || current.GetName() == "" {
		return fmt.Errorf("stale_plan: target no longer exists")
	}
	if string(current.GetUID()) != pre.UID || current.GetResourceVersion() != pre.ResourceVersion || current.GetGeneration() != pre.Generation {
		return fmt.Errorf("stale_plan: target changed since planning")
	}
	return nil
}

func planTarget(kind string) (InventoryKind, schema.GroupVersionResource, string, error) {
	switch kind {
	case string(KindCluster):
		return KindCluster, kube.GVRLinstorCluster, "LinstorCluster", nil
	case string(KindSatelliteConfiguration):
		return KindSatelliteConfiguration, kube.GVRSatelliteCfg, "LinstorSatelliteConfiguration", nil
	case string(KindNodeConnection):
		return KindNodeConnection, kube.GVRNodeConnection, "LinstorNodeConnection", nil
	default:
		return "", schema.GroupVersionResource{}, "", fmt.Errorf("validation_error: unsupported kind %q", kind)
	}
}

func allowedOperation(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "create", "update", "reconcile":
		return true
	default:
		return false
	}
}

func renderDiff(current, desired map[string]any) (string, error) {
	currentJSON, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return "", err
	}
	desiredJSON, err := json.MarshalIndent(desired, "", "  ")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("current:\n%s\n\ndesired:\n%s", currentJSON, desiredJSON), nil
}
