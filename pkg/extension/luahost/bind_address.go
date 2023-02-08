package luahost

import (
	"net/mail"

	lua "github.com/yuin/gopher-lua"
)

const mailAddressName = "address"

func registerMailAddressType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(mailAddressName)
	ls.SetGlobal(mailAddressName, mt)

	// Static attributes.
	ls.SetField(mt, "new", ls.NewFunction(newMailAddress))

	// Methods.
	ls.SetField(mt, "__index", ls.NewFunction(mailAddressIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(mailAddressNewIndex))
}

func newMailAddress(ls *lua.LState) int {
	val := &mail.Address{
		Name:    ls.CheckString(1),
		Address: ls.CheckString(2),
	}
	ud := wrapMailAddress(ls, val)
	ls.Push(ud)

	return 1
}

func wrapMailAddress(ls *lua.LState, val *mail.Address) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(mailAddressName))

	return ud
}

func unwrapMailAddress(ud *lua.LUserData) (*mail.Address, bool) {
	val, ok := ud.Value.(*mail.Address)
	return val, ok
}

func checkMailAddress(ls *lua.LState, pos int) *mail.Address {
	ud := ls.CheckUserData(pos)
	if val, ok := ud.Value.(*mail.Address); ok {
		return val
	}
	ls.ArgError(1, mailAddressName+" expected")
	return nil
}

// Gets a field value from MailAddress user object.  This emulates a Lua table,
// allowing `msg.subject` instead of a Lua object syntax of `msg:subject()`.
func mailAddressIndex(ls *lua.LState) int {
	a := checkMailAddress(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "name":
		ls.Push(lua.LString(a.Name))
	case "address":
		ls.Push(lua.LString(a.Address))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// Sets a field value on MailAddress user object.  This emulates a Lua table,
// allowing `msg.subject = x` instead of a Lua object syntax of `msg:subject(x)`.
func mailAddressNewIndex(ls *lua.LState) int {
	a := checkMailAddress(ls, 1)
	index := ls.CheckString(2)

	switch index {
	case "name":
		a.Name = ls.CheckString(3)
	case "address":
		a.Address = ls.CheckString(3)
	default:
		ls.RaiseError("invalid index %q", index)
	}

	return 0
}
