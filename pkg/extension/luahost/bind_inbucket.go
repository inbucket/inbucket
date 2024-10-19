package luahost

import (
	"errors"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

const (
	inbucketName       = "inbucket"
	inbucketBeforeName = "inbucket_before"
	inbucketAfterName  = "inbucket_after"
)

// Inbucket is the primary Lua interface data structure.
type Inbucket struct {
	After  InbucketAfterFuncs
	Before InbucketBeforeFuncs
}

// InbucketAfterFuncs holds references to Lua extension functions to be called async
// after Inbucket handles an event.
type InbucketAfterFuncs struct {
	MessageDeleted *lua.LFunction
	MessageStored  *lua.LFunction
}

// InbucketBeforeFuncs holds references to Lua extension functions to be called
// before Inbucket handles an event.
type InbucketBeforeFuncs struct {
	MailFromAccepted *lua.LFunction
	MessageStored    *lua.LFunction
	RcptToAccepted   *lua.LFunction
}

func registerInbucketTypes(ls *lua.LState) {
	// inbucket type.
	mt := ls.NewTypeMetatable(inbucketName)
	ls.SetField(mt, "__index", ls.NewFunction(inbucketIndex))

	// inbucket global var.
	ud := wrapInbucket(ls, &Inbucket{})
	ls.SetGlobal(inbucketName, ud)

	// inbucket.after type.
	mt = ls.NewTypeMetatable(inbucketAfterName)
	ls.SetField(mt, "__index", ls.NewFunction(inbucketAfterIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(inbucketAfterNewIndex))

	// inbucket.before type.
	mt = ls.NewTypeMetatable(inbucketBeforeName)
	ls.SetField(mt, "__index", ls.NewFunction(inbucketBeforeIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(inbucketBeforeNewIndex))
}

func wrapInbucket(ls *lua.LState, val *Inbucket) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(inbucketName))

	return ud
}

func wrapInbucketAfter(ls *lua.LState, val *InbucketAfterFuncs) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(inbucketAfterName))

	return ud
}

func wrapInbucketBefore(ls *lua.LState, val *InbucketBeforeFuncs) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(inbucketBeforeName))

	return ud
}

func getInbucket(ls *lua.LState) (*Inbucket, error) {
	lv := ls.GetGlobal(inbucketName)
	if lv == nil {
		return nil, errors.New("inbucket object was nil")
	}

	ud, ok := lv.(*lua.LUserData)
	if !ok {
		return nil, fmt.Errorf("inbucket object was type %s instead of UserData", lv.Type())
	}

	val, ok := ud.Value.(*Inbucket)
	if !ok {
		return nil, fmt.Errorf("inbucket object (%v) could not be cast", ud.Value)
	}

	return val, nil
}

func checkInbucket(ls *lua.LState, pos int) *Inbucket {
	ud := ls.CheckUserData(pos)
	if val, ok := ud.Value.(*Inbucket); ok {
		return val
	}
	ls.ArgError(1, inbucketName+" expected")
	return nil
}

func checkInbucketAfter(ls *lua.LState, pos int) *InbucketAfterFuncs {
	ud := ls.CheckUserData(pos)
	if val, ok := ud.Value.(*InbucketAfterFuncs); ok {
		return val
	}
	ls.ArgError(1, inbucketAfterName+" expected")
	return nil
}

func checkInbucketBefore(ls *lua.LState, pos int) *InbucketBeforeFuncs {
	ud := ls.CheckUserData(pos)
	if val, ok := ud.Value.(*InbucketBeforeFuncs); ok {
		return val
	}
	ls.ArgError(1, inbucketBeforeName+" expected")
	return nil
}

// inbucket getter.
func inbucketIndex(ls *lua.LState) int {
	ib := checkInbucket(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "after":
		ls.Push(wrapInbucketAfter(ls, &ib.After))
	case "before":
		ls.Push(wrapInbucketBefore(ls, &ib.Before))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// inbucket.after getter.
func inbucketAfterIndex(ls *lua.LState) int {
	after := checkInbucketAfter(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "message_deleted":
		ls.Push(funcOrNil(after.MessageDeleted))
	case "message_stored":
		ls.Push(funcOrNil(after.MessageStored))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// inbucket.after setter.
func inbucketAfterNewIndex(ls *lua.LState) int {
	m := checkInbucketAfter(ls, 1)
	index := ls.CheckString(2)

	switch index {
	case "message_deleted":
		m.MessageDeleted = ls.CheckFunction(3)
	case "message_stored":
		m.MessageStored = ls.CheckFunction(3)
	default:
		ls.RaiseError("invalid inbucket.after index %q", index)
	}

	return 0
}

// inbucket.before getter.
func inbucketBeforeIndex(ls *lua.LState) int {
	before := checkInbucketBefore(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "mail_from_accepted":
		ls.Push(funcOrNil(before.MailFromAccepted))
	case "message_stored":
		ls.Push(funcOrNil(before.MessageStored))
	case "rcpt_to_accepted":
		ls.Push(funcOrNil(before.RcptToAccepted))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// inbucket.before setter.
func inbucketBeforeNewIndex(ls *lua.LState) int {
	m := checkInbucketBefore(ls, 1)
	index := ls.CheckString(2)

	switch index {
	case "mail_from_accepted":
		m.MailFromAccepted = ls.CheckFunction(3)
	case "message_stored":
		m.MessageStored = ls.CheckFunction(3)
	case "rcpt_to_accepted":
		m.RcptToAccepted = ls.CheckFunction(3)
	default:
		ls.RaiseError("invalid inbucket.before index %q", index)
	}

	return 0
}

func funcOrNil(f *lua.LFunction) lua.LValue {
	if f == nil {
		return lua.LNil
	}

	return f
}
