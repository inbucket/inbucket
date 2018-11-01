package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jhillyerd/inbucket/pkg/stringutil"
	"github.com/kelseyhightower/envconfig"
)

const (
	prefix      = "inbucket"
	tableFormat = `Inbucket is configured via the environment. The following environment variables
can be used:

KEY	DEFAULT	DESCRIPTION
{{range .}}{{usage_key .}}	{{usage_default .}}	{{usage_description .}}
{{end}}`
)

var (
	// Version of this build, set by main
	Version = ""

	// BuildDate for this build, set by main
	BuildDate = ""
)

// mbNaming represents a mailbox naming strategy.
type mbNaming int

// Mailbox naming strategies.
const (
	UnknownNaming mbNaming = iota
	LocalNaming
	FullNaming
)

// Decode a naming strategy from string.
func (n *mbNaming) Decode(v string) error {
	switch strings.ToLower(v) {
	case "local":
		*n = LocalNaming
	case "full":
		*n = FullNaming
	default:
		return fmt.Errorf("Unknown MailboxNaming strategy: %q", v)
	}
	return nil
}

// Root contains global configuration, and structs with for specific sub-systems.
type Root struct {
	LogLevel      string   `required:"true" default:"info" desc:"debug, info, warn, or error"`
	MailboxNaming mbNaming `required:"true" default:"local" desc:"Use local or full addressing"`
	SMTP          SMTP
	POP3          POP3
	Web           Web
	Storage       Storage
}

// SMTP contains the SMTP server configuration.
type SMTP struct {
	Addr            string        `required:"true" default:"0.0.0.0:2500" desc:"SMTP server IP4 host:port"`
	Domain          string        `required:"true" default:"inbucket" desc:"HELO domain"`
	MaxRecipients   int           `required:"true" default:"200" desc:"Maximum RCPT TO per message"`
	MaxMessageBytes int           `required:"true" default:"10240000" desc:"Maximum message size"`
	DefaultAccept   bool          `required:"true" default:"true" desc:"Accept all mail by default?"`
	AcceptDomains   []string      `desc:"Domains to accept mail for"`
	RejectDomains   []string      `desc:"Domains to reject mail for"`
	DefaultStore    bool          `required:"true" default:"true" desc:"Store all mail by default?"`
	StoreDomains    []string      `desc:"Domains to store mail for"`
	DiscardDomains  []string      `desc:"Domains to discard mail for"`
	Timeout         time.Duration `required:"true" default:"300s" desc:"Idle network timeout"`
	TLSEnabled      bool          `default:"false" desc:"Enable STARTTLS option"`
	TLSPrivKey      string        `default:"cert.key" desc:"X509 Private Key file for TLS Support"`
	TLSCert         string        `default:"cert.crt" desc:"X509 Public Certificate file for TLS Support"`
	Debug           bool          `ignored:"true"`
}

// POP3 contains the POP3 server configuration.
type POP3 struct {
	Addr    string        `required:"true" default:"0.0.0.0:1100" desc:"POP3 server IP4 host:port"`
	Domain  string        `required:"true" default:"inbucket" desc:"HELLO domain"`
	Timeout time.Duration `required:"true" default:"600s" desc:"Idle network timeout"`
	Debug   bool          `ignored:"true"`
}

// Web contains the HTTP server configuration.
type Web struct {
	Addr           string `required:"true" default:"0.0.0.0:9000" desc:"Web server IP4 host:port"`
	UIDir          string `required:"true" default:"ui" desc:"User interface dir"`
	GreetingFile   string `required:"true" default:"ui/greeting.html" desc:"Home page greeting HTML"`
	TemplateCache  bool   `required:"true" default:"true" desc:"Cache templates after first use?"`
	MailboxPrompt  string `required:"true" default:"@inbucket" desc:"Prompt next to mailbox input"`
	CookieAuthKey  string `desc:"Session cipher key (text)"`
	MonitorVisible bool   `required:"true" default:"true" desc:"Show monitor tab in UI?"`
	MonitorHistory int    `required:"true" default:"30" desc:"Monitor remembered messages"`
	PProf          bool   `required:"true" default:"false" desc:"Expose profiling tools on /debug/pprof"`
}

// Storage contains the mail store configuration.
type Storage struct {
	Type            string            `required:"true" default:"memory" desc:"Storage impl: file or memory"`
	Params          map[string]string `desc:"Storage impl parameters, see docs."`
	RetentionPeriod time.Duration     `required:"true" default:"24h" desc:"Duration to retain messages"`
	RetentionSleep  time.Duration     `required:"true" default:"50ms" desc:"Duration to sleep between mailboxes"`
	MailboxMsgCap   int               `required:"true" default:"500" desc:"Maximum messages per mailbox"`
}

// Process loads and parses configuration from the environment.
func Process() (*Root, error) {
	c := &Root{}
	err := envconfig.Process(prefix, c)
	c.LogLevel = strings.ToLower(c.LogLevel)
	stringutil.SliceToLower(c.SMTP.AcceptDomains)
	stringutil.SliceToLower(c.SMTP.RejectDomains)
	stringutil.SliceToLower(c.SMTP.StoreDomains)
	stringutil.SliceToLower(c.SMTP.DiscardDomains)
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
