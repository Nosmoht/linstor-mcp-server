package app

import "time"

type InventoryKind string

const (
	KindCluster                InventoryKind = "cluster"
	KindSatelliteConfiguration InventoryKind = "satellite_configuration"
	KindNodeConnection         InventoryKind = "node_connection"
	KindNode                   InventoryKind = "node"
	KindStoragePool            InventoryKind = "storage_pool"
	KindResourceDefinition     InventoryKind = "resource_definition"
	KindResource               InventoryKind = "resource"
)

type InventoryItem struct {
	Kind     InventoryKind  `json:"kind"`
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Title    string         `json:"title"`
	URI      string         `json:"uri"`
	Owner    string         `json:"owner"`
	Source   string         `json:"source"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Spec     map[string]any `json:"spec,omitempty"`
	Status   map[string]any `json:"status,omitempty"`
}

type ValidationIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type EnvironmentValidation struct {
	KubeContext         string            `json:"kube_context"`
	LinstorCluster      string            `json:"linstor_cluster"`
	ControllerURL       string            `json:"controller_url"`
	ControllerVersion   string            `json:"controller_version"`
	ControllerAPIVer    string            `json:"controller_api_version"`
	DefaultStorageClass string            `json:"default_storage_class,omitempty"`
	Checks              map[string]bool   `json:"checks"`
	Issues              []ValidationIssue `json:"issues,omitempty"`
}

type PlanPreconditions struct {
	Absent          bool   `json:"absent"`
	UID             string `json:"uid,omitempty"`
	ResourceVersion string `json:"resource_version,omitempty"`
	Generation      int64  `json:"generation,omitempty"`
	KubeContext     string `json:"kube_context"`
}

type PlanRecord struct {
	PlanID         string            `json:"plan_id"`
	CreatedAt      time.Time         `json:"created_at"`
	ExpiresAt      time.Time         `json:"expires_at"`
	KubeContext    string            `json:"kube_context"`
	LinstorCluster string            `json:"linstor_cluster"`
	Kind           InventoryKind     `json:"kind"`
	Name           string            `json:"name"`
	Operation      string            `json:"operation"`
	DesiredSpec    map[string]any    `json:"desired_spec"`
	Summary        string            `json:"summary"`
	Diff           string            `json:"diff"`
	Destructive    bool              `json:"destructive"`
	Preconditions  PlanPreconditions `json:"preconditions"`
	State          string            `json:"state"`
}

type JobRecord struct {
	JobID          string    `json:"job_id"`
	PlanID         string    `json:"plan_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Phase          string    `json:"phase"`
	ResultRef      string    `json:"result_ref,omitempty"`
	Error          string    `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
