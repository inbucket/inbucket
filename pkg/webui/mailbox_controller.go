package webui

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jhillyerd/inbucket/pkg/server/web"
	"github.com/jhillyerd/inbucket/pkg/storage"
	"github.com/jhillyerd/inbucket/pkg/stringutil"
	"github.com/jhillyerd/inbucket/pkg/webui/sanitize"
	"github.com/rs/zerolog/log"
)

// JSONMessage formats message data for the UI.
type JSONMessage struct {
	Mailbox     string              `json:"mailbox"`
	ID          string              `json:"id"`
	From        string              `json:"from"`
	To          []string            `json:"to"`
	Subject     string              `json:"subject"`
	Date        time.Time           `json:"date"`
	Size        int64               `json:"size"`
	Seen        bool                `json:"seen"`
	Header      map[string][]string `json:"header"`
	Text        string              `json:"text"`
	HTML        string              `json:"html"`
	Attachments []*JSONAttachment   `json:"attachments"`
}

// JSONAttachment formats attachment data for the UI.
type JSONAttachment struct {
	ID          string `json:"id"`
	FileName    string `json:"filename"`
	ContentType string `json:"content-type"`
}

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
	attachParts := msg.Attachments()
	attachments := make([]*JSONAttachment, len(attachParts))
	for i, part := range attachParts {
		attachments[i] = &JSONAttachment{
			ID:          strconv.Itoa(i),
			FileName:    part.FileName,
			ContentType: part.ContentType,
		}
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
		&JSONMessage{
			Mailbox:     name,
			ID:          msg.ID,
			From:        msg.From.String(),
			To:          stringutil.StringAddressList(msg.To),
			Subject:     msg.Subject,
			Date:        msg.Date,
			Size:        msg.Size,
			Seen:        msg.Seen,
			Header:      msg.Header(),
			Text:        web.TextToHTML(msg.Text()),
			HTML:        htmlBody,
			Attachments: attachments,
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
	// Render partial template
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	return web.RenderPartial("mailbox/_html.html", w, map[string]interface{}{
		"ctx":     ctx,
		"name":    name,
		"message": msg,
		"body":    template.HTML(msg.HTML()),
	})
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
		ctx.Session.AddFlash(err.Error(), "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	id := ctx.Vars["id"]
	numStr := ctx.Vars["num"]
	num, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		ctx.Session.AddFlash("Attachment number must be unsigned numeric", "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
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
		ctx.Session.AddFlash("Attachment number too high", "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	// Output attachment
	part := msg.Attachments()[num]
	w.Header().Set("Content-Type", part.ContentType)
	_, err = w.Write(part.Content)
	return err
}
