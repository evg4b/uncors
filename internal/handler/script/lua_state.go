package script

import (
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	luajson "layeh.com/gopher-json"
)

// newLuaState builds the VM a script runs in.
//
// The library set is stated here rather than inherited from lua.NewState's
// defaults: a user needs to know what a config file someone else wrote is able
// to do, and "whatever OpenLibs happens to include" is not an answer. base,
// package (for require), string, table, math, os and json are open; io, debug,
// coroutine and channel are not, so a script cannot read or write files.
func newLuaState() *lua.LState {
	luaState := lua.NewState(lua.Options{SkipOpenLibs: true})

	for _, library := range []struct {
		name   string
		loader lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.LoadLibName, lua.OpenPackage},
		{lua.StringLibName, lua.OpenString},
		{lua.TabLibName, lua.OpenTable},
		{lua.MathLibName, lua.OpenMath},
		{lua.OsLibName, lua.OpenOs},
	} {
		luaState.Push(luaState.NewFunction(library.loader))
		luaState.Push(lua.LString(library.name))
		luaState.Call(1, 0)
	}

	luaState.SetGlobal("_G", luaState.Get(lua.GlobalsIndex))
	luaState.PreloadModule("math", lua.OpenMath)
	luaState.PreloadModule("string", lua.OpenString)
	luaState.PreloadModule("table", lua.OpenTable)
	luaState.PreloadModule("os", lua.OpenOs)
	luajson.Preload(luaState)

	return luaState
}

// compileScript turns the script source into a reusable function prototype.
// Compiling at construction means a syntax error is reported once, with a
// location, instead of on every request that reaches the route.
func compileScript(source, name string) (*lua.FunctionProto, error) {
	chunk, err := parse.Parse(strings.NewReader(source), name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScriptCompilationFailed, err)
	}

	proto, err := lua.Compile(chunk, name)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScriptCompilationFailed, err)
	}

	return proto, nil
}
