package config

import (
	"errors"
	"slices"

	"github.com/evg4b/uncors/pkg/urlt"
)

type RewritingOption struct {
	From string    `yaml:"from"`
	To   string    `yaml:"to"`
	Host urlt.Host `yaml:"host"`
}

func (r RewritingOption) Clone() RewritingOption {
	return r
}

type RewriteOptions []RewritingOption

func (r RewriteOptions) Clone() RewriteOptions {
	return slices.Clone(r)
}

func (r RewritingOption) Validate(field string) error {
	var errs []error

	errs = append(errs, ValidatePath(joinPath(field, "from"), r.From, true))
	errs = append(errs, ValidatePath(joinPath(field, "to"), r.To, true))

	if r.Host != (urlt.Host{}) {
		errs = append(errs, ValidateHost(joinPath(field, "host"), r.Host))
	}

	return errors.Join(errs...)
}
