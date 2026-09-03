package di

import "github.com/evg4b/uncors/internal/contracts"

// noopOutput is the container's default console output.
//
// The real one is a presentation decision and is supplied by the composition
// root: main installs the console renderer, and interactive mode installs the
// sink that feeds the TUI. Defaulting to a null object is what lets the
// container - and therefore the whole service - stay free of the terminal
// libraries.
type noopOutput struct{}

func (*noopOutput) Write(p []byte) (int, error) { return len(p), nil }

func (*noopOutput) Info(any)                       {}
func (*noopOutput) Infof(string, ...any)           {}
func (*noopOutput) InfoBox(...string)              {}
func (*noopOutput) Error(any)                      {}
func (*noopOutput) Errorf(string, ...any)          {}
func (*noopOutput) ErrorBox(...string)             {}
func (*noopOutput) Warn(any)                       {}
func (*noopOutput) Warnf(string, ...any)           {}
func (*noopOutput) WarnBox(...string)              {}
func (*noopOutput) Print(any)                      {}
func (*noopOutput) Printf(string, ...any)          {}
func (*noopOutput) Request(*contracts.RequestData) {}

func (n *noopOutput) NewPrefixOutput(string) contracts.Output { return n }
