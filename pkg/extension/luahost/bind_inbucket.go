package luahost

import (
	"errors"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

const (
	inbucketName      = "inbucket"
	inbucketAfterName = "inbucket_after"

	// TODO remove?
	afterMessageDeletedFnName = "after_message_deleted"
	afterMessageStoredFnName  = "after_message_stored"
	beforeMailAcceptedFnName  = "before_mail_accepted"
)

type Inbucket struct {
	After InbucketAfterFuncs
}

type InbucketAfterFuncs struct {
	MessageStored *lua.LFunction
}

func registerInbucketTypes(ls *lua.LState) {
	// inbucket type.
	mt := ls.NewTypeMetatable(inbucketName)
	ls.SetField(mt, "__index", ls.NewFunction(inbucketIndex))

	// inbucket.after type.
	mt = ls.NewTypeMetatable(inbucketAfterName)
	ls.SetField(mt, "__index", ls.NewFunction(inbucketAfterIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(inbucketAfterNewIndex))

	// inbucket global.
	ud := wrapInbucket(ls, &Inbucket{
		After: InbucketAfterFuncs{},
	})
	ls.SetGlobal(inbucketName, ud)
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

// inbucket getter.
func inbucketIndex(ls *lua.LState) int {
	ib := checkInbucket(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "after":
		ls.Push(wrapInbucketAfter(ls, &ib.After))
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
	case "message_stored":
		m.MessageStored = ls.CheckFunction(3)
	default:
		ls.RaiseError("invalid inbucket.after index %q", index)
	}

	return 0
}

func funcOrNil(f *lua.LFunction) lua.LValue {
	if f == nil {
		return lua.LNil
	}

	return f
}
