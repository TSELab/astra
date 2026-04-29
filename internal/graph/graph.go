// internal/graph/graph.go
package graph

import (
	"fmt"
	"sort"
	"strings"
)

// To DOT renders an AstraGraph into Graphviz DOT format.
func ToDOT(g AstraGraph) string {
	var b strings.Builder
	b.WriteString("digraph astra {\n")

	// Artifacts
	for _, n := range g.Artifacts {
		b.WriteString(fmt.Sprintf(
			"  \"%s\" [label=\"%s\\n(Artifact)\" shape=box];\n",
			n.ID, n.Name))
	}
	// Steps
	for _, n := range g.Steps {
		b.WriteString(fmt.Sprintf(
			"  \"%s\" [label=\"%s\\n(Step)\" shape=diamond];\n",
			n.ID, n.Command))
	}
	// Principals
	for _, n := range g.Principals {
		b.WriteString(fmt.Sprintf(
			"  \"%s\" [label=\"%s\\n(Principal)\" shape=oval];\n",
			n.ID, n.ID))
	}
	// Resources
	for _, n := range g.Resources {
		b.WriteString(fmt.Sprintf(
			"  \"%s\" [label=\"%s\\n(Resource)\" shape=hexagon];\n",
			n.ID, n.ID))
	}

	// Edges
	for _, e := range g.Edges {
		b.WriteString(fmt.Sprintf(
			"  \"%s\" -> \"%s\" [label=\"%s\"];\n",
			e.Source, e.Target, e.Relation))
	}
	b.WriteString("}\n")
	return b.String()
}

func NewAstraGraph() AstraGraph {
	return AstraGraph{
		Artifacts:  map[string]Artifact{},
		Steps:      map[string]Step{},
		Principals: map[string]Principal{},
		Resources:  map[string]Resource{},
		Edges:      []Edge{},
	}
}

func FromExport(e ExportGraph) AstraGraph {
	g := NewAstraGraph()

	for _, a := range e.Artifacts {
		g.Artifacts[a.ID] = a
	}
	for _, s := range e.Steps {
		g.Steps[s.ID] = s
	}
	for _, p := range e.Principals {
		g.Principals[p.ID] = p
	}
	for _, r := range e.Resources {
		g.Resources[r.ID] = r
	}
	for _, ed := range e.Edges {
		g.Edges = append(g.Edges, ed)
	}

	return g
}

// ExportGraph is used for JSON output / deterministic ordering
type ExportGraph struct {
	Artifacts  []Artifact  `json:"artifacts"`
	Steps      []Step      `json:"steps"`
	Principals []Principal `json:"principals"`
	Resources  []Resource  `json:"resources"`
	Edges      []Edge      `json:"edges"`
}

func ToExport(g AstraGraph) ExportGraph {
	var out ExportGraph

	// Artifacts
	for _, a := range g.Artifacts {
		out.Artifacts = append(out.Artifacts, a)
	}
	sort.Slice(out.Artifacts, func(i, j int) bool {
		return out.Artifacts[i].ID < out.Artifacts[j].ID
	})

	// Steps
	for _, s := range g.Steps {
		out.Steps = append(out.Steps, s)
	}
	sort.Slice(out.Steps, func(i, j int) bool {
		return out.Steps[i].ID < out.Steps[j].ID
	})

	// Principals
	for _, p := range g.Principals {
		out.Principals = append(out.Principals, p)
	}
	sort.Slice(out.Principals, func(i, j int) bool {
		return out.Principals[i].ID < out.Principals[j].ID
	})

	// Resources
	for _, r := range g.Resources {
		out.Resources = append(out.Resources, r)
	}
	sort.Slice(out.Resources, func(i, j int) bool {
		return out.Resources[i].ID < out.Resources[j].ID
	})

	// Edges
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, e)
	}
	sort.Slice(out.Edges, func(i, j int) bool {
		if out.Edges[i].Source != out.Edges[j].Source {
			return out.Edges[i].Source < out.Edges[j].Source
		}
		if out.Edges[i].Relation != out.Edges[j].Relation {
			return out.Edges[i].Relation < out.Edges[j].Relation
		}
		return out.Edges[i].Target < out.Edges[j].Target
	})

	return out
}

//TODO
/*
Validates DAG properties
- Detect and reject cycles in the AStRA graph
- Enforce required structural invariants, e.g.:
		every Artifact has ≥1 producing Step
		every Step is associated with ≥1 Principal and Resource
		Ensure edge directions respect causal semantics

Add temporal reasoning
- Validate that edge directions are consistent with timestamps
- Detect temporal anomalies (e.g., artifact consumed before production)
- Enable reasoning about:
- Check step ordering -> run step to step order and verify the temporal order

*/
