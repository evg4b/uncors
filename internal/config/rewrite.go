package config

import (
	"errors"

	"github.com/evg4b/uncors/pkg/urlt"
)

type RewritingOption struct {
	From string    `yaml:"from"`
	To   string    `yaml:"to"`
	Host urlt.Host `yaml:"host,omitempty"`
}

type RewriteOptions []RewritingOption

func (r RewritingOption) Validate(field string) error {
	var errs []error

	errs = append(errs, ValidatePath(joinPath(field, "from"), r.From, true))
	errs = append(errs, ValidatePath(joinPath(field, "to"), r.To, true))

	if r.Host != (urlt.Host{}) {
		errs = append(errs, ValidateHost(joinPath(field, "host"), r.Host))
	}

	return errors.Join(errs...)
}
