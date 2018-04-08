package webui

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/server/web"
)

// RootIndex serves the Inbucket landing page
func RootIndex(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	greeting, err := ioutil.ReadFile(ctx.RootConfig.Web.GreetingFile)
	if err != nil {
		return fmt.Errorf("Failed to load greeting: %v", err)
	}
	// Get flash messages, save session
	errorFlash := ctx.Session.Flashes("errors")
	if err = ctx.Session.Save(req, w); err != nil {
		return err
	}
	// Render template
	return web.RenderTemplate("root/index.html", w, map[string]interface{}{
		"ctx":        ctx,
		"errorFlash": errorFlash,
		"greeting":   template.HTML(string(greeting)),
	})
}

// RootMonitor serves the Inbucket monitor page
func RootMonitor(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	if !ctx.RootConfig.Web.MonitorVisible {
		ctx.Session.AddFlash("Monitor is disabled in configuration", "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	// Get flash messages, save session
	errorFlash := ctx.Session.Flashes("errors")
	if err = ctx.Session.Save(req, w); err != nil {
		return err
	}
	// Render template
	return web.RenderTemplate("root/monitor.html", w, map[string]interface{}{
		"ctx":        ctx,
		"errorFlash": errorFlash,
	})
}

// RootMonitorMailbox serves the Inbucket monitor page for a particular mailbox
func RootMonitorMailbox(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	if !ctx.RootConfig.Web.MonitorVisible {
		ctx.Session.AddFlash("Monitor is disabled in configuration", "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		ctx.Session.AddFlash(err.Error(), "errors")
		_ = ctx.Session.Save(req, w)
		http.Redirect(w, req, web.Reverse("RootIndex"), http.StatusSeeOther)
		return nil
	}
	// Get flash messages, save session
	errorFlash := ctx.Session.Flashes("errors")
	if err = ctx.Session.Save(req, w); err != nil {
		return err
	}
	// Render template
	return web.RenderTemplate("root/monitor.html", w, map[string]interface{}{
		"ctx":        ctx,
		"errorFlash": errorFlash,
		"name":       name,
	})
}

// RootStatus serves the Inbucket status page
func RootStatus(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Get flash messages, save session
	errorFlash := ctx.Session.Flashes("errors")
	if err = ctx.Session.Save(req, w); err != nil {
		return err
	}
	// Render template
	return web.RenderTemplate("root/status.html", w, map[string]interface{}{
		"ctx":           ctx,
		"errorFlash":    errorFlash,
		"version":       config.Version,
		"buildDate":     config.BuildDate,
		"smtpListener":  ctx.RootConfig.SMTP.Addr,
		"pop3Listener":  ctx.RootConfig.POP3.Addr,
		"webListener":   ctx.RootConfig.Web.Addr,
		"smtpConfig":    ctx.RootConfig.SMTP,
		"storageConfig": ctx.RootConfig.Storage,
	})
}
