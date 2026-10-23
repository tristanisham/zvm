//go:build windows

// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	"os"

	// "os/exec"
	"strings"
	"syscall"

	// "github.com/charmbracelet/log"
	"github.com/charmbracelet/log"
	"github.com/nyaosorg/go-windows-junction"
	"golang.org/x/sys/windows"
)

// becomeAdmin attempts to re-run the current executable with administrative privileges using "runas".
func becomeAdmin() error {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 // SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		return err
	}

	return nil
}

// isAdmin checks if the current process has administrative privileges.
func isAdmin() bool {
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")

	if err != nil {
		return false
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Error("Failed to close file", "err", err)
		}
	}()

	return true
}

// Link is a wrapper around Go's os.Symlink and os.Link functions,
// On Windows, if Link is unable to create a symlink it will attempt to create a
// hardlink before trying its automatic privilege escalation.
func Link(oldname, newname string) error {
	if err := junction.Create(oldname, newname); err != nil {
		log.Error("Junction link failed", "err", err)

		return err
	}

	return nil
}

// Relink points the junction at newname to oldname, creating it if it does
// not exist. Windows cannot rename a junction over an existing one, so the old
// junction is removed first (os.Remove deletes the reparse point only, never
// the target's contents) and a new one is created; another process may briefly
// observe the link missing. An existing entry at newname that is not a link
// is never replaced.
func Relink(oldname, newname string) error {
	if fi, err := os.Lstat(newname); err == nil {
		if !IsLink(fi) {
			return &os.LinkError{Op: "relink", Old: oldname, New: newname, Err: os.ErrExist}
		}
		if err := os.Remove(newname); err != nil {
			return err
		}
	}
	return Link(oldname, newname)
}

// IsLink reports whether fi (from os.Lstat) describes a link created by Link.
// Since Go 1.23, os.Lstat reports junctions (mount points) as ModeIrregular
// rather than ModeSymlink, so both are accepted.
func IsLink(fi os.FileInfo) bool {
	return fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0
}
