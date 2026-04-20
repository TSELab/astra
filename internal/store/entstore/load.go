// Package entstore implements persistence logic for loading
// AStRA graphs from the database.
//
// This file contains logic for reconstructing graph structures
// from stored database records.
package entstore

import (
	"context"
	"fmt"

	"github.com/TSELab/astra/internal/graph"
	genent "github.com/TSELab/astra/internal/store/ent"
)

// LoadGraph retrieves a stored graph from the database and
// reconstructs it into an in-memory AstraGraph representation.
//
// The graphID parameter can be used to scope the query if
// multiple graphs are stored.
func (s *Store) LoadGraph(ctx context.Context, graphID string) (graph.AstraGraph, error) {
	var g graph.AstraGraph

	_ = graphID // reserved for future multi-graph support

	arts, err := loadArtifacts(ctx, s.client)
	if err != nil {
		return g, err
	}
	steps, err := loadSteps(ctx, s.client)
	if err != nil {
		return g, err
	}
	principals, err := loadPrincipals(ctx, s.client)
	if err != nil {
		return g, err
	}
	resources, err := loadResources(ctx, s.client)
	if err != nil {
		return g, err
	}
	edges, err := loadEdges(ctx, s.client)
	if err != nil {
		return g, err
	}

	g.Artifacts = arts
	g.Steps = steps
	g.Principals = principals
	g.Resources = resources
	g.Edges = edges

	return g, nil
}

func loadArtifacts(ctx context.Context, client *genent.Client) ([]graph.Artifact, error) {
	rows, err := client.Artifact.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query artifacts: %w", err)
	}

	out := make([]graph.Artifact, 0, len(rows))
	for _, a := range rows {
		out = append(out, graph.Artifact{
			ID:       a.AstraID,
			Kind:     a.Kind,
			Name:     a.Name,
			Version:  a.Version,
			Hash:     a.Hash,
			Size:     a.Size,
			Metadata: a.Metadata,
		})
	}
	return out, nil
}

func loadSteps(ctx context.Context, client *genent.Client) ([]graph.Step, error) {
	rows, err := client.Step.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query steps: %w", err)
	}

	out := make([]graph.Step, 0, len(rows))
	for _, st := range rows {
		out = append(out, graph.Step{
			ID:          st.AstraID,
			Command:     st.Command,
			Timestamp:   st.Timestamp,
			Arch:        st.Arch,
			Environment: st.Environment,
			Metadata:    st.Metadata,
		})
	}
	return out, nil
}

func loadPrincipals(ctx context.Context, client *genent.Client) ([]graph.Principal, error) {
	rows, err := client.Principal.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query principals: %w", err)
	}

	out := make([]graph.Principal, 0, len(rows))
	for _, p := range rows {
		out = append(out, graph.Principal{
			ID:       p.AstraID,
			Name:     p.Name,
			Trust:    p.Trust,
			Builder:  p.Builder,
			Metadata: p.Metadata,
		})
	}
	return out, nil
}

func loadResources(ctx context.Context, client *genent.Client) ([]graph.Resource, error) {
	rows, err := client.Resource.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query resources: %w", err)
	}

	out := make([]graph.Resource, 0, len(rows))
	for _, r := range rows {
		out = append(out, graph.Resource{
			ID:       r.AstraID,
			Type:     r.Type,
			URI:      r.URI,
			Format:   r.Format,
			Metadata: r.Metadata,
		})
	}
	return out, nil
}

func loadEdges(ctx context.Context, client *genent.Client) ([]graph.Edge, error) {
	rows, err := client.Edge.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}

	out := make([]graph.Edge, 0, len(rows))
	for _, e := range rows {
		out = append(out, graph.Edge{
			Source:   e.Source,
			Target:   e.Target,
			Relation: e.Relation,
		})
	}
	return out, nil
}
