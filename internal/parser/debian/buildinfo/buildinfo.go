/*
Package buildinfo implements the BuildinfoParser for AStRA.

BuildinfoParser parses Debian .buildinfo files and maps the build event
into AStRA's record structure. Each file produces one Record.

For a buildinfo file:
  - The dpkg-buildpackage invocation is represented as a step with ID
    step:build:deb/<source>@<version>.
  - The build origin (e.g. "Debian") is the principal; the PGP key ID
    is stored in its attrs when a signature is present.
  - The upstream source tarball is an input artifact (consumed by the step)
    with ID artifact:tarball:deb/<source>@<upstream-version>.
  - Installed build dependencies are resources (they constitute the build
    environment, not inputs that are transformed). IDs use purl format:
    pkg:deb/debian/<name>@<version>?arch=<arch>. The mapper wires them as
    principal --uses--> resource --carries_out--> step.
  - Output .deb files are output artifacts (produced by the step).
    IDs use purl format: artifact:pkg:deb/debian/<name>@<version>?arch=<arch>.
  - The buildinfo file itself is emitted as an output artifact with ID
    artifact:buildinfo:deb/<source>@<version>?arch=<arch>.
*/
package buildinfo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	parser "github.com/TSELab/astra/internal/parser"
)

type BuildinfoParser struct{}

func (p *BuildinfoParser) Parse(r io.Reader) (parser.Mapped, error) {
	return parseBuildinfo(r)
}

func stripEpoch(ver string) string {
	if i := strings.Index(ver, ":"); i >= 0 {
		return ver[i+1:]
	}
	return ver
}

