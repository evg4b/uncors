package events

type Command interface {
	isCommand()
}

type Restart struct{}

type Shutdown struct{}

func (Restart) isCommand()  {}
func (Shutdown) isCommand() {}
