package entstore

import (
	"context"
	"fmt"

	"github.com/TSELab/astra/internal/graph"
	genent "github.com/TSELab/astra/internal/store/ent"
	"github.com/TSELab/astra/internal/store/ent/artifact"
	"github.com/TSELab/astra/internal/store/ent/edge"
	"github.com/TSELab/astra/internal/store/ent/principal"
	"github.com/TSELab/astra/internal/store/ent/resource"
	"github.com/TSELab/astra/internal/store/ent/step"
)

// SaveGraph persists an AStRA graph into the database.
//
// The operation runs in a transaction so that either all nodes/edges
// are written successfully, or none are.
func (s *Store) SaveGraph(ctx context.Context, g graph.AstraGraph) (err error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = saveArtifacts(ctx, tx, g.Artifacts); err != nil {
		return err
	}
	if err = saveSteps(ctx, tx, g.Steps); err != nil {
		return err
	}
	if err = savePrincipals(ctx, tx, g.Principals); err != nil {
		return err
	}
	if err = saveResources(ctx, tx, g.Resources); err != nil {
		return err
	}
	if err = saveEdges(ctx, tx, g.Edges); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func saveArtifacts(ctx context.Context, tx *genent.Tx, items map[string]graph.Artifact) error {
	for _, a := range items {
		incoming := artifact.Completeness(a.Completeness)
		if incoming == "" {
			incoming = artifact.CompletenessComplete
		}

		existing, err := tx.Artifact.
			Query().
			Where(artifact.AstraIDEQ(a.ID)).
			Only(ctx)

		switch {
		case genent.IsNotFound(err):
			_, err = tx.Artifact.
				Create().
				SetAstraID(a.ID).
				SetKind(a.Kind).
				SetName(a.Name).
				SetVersion(a.Version).
				SetHash(a.Hash).
				SetSize(a.Size).
				SetMetadata(a.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create artifact %q: %w", a.ID, err)
			}

		case err != nil:
			return fmt.Errorf("query artifact %q: %w", a.ID, err)

		default:
			if graph.CompletenessRank(string(incoming)) <= graph.CompletenessRank(string(existing.Completeness)) {
				continue // stored is at least as complete — skip
			}
			_, err = existing.
				Update().
				SetKind(a.Kind).
				SetName(a.Name).
				SetVersion(a.Version).
				SetHash(a.Hash).
				SetSize(a.Size).
				SetMetadata(a.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("update artifact %q: %w", a.ID, err)
			}
		}
	}
	return nil
}

func saveSteps(ctx context.Context, tx *genent.Tx, items map[string]graph.Step) error {
	for _, st := range items {
		incoming := step.Completeness(st.Completeness)
		if incoming == "" {
			incoming = step.CompletenessComplete
		}

		existing, err := tx.Step.
			Query().
			Where(step.AstraIDEQ(st.ID)).
			Only(ctx)

		switch {
		case genent.IsNotFound(err):
			_, err = tx.Step.
				Create().
				SetAstraID(st.ID).
				SetCommand(st.Command).
				SetTimestamp(st.Timestamp).
				SetArch(st.Arch).
				SetEnvironment(st.Environment).
				SetMetadata(st.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create step %q: %w", st.ID, err)
			}

		case err != nil:
			return fmt.Errorf("query step %q: %w", st.ID, err)

		default:
			if graph.CompletenessRank(string(incoming)) <= graph.CompletenessRank(string(existing.Completeness)) {
				continue
			}
			_, err = existing.
				Update().
				SetCommand(st.Command).
				SetTimestamp(st.Timestamp).
				SetArch(st.Arch).
				SetEnvironment(st.Environment).
				SetMetadata(st.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("update step %q: %w", st.ID, err)
			}
		}
	}
	return nil
}

func savePrincipals(ctx context.Context, tx *genent.Tx, items map[string]graph.Principal) error {
	for _, p := range items {
		incoming := principal.Completeness(p.Completeness)
		if incoming == "" {
			incoming = principal.CompletenessComplete
		}

		existing, err := tx.Principal.
			Query().
			Where(principal.AstraIDEQ(p.ID)).
			Only(ctx)

		switch {
		case genent.IsNotFound(err):
			_, err = tx.Principal.
				Create().
				SetAstraID(p.ID).
				SetName(p.Name).
				SetTrust(p.Trust).
				SetBuilder(p.Builder).
				SetMetadata(p.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create principal %q: %w", p.ID, err)
			}

		case err != nil:
			return fmt.Errorf("query principal %q: %w", p.ID, err)

		default:
			if graph.CompletenessRank(string(incoming)) <= graph.CompletenessRank(string(existing.Completeness)) {
				continue
			}
			_, err = existing.
				Update().
				SetName(p.Name).
				SetTrust(p.Trust).
				SetBuilder(p.Builder).
				SetMetadata(p.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("update principal %q: %w", p.ID, err)
			}
		}
	}
	return nil
}

func saveResources(ctx context.Context, tx *genent.Tx, items map[string]graph.Resource) error {
	for _, r := range items {
		incoming := resource.Completeness(r.Completeness)
		if incoming == "" {
			incoming = resource.CompletenessComplete
		}

		existing, err := tx.Resource.
			Query().
			Where(resource.AstraIDEQ(r.ID)).
			Only(ctx)

		switch {
		case genent.IsNotFound(err):
			_, err = tx.Resource.
				Create().
				SetAstraID(r.ID).
				SetType(r.Type).
				SetURI(r.URI).
				SetFormat(r.Format).
				SetMetadata(r.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create resource %q: %w", r.ID, err)
			}

		case err != nil:
			return fmt.Errorf("query resource %q: %w", r.ID, err)

		default:
			if graph.CompletenessRank(string(incoming)) <= graph.CompletenessRank(string(existing.Completeness)) {
				continue
			}
			_, err = existing.
				Update().
				SetType(r.Type).
				SetURI(r.URI).
				SetFormat(r.Format).
				SetMetadata(r.Metadata).
				SetCompleteness(incoming).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("update resource %q: %w", r.ID, err)
			}
		}
	}
	return nil
}

func saveEdges(ctx context.Context, tx *genent.Tx, items []graph.Edge) error {
	for _, e := range items {
		existing, err := tx.Edge.
			Query().
			Where(
				edge.SourceEQ(e.Source),
				edge.TargetEQ(e.Target),
				edge.RelationEQ(e.Relation),
			).
			Only(ctx)

		switch {
		case genent.IsNotFound(err):
			_, err = tx.Edge.
				Create().
				SetSource(e.Source).
				SetTarget(e.Target).
				SetRelation(e.Relation).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create edge %q -> %q (%s): %w", e.Source, e.Target, e.Relation, err)
			}

		case err != nil:
			return fmt.Errorf("query edge %q -> %q (%s): %w", e.Source, e.Target, e.Relation, err)

		default:
			// Edge already exists; nothing to update.
			_ = existing
		}
	}
	return nil
}
