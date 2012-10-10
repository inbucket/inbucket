package controllers

import (
	"fmt"
	"github.com/jhillyerd/inbucket/app/inbucket"
	"github.com/robfig/revel"
)

type Mailbox struct {
	*rev.Controller
}

func (c Mailbox) Index(name string) rev.Result {
	return c.Redirect("/mailbox/list/%v", name)
}

func (c Mailbox) List(name string) rev.Result {
	title := fmt.Sprintf("Mailbox for %v", name)

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

	return c.Render(title, name, messages)
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
