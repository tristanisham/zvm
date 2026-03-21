// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// This is the (dummy) fallback implementation to use if the `noAutoUpgrades` tag was passed for building.
//go:build noAutoUpgrades

package cli

import (
	"github.com/tristanisham/zvm/cli/meta"
)

func (z *ZVM) Upgrade() error {
	return nil
}

func CanIUpgrade() (string, bool, error) {
	return meta.VERSION, false, nil
}
