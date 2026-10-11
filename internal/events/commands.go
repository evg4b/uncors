package events

// Command is a request from a frontend to the supervisor that runs the proxy.
type Command interface {
	isCommand()
}

// Restart asks to reload the configuration file and restart the proxy with it.
type Restart struct{}

// Shutdown asks to stop the proxy and exit.
type Shutdown struct{}

func (Restart) isCommand()  {}
func (Shutdown) isCommand() {}
