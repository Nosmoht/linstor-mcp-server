package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ntbc/linstor-mcp-server/internal/config"
	"github.com/ntbc/linstor-mcp-server/internal/kube"
	"github.com/ntbc/linstor-mcp-server/internal/linstor"
)

type Server struct {
	cfg       config.Config
	kube      *kube.Client
	linstor   *linstor.Client
	store     *Store
	inventory *InventoryService
	planner   *Planner
	mcp       *mcp.Server
	httpSrv   *http.Server
}

func New(ctx context.Context, cfg config.Config) (*Server, error) {
	kubeClient, err := kube.New(cfg.KubeContext)
	if err != nil {
		return nil, err
	}
	store, err := NewStore(cfg.StateDir)
	if err != nil {
		return nil, err
	}
	linstorClient, err := linstor.New(ctx, cfg, kubeClient)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	s := &Server{
		cfg:       cfg,
		kube:      kubeClient,
		linstor:   linstorClient,
		store:     store,
		inventory: NewInventoryService(kubeClient, linstorClient),
	}
	s.planner = NewPlanner(cfg, kubeClient, store)
	s.mcp = s.newMCPServer()
	return s, nil
}

func (s *Server) Close() {
	if s.httpSrv != nil {
		_ = s.httpSrv.Close()
	}
	if s.linstor != nil {
		s.linstor.Close()
	}
	if s.store != nil {
		_ = s.store.Close()
	}
}

func (s *Server) RunStdio(ctx context.Context) error {
	slog.Info("starting stdio MCP server", "profile", s.cfg.Profile, "kube_context", s.kube.CurrentCtx)
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) RunHTTP(ctx context.Context) error {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s.mcp
	}, nil)
	s.httpSrv = &http.Server{Addr: s.cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 30 * time.Second}
	go func() {
		<-ctx.Done()
		_ = s.httpSrv.Close()
	}()
	slog.Info("starting beta streamable HTTP MCP server", "addr", s.cfg.HTTPAddr)
	err := s.httpSrv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: s.cfg.ServerName, Version: s.cfg.ServerVersion}, &mcp.ServerOptions{
		CompletionHandler: s.complete,
	})

	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, Title: "Read"}
	idempotentWrite := &mcp.ToolAnnotations{IdempotentHint: true}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate_environment",
		Description: "Validate Kubernetes, LINSTOR controller connectivity, and homelab profile assumptions",
		Annotations: readOnly,
	}, s.validateEnvironment)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "inventory_list",
		Description: "List canonical LINSTOR inventory items with pagination and filters",
		Annotations: readOnly,
	}, s.inventoryList)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "inventory_get",
		Description: "Get a canonical LINSTOR inventory item by semantic kind and ID",
		Annotations: readOnly,
	}, s.inventoryGet)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_cluster_config",
		Description: "Create a 5-minute cluster-configuration plan for a Piraeus operator object without mutating the cluster",
		Annotations: idempotentWrite,
	}, s.planClusterConfig)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "apply_plan",
		Description: "Apply a previously created plan with stale-plan revalidation and idempotency protection",
		Annotations: idempotentWrite,
	}, s.applyPlan)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "job_get",
		Description: "Get the status of a plan application job",
		Annotations: readOnly,
	}, s.jobGet)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "job_cancel",
		Description: "Cancel a pending or running job if it has not already completed",
	}, s.jobCancel)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "cluster-resource",
		Title:       "LINSTOR Cluster",
		URITemplate: "linstor://clusters/{name}",
	}, s.readResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "satellite-configuration-resource",
		Title:       "LINSTOR Satellite Configuration",
		URITemplate: "linstor://satellite-configurations/{name}",
	}, s.readResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "node-connection-resource",
		Title:       "LINSTOR Node Connection",
		URITemplate: "linstor://node-connections/{name}",
	}, s.readResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "node-resource",
		Title:       "LINSTOR Node",
		URITemplate: "linstor://nodes/{name}",
	}, s.readResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "storage-pool-resource",
		Title:       "LINSTOR Storage Pool",
		URITemplate: "linstor://storage-pools/{node}/{pool}",
	}, s.readResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "resource-definition-resource",
		Title:       "LINSTOR Resource Definition",
		URITemplate: "linstor://resource-definitions/{name}",
	}, s.readResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "resource-resource",
		Title:       "LINSTOR Resource",
		URITemplate: "linstor://resources/{id}",
	}, s.readResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "job-resource",
		Title:       "LINSTOR Job",
		URITemplate: "linstor://jobs/{job_id}",
	}, s.readResource)

	return server
}

func (s *Server) validateEnvironment(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, EnvironmentValidation, error) {
	version, err := s.linstor.ControllerVersion(ctx)
	if err != nil {
		return nil, EnvironmentValidation{}, fmt.Errorf("connectivity_error: %w", err)
	}
	defaultSC, err := s.kube.DefaultStorageClass(ctx)
	if err != nil {
		return nil, EnvironmentValidation{}, fmt.Errorf("backend_error: %w", err)
	}
	cluster, err := s.kube.GetClusterScoped(ctx, kube.GVRLinstorCluster, s.cfg.LinstorCluster)
	if err != nil {
		return nil, EnvironmentValidation{}, fmt.Errorf("backend_error: failed to get LinstorCluster %q: %w", s.cfg.LinstorCluster, err)
	}
	checks := map[string]bool{
		"kube_context_matches":          s.kube.CurrentCtx == s.cfg.KubeContext,
		"linstor_cluster_exists":        cluster != nil,
		"controller_reachable":          version.Version != "",
		"default_storage_class_present": defaultSC != "",
	}
	out := EnvironmentValidation{
		KubeContext:         s.kube.CurrentCtx,
		LinstorCluster:      s.cfg.LinstorCluster,
		ControllerURL:       s.cfg.ControllerURL,
		ControllerVersion:   version.Version,
		ControllerAPIVer:    version.RestAPIVersion,
		DefaultStorageClass: defaultSC,
		Checks:              checks,
	}
	return nil, out, nil
}

