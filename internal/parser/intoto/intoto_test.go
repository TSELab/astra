package intoto_test

import (
	"strings"
	"testing"

	"github.com/TSELab/astra/internal/parser/intoto"
)

// linkJSON builds a minimal in-toto link file JSON string.
func linkJSON(name, keyid string, materials, products map[string]string) string {
	matJSON := mapToHashJSON(materials)
	prodJSON := mapToHashJSON(products)
	sigs := "[]"
	if keyid != "" {
		sigs = `[{"keyid":"` + keyid + `","sig":"dummysig"}]`
	}
	return `{
  "signed": {
    "_type": "link",
    "name": "` + name + `",
    "materials": ` + matJSON + `,
    "products": ` + prodJSON + `,
    "byproducts": {},
    "environment": {},
    "command": ["dpkg-buildpackage", "-b"]
  },
  "signatures": ` + sigs + `
}`
}

func mapToHashJSON(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, `"`+k+`": {"sha256": "`+v+`"}`)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func TestDebProductIDMatchesBuildinfo(t *testing.T) {
	// A .deb product must produce the same purl-based ID as buildinfo.
	j := linkJSON("build-package", "abc123", nil, map[string]string{
		"libarchive3_3.6.2-1_amd64.deb": "82cc6d094f9b7c872e5bc5c4613151a7a8c20ac1a3d7d6c128dca64da618857b",
	})
	p := &intoto.InTotoParser{}
	mapped, err := p.Parse(strings.NewReader(j))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(mapped.Mapped) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mapped.Mapped))
	}
	rec := mapped.Mapped[0]
	if len(rec.ArtifactsOut) != 1 {
		t.Fatalf("expected 1 product artifact, got %d", len(rec.ArtifactsOut))
	}
	got := rec.ArtifactsOut[0].ID
	want := "artifact:pkg:deb/debian/libarchive3@3.6.2-1?arch=amd64"
	if got != want {
		t.Errorf("product artifact ID:\ngot  %q\nwant %q", got, want)
	}
	if rec.ArtifactsOut[0].Completeness != "complete" {
		t.Errorf("product completeness: got %q, want complete", rec.ArtifactsOut[0].Completeness)
	}
	if rec.ArtifactsOut[0].Attrs["hash"] != "82cc6d094f9b7c872e5bc5c4613151a7a8c20ac1a3d7d6c128dca64da618857b" {
		t.Errorf("product hash mismatch: %q", rec.ArtifactsOut[0].Attrs["hash"])
	}
}

func TestNonDebMaterialGetsHashID(t *testing.T) {
	j := linkJSON("build-package", "", map[string]string{
		"some-source.c": "deadbeef1234",
	}, nil)
	p := &intoto.InTotoParser{}
	mapped, err := p.Parse(strings.NewReader(j))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	rec := mapped.Mapped[0]
	if len(rec.ArtifactsIn) != 1 {
		t.Fatalf("expected 1 material, got %d", len(rec.ArtifactsIn))
	}
	got := rec.ArtifactsIn[0].ID
	want := "artifact:intoto:build-package:material:deadbeef1234"
	if got != want {
		t.Errorf("material ID:\ngot  %q\nwant %q", got, want)
	}
	if rec.ArtifactsIn[0].Completeness != "incomplete" {
		t.Errorf("material completeness: got %q, want incomplete", rec.ArtifactsIn[0].Completeness)
	}
}

func TestUnsignedLink(t *testing.T) {
	j := linkJSON("build-package", "", nil, nil)
	p := &intoto.InTotoParser{}
	mapped, err := p.Parse(strings.NewReader(j))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	rec := mapped.Mapped[0]
	if rec.Principal.ID != "principal:intoto:unsigned" {
		t.Errorf("principal ID: got %q, want principal:intoto:unsigned", rec.Principal.ID)
	}
}

func TestSignedLinkUsesFirstKeyID(t *testing.T) {
	j := linkJSON("build-package", "abc123def456", nil, nil)
	p := &intoto.InTotoParser{}
	mapped, err := p.Parse(strings.NewReader(j))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	rec := mapped.Mapped[0]
	wantID := "principal:intoto:abc123def456"
	if rec.Principal.ID != wantID {
		t.Errorf("principal ID: got %q, want %q", rec.Principal.ID, wantID)
	}
	if rec.Principal.Attrs["keyid"] != "abc123def456" {
		t.Errorf("keyid attr: got %q, want abc123def456", rec.Principal.Attrs["keyid"])
	}
}

func TestStepIDStableAcrossReparses(t *testing.T) {
	j := linkJSON("build-package", "key1", map[string]string{"src.c": "abc"}, map[string]string{
		"pkg_1.0-1_amd64.deb": "def",
	})
	p := &intoto.InTotoParser{}
	m1, _ := p.Parse(strings.NewReader(j))
	m2, _ := p.Parse(strings.NewReader(j))
	if m1.Mapped[0].Step.ID != m2.Mapped[0].Step.ID {
		t.Errorf("step ID not stable: %q vs %q", m1.Mapped[0].Step.ID, m2.Mapped[0].Step.ID)
	}
}

func TestEmptyMaterialsAndProducts(t *testing.T) {
	j := linkJSON("empty-step", "", nil, nil)
	p := &intoto.InTotoParser{}
	mapped, err := p.Parse(strings.NewReader(j))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	rec := mapped.Mapped[0]
	if len(rec.ArtifactsIn) != 0 {
		t.Errorf("expected 0 materials, got %d", len(rec.ArtifactsIn))
	}
	if len(rec.ArtifactsOut) != 0 {
		t.Errorf("expected 0 products, got %d", len(rec.ArtifactsOut))
	}
}

func TestArchExtractedFromEnvironment(t *testing.T) {
	j := `{
  "signed": {
    "_type": "link",
    "name": "build",
    "materials": {},
    "products": {},
    "byproducts": {},
    "environment": {"variables": ["PATH=/usr/bin", "DEB_HOST_ARCH=amd64", "HOME=/root"]},
    "command": []
  },
  "signatures": []
}`
	p := &intoto.InTotoParser{}
	mapped, err := p.Parse(strings.NewReader(j))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	arch := mapped.Mapped[0].Step.Attrs["architecture"]
	if arch != "amd64" {
		t.Errorf("architecture: got %q, want amd64", arch)
	}
}

func TestStepIDStartsWithPrefix(t *testing.T) {
	j := linkJSON("my-step", "", nil, nil)
	p := &intoto.InTotoParser{}
	mapped, err := p.Parse(strings.NewReader(j))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	id := mapped.Mapped[0].Step.ID
	if !strings.HasPrefix(id, "step:linker:debtrace:my-step@") {
		t.Errorf("step ID %q does not have expected prefix", id)
	}
}
