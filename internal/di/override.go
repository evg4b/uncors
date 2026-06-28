package di

import "github.com/evg4b/uncors/internal/contracts"

func (c *Container) Override(action ContainerOption) {
	action(c)
}

func WithCliOutput(factory func() contracts.Output) ContainerOption {
	return func(c *Container) {
		c.cliOutput = newFactory(factory)
	}
}
