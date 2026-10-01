package luahost

import (
	"net/textproto"

	lua "github.com/yuin/gopher-lua"
)

const mimeHeaderName = "header"

func registerMIMEHeaderType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(mimeHeaderName)

	// Methods.
	ls.SetField(mt, "__index", ls.NewFunction(mimeHeaderIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(mimeHeaderNewIndex))
}

func wrapMIMEHeader(ls *lua.LState, val textproto.MIMEHeader) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(mimeHeaderName))

	return ud
}

// Checks there is a MIMEHeader at stack position `pos`, else throws Lua error.
func checkMIMEHeader(ls *lua.LState, pos int) textproto.MIMEHeader {
	ud := ls.CheckUserData(pos)
	if v, ok := ud.Value.(textproto.MIMEHeader); ok {
		return v
	}
	ls.ArgError(pos, mimeHeaderName+" expected")
	return nil
}

// Gets a header value from MIMEHeader user object.  This emulates a Lua table,
// allowing `msg.header["Subject"]` instead of a Lua object syntax of `msg:header("Subject")`.
func mimeHeaderIndex(ls *lua.LState) int {
	h := checkMIMEHeader(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	if vals := h.Values(field); len(vals) > 0 {
		ls.Push(lua.LString(vals[0]))
	} else {
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// Rejects writes to MIMEHeader user object, `msg.header["Subject"] = x` raises an error.
func mimeHeaderNewIndex(ls *lua.LState) int {
	checkMIMEHeader(ls, 1)
	index := ls.CheckString(2)

	ls.RaiseError("header %q is read-only", index)

	return 0
}
