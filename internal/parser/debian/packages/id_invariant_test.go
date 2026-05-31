package packages_test

// id_invariant_test.go asserts that the artifact ID produced for a given
// (package, version, arch) triple is byte-identical across the packages
// parser and the buildinfo parser.  This invariant is what makes in-toto
// and packages parser ingestions naturally wire up to buildinfo nodes
// through graph merging — if the ID format ever drifts between parsers,
// the linking silently breaks.

import (
	"fmt"
	"strings"
	"testing"

	buildinfoparser "github.com/TSELab/astra/internal/parser/debian/buildinfo"
	"github.com/TSELab/astra/internal/parser/debian/packages"
)

// minimalBuildinfo returns a .buildinfo snippet sufficient to parse one output
// artifact for the given package/version/arch.
func minimalBuildinfo(pkg, version, arch string) string {
	return fmt.Sprintf(`Format: 1.0
Source: %s
Version: %s
Build-Architecture: %s
Checksums-Sha256:
 deadbeef 12345 %s_%s_%s.deb

Installed-Build-Depends:
`, pkg, version, arch, pkg, version, arch)
}

// minimalPackagesStanza returns a Packages file stanza for the given
// package/version/arch with a dummy SHA256.
func minimalPackagesStanza(pkg, version, arch string) string {
	return fmt.Sprintf(`Package: %s
Version: %s
Architecture: %s
SHA256: deadbeef
`, pkg, version, arch)
}

// artifactIDFromBuildinfo parses a minimal buildinfo and returns the first
// output artifact ID produced by the buildinfo parser.
func artifactIDFromBuildinfo(t *testing.T, pkg, version, arch string) string {
	t.Helper()
	r := strings.NewReader(minimalBuildinfo(pkg, version, arch))
	mapped, err := (&buildinfoparser.BuildinfoParser{}).Parse(r)
	if err != nil {
		t.Fatalf("buildinfo parse: %v", err)
	}
	if len(mapped.Mapped) == 0 || len(mapped.Mapped[0].ArtifactsOut) == 0 {
		t.Fatal("buildinfo parser produced no output artifacts")
	}
	return mapped.Mapped[0].ArtifactsOut[0].ID
}

// artifactIDFromPackages parses a minimal Packages stanza and returns the
// first output artifact ID produced by the packages parser.
func artifactIDFromPackages(t *testing.T, pkg, version, arch string) string {
	t.Helper()
	r := strings.NewReader(minimalPackagesStanza(pkg, version, arch))
	mapped, err := (&packages.PackagesParser{}).Parse(r)
	if err != nil {
		t.Fatalf("packages parse: %v", err)
	}
	if len(mapped.Mapped) == 0 || len(mapped.Mapped[0].ArtifactsOut) == 0 {
		t.Fatal("packages parser produced no output artifacts")
	}
	return mapped.Mapped[0].ArtifactsOut[0].ID
}

func TestArtifactIDIdenticalAcrossParsers(t *testing.T) {
	cases := []struct {
		pkg, version, arch string
	}{
		{"libarchive13", "3.6.2-1", "amd64"},
		{"bash", "5.2-2", "amd64"},
		{"libarchive13", "1:3.6.2-1", "amd64"}, // epoch
		{"libarchive13", "3.6.2-1", "all"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%s_%s_%s", tc.pkg, tc.version, tc.arch), func(t *testing.T) {
			fromBuildinfo := artifactIDFromBuildinfo(t, tc.pkg, tc.version, tc.arch)
			fromPackages := artifactIDFromPackages(t, tc.pkg, tc.version, tc.arch)
			if fromBuildinfo != fromPackages {
				t.Errorf("artifact ID mismatch for %s@%s?arch=%s:\n  buildinfo: %q\n  packages:  %q",
					tc.pkg, tc.version, tc.arch, fromBuildinfo, fromPackages)
			}
		})
	}
}
