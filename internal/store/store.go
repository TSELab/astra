// Package store defines abstract interfaces for graph persistence.
//
// This abstraction decouples the application logic from the
// underlying storage implementation (e.g., Ent, in-memory, or others).
package store

import (
	"context"

	"github.com/TSELab/astra/internal/graph"
)

// Store defines the interface for persisting and retrieving
// AStRA graphs.
//
// Implementations are responsible for handling storage-specific
// concerns such as transactions and schema mapping.
type Store interface {
	SaveGraph(ctx context.Context, g graph.AstraGraph) error
	LoadGraph(ctx context.Context, graphID string) (graph.AstraGraph, error)
	Close() error
}
