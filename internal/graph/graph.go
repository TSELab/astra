package graph

//	Attrs map[string]string `json:"attrs"`

type Artifact struct {
	ID       string            `json:"id"`
	Kind     string            `json:"kind"`
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Hash     string            `json:"hash,omitempty"`
	Size     int64             `json:"size,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Step struct {
	ID          string            `json:"id"`
	Command     string            `json:"command"`
	Timestamp   string            `json:"timestamp"`
	Arch        string            `json:"architecture"`
	Environment map[string]string `json:"environment"`
	Consumed    []string          `json:"consumed_resources"`
	Outputs     []string          `json:"outputs"`
}

type Principal struct {
	ID       string            `json:"id"`
	Trust    string            `json:"trust_level"`
	Builder  string            `json:"builder"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Resource struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	URI    string `json:"uri"`
	Format string `json:"format"`
	UsedBy string `json:"used_by,omitempty"`
}

type Edge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

type AstraGraph struct {
	Artifacts  []Artifact  `json:"artifacts"`
	Steps      []Step      `json:"step"`
	Principals []Principal `json:"principals"`
	Resources  []Resource  `json:"resources"`
	Edges      []Edge      `json:"edges"`
}
