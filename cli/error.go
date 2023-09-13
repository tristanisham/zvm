package cli

import (
	"errors"
)

var (
	ErrMissingBundlePath  = errors.New("bundle download path not found")
	ErrUnsupportedSystem  = errors.New("unsupported system for Zig")
	ErrUnsupportedVersion = errors.New("unsupported Zig version")
	ErrMissingInstallPathEnv = errors.New("env 'ZVM_INSTALL' is not set")
	ErrFailedUpgrade = errors.New("failed to self-upgrade zvm")
)
