package config

import (
	"time"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type Response struct {
	Code    int               `mapstructure:"code"`
	Headers map[string]string `mapstructure:"headers"`
	Raw     string            `mapstructure:"raw"`
	File    string            `mapstructure:"file"`
	Delay   time.Duration     `mapstructure:"delay"`
}

func (r *Response) Clone() Response {
	return Response{
		Code:    r.Code,
		Headers: helpers.CloneMap(r.Headers),
		Raw:     r.Raw,
		File:    r.File,
		Delay:   r.Delay,
	}
}

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

type Mocks []Mock

func (m Mocks) Clone() Mocks {
	if m == nil {
		return nil
	}

	return lo.Map(m, func(item Mock, index int) Mock {
		return item.Clone()
	})
}
