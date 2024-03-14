package cli

import (
	"bytes"
	"os/exec"

	"github.com/charmbracelet/log"
)

// winElevatedRun elevates a process to run as an administrator on Windows.
func winElevatedRun(name string, arg ...string) (bool, error) {
	log.Debugf("Elevated process: %q", name)
	ok, err := run("cmd", nil, append([]string{"/C", name}, arg...)...)
	if err != nil {
		ok, err = run("elevate.cmd", nil, append([]string{"cmd", "/C", name}, arg...)...)
	}
	return ok, err
}

// run actually constructs the command to be ran from ZVM
func run(name string, dir *string, arg ...string) (bool, error) {
	c := exec.Command(name, arg...)
	if dir != nil {
		c.Dir = *dir
	}
	var stderr bytes.Buffer
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		return false, err
	}

	return true, nil
}
