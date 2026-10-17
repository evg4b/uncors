package script

import (
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
)

// newLuaState opens every gopher-lua standard library (lua.NewState does
// that) and preloads the json module.
func newLuaState() *lua.LState {
	luaState := lua.NewState()
	luajson.Preload(luaState)
	replaceOsExit(luaState)

	return luaState
}

// replaceOsExit makes os.exit end the script with an error: the original calls
// os.Exit and would stop UNCORS itself. The global os table is the one
// require("os") returns, so this covers both.
func replaceOsExit(luaState *lua.LState) {
	osTable, ok := luaState.GetGlobal(lua.OsLibName).(*lua.LTable)
	if !ok {
		return
	}

	osTable.RawSetString("exit", luaState.NewFunction(func(state *lua.LState) int {
		state.RaiseError("%s", ErrOsExit.Error())

		return 0
	}))
}
