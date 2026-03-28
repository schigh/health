package discovery

import (
	"fmt"
	"strings"
)

// Mermaid renders the graph as a Mermaid flowchart diagram.
func (g *Graph) Mermaid() string {
	var b strings.Builder
	b.WriteString("graph TD\n")

	for url, node := range g.Nodes {
		id := sanitizeID(url)
		label := node.Service
		if label == "" {
			label = url
		}

		style := ""
		switch node.Status {
		case "pass":
			style = fmt.Sprintf("    style %s fill:#4caf50,color:#fff\n", id)
		case "fail":
			style = fmt.Sprintf("    style %s fill:#f44336,color:#fff\n", id)
		case "warn":
			style = fmt.Sprintf("    style %s fill:#ff9800,color:#fff\n", id)
		default:
			style = fmt.Sprintf("    style %s fill:#9e9e9e,color:#fff\n", id)
		}

		b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", id, label))
		b.WriteString(style)

		for _, dep := range node.Dependencies {
			depID := sanitizeID(dep)
			b.WriteString(fmt.Sprintf("    %s --> %s\n", id, depID))
		}
	}

	return b.String()
}

// DOT renders the graph in Graphviz DOT format.
func (g *Graph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph health {\n")
	b.WriteString("    rankdir=TB;\n")
	b.WriteString("    node [shape=box, style=filled, fontname=\"sans-serif\"];\n\n")

	for url, node := range g.Nodes {
		id := sanitizeID(url)
		label := node.Service
		if label == "" {
			label = url
		}

		color := "#9e9e9e"
		switch node.Status {
		case "pass":
			color = "#4caf50"
		case "fail":
			color = "#f44336"
		case "warn":
			color = "#ff9800"
		}

		b.WriteString(fmt.Sprintf("    %s [label=\"%s\", fillcolor=\"%s\", fontcolor=\"white\"];\n", id, label, color))
	}

	b.WriteString("\n")

	for url, node := range g.Nodes {
		id := sanitizeID(url)
		for _, dep := range node.Dependencies {
			depID := sanitizeID(dep)
			b.WriteString(fmt.Sprintf("    %s -> %s;\n", id, depID))
		}
		_ = url
	}

	b.WriteString("}\n")
	return b.String()
}

// sanitizeID converts a URL into a valid Mermaid/DOT node ID.
func sanitizeID(s string) string {
	r := strings.NewReplacer(
		"://", "_",
		"/", "_",
		".", "_",
		":", "_",
		"-", "_",
	)
	return r.Replace(s)
}
