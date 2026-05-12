package mapper

import (
	"github.com/TSELab/astra/internal/graph"
	"github.com/TSELab/astra/internal/parser"
)

// ToAstraGraph converts parser.Mapped ([]Record) into a typed graph.AstraGraph.
// Relations emitted:
//
//	principal --uses--> resource
//	resource  --carries_out--> step
//	step  --consumes--> artifact
//	step      --produces--> artifact
func ToAstraGraph(m parser.Evidence) graph.AstraGraph {
	out := graph.NewAstraGraph()

	addEdge := func(src, dst, rel string, md map[string]string) {
		if src == "" || dst == "" || rel == "" {
			return
		}

		out.Edges = append(out.Edges, graph.Edge{
			Source:   src,
			Target:   dst,
			Relation: rel,
		})

		_ = md
	}

	for _, rec := range m.Records {
		if rec.Principal.ID != "" {
			if _, ok := out.Principals[rec.Principal.ID]; !ok {
				md := cloneMap(rec.Principal.Attrs)
				if md == nil {
					md = map[string]string{}
				}

				out.Principals[rec.Principal.ID] = graph.Principal{
					ID:       rec.Principal.ID,
					Trust:    "unknown",
					Builder:  "",
					Name:     rec.Principal.Label,
					Metadata: md,
				}
			}
		}

		if rec.Step.ID != "" {
			if _, ok := out.Steps[rec.Step.ID]; !ok {
				md := cloneMap(rec.Step.Attrs)
				if md == nil {
					md = map[string]string{}
				}

				out.Steps[rec.Step.ID] = graph.Step{
					ID:          rec.Step.ID,
					Command:     normalizeStepCommand(md),
					Timestamp:   normalizeTimestamp(md),
					Arch:        normalizeStepArch(md),
					Environment: map[string]string{},
					Metadata:    md,
				}
			}
		}

		for _, r := range rec.Resources {
			if r.ID == "" {
				continue
			}

			if _, ok := out.Resources[r.ID]; !ok {
				out.Resources[r.ID] = graph.Resource{
					ID:       r.ID,
					Type:     normalizeResourceType(r),
					URI:      normalizeResourceURI(r),
					Format:   normalizeResourceFormat(r),
					Metadata: cloneMap(r.Attrs),
				}
			}
		}

		for _, it := range rec.ArtifactsIn {
			if it.ID == "" {
				continue
			}

			if _, ok := out.Artifacts[it.ID]; !ok {
				out.Artifacts[it.ID] = normalizeArtifact(it)
			}

			addEdge(it.ID, rec.Step.ID, "consumes", nil)
		}

		for _, it := range rec.ArtifactsOut {
			if it.ID == "" {
				continue
			}

			if _, ok := out.Artifacts[it.ID]; !ok {
				out.Artifacts[it.ID] = normalizeArtifact(it)
			}

			addEdge(rec.Step.ID, it.ID, "produces", nil)
		}

		for _, r := range rec.Resources {
			addEdge(rec.Principal.ID, r.ID, "uses", nil)
			addEdge(r.ID, rec.Step.ID, "carries_out", nil)
		}
	}

	return out
}
