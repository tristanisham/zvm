package cli

import (
	"errors"
)

var (
	ErrMissingBundlePath  = errors.New("bundle download path not found")
	ErrUnsupportedSystem  = errors.New("unsupported system for Zig")
	ErrUnsupportedVersion = errors.New("unsupported Zig version")
)
