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
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")

	return err == nil
}

// Link is a wrapper around Go's os.Symlink and os.Link functions,
// On Windows, if Link is unable to create a symlink it will attempt to create a
// hardlink before trying its automatic privilege escalation.
func Link(oldname, newname string) error {
	if err := junction.Create(oldname, newname); err != nil {
		log.Error("Junction link failed")

		return err
	}

	return nil
}
