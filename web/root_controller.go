package web

import (
	"fmt"
	"github.com/jhillyerd/inbucket/config"
	"net/http"
)

func RootIndex(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	return RenderTemplate("root/index.html", w, map[string]interface{}{
		"ctx": ctx,
	})
}

func RootStatus(w http.ResponseWriter, req *http.Request, ctx *Context) (err error) {
	retentionMinutes := config.GetDataStoreConfig().RetentionMinutes
	smtpListener := fmt.Sprintf("%s:%d", config.GetSmtpConfig().Ip4address.String(),
		config.GetSmtpConfig().Ip4port)
	pop3Listener := fmt.Sprintf("%s:%d", config.GetPop3Config().Ip4address.String(),
		config.GetPop3Config().Ip4port)
	webListener := fmt.Sprintf("%s:%d", config.GetWebConfig().Ip4address.String(),
		config.GetWebConfig().Ip4port)
	return RenderTemplate("root/status.html", w, map[string]interface{}{
		"ctx":              ctx,
		"retentionMinutes": retentionMinutes,
		"smtpListener":     smtpListener,
		"pop3Listener":     pop3Listener,
		"webListener":      webListener,
	})
}
