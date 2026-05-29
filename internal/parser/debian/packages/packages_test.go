package packages_test

import (
	"strings"
	"testing"

	"github.com/TSELab/astra/internal/parser/debian/packages"
)

const singleStanza = `Package: libarchive13
Version: 3.6.2-1
Architecture: amd64
Filename: pool/main/liba/libarchive/libarchive13_3.6.2-1_amd64.deb
Size: 343196
SHA256: 82cc6d094f9b7c872e5bc5c4613151a7a8c20ac1a3d7d6c128dca64da618857b
Description: Multi-format archive and compression library
 This library provides an interface for reading and writing
 streaming archive formats.
`

func TestSingleEntry(t *testing.T) {
	p := &packages.PackagesParser{ArchiveURL: "https://deb.debian.org/debian"}
	mapped, err := p.Parse(strings.NewReader(singleStanza))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(mapped.Mapped) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mapped.Mapped))
	}
	rec := mapped.Mapped[0]

	// Artifact ID must match buildinfo output ID format exactly
	wantID := "artifact:pkg:deb/debian/libarchive13@3.6.2-1?arch=amd64"
	if rec.ArtifactsOut[0].ID != wantID {
		t.Errorf("artifact ID:\ngot  %q\nwant %q", rec.ArtifactsOut[0].ID, wantID)
	}

	// Hash must be present
	if rec.ArtifactsOut[0].Attrs["hash"] != "82cc6d094f9b7c872e5bc5c4613151a7a8c20ac1a3d7d6c128dca64da618857b" {
		t.Errorf("unexpected hash: %q", rec.ArtifactsOut[0].Attrs["hash"])
	}

	// Completeness must be complete
	if rec.ArtifactsOut[0].Completeness != "complete" {
		t.Errorf("expected completeness=complete, got %q", rec.ArtifactsOut[0].Completeness)
	}

	// Principal must match buildinfo principal
	if rec.Principal.ID != "principal:Debian" {
		t.Errorf("principal ID: got %q, want %q", rec.Principal.ID, "principal:Debian")
	}

	// Step ID format
	wantStep := "step:archive:deb/debian/libarchive13@3.6.2-1?arch=amd64"
	if rec.Step.ID != wantStep {
		t.Errorf("step ID:\ngot  %q\nwant %q", rec.Step.ID, wantStep)
	}

	// Resource ID
	wantResource := "archive:debian:https://deb.debian.org/debian"
	if len(rec.Resources) == 0 || rec.Resources[0].ID != wantResource {
		t.Errorf("resource ID: got %q, want %q", func() string {
			if len(rec.Resources) == 0 {
				return "<none>"
			}
			return rec.Resources[0].ID
		}(), wantResource)
	}
}

func TestMultipleStanzas(t *testing.T) {
	input := `Package: libarchive13
Version: 3.6.2-1
Architecture: amd64
SHA256: aaa

Package: libarchive-dev
Version: 3.6.2-1
Architecture: amd64
SHA256: bbb
`
	p := &packages.PackagesParser{}
	mapped, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(mapped.Mapped) != 2 {
		t.Fatalf("expected 2 records, got %d", len(mapped.Mapped))
	}
}

func TestMissingSHA256Skipped(t *testing.T) {
	input := `Package: libarchive13
Version: 3.6.2-1
Architecture: amd64
Filename: pool/main/liba/libarchive/libarchive13_3.6.2-1_amd64.deb
`
	p := &packages.PackagesParser{}
	mapped, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(mapped.Mapped) != 0 {
		t.Fatalf("expected 0 records (no SHA256), got %d", len(mapped.Mapped))
	}
}

func TestEpochVersionPreserved(t *testing.T) {
	input := `Package: bash
Version: 1:5.1-6
Architecture: amd64
SHA256: deadbeef
`
	p := &packages.PackagesParser{}
	mapped, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(mapped.Mapped) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mapped.Mapped))
	}
	got := mapped.Mapped[0].ArtifactsOut[0].ID
	want := "artifact:pkg:deb/debian/bash@1:5.1-6?arch=amd64"
	if got != want {
		t.Errorf("artifact ID with epoch:\ngot  %q\nwant %q", got, want)
	}
}

func TestMultiLineDescriptionDoesNotCorrupt(t *testing.T) {
	// Description with continuation lines immediately followed by another stanza
	input := `Package: libarchive13
Version: 3.6.2-1
Architecture: amd64
SHA256: abc123
Description: Short description
 Continuation line 1.
 Continuation line 2.

Package: libarchive-dev
Version: 3.6.2-1
Architecture: amd64
SHA256: def456
`
	p := &packages.PackagesParser{}
	mapped, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(mapped.Mapped) != 2 {
		t.Fatalf("expected 2 records, got %d", len(mapped.Mapped))
	}
	// Verify second record was parsed correctly (not corrupted by first's Description)
	if mapped.Mapped[1].ArtifactsOut[0].Attrs["hash"] != "def456" {
		t.Errorf("second record hash corrupted: %q", mapped.Mapped[1].ArtifactsOut[0].Attrs["hash"])
	}
}

func TestDefaultArchiveURL(t *testing.T) {
	input := `Package: bash
Version: 5.1-6
Architecture: amd64
SHA256: abc
`
	p := &packages.PackagesParser{} // no ArchiveURL set
	mapped, err := p.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(mapped.Mapped) != 1 {
		t.Fatalf("expected 1 record")
	}
	wantResource := "archive:debian:https://deb.debian.org/debian"
	if mapped.Mapped[0].Resources[0].ID != wantResource {
		t.Errorf("resource ID: got %q, want %q", mapped.Mapped[0].Resources[0].ID, wantResource)
	}
}
