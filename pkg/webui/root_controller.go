package webui

import (
	"fmt"
	"net/http"
	"os"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
)

// RootGreeting serves the Inbucket greeting.
func RootGreeting(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	greeting, err := os.ReadFile(ctx.RootConfig.Web.GreetingFile)
	if err != nil {
		return fmt.Errorf("failed to load greeting: %v", err)
	}

	w.Header().Set("Content-Type", "text/html")
	_, err = w.Write(greeting)
	return err
}

// RootStatus renders portions of the server configuration as JSON.
func RootStatus(w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	root := ctx.RootConfig
	retPeriod := ""
	if root.Storage.RetentionPeriod > 0 {
		retPeriod = root.Storage.RetentionPeriod.String()
	}

	return web.RenderJSON(w,
		&jsonServerConfig{
			Version:      config.Version,
			BuildDate:    config.BuildDate,
			POP3Listener: root.POP3.Addr,
			WebListener:  root.Web.Addr,
			SMTPConfig: jsonSMTPConfig{
				Addr:                root.SMTP.Addr,
				DefaultAccept:       root.SMTP.DefaultAccept,
				AcceptDomains:       root.SMTP.AcceptDomains,
				RejectDomains:       root.SMTP.RejectDomains,
				DefaultStore:        root.SMTP.DefaultStore,
				StoreDomains:        root.SMTP.StoreDomains,
				DiscardDomains:      root.SMTP.DiscardDomains,
				RejectOriginDomains: root.SMTP.RejectOriginDomains,
			},
			StorageConfig: jsonStorageConfig{
				MailboxMsgCap:   root.Storage.MailboxMsgCap,
				StoreType:       root.Storage.Type,
				RetentionPeriod: retPeriod,
			},
		})
}
