package config

import (
	"time"
)

type Response struct {
	Code       int               `mapstructure:"code"`
	Headers    map[string]string `mapstructure:"headers"`
	RawContent string            `mapstructure:"raw-content"`
	File       string            `mapstructure:"file"`
	Delay      time.Duration     `mapstructure:"delay"`
}

func (r Response) Clone() Response {
	return Response{
		Code:       r.Code,
		Headers:    r.Headers,
		RawContent: r.RawContent,
		File:       r.File,
		Delay:      r.Delay,
	}
}

type Mock struct {
	Path     string            `mapstructure:"path"`
	Method   string            `mapstructure:"method"`
	Queries  map[string]string `mapstructure:"queries"`
	Headers  map[string]string `mapstructure:"headers"`
	Response Response          `mapstructure:"response"`
}

func (m Mock) Clone() Mock {
	return Mock{
		Path:     m.Path,
		Method:   m.Method,
		Queries:  m.Queries,
		Headers:  m.Headers,
		Response: m.Response.Clone(),
	}
}
