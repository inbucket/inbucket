package model

import (
	"net/mail"
	"time"
)

// JSONMessageHeaderV1 contains the basic header data for a message
type JSONMessageHeaderV1 struct {
	Mailbox string    `json:"mailbox"`
	ID      string    `json:"id"`
	From    string    `json:"from"`
	To      []string  `json:"to"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
	Size    int64     `json:"size"`
}

// JSONMessageV1 contains the same data as the header plus a JSONMessageBody
type JSONMessageV1 struct {
	Mailbox     string                     `json:"mailbox"`
	ID          string                     `json:"id"`
	From        string                     `json:"from"`
	To          []string                   `json:"to"`
	Subject     string                     `json:"subject"`
	Date        time.Time                  `json:"date"`
	Size        int64                      `json:"size"`
	Body        *JSONMessageBodyV1         `json:"body"`
	Header      mail.Header                `json:"header"`
	Attachments []*JSONMessageAttachmentV1 `json:"attachments"`
}

type JSONMessageAttachmentV1 struct {
	FileName     string `json:"filename"`
	ContentType  string `json:"content-type"`
	DownloadLink string `json:"download-link"`
	ViewLink     string `json:"view-link"`
	MD5          string `json:"md5"`
}

// JSONMessageBodyV1 contains the Text and HTML versions of the message body
type JSONMessageBodyV1 struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}
