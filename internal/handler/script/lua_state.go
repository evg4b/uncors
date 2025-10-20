package script

import lua "github.com/yuin/gopher-lua"

// newLuaState creates and initializes a new Lua state with standard libraries.
func newLuaState() *lua.LState {
	luaState := lua.NewState()
	loadStandardLibraries(luaState)

	return luaState
}

// loadStandardLibraries loads the standard Lua libraries into the given state.
func loadStandardLibraries(luaState *lua.LState) {
	luaState.SetGlobal("_G", luaState.Get(lua.GlobalsIndex))
	luaState.PreloadModule("math", lua.OpenMath)
	luaState.PreloadModule("string", lua.OpenString)
	luaState.PreloadModule("table", lua.OpenTable)
	luaState.PreloadModule("os", lua.OpenOs)
}
