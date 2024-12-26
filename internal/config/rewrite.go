package config

type RewritingOption struct {
	From string `mapstructure:"from"`
	To   string `mapstructure:"to"`
	Host string `mapstructure:"host"`
}

func (r RewritingOption) Clone() RewritingOption {
	return RewritingOption{
		From: r.From,
		To:   r.To,
		Host: r.Host,
	}
}

type RewriteOptions []RewritingOption

func (r RewriteOptions) Clone() RewriteOptions {
	if r == nil {
		return nil
	}

	clone := make(RewriteOptions, len(r))
	for i, rewrite := range r {
		clone[i] = rewrite.Clone()
	}

	return clone
}
