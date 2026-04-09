package app

import "testing"

func TestParseResourceURIKinds(t *testing.T) {
	tests := []struct {
		uri      string
		wantKind string
		wantID   string
	}{
		{"linstor://clusters/homelab", string(KindCluster), "homelab"},
		{"linstor://satellite-configurations/homelab", string(KindSatelliteConfiguration), "homelab"},
		{"linstor://node-connections/a", string(KindNodeConnection), "a"},
		{"linstor://nodes/node-01", string(KindNode), "node-01"},
		{"linstor://resource-definitions/pvc-1", string(KindResourceDefinition), "pvc-1"},
		{"linstor://resources/pvc-1@node-01", string(KindResource), "pvc-1@node-01"},
		{"linstor://jobs/job_1", "job", "job_1"},
	}
	for _, tt := range tests {
		gotKind, gotID, err := parseResourceURI(tt.uri)
		if err != nil {
			t.Fatalf("parseResourceURI(%q) error = %v", tt.uri, err)
		}
		if gotKind != tt.wantKind || gotID != tt.wantID {
			t.Fatalf("parseResourceURI(%q) = (%q, %q), want (%q, %q)", tt.uri, gotKind, gotID, tt.wantKind, tt.wantID)
		}
	}
}

func TestParseResourceURIInvalid(t *testing.T) {
	if _, _, err := parseResourceURI("linstor://bad"); err == nil {
		t.Fatal("parseResourceURI() error = nil, want error")
	}
}
