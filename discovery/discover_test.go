package discovery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/schigh/health/v2/discovery"
)

func serveManifest(m discovery.Manifest) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != discovery.WellKnownPath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)
	}))
}

func TestFetchManifest(t *testing.T) {
	srv := serveManifest(discovery.Manifest{
		Service: "api",
		Status:  "pass",
		Checks: []discovery.CheckEntry{
			{Name: "postgres", Status: "healthy", Group: "database"},
		},
		Timestamp: time.Now(),
	})
	defer srv.Close()

	m, err := discovery.FetchManifest(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if m.Service != "api" {
		t.Fatalf("expected service 'api', got %q", m.Service)
	}
	if len(m.Checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(m.Checks))
	}
	if m.Checks[0].Name != "postgres" {
		t.Fatalf("expected check 'postgres', got %q", m.Checks[0].Name)
	}
}

func TestFetchManifest_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	_, err := discovery.FetchManifest(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFetchManifest_Unreachable(t *testing.T) {
	_, err := discovery.FetchManifest(context.Background(), "http://127.0.0.1:1",
		discovery.WithTimeout(100*time.Millisecond))
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

func TestDiscoverGraph_SingleNode(t *testing.T) {
	srv := serveManifest(discovery.Manifest{
		Service: "api",
		Status:  "pass",
		Checks: []discovery.CheckEntry{
			{Name: "postgres", Status: "healthy", Group: "database"},
		},
	})
	defer srv.Close()

	g, err := discovery.DiscoverGraph(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	node := g.Nodes[srv.URL]
	if node == nil {
		t.Fatal("root node not found")
	}
	if node.Service != "api" {
		t.Fatalf("expected service 'api', got %q", node.Service)
	}
}

func TestDiscoverGraph_MultiNode(t *testing.T) {
	// backend service
	backend := serveManifest(discovery.Manifest{
		Service: "backend",
		Status:  "pass",
		Checks:  []discovery.CheckEntry{{Name: "redis", Status: "healthy"}},
	})
	defer backend.Close()

	// frontend depends on backend via HTTP
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != discovery.WellKnownPath {
			http.NotFound(w, r)
			return
		}
		m := discovery.Manifest{
			Service: "frontend",
			Status:  "pass",
			Checks: []discovery.CheckEntry{
				{Name: "backend-api", Status: "healthy", DependsOn: []string{backend.URL}},
				{Name: "postgres", Status: "healthy"},
			},
		}
		json.NewEncoder(w).Encode(m)
	}))
	defer frontend.Close()

	g, err := discovery.DiscoverGraph(context.Background(), frontend.URL)
	if err != nil {
		t.Fatal(err)
	}

	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.Nodes))
	}

	frontNode := g.Nodes[frontend.URL]
	if frontNode == nil || frontNode.Service != "frontend" {
		t.Fatal("frontend node not found or wrong service")
	}
	if len(frontNode.Dependencies) != 1 || frontNode.Dependencies[0] != backend.URL {
		t.Fatalf("expected frontend to depend on backend, got %v", frontNode.Dependencies)
	}

	backNode := g.Nodes[backend.URL]
	if backNode == nil || backNode.Service != "backend" {
		t.Fatal("backend node not found or wrong service")
	}
}

func TestDiscoverGraph_UnreachableNode(t *testing.T) {
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != discovery.WellKnownPath {
			http.NotFound(w, r)
			return
		}
		m := discovery.Manifest{
			Service: "frontend",
			Status:  "warn",
			Checks: []discovery.CheckEntry{
				{Name: "dead-service", Status: "unhealthy", DependsOn: []string{"http://127.0.0.1:1"}},
			},
		}
		json.NewEncoder(w).Encode(m)
	}))
	defer frontend.Close()

	g, err := discovery.DiscoverGraph(context.Background(), frontend.URL,
		discovery.WithTimeout(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	// should have 2 nodes: frontend + unreachable node
	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.Nodes))
	}

	dead := g.Nodes["http://127.0.0.1:1"]
	if dead == nil {
		t.Fatal("unreachable node not recorded")
	}
	if dead.Status != "unknown" {
		t.Fatalf("expected 'unknown' status for unreachable node, got %q", dead.Status)
	}
}

func TestGraph_Mermaid(t *testing.T) {
	g := &discovery.Graph{
		Root: "http://api:8080",
		Nodes: map[string]*discovery.Node{
			"http://api:8080": {
				Service:      "api",
				URL:          "http://api:8080",
				Status:       "pass",
				Dependencies: []string{"http://db:5432"},
			},
			"http://db:5432": {
				Service: "database",
				URL:     "http://db:5432",
				Status:  "pass",
			},
		},
	}

	mermaid := g.Mermaid()
	if !strings.Contains(mermaid, "graph TD") {
		t.Fatal("expected Mermaid graph header")
	}
	if !strings.Contains(mermaid, "api") {
		t.Fatal("expected api node in Mermaid output")
	}
	if !strings.Contains(mermaid, "-->") {
		t.Fatal("expected edge in Mermaid output")
	}
}

func TestGraph_DOT(t *testing.T) {
	g := &discovery.Graph{
		Root: "http://api:8080",
		Nodes: map[string]*discovery.Node{
			"http://api:8080": {
				Service:      "api",
				URL:          "http://api:8080",
				Status:       "fail",
				Dependencies: []string{"http://db:5432"},
			},
		},
	}

	dot := g.DOT()
	if !strings.Contains(dot, "digraph health") {
		t.Fatal("expected DOT digraph header")
	}
	if !strings.Contains(dot, "#f44336") {
		t.Fatal("expected red color for failed node")
	}
}
