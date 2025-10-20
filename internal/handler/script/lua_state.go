package script

import lua "github.com/yuin/gopher-lua"

func newLuaState() *lua.LState {
	luaState := lua.NewState()
	loadStandardLibraries(luaState)
	return luaState
}

func loadStandardLibraries(luaState *lua.LState) {
	luaState.SetGlobal("_G", luaState.Get(lua.GlobalsIndex))
	luaState.PreloadModule("math", lua.OpenMath)
	luaState.PreloadModule("string", lua.OpenString)
	luaState.PreloadModule("table", lua.OpenTable)
	luaState.PreloadModule("os", lua.OpenOs)
}
