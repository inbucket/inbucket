package config

import (
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	prefix      = "inbucket"
	tableFormat = `Inbucket is configured via the environment. The following environment
variables can be used:

KEY	DEFAULT	REQUIRED	DESCRIPTION
{{range .}}{{usage_key .}}	{{usage_default .}}	{{usage_required .}}	{{usage_description .}}
{{end}}`
)

var (
	// Version of this build, set by main
	Version = ""

	// BuildDate for this build, set by main
	BuildDate = ""
)

// Root wraps all other configurations.
type Root struct {
	LogLevel string `required:"true" default:"INFO" desc:"TRACE, INFO, WARN, or ERROR"`
	SMTP     SMTP
	POP3     POP3
	Web      Web
	Storage  Storage
}

// SMTP contains the SMTP server configuration.
type SMTP struct {
	Addr            string        `required:"true" default:"0.0.0.0:2500" desc:"SMTP server IP4 host:port"`
	Domain          string        `required:"true" default:"inbucket" desc:"HELO domain"`
	DomainNoStore   string        `desc:"Load testing domain"`
	MaxRecipients   int           `required:"true" default:"200" desc:"Maximum RCPT TO per message"`
	MaxIdle         time.Duration `required:"true" default:"300s" desc:"Idle network timeout"`
	MaxMessageBytes int           `required:"true" default:"2048000" desc:"Maximum message size"`
	StoreMessages   bool          `required:"true" default:"true" desc:"Store incoming mail?"`
}

// POP3 contains the POP3 server configuration.
type POP3 struct {
	Addr    string        `required:"true" default:"0.0.0.0:1100" desc:"POP3 server IP4 host:port"`
	Domain  string        `required:"true" default:"inbucket" desc:"HELLO domain"`
	MaxIdle time.Duration `required:"true" default:"600s" desc:"Idle network timeout"`
}

// Web contains the HTTP server configuration.
type Web struct {
	Addr           string `required:"true" default:"0.0.0.0:9000" desc:"Web server IP4 host:port"`
	TemplateDir    string `required:"true" default:"themes/bootstrap/templates" desc:"Theme template dir"`
	TemplateCache  bool   `required:"true" default:"true" desc:"Cache templates after first use?"`
	PublicDir      string `required:"true" default:"themes/bootstrap/public" desc:"Theme public dir"`
	GreetingFile   string `required:"true" default:"themes/greeting.html" desc:"Home page greeting HTML"`
	MailboxPrompt  string `required:"true" default:"@inbucket" desc:"Prompt next to mailbox input"`
	CookieAuthKey  string `desc:"Session cipher key (text)"`
	MonitorVisible bool   `required:"true" default:"true" desc:"Show monitor tab in UI?"`
	MonitorHistory int    `required:"true" default:"30" desc:"Monitor remembered messages"`
}

// Storage contains the mail store configuration.
type Storage struct {
	Path            string        `required:"true" default:"/tmp/inbucket" desc:"Mail store path"`
	RetentionPeriod time.Duration `required:"true" default:"24h" desc:"Duration to retain messages"`
	RetentionSleep  time.Duration `required:"true" default:"100ms" desc:"Duration to sleep between deletes"`
	MailboxMsgCap   int           `required:"true" default:"500" desc:"Maximum messages per mailbox"`
}

// Process loads and parses configuration from the environment.
func Process() (*Root, error) {
	c := &Root{}
	err := envconfig.Process(prefix, c)
	return c, err
}

// Usage prints out the envconfig usage to Stderr.
func Usage() {
	tabs := tabwriter.NewWriter(os.Stderr, 1, 0, 4, ' ', 0)
	if err := envconfig.Usagef(prefix, &Root{}, tabs, tableFormat); err != nil {
		log.Fatalf("Unable to parse env config: %v", err)
	}
	tabs.Flush()
}
