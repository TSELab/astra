package graph

import (
	"fmt"

	"gonum.org/v1/gonum/graph/topo"
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
