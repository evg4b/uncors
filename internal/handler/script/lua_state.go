package script

import (
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
)

func newLuaState() *lua.LState {
	luaState := lua.NewState()
	luajson.Preload(luaState)
	replaceOsExit(luaState)

	return luaState
}

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
