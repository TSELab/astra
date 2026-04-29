package graph

import (
	"fmt"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"
)

// TopologicalOrder performs a topological sort of the directed graph g returning the 'from' to 'to'
// sort order. If a topological ordering is not possible, not a DAG error is returned
func TopologicalOrder(g AstraGraph) ([]string, error) {
	dg, ids := ToGonum(g)

	nodes, err := topo.Sort(dg)
	if err != nil {
		return nil, fmt.Errorf("graph is not a DAG: %w", err)
	}

	order := make([]string, 0, len(nodes))
	for _, n := range nodes {
		order = append(order, ids.IntToString[n.ID()])
	}

	return order, nil
}

func ReachableFrom(g AstraGraph, startID string) ([]string, error) {
	dg, ids := ToGonum(g)

	startIntID, ok := ids.StringToInt[startID]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", startID)
	}

	start := dg.Node(startIntID)
	if start == nil {
		return nil, fmt.Errorf("start node missing")
	}

	var out []string
	bfs := traverse.BreadthFirst{}

	bfs.Walk(dg, start, func(n gonumgraph.Node, depth int) bool {
		if n.ID() != start.ID() {
			out = append(out, ids.IntToString[n.ID()])
		}
		return false
	})

	return out, nil
}
