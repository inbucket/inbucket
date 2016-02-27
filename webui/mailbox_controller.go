package webui

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	"github.com/jhillyerd/inbucket/httpd"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/smtpd"
)

// JSONMessageHeader contains the basic header data for a message
type JSONMessageHeader struct {
	Mailbox string
	ID      string `json:"Id"`
	From    string
	Subject string
	Date    time.Time
	Size    int64
}

// JSONMessage contains the same data as the header plus a JSONMessageBody
type JSONMessage struct {
	Mailbox string
	ID      string `json:"Id"`
	From    string
	Subject string
	Date    time.Time
	Size    int64
	Body    *JSONMessageBody
	Header  mail.Header
}

// JSONMessageBody contains the Text and HTML versions of the message body
type JSONMessageBody struct {
	Text string
	HTML string `json:"Html"`
}

// MailboxIndex renders the index page for a particular mailbox
func MailboxIndex(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Form values must be validated manually
	name := req.FormValue("name")
	selected := req.FormValue("id")

	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	name, err = smtpd.ParseMailboxName(name)
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	// Remember this mailbox was visited
	RememberMailbox(ctx, name)

	return httpd.RenderTemplate("mailbox/index.html", w, map[string]interface{}{
		"ctx":      ctx,
		"name":     name,
		"selected": selected,
	})
}

// MailboxLink handles pretty links to a particular message. Renders a redirect
func MailboxLink(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	uri := fmt.Sprintf("%s?name=%s&id=%s", httpd.Reverse("MailboxIndex"), name, id)
	http.Redirect(w, req, uri, http.StatusSeeOther)
	return nil
}

// MailboxList renders a list of messages in a mailbox. Renders JSON or a partial
func MailboxList(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
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

	if ctx.IsJSON {
		jmessages := make([]*JSONMessageHeader, len(messages))
		for i, msg := range messages {
			jmessages[i] = &JSONMessageHeader{
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

	return httpd.RenderPartial("mailbox/_list.html", w, map[string]interface{}{
		"ctx":      ctx,
		"name":     name,
		"messages": messages,
	})
}

// MailboxShow renders a particular message from a mailbox. Renders JSON or a partial
func MailboxShow(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
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

	if ctx.IsJSON {
		return httpd.RenderJSON(w,
			&JSONMessage{
				Mailbox: name,
				ID:      msg.ID(),
				From:    msg.From(),
				Subject: msg.Subject(),
				Date:    msg.Date(),
				Size:    msg.Size(),
				Header:  header.Header,
				Body: &JSONMessageBody{
					Text: mime.Text,
					HTML: mime.HTML,
				},
			})
	}

	body := template.HTML(httpd.TextToHTML(mime.Text))
	htmlAvailable := mime.HTML != ""

	return httpd.RenderPartial("mailbox/_show.html", w, map[string]interface{}{
		"ctx":           ctx,
		"name":          name,
		"message":       msg,
		"body":          body,
		"htmlAvailable": htmlAvailable,
		"attachments":   mime.Attachments,
	})
}

// MailboxPurge deletes all messages from a mailbox.  Renders JSON or text/plain OK
func MailboxPurge(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
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

	if ctx.IsJSON {
		return httpd.RenderJSON(w, "OK")
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := io.WriteString(w, "OK"); err != nil {
		return err
	}
	return nil
}

// MailboxHTML displays the HTML content of a message. Renders a partial
func MailboxHTML(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
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
	mime, err := message.ReadBody()
	if err != nil {
		return fmt.Errorf("ReadBody(%q) failed: %v", id, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	return httpd.RenderPartial("mailbox/_html.html", w, map[string]interface{}{
		"ctx":     ctx,
		"name":    name,
		"message": message,
		// TODO: It is not really safe to render, need to sanitize.
		"body": template.HTML(mime.HTML),
	})
}

// MailboxSource displays the raw source of a message, including headers. Renders text/plain
func MailboxSource(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
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

// MailboxDownloadAttach sends the attachment to the client; disposition:
// attachment, type: application/octet-stream
func MailboxDownloadAttach(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	numStr := ctx.Vars["num"]
	num, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		ctx.Session.AddFlash("Attachment number must be unsigned numeric", "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
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
	body, err := message.ReadBody()
	if err != nil {
		return err
	}
	if int(num) >= len(body.Attachments) {
		ctx.Session.AddFlash("Attachment number too high", "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	part := body.Attachments[num]

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	if _, err := w.Write(part.Content()); err != nil {
		return err
	}
	return nil
}

// MailboxViewAttach sends the attachment to the client for online viewing
func MailboxViewAttach(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name, err := smtpd.ParseMailboxName(ctx.Vars["name"])
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	id := ctx.Vars["id"]
	numStr := ctx.Vars["num"]
	num, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		ctx.Session.AddFlash("Attachment number must be unsigned numeric", "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
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
	body, err := message.ReadBody()
	if err != nil {
		return err
	}
	if int(num) >= len(body.Attachments) {
		ctx.Session.AddFlash("Attachment number too high", "errors")
		http.Redirect(w, req, httpd.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	part := body.Attachments[num]

	w.Header().Set("Content-Type", part.ContentType())
	if _, err := w.Write(part.Content()); err != nil {
		return err
	}
	return nil
}

// MailboxDelete removes a particular message from a mailbox.  Renders JSON or plain/text OK
func MailboxDelete(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
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

	if ctx.IsJSON {
		return httpd.RenderJSON(w, "OK")
	}

	w.Header().Set("Content-Type", "text/plain")
	if _, err := io.WriteString(w, "OK"); err != nil {
		return err
	}
	return nil
}
