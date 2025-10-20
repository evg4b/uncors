package script

import lua "github.com/yuin/gopher-lua"

// newLuaState creates and initializes a new Lua state with standard libraries.
func newLuaState() *lua.LState {
	L := lua.NewState()
	loadStandardLibraries(L)
	return L
}

// loadStandardLibraries loads the standard Lua libraries into the given state.
func loadStandardLibraries(L *lua.LState) {
	L.SetGlobal("_G", L.Get(lua.GlobalsIndex))
	L.PreloadModule("math", lua.OpenMath)
	L.PreloadModule("string", lua.OpenString)
	L.PreloadModule("table", lua.OpenTable)
	L.PreloadModule("os", lua.OpenOs)
}
