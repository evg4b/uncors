package config

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a configuration duration.
//
// It exists so that the format uncors documents is the format uncors accepts:
// the documented `1m 30s` and `2s 500ms` spellings are what a reader copies, and
// time.ParseDuration rejects the space. Owning the type also lets a bad value
// say what a good one looks like.
//
//nolint:recvcheck // UnmarshalYAML has to take a pointer; the rest reads better as a value
type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var raw string

	err := value.Decode(&raw)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	parsed, err := ParseDuration(raw)
	if err != nil {
		return err
	}

	*d = parsed

	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return d.String(), nil
}

// ParseDuration reads a duration, tolerating the spaces between components that
// the documentation uses: "1m 30s" means the same as "1m30s".
func ParseDuration(raw string) (Duration, error) {
	compact := strings.Join(strings.Fields(raw), "")

	parsed, err := time.ParseDuration(compact)
	if err != nil {
		return 0, &ValidationError{
			fmt.Sprintf("%q is not a valid duration (expected for example 500ms, 30s, 1m30s or 1h 30m)", raw),
		}
	}

	return Duration(parsed), nil
}
