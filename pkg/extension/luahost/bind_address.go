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
	ls.SetField(mt, "__index", ls.SetFuncs(ls.NewTable(), mailAddressMethods))
}

var mailAddressMethods = map[string]lua.LGFunction{
	"address": mailAddressGetSetAddress,
	"name":    mailAddressGetSetName,
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

func checkMailAddress(ls *lua.LState) *mail.Address {
	ud := ls.CheckUserData(1)
	if val, ok := ud.Value.(*mail.Address); ok {
		return val
	}
	ls.ArgError(1, mailAddressName+" expected")
	return nil
}

func mailAddressGetSetAddress(ls *lua.LState) int {
	val := checkMailAddress(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.Address = ls.CheckString(2)
		return 0
	}

	// Getter.
	ls.Push(lua.LString(val.Address))
	return 1
}

func mailAddressGetSetName(ls *lua.LState) int {
	val := checkMailAddress(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.Name = ls.CheckString(2)
		return 0
	}

	// Getter.
	ls.Push(lua.LString(val.Name))
	return 1
}
