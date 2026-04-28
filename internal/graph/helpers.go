// internal/graph/helpers.go
package graph

func (g AstraGraph) AllNodes() []Node {
	var nodes []Node

	for _, a := range g.Artifacts {
		nodes = append(nodes, a)
	}
	for _, s := range g.Steps {
		nodes = append(nodes, s)
	}
	for _, r := range g.Resources {
		nodes = append(nodes, r)
	}
	for _, p := range g.Principals {
		nodes = append(nodes, p)
	}

	return nodes
}

func (g AstraGraph) NodeByID(id string) (Node, bool) {
	if n, ok := g.Artifacts[id]; ok {
		return n, true
	}
	if n, ok := g.Steps[id]; ok {
		return n, true
	}
	if n, ok := g.Resources[id]; ok {
		return n, true
	}
	if n, ok := g.Principals[id]; ok {
		return n, true
	}
	return nil, false
}
