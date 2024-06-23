package validators

import (
	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/evg4b/uncors/internal/fakedata"
	"github.com/gobuffalo/validate"
)

type FakedataNodeValidator struct {
	Field string
	Value *fakedata.Node
}

func (c *FakedataNodeValidator) IsValid(errors *validate.Errors) {
	errors.Append(validate.Validate(&base.StringEnumValidator{
		Field:   joinPath(c.Field, "type"),
		Value:   c.Value.Type,
		Options: fakedata.GetTypes(),
	}))
}
