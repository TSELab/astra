package parser

// Normalized is the container for normalized events
type Normalized struct {
	Source       string            `json:"source"`
	NormalizedAt int64             `json:"normalized_at"`
	Events       []NormalizedEvent `json:"events"`
}

// NormalizedEvent represents a single normalized commit event
type NormalizedEvent struct {
	EventType     string            `json:"event_type"`
	ID            string            `json:"id"`
	Timestamp     string            `json:"timestamp"`
	Author        string            `json:"author"`
	Email         string            `json:"email"`
	Message       string            `json:"message"`
	Repo          string            `json:"repo"`
	Ref           string            `json:"ref"`
	FilesAdded    []string          `json:"files_added,omitempty"`
	FilesModified []string          `json:"files_modified,omitempty"`
	FilesRemoved  []string          `json:"files_removed,omitempty"`
	Parents       []string          `json:"parents,omitempty"`
	Inputs        []string          `json:"inputs,omitempty"`
	Outputs       []string          `json:"outputs,omitempty"`
	Extras        map[string]string `json:"extras,omitempty"`
}

// Parser is the interface every parser must satisfy
type Parser interface {
	Parse(path string) (Normalized, error)
}
