// internal/graph/graph.go
package graph

import (
	"fmt"
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
