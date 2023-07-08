package config

import (
	"time"

	"github.com/evg4b/uncors/internal/helpers"
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
