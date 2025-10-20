package script

import "errors"

// ErrScriptFileNotFound is returned when the specified script file doesn't exist.
var ErrScriptFileNotFound = errors.New("script file not found")
