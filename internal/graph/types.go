// internal/graph/types.go
package graph

type NodeType string

const (
	ArtifactNode  NodeType = "artifact"
	StepNode      NodeType = "step"
	ResourceNode  NodeType = "resource"
	PrincipalNode NodeType = "principal"
)

// Completeness states for graph nodes.
const (
	Complete   = "complete"
	Incomplete = "incomplete"
)

type Node interface {
	GetID() string
	GetType() NodeType
}

type Artifact struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Hash         string            `json:"hash,omitempty"`
	Size         int64             `json:"size,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Completeness string            `json:"completeness,omitempty"`
}

func (a Artifact) GetID() string     { return a.ID }
func (a Artifact) GetType() NodeType { return ArtifactNode }

type Step struct {
	ID           string            `json:"id"`
	Command      string            `json:"command"`
	Timestamp    string            `json:"timestamp"`
	Arch         string            `json:"architecture"`
	Environment  map[string]string `json:"environment"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Completeness string            `json:"completeness,omitempty"`
}

func (s Step) GetID() string     { return s.ID }
func (s Step) GetType() NodeType { return StepNode }

type Resource struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	URI          string            `json:"uri"`
	Format       string            `json:"format"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Completeness string            `json:"completeness,omitempty"`
}

func (r Resource) GetID() string     { return r.ID }
func (r Resource) GetType() NodeType { return ResourceNode }

type Principal struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Trust        string            `json:"trust_level"`
	Builder      string            `json:"builder"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Completeness string            `json:"completeness,omitempty"`
}

func (p Principal) GetID() string     { return p.ID }
func (p Principal) GetType() NodeType { return PrincipalNode }

type Edge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

type AstraGraph struct {
	Artifacts  map[string]Artifact  `json:"artifacts"`
	Steps      map[string]Step      `json:"steps"`
	Resources  map[string]Resource  `json:"resources"`
	Principals map[string]Principal `json:"principals"`
	Edges      []Edge               `json:"edges"`
}
