package config

import (
	"container/list"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/robfig/config"
)

// SMTPConfig contains the SMTP server configuration - not using pointers
// so that we can pass around copies of the object safely.
type SMTPConfig struct {
	IP4address      net.IP
	IP4port         int
	Domain          string
	DomainNoStore   string
	MaxRecipients   int
	MaxIdleSeconds  int
	MaxMessageBytes int
	StoreMessages   bool
}

// POP3Config contains the POP3 server configuration
type POP3Config struct {
	IP4address     net.IP
	IP4port        int
	Domain         string
	MaxIdleSeconds int
}

// WebConfig contains the HTTP server configuration
type WebConfig struct {
	IP4address    net.IP
	IP4port       int
	TemplateDir   string
	TemplateCache bool
	PublicDir     string
	GreetingFile  string
	CookieAuthKey string
}

// DataStoreConfig contains the mail store configuration
type DataStoreConfig struct {
	Path             string
	RetentionMinutes int
	RetentionSleep   int
	MailboxMsgCap    int
}

var (
	// Version of this build, set by main
	Version = ""

	// BuildDate for this build, set by main
	BuildDate = ""

	// Config is our global robfig/config object
	Config *config.Config

	// Parsed specific configs
	smtpConfig      *SMTPConfig
	pop3Config      *POP3Config
	webConfig       *WebConfig
	dataStoreConfig *DataStoreConfig
)

// GetSMTPConfig returns a copy of the SmtpConfig object
func GetSMTPConfig() SMTPConfig {
	return *smtpConfig
}

// GetPOP3Config returns a copy of the Pop3Config object
func GetPOP3Config() POP3Config {
	return *pop3Config
}

// GetWebConfig returns a copy of the WebConfig object
func GetWebConfig() WebConfig {
	return *webConfig
}

// GetDataStoreConfig returns a copy of the DataStoreConfig object
func GetDataStoreConfig() DataStoreConfig {
	return *dataStoreConfig
}

// LoadConfig loads the specified configuration file into inbucket.Config
// and performs validations on it.
func LoadConfig(filename string) error {
	var err error
	Config, err = config.ReadDefault(filename)
	if err != nil {
		return err
	}

	messages := list.New()

	// Validate sections
	requireSection(messages, "logging")
	requireSection(messages, "smtp")
	requireSection(messages, "pop3")
	requireSection(messages, "web")
	requireSection(messages, "datastore")
	if messages.Len() > 0 {
		fmt.Fprintln(os.Stderr, "Error(s) validating configuration:")
		for e := messages.Front(); e != nil; e = e.Next() {
			fmt.Fprintln(os.Stderr, " -", e.Value.(string))
		}
		return fmt.Errorf("Failed to validate configuration")
	}

	// Validate options
	requireOption(messages, "logging", "level")
	requireOption(messages, "smtp", "ip4.address")
	requireOption(messages, "smtp", "ip4.port")
	requireOption(messages, "smtp", "domain")
	requireOption(messages, "smtp", "max.recipients")
	requireOption(messages, "smtp", "max.idle.seconds")
	requireOption(messages, "smtp", "max.message.bytes")
	requireOption(messages, "smtp", "store.messages")
	requireOption(messages, "pop3", "ip4.address")
	requireOption(messages, "pop3", "ip4.port")
	requireOption(messages, "pop3", "domain")
	requireOption(messages, "pop3", "max.idle.seconds")
	requireOption(messages, "web", "ip4.address")
	requireOption(messages, "web", "ip4.port")
	requireOption(messages, "web", "template.dir")
	requireOption(messages, "web", "template.cache")
	requireOption(messages, "web", "public.dir")
	requireOption(messages, "datastore", "path")
	requireOption(messages, "datastore", "retention.minutes")
	requireOption(messages, "datastore", "retention.sleep.millis")
	requireOption(messages, "datastore", "mailbox.message.cap")

	// Return error if validations failed
	if messages.Len() > 0 {
		fmt.Fprintln(os.Stderr, "Error(s) validating configuration:")
		for e := messages.Front(); e != nil; e = e.Next() {
			fmt.Fprintln(os.Stderr, " -", e.Value.(string))
		}
		return fmt.Errorf("Failed to validate configuration")
	}

	if err = parseSMTPConfig(); err != nil {
		return err
	}

	if err = parsePOP3Config(); err != nil {
		return err
	}

	if err = parseWebConfig(); err != nil {
		return err
	}

	if err = parseDataStoreConfig(); err != nil {
		return err
	}

	return nil
}

