//go:build noAutoUpgrades

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package meta

import "testing"

// TestNoAutoUpgrades_enabled confirms that the noAutoUpgrades build tag
// disables self-upgrade support.
func TestNoAutoUpgrades_enabled(t *testing.T) {
	if !NoAutoUpgrades {
		t.Error("NoAutoUpgrades should be true when built with the noAutoUpgrades tag")
	}
}
