package win32

import "errors"

// ErrUnsupported is returned by every function off Windows.
var ErrUnsupported = errors.New("win32: only available on Windows")
