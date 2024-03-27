package meta

import "errors"

var (
	ErrWinEscToAdmin = errors.New("unable to rerun as Windows Administrator")
	ErrEscalatedSymlink = errors.New("unable to symlink as Administrator")
)