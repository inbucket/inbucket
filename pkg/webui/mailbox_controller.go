package webui

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/jhillyerd/inbucket/pkg/server/web"
	"github.com/jhillyerd/inbucket/pkg/storage"
	"github.com/jhillyerd/inbucket/pkg/webui/sanitize"
	"github.com/rs/zerolog/log"
)

// MailboxIndex renders the index page for a particular mailbox
func MailboxIndex(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Form values must be validated manually
	name := req.FormValue("name")
	selected := req.FormValue("id")
	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	name, err = ctx.Manager.MailboxForAddress(name)
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	// Remember this mailbox was visited
	RememberMailbox(ctx, name)
	// Get flash messages, save session
	errorFlash := ctx.Session.Flashes("errors")
	if err = ctx.Session.Save(req, w); err != nil {
		return err
	}
	// Render template
	return web.RenderTemplate("mailbox/index.html", w, map[string]interface{}{
		"ctx":        ctx,
		"errorFlash": errorFlash,
		"name":       name,
		"selected":   selected,
	})
}

// MailboxIndexFriendly handles pretty links to a particular mailbox. Renders a redirect
func MailboxIndexFriendly(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	// Build redirect
	uri := fmt.Sprintf("%s?name=%s", web.Reverse("MailboxIndex"), name)
	http.Redirect(w, req, uri, http.StatusSeeOther)
	return nil
}

// MailboxLink handles pretty links to a particular message. Renders a redirect
func MailboxLink(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	// Build redirect
	uri := fmt.Sprintf("%s?name=%s&id=%s", web.Reverse("MailboxIndex"), name, id)
	http.Redirect(w, req, uri, http.StatusSeeOther)
	return nil
}

// MailboxList renders a list of messages in a mailbox. Renders a partial
func MailboxList(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	messages, err := ctx.Manager.GetMetadata(name)
	if err != nil {
		// This doesn't indicate empty, likely an IO error
		return fmt.Errorf("Failed to get messages for %v: %v", name, err)
	}
	// Render partial template
	return web.RenderPartial("mailbox/_list.html", w, map[string]interface{}{
		"ctx":      ctx,
		"name":     name,
		"messages": messages,
	})
}

// MailboxShow renders a particular message from a mailbox. Renders an HTML partial
func MailboxShow(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
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
	body := template.HTML(web.TextToHTML(msg.Text()))
	htmlAvailable := msg.HTML() != ""
	var htmlBody template.HTML
	if htmlAvailable {
		if str, err := sanitize.HTML(msg.HTML()); err == nil {
			htmlBody = template.HTML(str)
		} else {
			// Soft failure, render empty tab.
			log.Warn().Str("module", "webui").Str("mailbox", name).Str("id", id).Err(err).
				Msg("HTML sanitizer failed")
		}
	}
	// Render partial template
	return web.RenderPartial("mailbox/_show.html", w, map[string]interface{}{
		"ctx":           ctx,
		"name":          name,
		"message":       msg,
		"body":          body,
		"htmlAvailable": htmlAvailable,
		"htmlBody":      htmlBody,
		"mimeErrors":    msg.MIMEErrors(),
		"attachments":   msg.Attachments(),
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

// MailboxDownloadAttach sends the attachment to the client; disposition:
// attachment, type: application/octet-stream
func MailboxDownloadAttach(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Don't have to validate these aren't empty, Gorilla returns 404
	id := ctx.Vars["id"]
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
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
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")
	_, err = w.Write(msg.Attachments()[num].Content)
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
