package validators

import (
	"github.com/evg4b/uncors/internal/fakedata"
	"github.com/gobuffalo/validate"
)

type FakedataNodeValidator struct {
	Field string
	Value *fakedata.Node
}

func (c *FakedataNodeValidator) IsValid(errors *validate.Errors) {

}
