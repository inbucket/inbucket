package model

// JSONMessageIDV2 uniquely identifies a message.
type JSONMessageIDV2 struct {
	Mailbox string `json:"mailbox"`
	ID      string `json:"id"`
}

// JSONMonitorEventV2 contains events for the Inbucket mailbox and monitor tabs.
type JSONMonitorEventV2 struct {
	// Event variant: `message-deleted`, `message-stored`.
	Variant    string               `json:"variant"`
	Identifier *JSONMessageIDV2     `json:"identifier"`
	Header     *JSONMessageHeaderV1 `json:"header"`
}
