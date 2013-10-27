package web

import (
	"fmt"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/smtpd"
	"html/template"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"time"
)

type JsonMessageHeader struct {
	Mailbox, Id, From, Subject string
	Date                       time.Time
	Size                       int64
}

type JsonMessage struct {
	Mailbox, Id, From, Subject string
	Date                       time.Time
	Size                       int64
	Body                       *JsonMessageBody
	Header                     mail.Header
}

type JsonMessageBody struct {
	Text, Html string
}

func MailboxIndex(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Form values must be validated manually
	name := req.FormValue("name")
	selected := req.FormValue("id")

	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	name = smtpd.ParseMailboxName(name)

	return RenderTemplate("mailbox/index.html", w, map[string]interface{}{
		"ctx":  ctx,
		"name": name,
		"selected": selected,
	})
}

func MailboxLink(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])
	id := ctx.Vars["id"]

	uri := fmt.Sprintf("%s?name=%s&id=%s", reverse("MailboxIndex"), name, id)
	http.Redirect(w, req, uri, http.StatusSeeOther)
	return nil
}

func MailboxList(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return fmt.Errorf("Failed to get mailbox for %v: %v", name, err)
	}
	messages, err := mb.GetMessages()
	if err != nil {
		return fmt.Errorf("Failed to get messages for %v: %v", name, err)
	}
	log.LogTrace("Got %v messsages", len(messages))

	if ctx.IsJson {
		jmessages := make([]*JsonMessageHeader, len(messages))
		for i, msg := range messages {
			jmessages[i] = &JsonMessageHeader{
				Mailbox: name,
				Id:      msg.Id(),
				From:    msg.From(),
				Subject: msg.Subject(),
				Date:    msg.Date(),
				Size:    msg.Size(),
			}
		}
		return RenderJson(w, jmessages)
	}

	return RenderPartial("mailbox/_list.html", w, map[string]interface{}{
		"ctx":      ctx,
		"name":     name,
		"messages": messages,
	})
}

func MailboxShow(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])
	id := ctx.Vars["id"]

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return fmt.Errorf("MailboxFor('%v'): %v", name, err)
	}
	msg, err := mb.GetMessage(id)
	if err != nil {
		return fmt.Errorf("GetMessage() failed: %v", err)
	}
	header, err := msg.ReadHeader()
	if err != nil {
		return fmt.Errorf("ReadHeader() failed: %v", err)
	}
	mime, err := msg.ReadBody()
	if err != nil {
		return fmt.Errorf("ReadBody() failed: %v", err)
	}

	if ctx.IsJson {
		return RenderJson(w,
			&JsonMessage{
				Mailbox: name,
				Id:      msg.Id(),
				From:    msg.From(),
				Subject: msg.Subject(),
				Date:    msg.Date(),
				Size:    msg.Size(),
				Header:  header.Header,
				Body: &JsonMessageBody{
					Text: mime.Text,
					Html: mime.Html,
				},
			})
	}

	body := template.HTML(textToHtml(mime.Text))
	htmlAvailable := mime.Html != ""

	return RenderPartial("mailbox/_show.html", w, map[string]interface{}{
		"ctx":           ctx,
		"name":          name,
		"message":       msg,
		"body":          body,
		"htmlAvailable": htmlAvailable,
		"attachments":   mime.Attachments,
	})
}

func MailboxPurge(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return fmt.Errorf("MailboxFor('%v'): %v", name, err)
	}
	if err := mb.Purge(); err != nil {
		return fmt.Errorf("Mailbox(%q) Purge: %v", name, err)
	}
	log.LogTrace("Purged mailbox for %q", name)

	if ctx.IsJson {
		return RenderJson(w, "OK")
	}

	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "OK")
	return nil
}

func MailboxHtml(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])
	id := ctx.Vars["id"]

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return err
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return err
	}
	mime, err := message.ReadBody()
	if err != nil {
		return err
	}

	return RenderPartial("mailbox/_html.html", w, map[string]interface{}{
		"ctx":     ctx,
		"name":    name,
		"message": message,
		// TODO: It is not really safe to render, need to sanitize.
		"body": template.HTML(mime.Html),
	})
}

func MailboxSource(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])
	id := ctx.Vars["id"]

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return err
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return err
	}
	raw, err := message.ReadRaw()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, *raw)
	return nil
}

func MailboxDownloadAttach(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])
	id := ctx.Vars["id"]
	numStr := ctx.Vars["num"]
	num, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		ctx.Session.AddFlash("Attachment number must be unsigned numeric", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return err
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return err
	}
	body, err := message.ReadBody()
	if err != nil {
		return err
	}
	if int(num) >= len(body.Attachments) {
		ctx.Session.AddFlash("Attachment number too high", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	part := body.Attachments[num]

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	w.Write(part.Content())
	return nil
}

func MailboxViewAttach(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])
	id := ctx.Vars["id"]
	numStr := ctx.Vars["num"]
	num, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		ctx.Session.AddFlash("Attachment number must be unsigned numeric", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return err
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return err
	}
	body, err := message.ReadBody()
	if err != nil {
		return err
	}
	if int(num) >= len(body.Attachments) {
		ctx.Session.AddFlash("Attachment number too high", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	part := body.Attachments[num]

	w.Header().Set("Content-Type", part.ContentType())
	w.Write(part.Content())
	return nil
}

func MailboxDelete(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name := smtpd.ParseMailboxName(ctx.Vars["name"])
	id := ctx.Vars["id"]

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return err
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return err
	}
	err = message.Delete()
	if err != nil {
		return err
	}

	if ctx.IsJson {
		return RenderJson(w, "OK")
	}

	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "OK")
	return nil
}
