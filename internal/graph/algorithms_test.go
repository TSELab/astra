package graph

import (
	"testing"
)

func TestReachableFrom(t *testing.T) {
	g := NewAstraGraph()

	// A -> B -> D
	// A -> C

	g.Artifacts["A"] = Artifact{ID: "A"}
	g.Artifacts["B"] = Artifact{ID: "B"}
	g.Artifacts["C"] = Artifact{ID: "C"}
	g.Artifacts["D"] = Artifact{ID: "D"}

	g.Edges = []Edge{
		{Source: "A", Target: "B", Relation: "produces"},
		{Source: "A", Target: "C", Relation: "produces"},
		{Source: "B", Target: "D", Relation: "produces"},
	}

	reachable, err := ReachableFrom(g, "A")
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"B": true,
		"C": true,
		"D": true,
	}

	if len(reachable) != len(expected) {
		t.Fatalf("expected %d reachable nodes, got %d", len(expected), len(reachable))
	}

	for _, id := range reachable {
		if !expected[id] {
			t.Fatalf("unexpected reachable node: %s", id)
		}
	}
}

func TestTopologicalOrder(t *testing.T) {
	g := NewAstraGraph()

	// A -> B -> D
	// A -> C
	g.Artifacts["A"] = Artifact{ID: "A"}
	g.Artifacts["B"] = Artifact{ID: "B"}
	g.Artifacts["C"] = Artifact{ID: "C"}
	g.Artifacts["D"] = Artifact{ID: "D"}

	g.Edges = []Edge{
		{Source: "A", Target: "B", Relation: "produces"},
		{Source: "A", Target: "C", Relation: "produces"},
		{Source: "B", Target: "D", Relation: "produces"},
	}

	order, err := TopologicalOrder(g)
	if err != nil {
		t.Fatal(err)
	}

	pos := map[string]int{}
	for i, id := range order {
		pos[id] = i
	}

	assertBefore := func(a, b string) {
		if pos[a] >= pos[b] {
			t.Fatalf("expected %s before %s, got order %v", a, b, order)
		}
	}

	assertBefore("A", "B")
	assertBefore("A", "C")
	assertBefore("B", "D")
}