func (s *Server) inventoryList(ctx context.Context, _ *mcp.CallToolRequest, in ListInput) (*mcp.CallToolResult, ListOutput, error) {
	out, err := s.inventory.List(ctx, in)
	if err != nil {
		return nil, ListOutput{}, err
	}
	return nil, out, nil
}

type InventoryGetInput struct {
	Kind string `json:"kind" jsonschema:"canonical kind name"`
	ID   string `json:"id" jsonschema:"semantic object ID"`
}

type InventoryGetOutput struct {
	Item InventoryItem `json:"item"`
}

func (s *Server) inventoryGet(ctx context.Context, _ *mcp.CallToolRequest, in InventoryGetInput) (*mcp.CallToolResult, InventoryGetOutput, error) {
	item, err := s.inventory.Get(ctx, InventoryKind(strings.ToLower(in.Kind)), in.ID)
	if err != nil {
		return nil, InventoryGetOutput{}, err
	}
	return nil, InventoryGetOutput{Item: *item}, nil
}

func (s *Server) planClusterConfig(ctx context.Context, _ *mcp.CallToolRequest, in PlanClusterConfigInput) (*mcp.CallToolResult, PlanClusterConfigOutput, error) {
	out, err := s.planner.PlanClusterConfig(ctx, in)
	if err != nil {
		return nil, PlanClusterConfigOutput{}, err
	}
	return nil, out, nil
}

func (s *Server) applyPlan(ctx context.Context, _ *mcp.CallToolRequest, in ApplyPlanInput) (*mcp.CallToolResult, ApplyPlanOutput, error) {
	out, err := s.planner.ApplyPlan(ctx, in)
	if err != nil {
		return nil, ApplyPlanOutput{}, err
	}
	return nil, out, nil
}

func (s *Server) jobGet(ctx context.Context, _ *mcp.CallToolRequest, in JobInput) (*mcp.CallToolResult, JobOutput, error) {
	out, err := s.planner.GetJob(ctx, in)
	if err != nil {
		return nil, JobOutput{}, err
	}
	return nil, out, nil
}

func (s *Server) jobCancel(ctx context.Context, _ *mcp.CallToolRequest, in JobInput) (*mcp.CallToolResult, JobOutput, error) {
	out, err := s.planner.CancelJob(ctx, in)
	if err != nil {
		return nil, JobOutput{}, err
	}
	return nil, out, nil
}

func (s *Server) readResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	kind, id, err := parseResourceURI(req.Params.URI)
	if err != nil {
		return nil, err
	}
	var text string
	if kind == "job" {
		job, err := s.store.GetJob(ctx, id)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, fmt.Errorf("validation_error: job %q not found", id)
		}
		b, _ := json.MarshalIndent(job, "", "  ")
		text = string(b)
	} else {
		item, err := s.inventory.Get(ctx, InventoryKind(kind), id)
		if err != nil {
			return nil, err
		}
		b, _ := json.MarshalIndent(item, "", "  ")
		text = string(b)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: req.Params.URI, MIMEType: "application/json", Text: text},
		},
	}, nil
}

func (s *Server) complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	var values []string
	switch req.Params.Argument.Name {
	case "kind":
		values = []string{
			string(KindCluster),
			string(KindSatelliteConfiguration),
			string(KindNodeConnection),
			string(KindNode),
			string(KindStoragePool),
			string(KindResourceDefinition),
			string(KindResource),
		}
	case "name", "id":
		list, err := s.inventory.List(ctx, ListInput{Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			values = append(values, item.ID)
		}
	}
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{
			Values: values,
			Total:  len(values),
		},
	}, nil
}

func parseResourceURI(uri string) (string, string, error) {
	trimmed := strings.TrimPrefix(uri, "linstor://")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("validation_error: unsupported resource uri %q", uri)
	}
	switch parts[0] {
	case "clusters":
		return string(KindCluster), parts[1], nil
	case "satellite-configurations":
		return string(KindSatelliteConfiguration), parts[1], nil
	case "node-connections":
		return string(KindNodeConnection), parts[1], nil
	case "nodes":
		return string(KindNode), parts[1], nil
	case "storage-pools":
		if len(parts) < 3 {
			return "", "", fmt.Errorf("validation_error: invalid storage-pool resource uri")
		}
		return string(KindStoragePool), parts[1] + "/" + parts[2], nil
	case "resource-definitions":
		return string(KindResourceDefinition), parts[1], nil
	case "resources":
		return string(KindResource), parts[1], nil
	case "jobs":
		return "job", parts[1], nil
	default:
		return "", "", fmt.Errorf("validation_error: unsupported resource uri %q", uri)
	}
}
