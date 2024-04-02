//go:build windows

// Copyright 2022 Tristan Isham. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
package meta

import (
	// "bytes"
	"errors"
	"os"
	"runtime"

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

	var showCmd int32 = 1 //SW_NORMAL

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

// Symlink is a wrapper around Go's os.Symlink,
// but with automatic privilege escalation on windows
// for systems that do not support non-admin symlinks.
func Symlink(oldname, newname string) error {
	if err := os.Symlink(oldname, newname); err != nil {

		if errors.Is(err, &os.LinkError{}) {
			if runtime.GOOS == "windows" {
				if !isAdmin() {
					if err := becomeAdmin(); err != nil {
						if err := os.Symlink(oldname, newname); err != nil {
							return errors.Join(ErrEscalatedSymlink, err)
						}
					} else {
						return errors.Join(ErrWinEscToAdmin, err)
					}
				}
			}
		}

	}

	return nil
}
