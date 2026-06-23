package di

import "github.com/evg4b/uncors/internal/contracts"

type OverrideFunc func(c *Container)

func (c *Container) Override(action OverrideFunc) {
	action(c)
}

func OverrideCliOutput(factory func() contracts.Output) OverrideFunc {
	return func(c *Container) {
		c.cliOutput = newFactory(factory)
	}
}
