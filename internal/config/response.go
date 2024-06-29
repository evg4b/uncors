package config

import (
	"time"

	"github.com/evg4b/uncors/pkg/fakedata"

	"github.com/evg4b/uncors/internal/helpers"
)

type Response struct {
	Code    int               `mapstructure:"code"`
	Headers map[string]string `mapstructure:"headers"`
	Delay   time.Duration     `mapstructure:"delay"`
	Raw     string            `mapstructure:"raw"`
	File    string            `mapstructure:"file"`
	Fake    *fakedata.Node    `mapstructure:"fake"`
}

func (r *Response) Clone() Response {
	return Response{
		Code:    r.Code,
		Headers: helpers.CloneMap(r.Headers),
		Raw:     r.Raw,
		File:    r.File,
		Delay:   r.Delay,
		Fake:    r.Fake,
	}
}

func (r *Response) IsRaw() bool {
	return len(r.Raw) > 0
}

func (r *Response) IsFile() bool {
	return len(r.File) > 0
}

func (r *Response) IsFake() bool {
	return r.Fake != nil
}
