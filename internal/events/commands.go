package events

type Command interface {
	isCommand()
}

type Restart struct{}

type Shutdown struct{}

func (Restart) isCommand() {
	// Seals Command: only types in this package implement it.
}

func (Shutdown) isCommand() {
	// Seals Command: only types in this package implement it.
}
