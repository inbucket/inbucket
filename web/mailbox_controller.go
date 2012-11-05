package web

import (
	"fmt"
	"github.com/jhillyerd/inbucket/log"
	"html/template"
	"io"
	"net/http"
)

func MailboxIndex(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	name := req.FormValue("name")
	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	return RenderTemplate("mailbox/index.html", w, map[string]interface{}{
		"ctx":  ctx,
		"name": name,
	})
}

func MailboxList(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	name := ctx.Vars["name"]
	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return fmt.Errorf("Failed to get mailbox for %v: %v", name, err)
	}
	messages, err := mb.GetMessages()
	if err != nil {
		return fmt.Errorf("Failed to get messages for %v: %v", name, err)
	}
	log.Trace("Got %v messsages", len(messages))

	return RenderPartial("mailbox/_list.html", w, map[string]interface{}{
		"ctx":      ctx,
		"name":     name,
		"messages": messages,
	})
}

func MailboxShow(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	name := ctx.Vars["name"]
	id := ctx.Vars["id"]
	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	if len(id) == 0 {
		ctx.Session.AddFlash("Message ID is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}

	mb, err := ctx.DataStore.MailboxFor(name)
	if err != nil {
		return fmt.Errorf("MailboxFor('%v'): %v", name, err)
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return fmt.Errorf("GetMessage() failed: %v", err)
	}
	_, mime, err := message.ReadBody()
	if err != nil {
		return fmt.Errorf("ReadBody() failed: %v", err)
	}
	body := template.HTML(textToHtml(mime.Text))
	htmlAvailable := mime.Html != ""

	return RenderPartial("mailbox/_show.html", w, map[string]interface{}{
		"ctx":           ctx,
		"name":          name,
		"message":       message,
		"body":          body,
		"htmlAvailable": htmlAvailable,
	})
}

func MailboxHtml(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	name := ctx.Vars["name"]
	id := ctx.Vars["id"]
	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	if len(id) == 0 {
		ctx.Session.AddFlash("Message ID is required", "errors")
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
	_, mime, err := message.ReadBody()
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
	name := ctx.Vars["name"]
	id := ctx.Vars["id"]
	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	if len(id) == 0 {
		ctx.Session.AddFlash("Message ID is required", "errors")
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
	raw, err := message.ReadRaw()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, *raw)
	return nil
}

func MailboxDelete(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	name := ctx.Vars["name"]
	id := ctx.Vars["id"]
	if len(name) == 0 {
		ctx.Session.AddFlash("Account name is required", "errors")
		http.Redirect(w, req, reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	if len(id) == 0 {
		ctx.Session.AddFlash("Message ID is required", "errors")
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
	err = message.Delete()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "OK")
	return nil
}
