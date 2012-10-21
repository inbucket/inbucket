package web

import (
	"github.com/jhillyerd/inbucket/app/inbucket"
	"github.com/robfig/revel"
	"html/template"
)

type Mailbox struct {
	*rev.Controller
}

func (c Mailbox) Index(name string) rev.Result {
	c.Validation.Required(name).Message("Account name is required")

	if c.Validation.HasErrors() {
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(Application.Index)
	}

	return c.Render(name)
}

func (c Mailbox) List(name string) rev.Result {
	c.Validation.Required(name).Message("Account name is required")

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
	messages, err := mb.GetMessages()
	if err != nil {
		return c.RenderError(err)
	}
	rev.INFO.Printf("Got %v messsages", len(messages))

	c.Response.Out.Header().Set("Expires", "-1")
	return c.Render(name, messages)
}

func (c Mailbox) Show(name string, id string) rev.Result {
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
