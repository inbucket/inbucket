package luahost

import (
	lua "github.com/yuin/gopher-lua"
)

const policyName = "policy"

func registerPolicyType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(policyName)
	ls.SetGlobal(policyName, mt)

	// Static attributes.
	ls.SetField(mt, "allow", lua.LTrue)
	ls.SetField(mt, "deny", lua.LFalse)
	ls.SetField(mt, "defer", lua.LNil)
}
