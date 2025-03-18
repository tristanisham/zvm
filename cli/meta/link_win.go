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
	"golang.org/x/sys/windows"
)

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

		if err := os.Link(oldname, newname); err != nil {
			return errors.Join(ErrEscalatedSymlink, errors.New("cannot create hardlink"))
		} else {
			// Check if already admin first
			if isAdmin() {
				if err := os.Symlink(oldname, newname); err != nil {
					return errors.Join(ErrEscalatedSymlink, err)
				}
				return nil
			} else {
				// If not already admin, try to become admin
				if err := becomeAdmin(); err != nil {
					if err := os.Symlink(oldname, newname); err != nil {
						return errors.Join(ErrEscalatedSymlink, err)
					}
				}
			}
		}

	}

	return nil
}
