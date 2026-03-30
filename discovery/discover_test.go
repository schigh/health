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

func TestDiscoverGraph_MaxDepth(t *testing.T) {
	// backend that depends on itself (cycle), which would infinite loop without max depth
	var backend *httptest.Server
	backend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != discovery.WellKnownPath {
			http.NotFound(w, r)
			return
		}
		m := discovery.Manifest{
			Service: "backend",
			Status:  "pass",
			Checks: []discovery.CheckEntry{
				{Name: "self", Status: "healthy", DependsOn: []string{backend.URL}},
			},
		}
		json.NewEncoder(w).Encode(m)
	}))
	defer backend.Close()

	g, err := discovery.DiscoverGraph(context.Background(), backend.URL,
		discovery.WithMaxDepth(2))
	if err != nil {
		t.Fatal(err)
	}

	// should have visited the node once (cycle detected via seen set)
	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
}

func TestFetchManifest_WithClient(t *testing.T) {
	srv := serveManifest(discovery.Manifest{
		Service: "api",
		Status:  "pass",
	})
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	m, err := discovery.FetchManifest(context.Background(), srv.URL,
		discovery.WithClient(client))
	if err != nil {
		t.Fatal(err)
	}
	if m.Service != "api" {
		t.Fatalf("expected service 'api', got %q", m.Service)
	}
}

func TestFetchManifest_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != discovery.WellKnownPath {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := discovery.FetchManifest(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGraph_Mermaid_AllStatuses(t *testing.T) {
	g := &discovery.Graph{
		Root: "http://api:8080",
		Nodes: map[string]*discovery.Node{
			"http://api:8080": {
				Service: "api",
				URL:     "http://api:8080",
				Status:  "pass",
			},
			"http://db:5432": {
				Service: "database",
				URL:     "http://db:5432",
				Status:  "fail",
			},
			"http://cache:6379": {
				Service: "cache",
				URL:     "http://cache:6379",
				Status:  "warn",
			},
			"http://unknown:9999": {
				URL:    "http://unknown:9999",
				Status: "unknown",
			},
		},
	}

	mermaid := g.Mermaid()
	if !strings.Contains(mermaid, "#4caf50") {
		t.Error("expected green for pass")
	}
	if !strings.Contains(mermaid, "#f44336") {
		t.Error("expected red for fail")
	}
	if !strings.Contains(mermaid, "#ff9800") {
		t.Error("expected orange for warn")
	}
	if !strings.Contains(mermaid, "#9e9e9e") {
		t.Error("expected grey for unknown")
	}
	// node without service name should use URL as label
	if !strings.Contains(mermaid, "http://unknown:9999") {
		t.Error("expected URL as label for node without service name")
	}
}

func TestGraph_DOT_AllStatuses(t *testing.T) {
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
				URL:    "http://db:5432",
				Status: "warn",
			},
		},
	}

	dot := g.DOT()
	if !strings.Contains(dot, "#4caf50") {
		t.Error("expected green for pass")
	}
	if !strings.Contains(dot, "#ff9800") {
		t.Error("expected orange for warn")
	}
	if !strings.Contains(dot, "->") {
		t.Error("expected edge in DOT output")
	}
	// node without service should use URL as label
	if !strings.Contains(dot, "http://db:5432") {
		t.Error("expected URL as label for node without service name")
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
