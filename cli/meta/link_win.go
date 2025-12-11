//go:build windows

// Copyright 2025 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	// "bytes"
	"errors"
	"os"

	// "os/exec"
	"strings"
	"syscall"

	// "github.com/charmbracelet/log"
	"github.com/charmbracelet/log"
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
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")

	return err == nil
}

// Link is a wrapper around Go's os.Symlink and os.Link functions,
// On Windows, if Link is unable to create a symlink it will attempt to create a
// hardlink before trying its automatic privilege escalation.
func Link(oldname, newname string) error {
	// Attempt to do a regular symlink if allowed by user's permissions
	if err := os.Symlink(oldname, newname); err != nil {
		// If that fails, try to create an old hardlink.
		if err := os.Link(oldname, newname); err == nil {
			return nil
		}
		// If creating a hardlink fails, check to see if the user is an admin.
		// If they're not an admin, try to become an admin and retry making a symlink.
		if !isAdmin() {
			log.Error("Symlink & Hardlink failed", "admin", false)

			// If not already admin, try to become admin
			if adminErr := becomeAdmin(); adminErr != nil {
				return errors.Join(ErrWinEscToAdmin, adminErr, err)
			}

			if err := os.Symlink(oldname, newname); err != nil {
				if err := os.Link(oldname, newname); err == nil {
					return nil
				}

				return errors.Join(ErrEscalatedSymlink, ErrEscalatedHardlink, err)
			}

			return nil
		}

		return errors.Join(ErrEscalatedSymlink, ErrEscalatedHardlink, err)

	}

	return nil
}
