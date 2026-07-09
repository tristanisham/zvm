package cli

import "testing"

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
