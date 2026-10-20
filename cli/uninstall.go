// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// Uninstall removes the specified Zig version from the ZVM base directory.
// Only directories that hold a Zig install are removed; ZVM's own files
// (bin, self, settings.json) and anything else in the base directory are
// left alone.
func (z *ZVM) Uninstall(version string) error {
	// Resolve shorthand against locally installed versions
	if installedVersions, err := z.GetInstalledVersions(); err == nil {
		if resolved, resolveErr := resolveVersionShorthand(version, installedVersions); resolveErr == nil && resolved != version {
			log.Debug("resolved version shorthand", "input", version, "resolved", resolved)
			fmt.Printf("Resolved %q to %s\n", version, resolved)
			version = resolved
		}
	}

	root, err := os.OpenRoot(z.baseDir)
	if err != nil {
		return err
	}
	defer root.Close()

	if isZigInstall(root, version) {
		if err := root.RemoveAll(version); err != nil {
			return err
		}
		fmt.Printf("✔ Uninstalled %s\nRun `zvm ls` to view installed versions.\n", version)
		return nil
	}
	fmt.Printf("Version: %s not found locally.\nHere are your installed versions:\n", version)
	return z.ListVersions()
}

// isZigInstall reports whether name is a top-level directory of root that
// holds a Zig install, i.e. a real directory (not a link) containing a zig or
// zig.exe file. Both executable names are accepted because --target-os can
// install a Windows build on any host.
func isZigInstall(root *os.Root, name string) bool {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || !filepath.IsLocal(name) {
		return false
	}

	switch name {
	case "bin", "self":
		return false
	}

	info, err := root.Lstat(name)
	if err != nil || !info.IsDir() {
		return false
	}

	for _, exe := range []string{"zig", "zig.exe"} {
		if info, err := root.Lstat(filepath.Join(name, exe)); err == nil && info.Mode().IsRegular() {
			return true
		}
	}

	return false
}
