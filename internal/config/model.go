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

type Mock struct {
	Path     string            `mapstructure:"path"`
	Method   string            `mapstructure:"method"`
	Queries  map[string]string `mapstructure:"queries"`
	Headers  map[string]string `mapstructure:"headers"`
	Response Response          `mapstructure:"response"`
}
