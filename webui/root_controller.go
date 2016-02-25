package webui

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/httpd"
)

// RootIndex serves the Inbucket landing page
func RootIndex(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	greeting, err := ioutil.ReadFile(config.GetWebConfig().GreetingFile)
	if err != nil {
		return fmt.Errorf("Failed to load greeting: %v", err)
	}

	return httpd.RenderTemplate("root/index.html", w, map[string]interface{}{
		"ctx":      ctx,
		"greeting": template.HTML(string(greeting)),
	})
}

// RootStatus serves the Inbucket status page
func RootStatus(w http.ResponseWriter, req *http.Request, ctx *httpd.Context) (err error) {
	retentionMinutes := config.GetDataStoreConfig().RetentionMinutes
	smtpListener := fmt.Sprintf("%s:%d", config.GetSMTPConfig().IP4address.String(),
		config.GetSMTPConfig().IP4port)
	pop3Listener := fmt.Sprintf("%s:%d", config.GetPOP3Config().IP4address.String(),
		config.GetPOP3Config().IP4port)
	webListener := fmt.Sprintf("%s:%d", config.GetWebConfig().IP4address.String(),
		config.GetWebConfig().IP4port)
	return httpd.RenderTemplate("root/status.html", w, map[string]interface{}{
		"ctx":              ctx,
		"version":          config.Version,
		"buildDate":        config.BuildDate,
		"retentionMinutes": retentionMinutes,
		"smtpListener":     smtpListener,
		"pop3Listener":     pop3Listener,
		"webListener":      webListener,
	})
}
