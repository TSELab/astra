package graph

import (
	"math"
	"reflect"
	"slices"
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

func TestDegreeAnalysis(t *testing.T) {
	g := NewAstraGraph()

	// Add nodes
	g.Artifacts["A"] = Artifact{ID: "A"}
	g.Artifacts["B"] = Artifact{ID: "B"}
	g.Artifacts["C"] = Artifact{ID: "C"}

	// Add edges: A→B, B→C, A→C
	g.Edges = append(g.Edges,
		Edge{Source: "A", Target: "B"},
		Edge{Source: "B", Target: "C"},
		Edge{Source: "A", Target: "C"},
	)

	degrees := DegreeAnalysis(g)

	tests := map[string]Degree{
		"A": {In: 0, Out: 2, Total: 2},
		"B": {In: 1, Out: 1, Total: 2},
		"C": {In: 2, Out: 0, Total: 2},
	}

	for node, expected := range tests {
		got, ok := degrees[node]
		if !ok {
			t.Fatalf("missing node %s in result", node)
		}

		if got != expected {
			t.Errorf("node %s: expected %+v, got %+v", node, expected, got)
		}
	}
}

func TestDegreeAnalysisMixedNodeTypes(t *testing.T) {
	g := NewAstraGraph()

	// Nodes from different AStRA types
	g.Principals["P1"] = Principal{ID: "P1"}
	g.Resources["R1"] = Resource{ID: "R1"}
	g.Steps["S1"] = Step{ID: "S1"}
	g.Artifacts["A1"] = Artifact{ID: "A1"}
	g.Artifacts["A2"] = Artifact{ID: "A2"}

	// AStRA-style causal chain:
	// P1 -> R1 -> S1
	// A1 -> S1
	// S1 -> A2
	g.Edges = append(g.Edges,
		Edge{Source: "P1", Target: "R1", Relation: "uses"},
		Edge{Source: "R1", Target: "S1", Relation: "carries_out"},
		Edge{Source: "A1", Target: "S1", Relation: "consumes"},
		Edge{Source: "S1", Target: "A2", Relation: "produces"},
	)

	got := DegreeAnalysis(g)

	expected := map[string]Degree{
		"P1": {In: 0, Out: 1, Total: 1},
		"R1": {In: 1, Out: 1, Total: 2},
		"S1": {In: 2, Out: 1, Total: 3},
		"A1": {In: 0, Out: 1, Total: 1},
		"A2": {In: 1, Out: 0, Total: 1},
	}

	for id, want := range expected {
		gotDegree, ok := got[id]
		if !ok {
			t.Fatalf("missing degree result for node %q", id)
		}

		if gotDegree != want {
			t.Errorf("node %q: expected %+v, got %+v", id, want, gotDegree)
		}
	}

	if len(got) != len(expected) {
		t.Errorf("expected %d degree results, got %d: %+v", len(expected), len(got), got)
	}
}

func TestShortestPath(t *testing.T) {
	// Build test graph
	g := NewAstraGraph()

	// Add nodes
	g.Principals["P1"] = Principal{ID: "P1"}
	g.Principals["P2"] = Principal{ID: "P2"}
	g.Resources["R1"] = Resource{ID: "R1"}
	g.Resources["R2"] = Resource{ID: "R2"}
	g.Steps["S1"] = Step{ID: "S1"}
	g.Steps["S2"] = Step{ID: "S2"}
	g.Artifacts["A1"] = Artifact{ID: "A1"}
	g.Artifacts["A2"] = Artifact{ID: "A2"}
	g.Artifacts["A3"] = Artifact{ID: "A3"}

	// AStRA-style causal chain:
	// P1 -> R1 -> S1
	// A1 -> S1
	// S1 -> A2
	// P2 -> R2 -> S2
	// A2 -> S2
	// S2 -> A3
	// Add edges (causal relationships)
	g.Edges = append(g.Edges,
		Edge{Source: "P1", Target: "R1", Relation: "uses"},
		Edge{Source: "R1", Target: "S1", Relation: "carries_out"},
		Edge{Source: "A1", Target: "S1", Relation: "consumes"},
		Edge{Source: "S1", Target: "A2", Relation: "produces"},
		Edge{Source: "P2", Target: "R2", Relation: "uses"},
		Edge{Source: "R2", Target: "S2", Relation: "carries_out"},
		Edge{Source: "A2", Target: "S2", Relation: "consumes"},
		Edge{Source: "S2", Target: "A3", Relation: "produces"},
	)

	// Run shortest path
	path, ok := ShortestPath(g, "P1", "A3")

	// Expected path
	expected := []string{
		"P1",
		"R1",
		"S1",
		"A2",
		"S2",
		"A3",
	}

	// Assertions
	if !ok {
		t.Fatalf("expected path to exist, but got none")
	}

	if !reflect.DeepEqual(path, expected) {
		t.Fatalf("unexpected path.\nGot: %v\nWant: %v", path, expected)
	}
}

func TestShortestPath_InvalidNode(t *testing.T) {
	g := NewAstraGraph()

	path, ok := ShortestPath(g, "missing", "x")

	if ok {
		t.Fatalf("expected failure for missing node, got %v", path)
	}
}

func TestShortestPath_NoPath(t *testing.T) {
	g := NewAstraGraph()

	g.Principals["P1"] = Principal{ID: "P1"}
	g.Artifacts["A1"] = Artifact{ID: "A1"}

	// no edges

	path, ok := ShortestPath(g, "P1", "A1")

	if ok {
		t.Fatalf("expected no path, got %v", path)
	}
}

func TestBetweennessCentrality_Star(t *testing.T) {
	g := NewAstraGraph()

	nodes := []string{"A", "B", "C", "D", "E"}
	for _, n := range nodes {
		g.Artifacts[n] = Artifact{ID: n}
	}

	//     A
	//     ↓
	// B → C → D
	//     ↑
	//     E
	g.Edges = []Edge{
		{"A", "C", ""},
		{"B", "C", ""},
		{"C", "D", ""},
		{"E", "C", ""},
	}

	bc := BetweennessCentrality(g)

	// Center should dominate
	center := bc["C"]

	// C = highest (critical bridge)
	// others = 0
	for n, v := range bc {
		if n == "C" {
			continue
		}
		if v != 0 {
			t.Errorf("expected %s = 0, got %v", n, v)
		}
	}
	if center <= 0 {
		t.Errorf("expected center > 0, got %v", center)
	}
}

func assertApproxEqual(t *testing.T, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
func TestBetweennessCentrality_ByHandRanking(t *testing.T) {
	g := NewAstraGraph()

	nodes := []string{"A", "B", "C", "D", "E"}
	for _, n := range nodes {
		g.Artifacts[n] = Artifact{ID: n}
	}

	// Diamond into a final sink:
	//
	//      B
	//    ↗   ↘
	// A       D → E
	//    ↘   ↗
	//      C
	//
	// By hand, shortest paths:
	// A→D has two shortest paths: A-B-D and A-C-D.
	// A→E has two shortest paths: A-B-D-E and A-C-D-E.
	// B→E goes through D.
	// C→E goes through D.
	//
	// So D should rank highest.
	// B and C should tie.
	// A and E should be zero.
	g.Edges = []Edge{
		{Source: "A", Target: "B"},
		{Source: "A", Target: "C"},
		{Source: "B", Target: "D"},
		{Source: "C", Target: "D"},
		{Source: "D", Target: "E"},
	}

	bc := BetweennessCentrality(g)

	if bc["D"] <= bc["B"] {
		t.Fatalf("expected D to rank higher than B, got D=%v B=%v", bc["D"], bc["B"])
	}

	if bc["D"] <= bc["C"] {
		t.Fatalf("expected D to rank higher than C, got D=%v C=%v", bc["D"], bc["C"])
	}

	assertApproxEqual(t, bc["B"], bc["C"])

	if bc["A"] != 0 {
		t.Fatalf("expected A to have betweenness 0, got %v", bc["A"])
	}

	if bc["E"] != 0 {
		t.Fatalf("expected E to have betweenness 0, got %v", bc["E"])
	}
}

func TestAncestors_Star(t *testing.T) {
	g := NewAstraGraph()

	nodes := []string{"A", "B", "C", "D", "E"}
	for _, n := range nodes {
		g.Artifacts[n] = Artifact{ID: n}
	}

	//     A
	//     ↓
	// B → C → D
	//     ↑
	//     E
	g.Edges = []Edge{
		{"A", "C", ""},
		{"B", "C", ""},
		{"C", "D", ""},
		{"E", "C", ""},
	}

	an := Ancestors(g, "C")

	ancestors := []string{"A", "B", "E"}

	if len(an) != len(ancestors) {
		t.Errorf("expected %v got %v", len(ancestors), len(an))
	}
	for _, v := range an {

		if slices.Contains(ancestors, v) {
			continue
		}
	}

}

func TestAncestors_Recursive(t *testing.T) {
	g := NewAstraGraph()

	nodes := []string{"A", "B", "C", "D", "E"}
	for _, n := range nodes {
		g.Artifacts[n] = Artifact{ID: n}
	}

	//   A → B → C → D
	//			 ↑
	//  		 E

	g.Edges = []Edge{
		{"A", "B", ""},
		{"B", "C", ""},
		{"C", "D", ""},
		{"E", "C", ""},
	}

	got := Ancestors(g, "D")

	want := []string{"A", "B", "C", "E"}

	if len(got) != len(want) {
		t.Errorf("expected %v got %v", len(want), len(got))
	}
	for _, v := range got {

		if slices.Contains(want, v) {
			continue
		}
	}

}

func TestDeadNodes(t *testing.T) {
	g := NewAstraGraph()

	g.Artifacts["A"] = Artifact{ID: "A"}
	g.Artifacts["B"] = Artifact{ID: "B"}
	g.Artifacts["C"] = Artifact{ID: "C"}
	g.Artifacts["D"] = Artifact{ID: "D"}

	g.Edges = []Edge{
		{Source: "A", Target: "B"},
	}

	got := DeadNodes(g) // rename to your actual function if different
	want := []string{"C", "D"}

	if len(got) != len(want) {
		t.Errorf("expected %v got %v", len(want), len(got))
	}
	for _, v := range got {

		if slices.Contains(want, v) {
			continue
		}
	}
}
