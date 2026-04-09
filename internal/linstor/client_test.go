package linstor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestControllerVersion(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/controller/version" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ControllerVersion{Version: "1.33.1", RestAPIVersion: "1.27.0"})
	}))
	defer srv.Close()

	client := &Client{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}
	got, err := client.ControllerVersion(context.Background())
	if err != nil {
		t.Fatalf("ControllerVersion() error = %v", err)
	}
	if got.Version != "1.33.1" {
		t.Fatalf("ControllerVersion() version = %q", got.Version)
	}
}

func TestNodes(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/nodes" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]Node{{Name: "node-01", ConnectionStatus: "ONLINE"}})
	}))
	defer srv.Close()

	client := &Client{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}
	got, err := client.Nodes(context.Background())
	if err != nil {
		t.Fatalf("Nodes() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "node-01" {
		t.Fatalf("Nodes() = %+v", got)
	}
}
