package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// WellKnownPath is the standard endpoint for health manifests.
	WellKnownPath = "/.well-known/health"

	// DefaultTimeout for fetching a single manifest.
	DefaultTimeout = 5 * time.Second

	// DefaultMaxDepth for graph traversal.
	DefaultMaxDepth = 10
)

// DiscoverOption configures the discovery process.
type DiscoverOption func(*discoverConfig)

type discoverConfig struct {
	client   *http.Client
	timeout  time.Duration
	maxDepth int
}

// WithClient sets a custom HTTP client for discovery requests.
func WithClient(c *http.Client) DiscoverOption {
	return func(cfg *discoverConfig) { cfg.client = c }
}

// WithTimeout sets the per-request timeout for manifest fetches.
func WithTimeout(d time.Duration) DiscoverOption {
	return func(cfg *discoverConfig) { cfg.timeout = d }
}

// WithMaxDepth sets the maximum traversal depth. Default is 10.
func WithMaxDepth(n int) DiscoverOption {
	return func(cfg *discoverConfig) { cfg.maxDepth = n }
}

// FetchManifest fetches a single service's health manifest from its
// /.well-known/health endpoint.
func FetchManifest(ctx context.Context, baseURL string, opts ...DiscoverOption) (*Manifest, error) {
	cfg := &discoverConfig{timeout: DefaultTimeout}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.client == nil {
		cfg.client = &http.Client{Timeout: cfg.timeout}
	}

	url := strings.TrimRight(baseURL, "/") + WellKnownPath

	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := cfg.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	var m Manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode %s: %w", url, err)
	}
	return &m, nil
}

// DiscoverGraph walks the dependency graph starting from rootURL by
// fetching each service's manifest and following HTTP-based DependsOn
// entries. It returns the full discovered graph.
//
// Only DependsOn entries that look like URLs (start with "http://" or
// "https://") are followed. Non-URL dependencies (like "postgres" or
// "redis") are recorded but not traversed.
func DiscoverGraph(ctx context.Context, rootURL string, opts ...DiscoverOption) (*Graph, error) {
	cfg := &discoverConfig{timeout: DefaultTimeout, maxDepth: DefaultMaxDepth}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.client == nil {
		cfg.client = &http.Client{Timeout: cfg.timeout}
	}

	g := &Graph{
		Root:      rootURL,
		Nodes:     make(map[string]*Node),
		Timestamp: time.Now(),
	}

	// BFS traversal
	type queueItem struct {
		url   string
		depth int
	}
	queue := []queueItem{{url: rootURL, depth: 0}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		// skip if already visited
		if _, seen := g.Nodes[item.url]; seen {
			continue
		}

		// skip if too deep
		if item.depth > cfg.maxDepth {
			continue
		}

		manifest, err := FetchManifest(ctx, item.url, WithClient(cfg.client), WithTimeout(cfg.timeout))
		if err != nil {
			// record the node as unreachable but don't fail the whole graph
			g.Nodes[item.url] = &Node{
				URL:    item.url,
				Status: "unknown",
			}
			continue
		}

		node := &Node{
			Service: manifest.Service,
			URL:     item.url,
			Status:  manifest.Status,
			Checks:  manifest.Checks,
		}

		// collect HTTP dependencies for traversal
		for _, check := range manifest.Checks {
			for _, dep := range check.DependsOn {
				if isHTTPURL(dep) {
					node.Dependencies = append(node.Dependencies, dep)
					queue = append(queue, queueItem{url: dep, depth: item.depth + 1})
				}
			}
		}

		g.Nodes[item.url] = node
	}

	return g, nil
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
