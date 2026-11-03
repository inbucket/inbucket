package luahost

import (
	"fmt"
	"net/mail"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	lua "github.com/yuin/gopher-lua"
)

const inboundMessageName = "inbound_message"

func registerInboundMessageType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(inboundMessageName)
	ls.SetGlobal(inboundMessageName, mt)

	// Static attributes.
	ls.SetField(mt, "new", ls.NewFunction(newInboundMessage))

	// Methods.
	ls.SetField(mt, "__index", ls.NewFunction(inboundMessageIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(inboundMessageNewIndex))
}

func newInboundMessage(ls *lua.LState) int {
	val := &event.InboundMessage{}
	ud := wrapInboundMessage(ls, val)
	ls.Push(ud)

	return 1
}

func wrapInboundMessage(ls *lua.LState, val *event.InboundMessage) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(inboundMessageName))

	return ud
}

// Checks there is an InboundMessage at stack position `pos`, else throws Lua error.
func checkInboundMessage(ls *lua.LState, pos int) *event.InboundMessage {
	ud := ls.CheckUserData(pos)
	if v, ok := ud.Value.(*event.InboundMessage); ok {
		return v
	}
	ls.ArgError(pos, inboundMessageName+" expected")
	return nil
}

func unwrapInboundMessage(lv lua.LValue) (*event.InboundMessage, error) {
	if ud, ok := lv.(*lua.LUserData); ok {
		if v, ok := ud.Value.(*event.InboundMessage); ok {
			return v, nil
		}
	}

	return nil, fmt.Errorf("expected InboundMessage, got %q", lv.Type().String())
}

// Gets a field value from InboundMessage user object.  This emulates a Lua table,
// allowing `msg.subject` instead of a Lua object syntax of `msg:subject()`.
func inboundMessageIndex(ls *lua.LState) int {
	m := checkInboundMessage(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "mailboxes":
		lt := &lua.LTable{}
		for _, v := range m.Mailboxes {
			lt.Append(lua.LString(v))
		}
		ls.Push(lt)
	case "from":
		ls.Push(wrapMailAddress(ls, m.From))
	case "to":
		lt := &lua.LTable{}
		for _, v := range m.To {
			addr := v
			lt.Append(wrapMailAddress(ls, addr))
		}
		ls.Push(lt)
	case "subject":
		ls.Push(lua.LString(m.Subject))
	case "size":
		ls.Push(lua.LNumber(m.Size))
	case "header":
		ls.Push(wrapHeader(ls, m.Headers))
	case "labels":
		ls.Push(wrapLabels(m.Labels))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// Sets a field value on InboundMessage user object.  This emulates a Lua table,
// allowing `msg.subject = x` instead of a Lua object syntax of `msg:subject(x)`.
func inboundMessageNewIndex(ls *lua.LState) int {
	m := checkInboundMessage(ls, 1)
	index := ls.CheckString(2)

	switch index {
	case "mailboxes":
		lt := ls.CheckTable(3)
		mailboxes := make([]string, 0, 16)
		lt.ForEach(func(k, lv lua.LValue) {
			if mb, ok := lv.(lua.LString); ok {
				mailboxes = append(mailboxes, string(mb))
			}
		})
		m.Mailboxes = mailboxes
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
	case "subject":
		m.Subject = ls.CheckString(3)
	case "size":
		ls.RaiseError("size is read-only")
	case "header":
		ls.RaiseError("header is read-only")
	case "labels":
		m.Labels = checkLabels(ls, 3)
	default:
		ls.RaiseError("invalid index %q", index)
	}

	return 0
}

// Converts labels to a Lua array of `{name, description, color}` tables.
func wrapLabels(labels []event.Label) *lua.LTable {
	lt := &lua.LTable{}
	for _, label := range labels {
		entry := &lua.LTable{}
		entry.RawSetString("name", lua.LString(label.Name))
		entry.RawSetString("description", lua.LString(label.Description))
		entry.RawSetString("color", lua.LString(label.Color))
		lt.Append(entry)
	}

	return lt
}

// Checks there is an array of label tables or nil at stack position `pos`, else throws Lua
// error.  Each label must have a non-empty `name` string and may have `description` and `color`
// strings.
func checkLabels(ls *lua.LState, pos int) []event.Label {
	if ls.Get(pos) == lua.LNil {
		return nil
	}
	lt := ls.CheckTable(pos)
	keys := 0
	lt.ForEach(func(_, _ lua.LValue) { keys++ })
	if keys != lt.Len() {
		ls.ArgError(pos, "labels must be an array of label tables")
		return nil
	}

	labels := make([]event.Label, 0, lt.Len())
	for i := 1; i <= lt.Len(); i++ {
		entry, ok := lt.RawGetInt(i).(*lua.LTable)
		if !ok {
			ls.ArgError(pos, fmt.Sprintf("labels[%d] must be a table", i))
			return nil
		}

		name, ok := entry.RawGetString("name").(lua.LString)
		if !ok || name == "" {
			ls.ArgError(pos, fmt.Sprintf("labels[%d].name must be a non-empty string", i))
			return nil
		}

		label := event.Label{Name: string(name)}
		for _, field := range []string{"description", "color"} {
			switch fv := entry.RawGetString(field).(type) {
			case *lua.LNilType:
			case lua.LString:
				if field == "description" {
					label.Description = string(fv)
				} else {
					label.Color = string(fv)
				}
			default:
				ls.ArgError(pos, fmt.Sprintf("labels[%d].%s must be a string", i, field))
				return nil
			}
		}

		labels = append(labels, label)
	}

	return labels
}
