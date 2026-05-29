package debutil_test

import (
	"testing"

	"github.com/TSELab/astra/internal/parser/debian/debutil"
)

func TestParseDebFilename(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPkg     string
		wantVersion string
		wantArch    string
		wantOK      bool
	}{
		{
			name:        "normal deb",
			input:       "libarchive3_3.6.2-1_amd64.deb",
			wantPkg:     "libarchive3",
			wantVersion: "3.6.2-1",
			wantArch:    "amd64",
			wantOK:      true,
		},
		{
			name:        "arch all",
			input:       "libarchive3_3.6.2-1_all.deb",
			wantPkg:     "libarchive3",
			wantVersion: "3.6.2-1",
			wantArch:    "all",
			wantOK:      true,
		},
		{
			name:        "epoch in version",
			input:       "libarchive3_1:3.6.2-1_amd64.deb",
			wantPkg:     "libarchive3",
			wantVersion: "1:3.6.2-1",
			wantArch:    "amd64",
			wantOK:      true,
		},
		{
			name:    "orig tarball not a deb",
			input:   "libarchive_3.6.2.orig.tar.xz",
			wantOK:  false,
		},
		{
			name:    "plain source file",
			input:   "some-source.c",
			wantOK:  false,
		},
		{
			name:    "missing arch (only two underscores missing third part)",
			input:   "libarchive3_3.6.2-1.deb",
			wantOK:  false,
		},
		{
			name:    "empty package name",
			input:   "_3.6.2-1_amd64.deb",
			wantOK:  false,
		},
		{
			name:    "empty string",
			input:   "",
			wantOK:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pkg, version, arch, ok := debutil.ParseDebFilename(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ParseDebFilename(%q): got ok=%v, want ok=%v", tc.input, ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if pkg != tc.wantPkg {
				t.Errorf("pkg: got %q, want %q", pkg, tc.wantPkg)
			}
			if version != tc.wantVersion {
				t.Errorf("version: got %q, want %q", version, tc.wantVersion)
			}
			if arch != tc.wantArch {
				t.Errorf("arch: got %q, want %q", arch, tc.wantArch)
			}
		})
	}
}
