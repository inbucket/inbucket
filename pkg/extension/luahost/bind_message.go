package luahost

import (
	"net/mail"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	lua "github.com/yuin/gopher-lua"
)

const messageMetadataName = "message_metadata"

func registerMessageMetadataType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(messageMetadataName)
	ls.SetGlobal(messageMetadataName, mt)

	// Static attributes.
	ls.SetField(mt, "new", ls.NewFunction(newMessageMetadata))

	// Methods.
	ls.SetField(mt, "__index", ls.NewFunction(messageMetadataIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(messageMetadataNewIndex))
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

func checkMessageMetadata(ls *lua.LState, pos int) *event.MessageMetadata {
	ud := ls.CheckUserData(pos)
	if v, ok := ud.Value.(*event.MessageMetadata); ok {
		return v
	}
	ls.ArgError(1, messageMetadataName+" expected")
	return nil
}

// Gets a field value from MessageMetadata user object.  This emulates a Lua table,
// allowing `msg.subject` instead of a Lua object syntax of `msg:subject()`.
func messageMetadataIndex(ls *lua.LState) int {
	m := checkMessageMetadata(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "mailbox":
		ls.Push(lua.LString(m.Mailbox))
	case "id":
		ls.Push(lua.LString(m.ID))
	case "from":
		ls.Push(wrapMailAddress(ls, m.From))
	case "to":
		lt := &lua.LTable{}
		for _, v := range m.To {
			lt.Append(wrapMailAddress(ls, v))
		}
		ls.Push(lt)
	case "date":
		ls.Push(lua.LNumber(m.Date.Unix()))
	case "subject":
		ls.Push(lua.LString(m.Subject))
	case "size":
		ls.Push(lua.LNumber(m.Size))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// Sets a field value on MessageMetadata user object.  This emulates a Lua table,
// allowing `msg.subject = x` instead of a Lua object syntax of `msg:subject(x)`.
func messageMetadataNewIndex(ls *lua.LState) int {
	m := checkMessageMetadata(ls, 1)
	index := ls.CheckString(2)

	switch index {
	case "mailbox":
		m.Mailbox = ls.CheckString(3)
	case "id":
		m.ID = ls.CheckString(3)
	case "from":
		m.From = checkMailAddress(ls, 3)
	case "to":
		lt := ls.CheckTable(3)
		to := make([]*mail.Address, 0, 16)
		lt.ForEach(func(k, lv lua.LValue) {
			if ud, ok := lv.(*lua.LUserData); ok {
				// TODO should fail if wrong type + test.
				if entry, ok := unwrapMailAddress(ud); ok {
					to = append(to, entry)
				}
			}
		})
		m.To = to
	case "date":
		m.Date = time.Unix(ls.CheckInt64(3), 0)
	case "subject":
		m.Subject = ls.CheckString(3)
	case "size":
		m.Size = ls.CheckInt64(3)
	default:
		ls.RaiseError("invalid index %q", index)
	}

	return 0
}
