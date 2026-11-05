// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"testing"
)

func TestExtractInstall(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  installRequest
	}{
		{"Package only", "zig", installRequest{Package: "zig"}},
		{"Package with version", "zig@0.12.0", installRequest{Package: "zig", Version: "0.12.0"}},
		{"Site and package", "github:tristanisham/myrepo", installRequest{Site: "github", Package: "tristanisham/myrepo"}},
		{"Site, package, and version", "github:tristanisham/myrepo@main", installRequest{Site: "github", Package: "tristanisham/myrepo", Version: "main"}},
		{"Multiple separators", "zpm:zl:s@te@st", installRequest{Site: "zpm", Package: "zl:s", Version: "te@st"}},
		{"Multiple colons without version", "zpm:zl:s", installRequest{Site: "zpm", Package: "zl:s"}},
		{"Package with numeric version", "zls@11", installRequest{Package: "zls", Version: "11"}},
		{"Empty string", "", installRequest{}},
		{"Only at symbol", "@", installRequest{Version: ""}},
		{"Only colon symbol", ":", installRequest{Site: "", Package: ""}},
		{"Site with empty package", "site:", installRequest{Site: "site", Package: ""}},
		{"Site with empty package and version", "site:@1.0", installRequest{Site: "site", Package: "", Version: "1.0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractInstall(tt.input)
			if got.Site != tt.want.Site || got.Package != tt.want.Package || got.Version != tt.want.Version {
				t.Errorf("ExtractInstall(%q) got = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}
