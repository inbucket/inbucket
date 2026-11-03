package luahost

import (
	"net/mail"
	"net/textproto"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundMessageGetters(t *testing.T) {
	want := &event.InboundMessage{
		Mailboxes: []string{"mb1", "mb2"},
		From:      &mail.Address{Name: "name1", Address: "addr1"},
		To: []*mail.Address{
			{Name: "name2", Address: "addr2"},
			{Name: "name3", Address: "addr3"},
		},
		Subject: "subj1",
		Size:    42,
		Headers: textproto.MIMEHeader{
			"Subject":       {"subj1"},
			"X-Spam-Status": {"Yes"},
		},
		Labels: []event.Label{
			{Name: "Urgent", Description: "X-Priority: 1", Color: "tomato"},
			{Name: "Newsletter"},
		},
	}
	script := `
		assert(msg, "msg should not be nil")

		assert_eq(#msg.labels, 2, "#msg.labels")
		assert_eq(msg.labels[1].name, "Urgent", "labels[1].name")
		assert_eq(msg.labels[1].description, "X-Priority: 1", "labels[1].description")
		assert_eq(msg.labels[1].color, "tomato", "labels[1].color")
		assert_eq(msg.labels[2].name, "Newsletter", "labels[2].name")
		assert_eq(msg.labels[2].description, "", "labels[2].description")
		assert_eq(msg.labels[2].color, "", "labels[2].color")

		assert_eq(msg.mailboxes, {"mb1", "mb2"})
		assert_eq(msg.subject, "subj1")
		assert_eq(msg.size, 42, "msg.size")

		assert_eq(msg.header.subject, "subj1")
		assert_eq(msg.header["x-spam-status"], "Yes")
		assert(msg.header["x-not-registered"] == nil, "unregistered header should be nil")

		assert_eq(msg.from.name, "name1", "from.name")
		assert_eq(msg.from.address, "addr1", "from.address")

		assert_eq(#msg.to, 2, "#msg.to")
		assert_eq(msg.to[1].name, "name2", "to[1].name")
		assert_eq(msg.to[1].address, "addr2", "to[1].address")
		assert_eq(msg.to[2].name, "name3", "to[2].name")
		assert_eq(msg.to[2].address, "addr3", "to[2].address")
	`

	ls, _ := test.NewLuaState()
	registerInboundMessageType(ls)
	registerMailAddressType(ls)
	registerHeaderType(ls)
	ls.SetGlobal("msg", wrapInboundMessage(ls, want))
	require.NoError(t, ls.DoString(script))
}

func TestInboundMessageHeaderReadOnly(t *testing.T) {
	script := `
		msg.header = {}
	`

	got := &event.InboundMessage{}
	ls, _ := test.NewLuaState()
	registerInboundMessageType(ls)
	registerHeaderType(ls)
	ls.SetGlobal("msg", wrapInboundMessage(ls, got))
	err := ls.DoString(script)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "header is read-only")
}

func TestInboundMessageNoLabels(t *testing.T) {
	script := `
		assert_eq(#msg.labels, 0, "labels should be empty")
	`

	ls, _ := test.NewLuaState()
	registerInboundMessageType(ls)
	ls.SetGlobal("msg", wrapInboundMessage(ls, &event.InboundMessage{}))
	require.NoError(t, ls.DoString(script))
}

