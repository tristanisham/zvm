// Copyright 2026 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import "io"

// RunSettingsTUI prints settings on Plan 9, where Bubble Tea has no terminal
// backend. SettingsCommand still supports reading and updating individual keys.
func (z *ZVM) RunSettingsTUI(_ io.Reader, out io.Writer) error {
	return printSettingJSON(out, z.Settings, false)
}
