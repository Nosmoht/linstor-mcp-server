package app

import "testing"

func TestDecodeCursor(t *testing.T) {
	cursor := encodeCursor(25)
	got, err := decodeCursor(cursor)
	if err != nil {
		t.Fatalf("decodeCursor() error = %v", err)
	}
	if got != 25 {
		t.Fatalf("decodeCursor() = %d, want 25", got)
	}
}

func TestDecodeCursorInvalid(t *testing.T) {
	if _, err := decodeCursor("%%%"); err == nil {
		t.Fatal("decodeCursor() error = nil, want error")
	}
}

func TestFilterItems(t *testing.T) {
	items := []InventoryItem{
		{Kind: KindNode, ID: "node-01", Name: "node-01", Owner: "linstor_controller", Metadata: map[string]any{"node": "node-01"}},
		{Kind: KindCluster, ID: "homelab", Name: "homelab", Owner: "piraeus_operator"},
		{Kind: KindStoragePool, ID: "node-01/lvm-thick", Name: "LVM-THICK", Owner: "diagnostic_mirror", Metadata: map[string]any{"node": "node-01"}},
	}

	got := filterItems(items, ListInput{Kind: "node", NamePrefix: "node", Owner: "linstor_controller"})
	if len(got) != 1 {
		t.Fatalf("len(filterItems()) = %d, want 1", len(got))
	}
	if got[0].ID != "node-01" {
		t.Fatalf("filtered item ID = %q, want node-01", got[0].ID)
	}

	got = filterItems(items, ListInput{Node: "node-01"})
	if len(got) != 2 {
		t.Fatalf("len(filterItems(node)) = %d, want 2", len(got))
	}
}

func TestParseResourceURI(t *testing.T) {
	kind, id, err := parseResourceURI("linstor://storage-pools/node-01/lvm-thick")
	if err != nil {
		t.Fatalf("parseResourceURI() error = %v", err)
	}
	if kind != string(KindStoragePool) || id != "node-01/lvm-thick" {
		t.Fatalf("parseResourceURI() = (%q, %q), want (%q, %q)", kind, id, KindStoragePool, "node-01/lvm-thick")
	}
}

func TestHelpers(t *testing.T) {
	if got := kindURIPlural(KindCluster); got != "cluster" {
		t.Fatalf("kindURIPlural(cluster) = %q", got)
	}
	if got := firstNonEmpty("", "x", "y"); got != "x" {
		t.Fatalf("firstNonEmpty() = %q, want x", got)
	}
}