func parseBuildinfo(r io.Reader) (parser.Mapped, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return parser.Mapped{}, err
	}

	// Pre-scan: collect header fields that may appear after Checksums-Sha256
	var source, version, buildDate, buildOrigin, buildArch string
	pre := bufio.NewScanner(bytes.NewReader(raw))
	for pre.Scan() {
		line := pre.Text()
		switch {
		case strings.HasPrefix(line, "Source:"):
			source = strings.TrimSpace(strings.TrimPrefix(line, "Source:"))
		case strings.HasPrefix(line, "Version:"):
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		case strings.HasPrefix(line, "Build-Architecture:"):
			buildArch = strings.TrimSpace(strings.TrimPrefix(line, "Build-Architecture:"))
		case strings.HasPrefix(line, "Build-Date:"):
			buildDate = strings.TrimSpace(strings.TrimPrefix(line, "Build-Date:"))
		case strings.HasPrefix(line, "Build-Origin:"):
			buildOrigin = strings.TrimSpace(strings.TrimPrefix(line, "Build-Origin:"))
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(raw))

	var outputItems []parser.ArtifactItem
	var depItems []parser.ResourceItem
	var pgpLines []string
	seenDeps := map[string]bool{}

	outputSection, dependsSection, pgpSection := false, false, false
	re := regexp.MustCompile(`([a-zA-Z0-9.+:~\-]+) \(= ([^\)]+)\)`)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "Checksums-Sha256:"):
			outputSection = true
			continue
		case strings.HasPrefix(line, "Installed-Build-Depends:"):
			dependsSection = true
			continue
		case strings.HasPrefix(line, "-----BEGIN PGP SIGNATURE-----"):
			pgpSection = true
		case strings.HasPrefix(line, "-----END PGP SIGNATURE-----"):
			pgpSection = false
			continue
		case outputSection && strings.TrimSpace(line) == "":
			outputSection = false
		case dependsSection && strings.TrimSpace(line) == "":
			dependsSection = false
		}

		if pgpSection {
			pgpLines = append(pgpLines, line)
			continue
		}

		if outputSection && strings.HasSuffix(strings.TrimSpace(line), ".deb") {
			parts := strings.Fields(line)
			if len(parts) == 3 {
				hash, size, filename := parts[0], parts[1], parts[2]
				pkgName := strings.SplitN(filename, "_", 2)[0]
				purl := fmt.Sprintf("pkg:deb/debian/%s@%s?arch=%s", pkgName, version, buildArch)
				outputItems = append(outputItems, parser.ArtifactItem{
					ID:    fmt.Sprintf("artifact:%s", purl),
					Label: filename,
					Kind:  "deb",
					Attrs: map[string]string{
						"purl":     purl,
						"hash":     hash,
						"size":     size,
						"filename": filename,
						"version":  version,
					},
				})
			}
		} else if dependsSection && strings.TrimSpace(line) != "" {
			matches := re.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				pkg := strings.TrimSpace(match[1])
				ver := strings.TrimSpace(match[2])
				purl := fmt.Sprintf("pkg:deb/debian/%s@%s?arch=%s", pkg, ver, buildArch)
				if !seenDeps[purl] {
					seenDeps[purl] = true
					depItems = append(depItems, parser.ResourceItem{
						ID:    purl,
						Label: pkg,
						Kind:  "deb",
						Attrs: map[string]string{
							"purl":    purl,
							"version": ver,
						},
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return parser.Mapped{}, err
	}

	// Extract PGP signing key ID from the signature block.
	var keyID string
	if len(pgpLines) > 0 {
		pgpText := strings.Join(pgpLines, "\n")
		if pgpMsg, err := crypto.NewPGPMessageFromArmored(pgpText); err == nil {
			if ids, ok := pgpMsg.HexSignatureKeyIDs(); ok && len(ids) > 0 {
				keyID = ids[0]
			}
		}
	}

	upstreamVersion := strings.SplitN(stripEpoch(version), "-", 2)[0]
	// Tarball artifact consumed by the build step. ID matches the artifact:tarball: scheme
	// used by the intoto parser for .orig.tar.<ext> products — same ID means the same
	// node in the graph when both documents are ingested.
	tarballArtifact := parser.ArtifactItem{
		ID:           fmt.Sprintf("artifact:tarball:deb/%s@%s", source, upstreamVersion),
		Label:        fmt.Sprintf("%s_%s.orig.tar.xz", source, upstreamVersion),
		Kind:         "tarball",
		Completeness: "incomplete",
		Attrs: map[string]string{
			"source":  source,
			"version": upstreamVersion,
			"format":  "orig.tar.xz",
		},
	}

	principalAttrs := map[string]string{}
	if keyID != "" {
		principalAttrs["pgp_key_id"] = keyID
	}
	principal := parser.PrincipalItem{
		ID:    fmt.Sprintf("principal:%s", buildOrigin),
		Label: "Debian Build Infrastructure",
		Kind:  "principal",
		Attrs: principalAttrs,
	}

	buildinfoArtifact := parser.ArtifactItem{
		ID:           fmt.Sprintf("artifact:buildinfo:deb/%s@%s?arch=%s", source, version, buildArch),
		Label:        fmt.Sprintf("%s_%s_%s.buildinfo", source, version, buildArch),
		Kind:         "buildinfo",
		Completeness: "complete",
		Attrs: map[string]string{
			"source":  source,
			"version": version,
			"arch":    buildArch,
		},
	}

	rec := parser.Record{
		Step: parser.StepItem{
			ID:    fmt.Sprintf("step:build:deb/%s@%s", source, version),
			Label: "dpkg-buildpackage",
			Kind:  "build",
			Attrs: map[string]string{
				"command":      "dpkg-buildpackage",
				"timestamp":    buildDate,
				"architecture": buildArch,
			},
		},
		Principal:    principal,
		ArtifactsIn:  []parser.ArtifactItem{tarballArtifact},
		ArtifactsOut: append(outputItems, buildinfoArtifact),
		Resources:    depItems,
	}

	return parser.Mapped{
		Source:       "buildinfo",
		NormalizedAt: time.Now().Unix(),
		Mapped:       []parser.Record{rec},
	}, nil
}
