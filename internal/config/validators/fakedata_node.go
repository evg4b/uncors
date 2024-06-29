package validators

import (
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/evg4b/uncors/pkg/fakedata"
	"github.com/gobuffalo/validate"
)

type FakedataNodeValidator struct {
	Field string
	Value *fakedata.Node
	Root  bool
}

func (c *FakedataNodeValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(&base.StringEnumValidator{
		Field:   joinPath(c.Field, "type"),
		Value:   c.Value.Type,
		Options: fakedata.GetTypes(),
	}))

	if !c.Root && c.Value.Seed != uint64(0) {
		errors.Add(joinPath(c.Field, "seed"), "property 'seed' is not allowed in nested nodes")
	}

	if c.Value.Type == "object" {
		for key, node := range c.Value.Properties {
			errors.Append(validate.Validate(&FakedataNodeValidator{
				Field: joinPath(c.Field, key),
				Value: &node,
			}))
		}
	}

	if c.Value.Type == "array" {
		errors.Append(validate.Validate(&FakedataNodeValidator{
			Field: joinPath(c.Field, "item"),
			Value: c.Value.Item,
		}))

		if c.Value.Count < 0 {
			errors.Add(joinPath(c.Field, "count"), "property 'count' must be greater than or equal to 0")
		}
	}
}