func TestInboundMessageLabelsSetter(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   []event.Label
	}{
		{
			name: "name and description",
			script: `msg.labels = {
				{ name = "Urgent", description = "X-Priority: 1" },
				{ name = "Newsletter", description = "Has List-Unsubscribe" },
			}`,
			want: []event.Label{
				{Name: "Urgent", Description: "X-Priority: 1"},
				{Name: "Newsletter", Description: "Has List-Unsubscribe"},
			},
		},
		{
			name:   "name only",
			script: `msg.labels = { { name = "Billing" } }`,
			want:   []event.Label{{Name: "Billing"}},
		},
		{
			name: "colors",
			script: `msg.labels = {
				{ name = "Urgent", color = "tomato" },
				{ name = "Account", color = "mediumseagreen" },
				{ name = "Newsletter", color = "#8e7cc3" },
			}`,
			want: []event.Label{
				{Name: "Urgent", Color: "tomato"},
				{Name: "Account", Color: "mediumseagreen"},
				{Name: "Newsletter", Color: "#8e7cc3"},
			},
		},
		{
			name:   "empty table",
			script: `msg.labels = {}`,
			want:   []event.Label{},
		},
		{
			name:   "nil clears",
			script: `msg.labels = { { name = "Urgent" } }; msg.labels = nil`,
			want:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := &event.InboundMessage{}
			ls, _ := test.NewLuaState()
			registerInboundMessageType(ls)
			ls.SetGlobal("msg", wrapInboundMessage(ls, got))
			require.NoError(t, ls.DoString(tc.script))
			assert.Equal(t, tc.want, got.Labels)
		})
	}
}

func TestInboundMessageLabelsSetterErrors(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		wantErr string
	}{
		{
			name:    "not a table",
			script:  `msg.labels = "Urgent"`,
			wantErr: "table expected",
		},
		{
			name:    "single label instead of array",
			script:  `msg.labels = { name = "Urgent" }`,
			wantErr: "labels must be an array of label tables",
		},
		{
			name:    "keyed entry mixed into array",
			script:  `msg.labels = { { name = "Urgent" }, extra = { name = "Billing" } }`,
			wantErr: "labels must be an array of label tables",
		},
		{
			name:    "array with a hole",
			script:  `msg.labels = { [1] = { name = "Urgent" }, [3] = { name = "Billing" } }`,
			wantErr: "labels must be an array of label tables",
		},
		{
			name:    "entry not a table",
			script:  `msg.labels = { "Urgent" }`,
			wantErr: "labels[1] must be a table",
		},
		{
			name:    "missing name",
			script:  `msg.labels = { { description = "X-Priority: 1" } }`,
			wantErr: "labels[1].name must be a non-empty string",
		},
		{
			name:    "empty name",
			script:  `msg.labels = { { name = "" } }`,
			wantErr: "labels[1].name must be a non-empty string",
		},
		{
			name:    "name not a string",
			script:  `msg.labels = { { name = 42 } }`,
			wantErr: "labels[1].name must be a non-empty string",
		},
		{
			name:    "description not a string",
			script:  `msg.labels = { { name = "Urgent" }, { name = "Newsletter", description = true } }`,
			wantErr: "labels[2].description must be a string",
		},
		{
			name:    "color not a string",
			script:  `msg.labels = { { name = "Urgent", color = 1 } }`,
			wantErr: "labels[1].color must be a string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := &event.InboundMessage{}
			ls, _ := test.NewLuaState()
			registerInboundMessageType(ls)
			ls.SetGlobal("msg", wrapInboundMessage(ls, got))
			err := ls.DoString(tc.script)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Nil(t, got.Labels, "labels should not be set on error")
		})
	}
}

func TestInboundMessageSetters(t *testing.T) {
	want := &event.InboundMessage{
		Mailboxes: []string{"mb1", "mb2"},
		From:      &mail.Address{Name: "name1", Address: "addr1"},
		To: []*mail.Address{
			{Name: "name2", Address: "addr2"},
			{Name: "name3", Address: "addr3"},
		},
		Subject: "subj1",
	}
	script := `
		assert(msg, "msg should not be nil")

		msg.mailboxes = {"mb1", "mb2"}
		msg.subject = "subj1"
		msg.from = address.new("name1", "addr1")
		msg.to = { address.new("name2", "addr2"), address.new("name3", "addr3") }
	`

	got := &event.InboundMessage{}
	ls, _ := test.NewLuaState()
	registerInboundMessageType(ls)
	registerMailAddressType(ls)
	ls.SetGlobal("msg", wrapInboundMessage(ls, got))
	require.NoError(t, ls.DoString(script))

	assert.Equal(t, want, got)
}
