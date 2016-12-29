package rest

import (
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"time"

	"github.com/jhillyerd/inbucket/httpd"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/smtpd"
	"strconv"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
)

// JSONMessageHeaderV1 contains the basic header data for a message
type JSONMessageHeaderV1 struct {
	Mailbox string    `json:"mailbox"`
	ID      string    `json:"id"`
	From    string    `json:"from"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
	Size    int64     `json:"size"`
}

// JSONMessageV1 contains the same data as the header plus a JSONMessageBody
type JSONMessageV1 struct {
	Mailbox string             `json:"mailbox"`
	ID      string             `json:"id"`
	From    string             `json:"from"`
	Subject string             `json:"subject"`
	Date    time.Time          `json:"date"`
	Size    int64              `json:"size"`
	Body    *JSONMessageBodyV1 `json:"body"`
	Header  mail.Header        `json:"header"`
	Attachments []*JSONMessageAttachmentV1 `json:"attachments"`
}

type JSONMessageAttachmentV1 struct {
	FileName     string                `json:"filename"`
	ContentType  string        `json:"content-type"`
	DownloadLink string        `json:"download-link"`
	ViewLink     string        `json:"view-link"`
	MD5	     string	   `json:"md5"`
}

// JSONMessageBodyV1 contains the Text and HTML versions of the message body
type JSONMessageBodyV1 struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

// MailboxListV1 renders a list of messages in a mailbox
func MailboxListV1(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		return err
	}
	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		// This doesn't indicate not found, likely an IO error
		return fmt.Errorf("Failed to get mailbox for %q: %v", name, err)
	}
	messages, err := mb.GetMessages()
	if err != nil {
		// This doesn't indicate empty, likely an IO error
		return fmt.Errorf("Failed to get messages for %v: %v", name, err)
	}
	log.Tracef("Got %v messsages", len(messages))

	jmessages := make([]*JSONMessageHeaderV1, len(messages))
	for i, msg := range messages {
		jmessages[i] = &JSONMessageHeaderV1{
			Mailbox: name,
			ID:      msg.ID(),
			From:    msg.From(),
			Subject: msg.Subject(),
			Date:    msg.Date(),
			Size:    msg.Size(),
		}
	}
	return httpd.RenderJSON(w, jmessages)
}

// MailboxShowV1 renders a particular message from a mailbox
func MailboxShowV1(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		return err
	}
	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		// This doesn't indicate not found, likely an IO error
		return fmt.Errorf("Failed to get mailbox for %q: %v", name, err)
	}
	msg, err := mb.GetMessage(id)
	if err == smtpd.ErrNotExist {
		http.NotFound(w, req)
		return nil
	}
	if err != nil {
		// This doesn't indicate empty, likely an IO error
		return fmt.Errorf("GetMessage(%q) failed: %v", id, err)
	}
	header, err := msg.ReadHeader()
	if err != nil {
		return fmt.Errorf("ReadHeader(%q) failed: %v", id, err)
	}
	mime, err := msg.ReadBody()
	if err != nil {
		return fmt.Errorf("ReadBody(%q) failed: %v", id, err)
	}

	attachments := make([]*JSONMessageAttachmentV1, len(mime.Attachments))
	for i, att := range mime.Attachments {
		var content []byte
		content, err = ioutil.ReadAll(att)
		var checksum = md5.Sum(content)
		attachments[i] = &JSONMessageAttachmentV1{
			ContentType: att.ContentType,
			FileName: att.FileName,
			DownloadLink:  "http://" + req.Host + "/mailbox/dattach/" + name + "/" + id + "/" + strconv.Itoa(i) + "/" + att.FileName,
			ViewLink: "http://" + req.Host + "/mailbox/vattach/" + name + "/" + id + "/" + strconv.Itoa(i) + "/" + att.FileName,
			MD5: hex.EncodeToString(checksum[:]),
		}
	}

	return httpd.RenderJSON(w,
		&JSONMessageV1{
			Mailbox: name,
			ID:      msg.ID(),
			From:    msg.From(),
			Subject: msg.Subject(),
			Date:    msg.Date(),
			Size:    msg.Size(),
			Header:  header.Header,
			Body: &JSONMessageBodyV1{
				Text: mime.Text,
				HTML: mime.HTML,
			},
			Attachments: attachments,
		})
}

// MailboxPurgeV1 deletes all messages from a mailbox
func MailboxPurgeV1(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		return err
	}
	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		// This doesn't indicate not found, likely an IO error
		return fmt.Errorf("Failed to get mailbox for %q: %v", name, err)
	}
	// Delete all messages
	err = mb.Purge()
	if err != nil {
		return fmt.Errorf("Mailbox(%q) purge failed: %v", name, err)
	}
	log.Tracef("HTTP purged mailbox for %q", name)

	return httpd.RenderJSON(w, "OK")
}

// MailboxSourceV1 displays the raw source of a message, including headers. Renders text/plain
func MailboxSourceV1(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		return err
	}
	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		// This doesn't indicate not found, likely an IO error
		return fmt.Errorf("Failed to get mailbox for %q: %v", name, err)
	}
	message, err := mb.GetMessage(id)
	if err == smtpd.ErrNotExist {
		http.NotFound(w, req)
		return nil
	}
	if err != nil {
		// This doesn't indicate missing, likely an IO error
		return fmt.Errorf("GetMessage(%q) failed: %v", id, err)
	}
	raw, err := message.ReadRaw()
	if err != nil {
		return fmt.Errorf("ReadRaw(%q) failed: %v", id, err)
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := io.WriteString(w, *raw); err != nil {
		return err
	}
	return nil
}

// MailboxDeleteV1 removes a particular message from a mailbox.  Renders JSON or plain/text OK
func MailboxDeleteV1(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		return err
	}
	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		// This doesn't indicate not found, likely an IO error
		return fmt.Errorf("Failed to get mailbox for %q: %v", name, err)
	}
	message, err := mb.GetMessage(id)
	if err == smtpd.ErrNotExist {
		http.NotFound(w, req)
		return nil
	}
	if err != nil {
		// This doesn't indicate missing, likely an IO error
		return fmt.Errorf("GetMessage(%q) failed: %v", id, err)
	}
	err = message.Delete()
	if err != nil {
		return fmt.Errorf("Delete(%q) failed: %v", id, err)
	}

	return httpd.RenderJSON(w, "OK")
}
