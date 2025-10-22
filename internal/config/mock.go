package config

import (
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type Mock struct {
	RequestMatcher `mapstructure:",squash"`
	Response       Response `mapstructure:"response"`
}

func (m *Mock) Clone() Mock {
	return Mock{
		RequestMatcher: m.RequestMatcher.Clone(),
		Response:       m.Response.Clone(),
	}
}

func (m *Mock) String() string {
	method := "*"
	if m.Method != "" {
		method = m.Method
	}

	return helpers.Sprintf("[%s %d] %s", method, m.Response.Code, m.Path)
}

type Mocks []Mock

func (m Mocks) Clone() Mocks {
	if m == nil {
		return nil
	}

	return lo.Map(m, func(item Mock, _ int) Mock {
		return item.Clone()
	})
}
