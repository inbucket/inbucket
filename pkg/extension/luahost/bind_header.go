package luahost

import (
	"net/textproto"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

const headerName = "inbound_header"

func registerHeaderType(ls *lua.LState) {
	// header type.
	mt := ls.NewTypeMetatable(headerName)

	// Methods.
	ls.SetField(mt, "__index", ls.NewFunction(headerIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(headerNewIndex))
}

func wrapHeader(ls *lua.LState, val textproto.MIMEHeader) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(headerName))

	return ud
}

// Checks there is a MIMEHeader at stack position `pos`, else throws Lua error.
func checkHeader(ls *lua.LState, pos int) textproto.MIMEHeader {
	ud := ls.CheckUserData(pos)
	if v, ok := ud.Value.(textproto.MIMEHeader); ok {
		return v
	}
	ls.ArgError(pos, headerName+" expected")
	return nil
}

// Gets a header value by name.  Header names are case-insensitive; the first
// value is returned when a header was set more than once, and nil when the
// header is absent or unset.  This emulates a Lua table, allowing
// `msg.header["x-spam-status"]`.
func headerIndex(ls *lua.LState) int {
	h := checkHeader(ls, 1)
	name := ls.CheckString(2)

	// Exact match first, then case-insensitive.  Decoded keys are canonicalized
	// by enmime using email-specific rules that differ from net/textproto, so
	// comparing against the stored keys is more reliable than canonicalizing
	// the requested name.  The header set is small, a scan is cheap.
	if values, ok := h[name]; ok && len(values) > 0 {
		ls.Push(lua.LString(values[0]))
		return 1
	}
	for key, values := range h {
		if len(values) > 0 && strings.EqualFold(key, name) {
			ls.Push(lua.LString(values[0]))
			return 1
		}
	}

	// Header absent, or registered but unset on this message.
	ls.Push(lua.LNil)

	return 1
}

// Rejects writes; headers are read-only.
func headerNewIndex(ls *lua.LState) int {
	checkHeader(ls, 1)
	name := ls.CheckString(2)
	ls.RaiseError("header %q is read-only", name)

	return 0
}
