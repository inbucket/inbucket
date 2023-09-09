package webui

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/inbucket/inbucket/v3/pkg/server/web"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/stringutil"
	"github.com/inbucket/inbucket/v3/pkg/webui/sanitize"
	"github.com/rs/zerolog/log"
)

// MailboxMessage outputs a particular message as JSON for the UI.
func MailboxMessage(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	id := ctx.Vars["id"]
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	msg, err := ctx.Manager.GetMessage(name, id)
	if err != nil && err != storage.ErrNotExist {
		return fmt.Errorf("GetMessage(%q) failed: %v", id, err)
	}
	if msg == nil {
		http.NotFound(w, req)
		return nil
	}

	attachments := make([]*jsonAttachment, 0)
	for i, part := range msg.Attachments() {
		attachments = append(attachments, &jsonAttachment{
			ID:          strconv.Itoa(i),
			FileName:    part.FileName,
			ContentType: part.ContentType,
		})
	}

	mimeErrors := make([]*jsonMIMEError, 0)
	for _, e := range msg.MIMEErrors() {
		mimeErrors = append(mimeErrors, &jsonMIMEError{
			Name:   e.Name,
			Detail: e.Detail,
			Severe: e.Severe,
		})
	}

	// Sanitize HTML body.
	htmlBody := ""
	if msg.HTML() != "" {
		if str, err := sanitize.HTML(msg.HTML()); err == nil {
			htmlBody = str
		} else {
			htmlBody = "Inbucket HTML sanitizer failed."
			log.Warn().Str("module", "webui").Str("mailbox", name).Str("id", id).Err(err).
				Msg("HTML sanitizer failed")
		}
	}

	return web.RenderJSON(w,
		&jsonMessage{
			Mailbox:     name,
			ID:          msg.ID,
			From:        stringutil.StringAddress(msg.From),
			To:          stringutil.StringAddressList(msg.To),
			Subject:     msg.Subject,
			Date:        msg.Date,
			PosixMillis: msg.Date.UnixNano() / 1000000,
			Size:        msg.Size,
			Seen:        msg.Seen,
			Header:      msg.Header(),
			Text:        web.TextToHTML(msg.Text()),
			HTML:        htmlBody,
			Attachments: attachments,
			Errors:      mimeErrors,
		})
}

// MailboxHTML displays the HTML content of a message. Renders a partial
func MailboxHTML(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	msg, err := ctx.Manager.GetMessage(name, id)
	if err == storage.ErrNotExist {
		http.NotFound(w, req)
		return nil
	}
	if err != nil {
		// This doesn't indicate empty, likely an IO error
		return fmt.Errorf("GetMessage(%q) failed: %v", id, err)
	}
	// Render HTML
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	_, err = w.Write([]byte(msg.HTML()))
	return err
}

// MailboxSource displays the raw source of a message, including headers. Renders text/plain
func MailboxSource(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	r, err := ctx.Manager.SourceReader(name, id)
	if err == storage.ErrNotExist {
		http.NotFound(w, req)
		return nil
	}
	if err != nil {
		// This doesn't indicate missing, likely an IO error
		return fmt.Errorf("SourceReader(%q) failed: %v", id, err)
	}
	// Output message source
	w.Header().Set("Content-Type", "text/plain")
	_, err = io.Copy(w, r)
	return err
}

// MailboxViewAttach sends the attachment to the client for online viewing
func MailboxViewAttach(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	id := ctx.Vars["id"]
	numStr := ctx.Vars["num"]
	num, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		return err
	}
	msg, err := ctx.Manager.GetMessage(name, id)
	if err == storage.ErrNotExist {
		http.NotFound(w, req)
		return nil
	}
	if err != nil {
		// This doesn't indicate empty, likely an IO error
		return fmt.Errorf("GetMessage(%q) failed: %v", id, err)
	}
	if int(num) >= len(msg.Attachments()) {
		return errors.New("requested attachment number does not exist")
	}
	// Output attachment
	part := msg.Attachments()[num]
	w.Header().Set("Content-Type", part.ContentType)
	_, err = w.Write(part.Content)
	return err
}
