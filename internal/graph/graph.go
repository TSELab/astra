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

// constructor to generate new AstraGraph
func NewAstraGraph() AstraGraph {
	return AstraGraph{
		Artifacts:  map[string]Artifact{},
		Steps:      map[string]Step{},
		Principals: map[string]Principal{},
		Resources:  map[string]Resource{},
		Edges:      []Edge{},
	}
}

// ExportGraph is used for JSON output / deterministic ordering
type ExportGraph struct {
	Artifacts  []Artifact  `json:"artifacts"`
	Steps      []Step      `json:"steps"`
	Principals []Principal `json:"principals"`
	Resources  []Resource  `json:"resources"`
	Edges      []Edge      `json:"edges"`
}

// ToExport converts AstraGraph to ExportGraph that is used for JSON output / deterministic ordering
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

// FromExport convert ExportGraph to AstraGraph, used to visualize/analyze json format graphs
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

// CompletenessRank returns a numeric rank for ordering completeness states.
// complete(1) outranks incomplete(0).
func CompletenessRank(c string) int {
	if c == Complete {
		return 1
	}
	return 0
}

// Merge combines two AstraGraph structs into one.
// Nodes are merged by ID with completeness-aware upgrade:
// an incoming node replaces a stored node only when it is more complete.
// Edges are deduplicated by source|relation|target.
func Merge(a, b AstraGraph) AstraGraph {
	for id, v := range b.Artifacts {
		if existing, ok := a.Artifacts[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Artifacts[id] = v
		}
	}
	for id, v := range b.Steps {
		if existing, ok := a.Steps[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Steps[id] = v
		}
	}
	for id, v := range b.Resources {
		if existing, ok := a.Resources[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Resources[id] = v
		}
	}
	for id, v := range b.Principals {
		if existing, ok := a.Principals[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Principals[id] = v
		}
	}

	seen := map[string]bool{}
	for _, e := range a.Edges {
		seen[e.Source+"|"+e.Relation+"|"+e.Target] = true
	}
	for _, e := range b.Edges {
		k := e.Source + "|" + e.Relation + "|" + e.Target
		if !seen[k] {
			seen[k] = true
			a.Edges = append(a.Edges, e)
		}
	}

	return a
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
