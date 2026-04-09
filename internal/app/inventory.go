package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ntbc/linstor-mcp-server/internal/kube"
	"github.com/ntbc/linstor-mcp-server/internal/linstor"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type InventoryService struct {
	kube    *kube.Client
	linstor *linstor.Client
}

func NewInventoryService(k *kube.Client, l *linstor.Client) *InventoryService {
	return &InventoryService{kube: k, linstor: l}
}

type ListInput struct {
	Kind       string `json:"kind,omitempty" jsonschema:"inventory kind filter"`
	NamePrefix string `json:"name_prefix,omitempty" jsonschema:"case-insensitive prefix filter"`
	Node       string `json:"node,omitempty" jsonschema:"node filter for node-bound objects"`
	Owner      string `json:"owner,omitempty" jsonschema:"owner filter"`
	Limit      int    `json:"limit,omitempty" jsonschema:"maximum number of items to return"`
	Cursor     string `json:"cursor,omitempty" jsonschema:"opaque pagination cursor"`
}

type ListOutput struct {
	Items      []InventoryItem `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
	Total      int             `json:"total"`
}

func (s *InventoryService) List(ctx context.Context, in ListInput) (ListOutput, error) {
	items, err := s.collectAll(ctx)
	if err != nil {
		return ListOutput{}, err
	}
	items = filterItems(items, in)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind == items[j].Kind {
			return items[i].ID < items[j].ID
		}
		return items[i].Kind < items[j].Kind
	})

	offset, err := decodeCursor(in.Cursor)
	if err != nil {
		return ListOutput{}, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset > len(items) {
		offset = len(items)
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	out := ListOutput{
		Items: items[offset:end],
		Total: len(items),
	}
	if end < len(items) {
		out.NextCursor = encodeCursor(end)
	}
	return out, nil
}

func (s *InventoryService) Get(ctx context.Context, kind InventoryKind, id string) (*InventoryItem, error) {
	items, err := s.collectAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Kind == kind && item.ID == strings.ToLower(id) {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("validation_error: %s %q not found", kind, id)
}

func (s *InventoryService) collectAll(ctx context.Context) ([]InventoryItem, error) {
	var items []InventoryItem

	clusterList, err := s.kube.ListClusterScoped(ctx, kube.GVRLinstorCluster)
	if err != nil {
		return nil, err
	}
	items = append(items, fromPiraeusList(KindCluster, clusterList)...)

	satCfgList, err := s.kube.ListClusterScoped(ctx, kube.GVRSatelliteCfg)
	if err != nil {
		return nil, err
	}
	items = append(items, fromPiraeusList(KindSatelliteConfiguration, satCfgList)...)

	nodeConnList, err := s.kube.ListClusterScoped(ctx, kube.GVRNodeConnection)
	if err != nil {
		return nil, err
	}
	items = append(items, fromPiraeusList(KindNodeConnection, nodeConnList)...)

	nodes, err := s.linstor.Nodes(ctx)
	if err != nil {
		return nil, err
	}
	for _, n := range nodes {
		id := strings.ToLower(n.Name)
		items = append(items, InventoryItem{
			Kind:   KindNode,
			ID:     id,
			Name:   n.Name,
			Title:  n.Name,
			URI:    "linstor://nodes/" + id,
			Owner:  "linstor_controller",
			Source: "controller",
			Metadata: map[string]any{
				"type":              n.Type,
				"connection_status": n.ConnectionStatus,
				"storage_providers": n.StorageProviders,
				"resource_layers":   n.ResourceLayers,
			},
		})
	}

	pools, err := s.kube.ListClusterScoped(ctx, kube.GVRInternalNodeStorPool)
	if err == nil {
		items = append(items, fromStoragePools(pools)...)
	}
	rdefs, err := s.kube.ListClusterScoped(ctx, kube.GVRInternalResourceDef)
	if err == nil {
		items = append(items, fromResourceDefinitions(rdefs)...)
	}
	resources, err := s.kube.ListClusterScoped(ctx, kube.GVRInternalResources)
	if err == nil {
		items = append(items, fromResources(resources)...)
	}

	return items, nil
}

func fromPiraeusList(kind InventoryKind, list *unstructured.UnstructuredList) []InventoryItem {
	items := make([]InventoryItem, 0, len(list.Items))
	for _, obj := range list.Items {
		name := obj.GetName()
		id := strings.ToLower(name)
		spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
		status, _, _ := unstructured.NestedMap(obj.Object, "status")
		items = append(items, InventoryItem{
			Kind:   kind,
			ID:     id,
			Name:   name,
			Title:  name,
			URI:    fmt.Sprintf("linstor://%ss/%s", kindURIPlural(kind), id),
			Owner:  "piraeus_operator",
			Source: "kubernetes",
			Spec:   spec,
			Status: status,
		})
	}
	return items
}

func fromStoragePools(list *unstructured.UnstructuredList) []InventoryItem {
	items := make([]InventoryItem, 0, len(list.Items))
	for _, obj := range list.Items {
		spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
		node := strings.ToLower(strings.TrimSpace(asString(spec["node_name"])))
		pool := strings.ToLower(strings.TrimSpace(asString(spec["pool_name"])))
		id := node + "/" + pool
		items = append(items, InventoryItem{
			Kind:   KindStoragePool,
			ID:     id,
			Name:   asString(spec["pool_name"]),
			Title:  fmt.Sprintf("%s/%s", asString(spec["node_name"]), asString(spec["pool_name"])),
			URI:    fmt.Sprintf("linstor://storage-pools/%s/%s", node, pool),
			Owner:  "diagnostic_mirror",
			Source: "internal_mirror",
			Spec:   spec,
			Metadata: map[string]any{
				"node":   asString(spec["node_name"]),
				"driver": asString(spec["driver_name"]),
			},
		})
	}
	return items
}

func fromResourceDefinitions(list *unstructured.UnstructuredList) []InventoryItem {
	items := make([]InventoryItem, 0, len(list.Items))
	for _, obj := range list.Items {
		spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
		name := strings.ToLower(firstNonEmpty(asString(spec["resource_dsp_name"]), asString(spec["resource_name"])))
		items = append(items, InventoryItem{
			Kind:   KindResourceDefinition,
			ID:     name,
			Name:   firstNonEmpty(asString(spec["resource_dsp_name"]), asString(spec["resource_name"])),
			Title:  firstNonEmpty(asString(spec["resource_dsp_name"]), asString(spec["resource_name"])),
			URI:    fmt.Sprintf("linstor://resource-definitions/%s", name),
			Owner:  "diagnostic_mirror",
			Source: "internal_mirror",
			Spec:   spec,
			Metadata: map[string]any{
				"resource_group_name": asString(spec["resource_group_name"]),
				"layer_stack":         spec["layer_stack"],
			},
		})
	}
	return items
}

func fromResources(list *unstructured.UnstructuredList) []InventoryItem {
	items := make([]InventoryItem, 0, len(list.Items))
	for _, obj := range list.Items {
		spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
		resource := strings.ToLower(asString(spec["resource_name"]))
		node := strings.ToLower(asString(spec["node_name"]))
		id := resource + "@" + node
		items = append(items, InventoryItem{
			Kind:   KindResource,
			ID:     id,
			Name:   asString(spec["resource_name"]),
			Title:  fmt.Sprintf("%s on %s", asString(spec["resource_name"]), asString(spec["node_name"])),
			URI:    fmt.Sprintf("linstor://resources/%s", id),
			Owner:  "diagnostic_mirror",
			Source: "internal_mirror",
			Spec:   spec,
			Metadata: map[string]any{
				"node":           asString(spec["node_name"]),
				"resource_flags": spec["resource_flags"],
			},
		})
	}
	return items
}

func filterItems(items []InventoryItem, in ListInput) []InventoryItem {
	var out []InventoryItem
	kind := strings.TrimSpace(strings.ToLower(in.Kind))
	namePrefix := strings.TrimSpace(strings.ToLower(in.NamePrefix))
	node := strings.TrimSpace(strings.ToLower(in.Node))
	owner := strings.TrimSpace(strings.ToLower(in.Owner))
	for _, item := range items {
		if kind != "" && string(item.Kind) != kind {
			continue
		}
		if namePrefix != "" && !strings.HasPrefix(strings.ToLower(item.Name), namePrefix) && !strings.HasPrefix(strings.ToLower(item.ID), namePrefix) {
			continue
		}
		if owner != "" && strings.ToLower(item.Owner) != owner {
			continue
		}
		if node != "" {
			itemNode := strings.ToLower(asString(item.Metadata["node"]))
			if itemNode == "" {
				itemNode = strings.ToLower(asString(item.Spec["node_name"]))
			}
			if itemNode != node {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("validation_error: invalid cursor")
	}
	offset, err := strconv.Atoi(string(raw))
	if err != nil {
		return 0, fmt.Errorf("validation_error: invalid cursor")
	}
	if offset < 0 {
		return 0, fmt.Errorf("validation_error: invalid cursor")
	}
	return offset, nil
}

func kindURIPlural(kind InventoryKind) string {
	switch kind {
	case KindCluster:
		return "cluster"
	case KindSatelliteConfiguration:
		return "satellite-configuration"
	case KindNodeConnection:
		return "node-connection"
	default:
		return string(kind)
	}
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		b, _ := json.Marshal(v)
		return strings.Trim(string(b), `"`)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
