package controllers

import (
	"github.com/jhillyerd/inbucket/app/smtpd"
	"github.com/robfig/revel"
)

type Application struct {
	*rev.Controller
}

func (c Application) Index() rev.Result {
	return c.Render()
}

type SmtpdPlugin struct {
	rev.EmptyPlugin
	server *smtpd.Server
}

func (p SmtpdPlugin) OnAppStart() {
	domain := rev.Config.StringDefault("smtpd.domain", "localhost")
	port := rev.Config.IntDefault("smtpd.port", 2500)
	rev.INFO.Printf("SMTP Daemon plugin init {domain: %v, port: %v}", domain, port)
	p.server = smtpd.New(domain, port)
	go p.server.Start()
}

func init() {
	rev.RegisterPlugin(SmtpdPlugin{})
}
