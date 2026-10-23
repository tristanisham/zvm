//go:build !noAutoUpgrades

// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package meta

import "testing"

// TestNoAutoUpgrades_default confirms that in a normal build (without the
// noAutoUpgrades build tag), the flag is false so self-upgrade is enabled.
func TestNoAutoUpgrades_default(t *testing.T) {
	if NoAutoUpgrades {
		t.Error("NoAutoUpgrades should be false in default build")
	}
}