// parseLoggingConfig trying to catch config errors early
func parseLoggingConfig() error {
	section := "logging"

	option := "level"
	str, err := Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	switch strings.ToUpper(str) {
	case "TRACE", "INFO", "WARN", "ERROR":
	default:
		return fmt.Errorf("Invalid value provided for [%v]%v: '%v'", section, option, str)
	}
	return nil
}

// parseSMTPConfig trying to catch config errors early
func parseSMTPConfig() error {
	smtpConfig = new(SMTPConfig)
	section := "smtp"

	// Parse IP4 address only, error on IP6.
	option := "ip4.address"
	str, err := Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	addr := net.ParseIP(str)
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, str)
	}
	addr = addr.To4()
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v' not IPv4!", section, option, str)
	}
	smtpConfig.IP4address = addr

	option = "ip4.port"
	smtpConfig.IP4port, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	option = "domain"
	str, err = Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	smtpConfig.Domain = str

	option = "domain.nostore"
	if Config.HasOption(section, option) {
		str, err = Config.String(section, option)
		if err != nil {
			return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
		}
		smtpConfig.DomainNoStore = str
	}

	option = "max.recipients"
	smtpConfig.MaxRecipients, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	option = "max.idle.seconds"
	smtpConfig.MaxIdleSeconds, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	option = "max.message.bytes"
	smtpConfig.MaxMessageBytes, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	option = "store.messages"
	flag, err := Config.Bool(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	smtpConfig.StoreMessages = flag

	return nil
}

// parsePOP3Config trying to catch config errors early
func parsePOP3Config() error {
	pop3Config = new(POP3Config)
	section := "pop3"

	// Parse IP4 address only, error on IP6.
	option := "ip4.address"
	str, err := Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	addr := net.ParseIP(str)
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, str)
	}
	addr = addr.To4()
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v' not IPv4!", section, option, str)
	}
	pop3Config.IP4address = addr

	option = "ip4.port"
	pop3Config.IP4port, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	option = "domain"
	str, err = Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	pop3Config.Domain = str

	option = "max.idle.seconds"
	pop3Config.MaxIdleSeconds, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	return nil
}

// parseWebConfig trying to catch config errors early
func parseWebConfig() error {
	webConfig = new(WebConfig)
	section := "web"

	// Parse IP4 address only, error on IP6.
	option := "ip4.address"
	str, err := Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	addr := net.ParseIP(str)
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, str)
	}
	addr = addr.To4()
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v' not IPv4!", section, option, str)
	}
	webConfig.IP4address = addr

	option = "ip4.port"
	webConfig.IP4port, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	option = "template.dir"
	str, err = Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	webConfig.TemplateDir = str

	option = "template.cache"
	flag, err := Config.Bool(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	webConfig.TemplateCache = flag

	option = "public.dir"
	str, err = Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	webConfig.PublicDir = str

	option = "greeting.file"
	str, err = Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	webConfig.GreetingFile = str

	option = "cookie.auth.key"
	if Config.HasOption(section, option) {
		str, err = Config.String(section, option)
		if err != nil {
			return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
		}
		webConfig.CookieAuthKey = str
	}

	return nil
}

// parseDataStoreConfig trying to catch config errors early
func parseDataStoreConfig() error {
	dataStoreConfig = new(DataStoreConfig)
	section := "datastore"

	option := "path"
	str, err := Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	dataStoreConfig.Path = str

	option = "retention.minutes"
	dataStoreConfig.RetentionMinutes, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	option = "retention.sleep.millis"
	dataStoreConfig.RetentionSleep, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	option = "mailbox.message.cap"
	dataStoreConfig.MailboxMsgCap, err = Config.Int(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}

	return nil
}

// requireSection checks that a [section] is defined in the configuration file,
// appending a message if not.
func requireSection(messages *list.List, section string) {
	if !Config.HasSection(section) {
		messages.PushBack(fmt.Sprintf("Config section [%v] is required", section))
	}
}

// requireOption checks that 'option' is defined in [section] of the config file,
// appending a message if not.
func requireOption(messages *list.List, section string, option string) {
	if !Config.HasOption(section, option) {
		messages.PushBack(fmt.Sprintf("Config option '%v' is required in section [%v]", option, section))
	}
}
