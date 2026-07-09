package cli

import (
	"testing"

	"github.com/tristanisham/clr"
)

func TestFormatAliasColumn(t *testing.T) {
	tests := []struct {
		name    string
		aliases []string
		want    string
	}{
		{name: "none", aliases: nil, want: ""},
		{name: "one", aliases: []string{"a"}, want: "a"},
		{name: "three", aliases: []string{"a", "b", "c"}, want: "a, b, c"},
		{name: "more than three", aliases: []string{"a", "b", "c", "d"}, want: "a, b, c... (4)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAliasColumn(tt.aliases); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestFormatVersionLine(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		aliasCol string
		active   bool
		useColor bool
		want     string
	}{
		{
			name:     "inactive without aliases",
			version:  "0.13.0",
			aliasCol: "",
			active:   false,
			useColor: false,
			want:     "0.13.0",
		},
		{
			name:     "inactive with aliases",
			version:  "0.13.0",
			aliasCol: "stable",
			active:   false,
			useColor: false,
			want:     "0.13.0  stable",
		},
		{
			name:     "active without color",
			version:  "0.13.0",
			aliasCol: "",
			active:   true,
			useColor: false,
			want:     "0.13.0 [x]",
		},
		{
			name:     "active with color",
			version:  "0.13.0",
			aliasCol: "",
			active:   true,
			useColor: true,
			want:     clr.Green("0.13.0"),
		},
		{
			name:     "active with color and aliases",
			version:  "0.13.0",
			aliasCol: "stable",
			active:   true,
			useColor: true,
			want:     clr.Green("0.13.0") + "  " + clr.Blue("stable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatVersionLine(tt.version, tt.aliasCol, tt.active, tt.useColor); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
