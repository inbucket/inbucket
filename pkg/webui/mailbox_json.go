package webui

import "time"

// jsonMessage formats message data for the UI.
type jsonMessage struct {
	Mailbox     string              `json:"mailbox"`
	ID          string              `json:"id"`
	From        string              `json:"from"`
	To          []string            `json:"to"`
	Subject     string              `json:"subject"`
	Date        time.Time           `json:"date"`
	PosixMillis int64               `json:"posix-millis"`
	Size        int64               `json:"size"`
	Seen        bool                `json:"seen"`
	Header      map[string][]string `json:"header"`
	Text        string              `json:"text"`
	HTML        string              `json:"html"`
	Attachments []*jsonAttachment   `json:"attachments"`
	Errors      []*jsonMIMEError    `json:"errors"`
}

// jsonAttachment formats attachment data for the UI.
type jsonAttachment struct {
	ID          string `json:"id"`
	FileName    string `json:"filename"`
	ContentType string `json:"content-type"`
}

type jsonMIMEError struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Severe bool   `json:"severe"`
}
