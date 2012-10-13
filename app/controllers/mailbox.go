package controllers

import (
	"github.com/jhillyerd/inbucket/app/inbucket"
	"github.com/robfig/revel"
)

type Mailbox struct {
	*rev.Controller
}

func (c Mailbox) Index(name string) rev.Result {
	return c.Render(name)
}

func (c Mailbox) List(name string) rev.Result {
	ds := inbucket.NewDataStore()
	mb, err := ds.MailboxFor(name)
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}
	messages, err := mb.GetMessages()
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}
	rev.INFO.Printf("Got %v messsages", len(messages))

	return c.Render(name, messages)
}

func (c Mailbox) Show(name string, id string) rev.Result {
	ds := inbucket.NewDataStore()
	mb, err := ds.MailboxFor(name)
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}
	_, body, err := message.ReadBody()
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}

	return c.Render(name, message, body)
}

func (c Mailbox) Delete(name string, id string) rev.Result {
	ds := inbucket.NewDataStore()
	mb, err := ds.MailboxFor(name)
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}
	message, err := mb.GetMessage(id)
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}
	err = message.Delete()
	if err != nil {
		rev.ERROR.Printf(err.Error())
		c.Flash.Error(err.Error())
		return c.Redirect(Application.Index)
	}
	return c.RenderText("OK")
}
