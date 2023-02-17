package model

import (
	"time"
)

// JSONMessageHeaderV1 contains the basic header data for a message.
type JSONMessageHeaderV1 struct {
	Mailbox     string    `json:"mailbox"`
	ID          string    `json:"id"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	Subject     string    `json:"subject"`
	Date        time.Time `json:"date"`
	PosixMillis int64     `json:"posix-millis"`
	Size        int64     `json:"size"`
	Seen        bool      `json:"seen"`
}

// JSONMessageV1 contains the same data as the header plus a JSONMessageBody.
type JSONMessageV1 struct {
	Mailbox     string                     `json:"mailbox"`
	ID          string                     `json:"id"`
	From        string                     `json:"from"`
	To          []string                   `json:"to"`
	Subject     string                     `json:"subject"`
	Date        time.Time                  `json:"date"`
	PosixMillis int64                      `json:"posix-millis"`
	Size        int64                      `json:"size"`
	Seen        bool                       `json:"seen"`
	Body        *JSONMessageBodyV1         `json:"body"`
	Header      map[string][]string        `json:"header"`
	Attachments []*JSONMessageAttachmentV1 `json:"attachments"`
}

// JSONMessageAttachmentV1 contains information about a MIME attachment.
type JSONMessageAttachmentV1 struct {
	FileName     string `json:"filename"`
	ContentType  string `json:"content-type"`
	DownloadLink string `json:"download-link"`
	ViewLink     string `json:"view-link"`
	MD5          string `json:"md5"`
}

// JSONMessageBodyV1 contains the Text and HTML versions of the message body.
type JSONMessageBodyV1 struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}
