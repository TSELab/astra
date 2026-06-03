package parser

import "io"

// Parser is the interface every parser must satisfy
type Parser interface {
	Parse(r io.Reader) (Mapped, error)
}

type StepItem struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Kind         string            `json:"kind"`
	Attrs        map[string]string `json:"attrs"`
	Completeness string            `json:"completeness,omitempty"`
}

type PrincipalItem struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Kind         string            `json:"kind"`
	Attrs        map[string]string `json:"attrs"`
	Completeness string            `json:"completeness,omitempty"`
}

type ArtifactItem struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Kind         string            `json:"kind"`
	Attrs        map[string]string `json:"attrs"`
	Completeness string            `json:"completeness,omitempty"`
}

type ResourceItem struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Kind         string            `json:"kind"`
	Attrs        map[string]string `json:"attrs"`
	Completeness string            `json:"completeness,omitempty"`
}

// Record holds one unit of parsed provenance
type Record struct {
	Step         StepItem       `json:"step"`
	Principal    PrincipalItem  `json:"principal"`
	ArtifactsIn  []ArtifactItem `json:"artifacts_in"`
	ArtifactsOut []ArtifactItem `json:"artifacts_out"`
	Resources    []ResourceItem `json:"resources"`
	// Dependencies are runtime or deployment-time artifact requirements.
	// The mapper emits a "depends" edge from each ArtifactsOut item to each
	// dependency, modelling artifact-to-artifact relationships without going
	// through a step.
	Dependencies []ArtifactItem `json:"dependencies,omitempty"`
}

// Mapped is the top-level output of a parser: Records plus metadata.
type Mapped struct {
	Mapped       []Record `json:"mapped"`
	Source       string   `json:"source"`
	NormalizedAt int64    `json:"normalized_at"`
}
