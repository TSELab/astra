package graph

import (
	"hash/fnv"

	"gonum.org/v1/gonum/graph/simple"
)

// NodeMeta stores metadata about an AStRA node after it is converted
// into a Gonum node.
type NodeMeta struct {
	ID   string
	Type NodeType
}

// IDMap keeps the relationship between AStRA string IDs and Gonum int64 IDs.
type IDMap struct {
	StringToInt map[string]int64
	IntToString map[int64]string
	IntToMeta   map[int64]NodeMeta
}

// ToGonum converts an AStRA graph into a Gonum directed graph.
//
// Gonum nodes use int64 IDs, so this function also returns an IDMap
// that lets callers translate between AStRA string IDs and Gonum IDs.
func ToGonum(g AstraGraph) (*simple.DirectedGraph, IDMap) {
	dg := simple.NewDirectedGraph()

	ids := IDMap{
		StringToInt: map[string]int64{},
		IntToString: map[int64]string{},
		IntToMeta:   map[int64]NodeMeta{},
	}

	addNode := func(n Node) {
		sid := n.GetID()
		nid := stableIntID(sid)

		ids.StringToInt[sid] = nid
		ids.IntToString[nid] = sid
		ids.IntToMeta[nid] = NodeMeta{
			ID:   sid,
			Type: n.GetType(),
		}

		dg.AddNode(simple.Node(nid))
	}

	for _, n := range g.AllNodes() {
		addNode(n)
	}

	for _, e := range g.Edges {
		fromID, ok1 := ids.StringToInt[e.Source]
		toID, ok2 := ids.StringToInt[e.Target]
		if !ok1 || !ok2 {
			continue
		}

		from := dg.Node(fromID)
		to := dg.Node(toID)
		if from == nil || to == nil {
			continue
		}

		dg.SetEdge(dg.NewEdge(from, to))
	}

	return dg, ids
}

// stableIntID converts a string ID into a stable positive int64 ID.
//
// This is needed because Gonum nodes are identified by int64 values,
// while AStRA nodes use string IDs.
// For the same input, the function returns the same ID every time.
func stableIntID(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}
