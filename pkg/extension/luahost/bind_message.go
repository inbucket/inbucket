package luahost

import (
	"net/mail"
	"time"

	"github.com/inbucket/inbucket/pkg/extension/event"
	lua "github.com/yuin/gopher-lua"
)

const messageMetadataName = "message_metadata"

func registerMessageMetadataType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(messageMetadataName)
	ls.SetGlobal(messageMetadataName, mt)

	// Static attributes.
	ls.SetField(mt, "new", ls.NewFunction(newMessageMetadata))

	// Methods.
	ls.SetField(mt, "__index", ls.SetFuncs(ls.NewTable(), messageMetadataMethods))
}

var messageMetadataMethods = map[string]lua.LGFunction{
	"mailbox": messageMetadataGetSetMailbox,
	"id":      messageMetadataGetSetID,
	"from":    messageMetadataGetSetFrom,
	"to":      messageMetadataGetSetTo,
	"subject": messageMetadataGetSetSubject,
	"date":    messageMetadataGetSetDate,
	"size":    messageMetadataGetSetSize,
}

func newMessageMetadata(ls *lua.LState) int {
	val := &event.MessageMetadata{}
	ud := wrapMessageMetadata(ls, val)
	ls.Push(ud)

	return 1
}

func wrapMessageMetadata(ls *lua.LState, val *event.MessageMetadata) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(messageMetadataName))

	return ud
}

func checkMessageMetadata(ls *lua.LState) *event.MessageMetadata {
	ud := ls.CheckUserData(1)
	if v, ok := ud.Value.(*event.MessageMetadata); ok {
		return v
	}
	ls.ArgError(1, messageMetadataName+" expected")
	return nil
}

func messageMetadataGetSetMailbox(ls *lua.LState) int {
	val := checkMessageMetadata(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.Mailbox = ls.CheckString(2)
		return 0
	}

	// Getter.
	ls.Push(lua.LString(val.Mailbox))
	return 1
}

func messageMetadataGetSetID(ls *lua.LState) int {
	val := checkMessageMetadata(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.ID = ls.CheckString(2)
		return 0
	}

	// Getter.
	ls.Push(lua.LString(val.ID))
	return 1
}

func messageMetadataGetSetFrom(ls *lua.LState) int {
	val := checkMessageMetadata(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.From = checkMailAddress(ls)
		return 0
	}

	// Getter.
	ls.Push(wrapMailAddress(ls, val.From))
	return 1
}

func messageMetadataGetSetTo(ls *lua.LState) int {
	val := checkMessageMetadata(ls)
	if ls.GetTop() == 2 {
		// Setter.
		lt := ls.CheckTable(2)
		to := make([]*mail.Address, lt.Len())
		lt.ForEach(func(k, lv lua.LValue) {
			if ud, ok := lv.(*lua.LUserData); ok {
				if entry, ok := unwrapMailAddress(ud); ok {
					to = append(to, entry)
				}
			}
		})
		val.To = to
		return 0
	}

	// Getter.
	lt := &lua.LTable{}
	for _, v := range val.To {
		lt.Append(wrapMailAddress(ls, v))
	}
	ls.Push(lt)
	return 1
}

func messageMetadataGetSetSubject(ls *lua.LState) int {
	val := checkMessageMetadata(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.Subject = ls.CheckString(2)
		return 0
	}

	// Getter.
	ls.Push(lua.LString(val.Subject))
	return 1
}

func messageMetadataGetSetDate(ls *lua.LState) int {
	val := checkMessageMetadata(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.Date = time.Unix(ls.CheckInt64(2), 0)
		return 0
	}

	// Getter.
	ls.Push(lua.LNumber(val.Date.Unix()))
	return 1
}

func messageMetadataGetSetSize(ls *lua.LState) int {
	val := checkMessageMetadata(ls)
	if ls.GetTop() == 2 {
		// Setter.
		val.Size = ls.CheckInt64(2)
		return 0
	}

	// Getter.
	ls.Push(lua.LNumber(val.Size))
	return 1
}
