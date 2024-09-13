package cli

import "testing"

func TestValidateVMUalias(t *testing.T) {
	if !validVmuAlis("mach") {
		t.Errorf("mach: should be true")
	}

	if !validVmuAlis("default") {
		t.Errorf("default: should be true")
	}
}
