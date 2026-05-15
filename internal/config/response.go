package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/helpers"
	"gopkg.in/yaml.v3"
)

type Response struct {
	Code    int               `yaml:"code"`
	Headers map[string]string `yaml:"headers"`
	Delay   time.Duration     `yaml:"-"`
	Raw     string            `yaml:"raw"`
	File    string            `yaml:"file"`
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

func (r *Response) IsRaw() bool {
	return len(r.Raw) > 0
}

func (r *Response) IsFile() bool {
	return len(r.File) > 0
}

// UnmarshalYAML implements custom decoding so that the "delay" field can be
// expressed as a human-readable duration string (e.g. "200ms", "1s 500ms").
// All other fields are decoded by the standard yaml.v3 machinery.
func (r *Response) UnmarshalYAML(value *yaml.Node) error {
	type responseRaw struct {
		Code    int               `yaml:"code"`
		Headers map[string]string `yaml:"headers"`
		Delay   string            `yaml:"delay"`
		Raw     string            `yaml:"raw"`
		File    string            `yaml:"file"`
	}

	var raw responseRaw

	err := value.Decode(&raw)
	if err != nil {
		return err
	}

	r.Code = raw.Code
	r.Headers = raw.Headers
	r.Raw = raw.Raw
	r.File = raw.File

	if raw.Delay == "" {
		return nil
	}

	dur, err := time.ParseDuration(strings.ReplaceAll(raw.Delay, " ", ""))
	if err != nil {
		return fmt.Errorf("invalid delay %q: %w", raw.Delay, err)
	}

	r.Delay = dur

	return nil
}
