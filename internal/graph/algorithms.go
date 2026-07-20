package graph

import (
	"fmt"
	"sort"

	gonumgraph "gonum.org/v1/gonum/graph"
	network "gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/topo"
	"gonum.org/v1/gonum/graph/traverse"
)

// TopologicalOrder returns a topological ordering of the AStRA graph,
// listing nodes from upstream ("from") to downstream ("to").
//
// If the graph contains a cycle (i.e., is not a DAG), an error is returned.
//
// In AStRA, a valid topological order represents a valid execution order
// of supply chain steps and dependencies. Violations may indicate structural
// inconsistencies or potential integrity issues in the pipeline.
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

// ReachableFrom returns all nodes that are reachable from the given start node
// using a breadth-first traversal of the AStRA graph.
//
// The function returns a list of AStRA node IDs excluding the start node itself.
// If the provided startID does not exist in the graph, an error is returned.
//
// In the context of AStRA, this operation is useful for modeling downstream
// impact and attack propagation. For example, if a node (e.g., a credential,
// build step, or resource) is compromised, ReachableFrom identifies all
// artifacts and steps that may be affected (i.e., the blast radius).
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

type Degree struct {
	In    int
	Out   int
	Total int
}

// DegreeAnalysis computes the in-degree, out-degree, and total degree for each
// node in the AStRA graph.
//
// The result is a map from AStRA node ID to its Degree metrics:
//   - In: number of incoming edges
//   - Out: number of outgoing edges
//   - Total: sum of in and out degrees
//
// In AStRA-based analysis, degree metrics help identify structurally important
// nodes. High in-degree may indicate heavily depended-upon artifacts or resources,
// while high out-degree may indicate nodes that influence many downstream elements.
func DegreeAnalysis(g AstraGraph) map[string]Degree {
	dg, ids := ToGonum(g)

	result := make(map[string]Degree)

	nodes := dg.Nodes()
	for nodes.Next() {
		n := nodes.Node()
		id := n.ID()

		from := dg.From(id)
		out := 0
		for from.Next() {
			out++
		}

		to := dg.To(id)
		in := 0
		for to.Next() {
			in++
		}

		astraID := ids.IntToString[id]

		result[astraID] = Degree{
			In:    in,
			Out:   out,
			Total: in + out,
		}
	}

	return result
}

// SourceNodes finds the soure nodes in the graoh
// a source node is where the flow starts in the graph
// which are nodes with in-degree =0 in the AStRA graph.
//
// The result is an array of source nodes
// In AStRA-based analysis, source nodes help identify
func SourceNodes(g AstraGraph) []string {
	return filterDegree(g, func(d Degree) bool {
		return d.In == 0
	})
}

// SinkNodes finds the nodes with out-degree =0 in the AStRA graph.
//
// The result is an array of sink nodes
func SinkNodes(g AstraGraph) []string {
	return filterDegree(g, func(d Degree) bool {
		return d.Out == 0
	})
}

// DeadNodes finds the nodes with in-degree =0, and out-degree=0 in the AStRA graph.
//
// The result is an array of dead nodes
func DeadNodes(g AstraGraph) []string {
	return filterDegree(g, func(d Degree) bool {
		return d.In == 0 && d.Out == 0
	})
}

// filterDegree returns all nodes whose Degree satisfies keep.
func filterDegree(g AstraGraph, keep func(Degree) bool) []string {
	degrees := DegreeAnalysis(g)

	var nodes []string
	for id, degree := range degrees {
		if keep(degree) {
			nodes = append(nodes, id)
		}
	}

	return nodes
}

// Ancestors returns all upstream nodes that can reach start.
// It follows incoming edges recursively using the Gonum graph representation.
// If start is not found in the graph, it returns an empty slice.
// The returned slice does not include start itself.
func Ancestors(g AstraGraph, start string) []string {
	dg, ids := ToGonum(g)

	startID, ok := ids.StringToInt[start]
	if !ok {
		return []string{}
	}

	visited := make(map[int64]bool)
	var result []string

	var dfs func(int64)
	dfs = func(id int64) {
		to := dg.To(id) // incoming neighbors = parents

		for to.Next() {
			parent := to.Node().ID()

			if visited[parent] {
				continue
			}

			visited[parent] = true
			result = append(result, ids.IntToString[parent])

			dfs(parent)
		}
	}

	dfs(startID)

	return result
}

// ShortestPath computes the shortest path between two nodes in the AStRA graph
// using Dijkstra's algorithm.
//
// It returns the sequence of AStRA node IDs representing the path from fromID
// to toID. The second return value indicates whether a path exists.
//
// If either node does not exist or no path is found, the function returns (nil, false).
//
// In AStRA, shortest paths can be used to analyze how quickly a compromise can
// propagate through the supply chain, or to trace the minimal sequence of steps
// linking a principal to a deployed artifact.
func ShortestPath(g AstraGraph, fromID string, toID string) ([]string, bool) {
	dg, ids := ToGonum(g)

	fromInt, ok := ids.StringToInt[fromID]
	if !ok {
		return nil, false
	}

	toInt, ok := ids.StringToInt[toID]
	if !ok {
		return nil, false
	}

	from := dg.Node(fromInt)
	to := dg.Node(toInt)
	if from == nil || to == nil {
		return nil, false
	}

	shortest, _ := path.DijkstraFrom(from, dg).To(to.ID())

	out := make([]string, 0, len(shortest))
	for _, n := range shortest {
		out = append(out, ids.IntToString[n.ID()])
	}

	return out, len(out) > 0
}

// BetweennessCentrality computes the betweenness centrality score for each node
// in the AStRA graph.
//
// Betweenness centrality measures how frequently a node appears on the shortest
// paths between other pairs of nodes. The result is a map from AStRA node ID to
// its centrality score.
//
// In AStRA-based risk analysis, nodes with high betweenness represent critical
// control points in the supply chain. These nodes act as bridges between different
// parts of the graph, and their compromise may allow an attacker to influence or
// disrupt multiple downstream paths.
// This operation can be expensive on large graphs because it computes shortest
// path information across many source-target pairs.`
func BetweennessCentrality(g AstraGraph) map[string]float64 {
	dg, ids := ToGonum(g)

	raw := network.Betweenness(dg)

	result := make(map[string]float64)
	for nid, score := range raw {
		result[ids.IntToString[nid]] = score
	}

	return result
}

type BetweennessResult struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

// Rank betweenness centrality results by centrality score in descending order
func RankBetweenness(bw map[string]float64) []BetweennessResult {
	results := make([]BetweennessResult, 0, len(bw))

	for gonumID, score := range bw {
		results = append(results, BetweennessResult{
			ID:    gonumID,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}
