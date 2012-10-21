package web

import (
	"github.com/jhillyerd/inbucket"
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
		return err
	}
	messages, err := mb.GetMessages()
	if err != nil {
		return err
	}
	inbucket.Trace("Got %v messsages", len(messages))

	return RenderPartial("mailbox/_list.html", w, map[string]interface{}{
		"ctx":      ctx,
		"name":     name,
		"messages": messages,
	})
}

/*
func (c Mailbox) Show(name string, id string) rev.Result {
func MailboxShow(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	c.Validation.Required(name).Message("Account name is required")
	c.Validation.Required(id).Message("Message ID is required")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Application.Index)
	}

	ds := inbucket.NewDataStore()
	mb, err := ds.MailboxFor(name)
	if err != nil {
		return c.RenderError(err)
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return c.RenderError(err)
	}
	_, mime, err := message.ReadBody()
	if err != nil {
		return c.RenderError(err)
	}
	body := template.HTML(inbucket.TextToHtml(mime.Text))
	htmlAvailable := mime.Html != ""

	c.Response.Out.Header().Set("Expires", "-1")
	return c.Render(name, message, body, htmlAvailable)
}

func (c Mailbox) Delete(name string, id string) rev.Result {
func MailboxDelete(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	c.Validation.Required(name).Message("Account name is required")
	c.Validation.Required(id).Message("Message ID is required")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Application.Index)
	}

	ds := inbucket.NewDataStore()
	mb, err := ds.MailboxFor(name)
	if err != nil {
		return c.RenderError(err)
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return c.RenderError(err)
	}
	err = message.Delete()
	if err != nil {
		return c.RenderError(err)
	}
	c.Response.Out.Header().Set("Expires", "-1")
	return c.RenderText("OK")
}

func (c Mailbox) Html(name string, id string) rev.Result {
func MailboxHtml(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	c.Validation.Required(name).Message("Account name is required")
	c.Validation.Required(id).Message("Message ID is required")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Application.Index)
	}

	ds := inbucket.NewDataStore()
	mb, err := ds.MailboxFor(name)
	if err != nil {
		return c.RenderError(err)
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return c.RenderError(err)
	}
	_, mime, err := message.ReadBody()
	if err != nil {
		return c.RenderError(err)
	}
	// Mark as safe to render HTML
	// TODO: It is not really safe to render, need to sanitize.
	body := template.HTML(mime.Html)

	c.Response.Out.Header().Set("Expires", "-1")
	return c.Render(name, message, body)
}

func (c Mailbox) Source(name string, id string) rev.Result {
func MailboxSource(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	c.Validation.Required(name).Message("Account name is required")
	c.Validation.Required(id).Message("Message ID is required")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Application.Index)
	}

	ds := inbucket.NewDataStore()
	mb, err := ds.MailboxFor(name)
	if err != nil {
		return c.RenderError(err)
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		return c.RenderError(err)
	}
	raw, err := message.ReadRaw()
	if err != nil {
		return c.RenderError(err)
	}

	c.Response.Out.Header().Set("Expires", "-1")
	return c.RenderText(*raw)
}
*/
