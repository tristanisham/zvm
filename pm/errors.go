package pm

import "errors"

var ErrMissingConfig = errors.New("no zvm.json found")
var ErrInvalidScheme = errors.New("invalid url scheme")