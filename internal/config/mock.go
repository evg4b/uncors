package config

import (
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type Mock struct {
	Path     string            `mapstructure:"path"`
	Method   string            `mapstructure:"method"`
	Queries  map[string]string `mapstructure:"queries"`
	Headers  map[string]string `mapstructure:"headers"`
	Response Response          `mapstructure:"response"`
}

func (m *Mock) Clone() Mock {
	return Mock{
		Path:     m.Path,
		Method:   m.Method,
		Queries:  helpers.CloneMap(m.Queries),
		Headers:  helpers.CloneMap(m.Headers),
		Response: m.Response.Clone(),
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
