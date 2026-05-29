// Package debutil provides shared utilities for Debian package parsers.
package debutil

import "strings"

// ParseDebFilename parses a Debian binary package filename of the form
//
//	<package>_<version>_<arch>.deb
//
// and returns the three components. Returns ok=false for any other filename form.
//
// Examples:
//
//	libarchive3_3.6.2-1_amd64.deb → ("libarchive3", "3.6.2-1", "amd64", true)
//	libarchive3_1:3.6.2-1_amd64.deb → ("libarchive3", "1:3.6.2-1", "amd64", true)
//	libarchive_3.6.2.orig.tar.xz  → ("", "", "", false)
func ParseDebFilename(name string) (pkg, version, arch string, ok bool) {
	if !strings.HasSuffix(name, ".deb") {
		return "", "", "", false
	}
	// Split on "_" into at most 3 parts: package, version, arch.
	// Debian package versions may contain letters and hyphens but not "_",
	// so exactly 3 "_"-delimited parts (after dropping the .deb suffix) is canonical.
	parts := strings.SplitN(strings.TrimSuffix(name, ".deb"), "_", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
