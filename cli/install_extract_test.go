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
		{
			name:  "Package only",
			input: "zig",
			want:  installRequest{Package: "zig"},
		},
		{
			name:  "Package with version",
			input: "zig@0.12.0",
			want:  installRequest{Package: "zig", Version: "0.12.0"},
		},
		{
			name:  "Site and package",
			input: "github:tristanisham/myrepo",
			want:  installRequest{Site: "github", Package: "tristanisham/myrepo"},
		},
		{
			name:  "Site, package, and version",
			input: "github:tristanisham/myrepo@main",
			want:  installRequest{Site: "github", Package: "tristanisham/myrepo", Version: "main"},
		},
		{
			name:  "Empty string",
			input: "",
			want:  installRequest{},
		},
		{
			name:  "Only at symbol",
			input: "@",
			want:  installRequest{Version: ""}, // Package will be empty, version is empty
		},
		{
			name:  "Only colon symbol",
			input: ":",
			want:  installRequest{Site: "", Package: ""}, // Site will be empty, package will be empty
		},
		{
			name:  "Site with empty package",
			input: "site:",
			want:  installRequest{Site: "site", Package: ""},
		},
		{
			name:  "Site with empty package and version",
			input: "site:@1.0",
			want:  installRequest{Site: "site", Package: "", Version: "1.0"},
		},
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
