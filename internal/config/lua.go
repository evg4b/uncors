package config

import (
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/samber/lo"
)

type LuaScript struct {
	Path    string            `mapstructure:"path"`
	Method  string            `mapstructure:"method"`
	Queries map[string]string `mapstructure:"queries"`
	Headers map[string]string `mapstructure:"headers"`
	Script  string            `mapstructure:"script"`
	File    string            `mapstructure:"file"`
}

func (l *LuaScript) Clone() LuaScript {
	return LuaScript{
		Path:    l.Path,
		Method:  l.Method,
		Queries: helpers.CloneMap(l.Queries),
		Headers: helpers.CloneMap(l.Headers),
		Script:  l.Script,
		File:    l.File,
	}
}

func (l *LuaScript) String() string {
	method := "*"
	if l.Method != "" {
		method = l.Method
	}

	scriptType := "inline"
	if l.File != "" {
		scriptType = "file: " + l.File
	}

	return helpers.Sprintf("[%s lua:%s] %s", method, scriptType, l.Path)
}

type LuaScripts []LuaScript

func (l LuaScripts) Clone() LuaScripts {
	if l == nil {
		return nil
	}

	return lo.Map(l, func(item LuaScript, _ int) LuaScript {
		return item.Clone()
	})
}
