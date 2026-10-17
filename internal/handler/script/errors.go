package script

import "errors"

var (
	ErrScriptFileNotFound  = errors.New("script file not found")
	ErrScriptTimeout       = errors.New("script timed out")
	ErrOsExit              = errors.New("os.exit is not supported in scripts")
	ErrRequestBodyTooLarge = errors.New("request body is too large for a script")
	ErrRequestBodyNotRead  = errors.New("failed to read the request body")
)
