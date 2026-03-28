package discovery

import "time"

// Manifest describes a service's health check configuration and current state.
// Served at /.well-known/health for zero-infrastructure dependency discovery.
type Manifest struct {
	// Service identifies this service instance.
	Service string `json:"service"`

	// Version of the service (optional, informational).
	Version string `json:"version,omitempty"`

	// Status is the aggregate service status: "pass", "fail", or "warn".
	Status string `json:"status"`

	// Checks describes each registered health check.
	Checks []CheckEntry `json:"checks"`

	// Timestamp of when this manifest was generated.
	Timestamp time.Time `json:"timestamp"`
}

// CheckEntry describes a single health check in the manifest.
type CheckEntry struct {
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	Group            string   `json:"group,omitempty"`
	ComponentType    string   `json:"componentType,omitempty"`
	AffectsLiveness  bool     `json:"affectsLiveness,omitempty"`
	AffectsReadiness bool     `json:"affectsReadiness,omitempty"`
	AffectsStartup   bool     `json:"affectsStartup,omitempty"`
	DependsOn        []string `json:"dependsOn,omitempty"`
	Duration         string   `json:"duration,omitempty"`
	LastCheck        string   `json:"lastCheck,omitempty"`
	Error            string   `json:"error,omitempty"`
}

// Node represents a service in a discovered dependency graph.
type Node struct {
	// Service name (from Manifest.Service).
	Service string `json:"service"`

	// URL this node was discovered at.
	URL string `json:"url"`

	// Status of this node: "pass", "fail", "warn", or "unknown".
	Status string `json:"status"`

	// Checks registered on this node.
	Checks []CheckEntry `json:"checks"`

	// Dependencies are the URLs of services this node depends on.
	// Populated by traversing DependsOn entries that have HTTP URLs.
	Dependencies []string `json:"dependencies,omitempty"`
}

// Graph is a discovered dependency graph rooted at a starting service.
type Graph struct {
	// Root is the URL of the starting service.
	Root string `json:"root"`

	// Nodes keyed by URL.
	Nodes map[string]*Node `json:"nodes"`

	// Timestamp of when this graph was built.
	Timestamp time.Time `json:"timestamp"`
}
