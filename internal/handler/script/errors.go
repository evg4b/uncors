package script

import "errors"

var (
	ErrScriptFileNotFound      = errors.New("script file not found")
	ErrScriptCompilationFailed = errors.New("script compilation failed")
	ErrScriptTimeout           = errors.New("script exceeded its timeout")
	ErrScriptNotConfigured     = errors.New("neither script nor file is configured")
)
