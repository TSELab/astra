/*
Package packages implements the PackagesParser for AStRA.

PackagesParser parses Debian Packages index files (binary package index from
a Debian archive or snapshot). Each stanza in the file produces one Record.

For each package stanza:
  - The act of publishing the package is represented as a step with ID
    step:archive:deb/debian/<name>@<version>?arch=<arch>.
  - "Debian" is the principal.
  - The archive URL is a resource (carries_out the step).
  - The packaged .deb is the output artifact with ID
    artifact:pkg:deb/debian/<name>@<version>?arch=<arch> and SHA256 hash
    from the stanza. Completeness is "complete".

The output artifact ID uses the same scheme as the buildinfo parser, so the
same .deb node is shared when both sources are ingested for the same package.

Stanzas missing Package, Version, Architecture, or SHA256 are skipped.
*/
package packages

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	parser "github.com/TSELab/astra/internal/parser"
)

// PackagesParser implements parser.Parser for Debian Packages index files.
type PackagesParser struct {
	// ArchiveURL is the base URL of the Debian archive serving this Packages
	// file. It is used to construct the Resource node ID and URI.
	// Defaults to "https://deb.debian.org/debian" if empty.
	ArchiveURL string
}

// entry holds parsed fields for a single Packages stanza.
type entry struct {
	pkg      string // Package:
	version  string // Version: (raw, epoch preserved)
	arch     string // Architecture:
	sha256   string // SHA256:
	size     string // Size: (raw string)
	filename string // Filename: path within archive
}

func (p *PackagesParser) archiveURL() string {
	if p.ArchiveURL != "" {
		return p.ArchiveURL
	}
	return "https://deb.debian.org/debian"
}

// Parse reads a Debian Packages file from r and returns a Mapped containing
// one Record per valid package stanza.
func (p *PackagesParser) Parse(r io.Reader) (parser.Mapped, error) {
	entries, err := parseStanzas(r)
	if err != nil {
		return parser.Mapped{}, err
	}

	archiveURL := p.archiveURL()
	var records []parser.Record
	for _, e := range entries {
		records = append(records, entryToRecord(e, archiveURL))
	}

	return parser.Mapped{
		Source:       "debian-packages",
		NormalizedAt: time.Now().Unix(),
		Mapped:       records,
	}, nil
}

// parseStanzas tokenizes r into stanzas and returns the valid entries.
func parseStanzas(r io.Reader) ([]entry, error) {
	sc := bufio.NewScanner(r)
	// Packages files contain long Description fields; raise the buffer to 1 MiB.
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	var entries []entry
	current := map[string]string{}

	flush := func() {
		e := extractEntry(current)
		if e.pkg != "" && e.version != "" && e.arch != "" && e.sha256 != "" {
			entries = append(entries, e)
		}
		current = map[string]string{}
	}

	var lastKey string
	for sc.Scan() {
		line := sc.Text()

		if line == "" {
			// Blank line = stanza boundary
			flush()
			lastKey = ""
			continue
		}

		// Continuation line (starts with space or tab): belongs to lastKey.
		// We don't need any multi-line field values, so just skip.
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			_ = lastKey
			continue
		}

		// Field line: "Key: value"
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		key := line[:colon]
		val := strings.TrimSpace(line[colon+1:])
		current[key] = val
		lastKey = key
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	// Flush any trailing stanza not followed by a blank line.
	flush()
	return entries, nil
}

// extractEntry converts a raw field map into an entry struct.
func extractEntry(fields map[string]string) entry {
	return entry{
		pkg:      fields["Package"],
		version:  fields["Version"],
		arch:     fields["Architecture"],
		sha256:   fields["SHA256"],
		size:     fields["Size"],
		filename: fields["Filename"],
	}
}

// entryToRecord builds an AStRA parser.Record from a single package entry.
func entryToRecord(e entry, archiveURL string) parser.Record {
	purl := fmt.Sprintf("pkg:deb/debian/%s@%s?arch=%s", e.pkg, e.version, e.arch)
	artifactID := "artifact:" + purl

	resourceID := "archive:debian:" + archiveURL

	return parser.Record{
		Step: parser.StepItem{
			ID:    fmt.Sprintf("step:archive:deb/debian/%s@%s?arch=%s", e.pkg, e.version, e.arch),
			Label: "Archived in Debian distribution",
			Kind:  "archive",
			Attrs: map[string]string{
				"command":      "archive",
				"archive_path": e.filename,
			},
		},
		Principal: parser.PrincipalItem{
			ID:    "principal:Debian",
			Label: "Debian Archive",
			Kind:  "principal",
			Attrs: map[string]string{},
		},
		ArtifactsIn: nil,
		ArtifactsOut: []parser.ArtifactItem{
			{
				ID:           artifactID,
				Label:        fmt.Sprintf("%s_%s_%s.deb", e.pkg, e.version, e.arch),
				Kind:         "deb",
				Completeness: "complete",
				Attrs: map[string]string{
					"purl":     purl,
					"hash":     e.sha256,
					"size":     e.size,
					"filename": e.filename,
					"version":  e.version,
				},
			},
		},
		Resources: []parser.ResourceItem{
			{
				ID:    resourceID,
				Label: archiveURL,
				Kind:  "archive",
				Attrs: map[string]string{
					"uri":    archiveURL,
					"format": "Packages",
				},
			},
		},
	}
}
