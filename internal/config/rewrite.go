package config

import (
	"slices"

	multierror "github.com/hashicorp/go-multierror"
)

type RewritingOption struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
	Host string `yaml:"host"`
}

func (r RewritingOption) Clone() RewritingOption {
	return r
}

type RewriteOptions []RewritingOption

func (r RewriteOptions) Clone() RewriteOptions {
	return slices.Clone(r)
}

func (r RewritingOption) Validate(field string) error {
	var errs *multierror.Error

	errs = multierror.Append(errs, ValidatePath(joinPath(field, "from"), r.From, true))
	errs = multierror.Append(errs, ValidatePath(joinPath(field, "to"), r.To, true))

	if r.Host != "" {
		errs = multierror.Append(errs, ValidateHost(joinPath(field, "host"), r.Host))
	}

	return joinErrors(errs)
}
