package cli

import "testing"

func TestStripExcessSlashes(t *testing.T) {
	old := "https://releases.zigtools.org//v1/zls/select-version"
	new := cleanURL(old)

	if new != "https://releases.zigtools.org/v1/zls/select-version" {
		t.Errorf("expected https://releases.zigtools.org/v1/zls/select-version. Got %s", new)
	}
}
