package config

import (
	"fmt"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/samber/lo"
	"github.com/spf13/afero"
)

type Mock struct {
	Matcher  RequestMatcher `yaml:",inline"`
	Response Response       `yaml:"response"`
}

func (m *Mock) Clone() Mock {
	return Mock{
		Matcher:  m.Matcher.Clone(),
		Response: m.Response.Clone(),
	}
}

func (m *Mock) String() string {
	method := "*"
	if m.Matcher.Method != "" {
		method = m.Matcher.Method
	}

	return fmt.Sprintf("[%s %d] %s", method, m.Response.Code, m.Matcher.Path)
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

func (m *Mock) Validate(field string, fs afero.Fs) error {
	var errs *multierror.Error

	errs = multierror.Append(errs, m.Matcher.Validate(field))
	errs = multierror.Append(errs, m.Response.Validate(joinPath(field, "response"), fs))

	return joinErrors(errs)
}
