package model

// JSONMonitorEventV2 contains events for the Inbucket mailbox and monitor tabs.
type JSONMonitorEventV2 struct {
	// Event variant: `message-deleted`, `message-stored`.
	Variant string               `json:"variant"`
	Header  *JSONMessageHeaderV1 `json:"header"`
}
