package config

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/robfig/config"
)

// SMTPConfig contains the SMTP server configuration - not using pointers so that we can pass around
// copies of the object safely.
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
	IP4address     net.IP
	IP4port        int
	TemplateDir    string
	TemplateCache  bool
	PublicDir      string
	GreetingFile   string
	CookieAuthKey  string
	MonitorVisible bool
	MonitorHistory int
}

// DataStoreConfig contains the mail store configuration
type DataStoreConfig struct {
	Path             string
	RetentionMinutes int
	RetentionSleep   int
	MailboxMsgCap    int
}

const (
	missingErrorFmt = "[%v] missing required option %q"
	parseErrorFmt   = "[%v] option %q error: %v"
)

var (
	// Version of this build, set by main
	Version = ""

	// BuildDate for this build, set by main
	BuildDate = ""

	// Config is our global robfig/config object
	Config   *config.Config
	logLevel string

	// Parsed specific configs
	smtpConfig      = &SMTPConfig{}
	pop3Config      = &POP3Config{}
	webConfig       = &WebConfig{}
	dataStoreConfig = &DataStoreConfig{}
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

// GetLogLevel returns the configured log level
func GetLogLevel() string {
	return logLevel
}

// LoadConfig loads the specified configuration file into inbucket.Config and performs validations
// on it.
func LoadConfig(filename string) error {
	var err error
	Config, err = config.ReadDefault(filename)
	if err != nil {
		return err
	}
	// Validation error messages
	messages := make([]string, 0)
	// Validate sections
	for _, s := range []string{"logging", "smtp", "pop3", "web", "datastore"} {
		if !Config.HasSection(s) {
			messages = append(messages,
				fmt.Sprintf("Config section [%v] is required", s))
		}
	}
	// Return immediately if config is missing entire sections
	if len(messages) > 0 {
		fmt.Fprintln(os.Stderr, "Error(s) validating configuration:")
		for _, m := range messages {
			fmt.Fprintln(os.Stderr, " -", m)
		}
		return fmt.Errorf("Failed to validate configuration")
	}
	// Load string config options
	stringOptions := []struct {
		section  string
		name     string
		target   *string
		required bool
	}{
		{"logging", "level", &logLevel, true},
		{"smtp", "domain", &smtpConfig.Domain, true},
		{"smtp", "domain.nostore", &smtpConfig.DomainNoStore, false},
		{"pop3", "domain", &pop3Config.Domain, true},
		{"web", "template.dir", &webConfig.TemplateDir, true},
		{"web", "public.dir", &webConfig.PublicDir, true},
		{"web", "greeting.file", &webConfig.GreetingFile, true},
		{"web", "cookie.auth.key", &webConfig.CookieAuthKey, false},
		{"datastore", "path", &dataStoreConfig.Path, true},
	}
	for _, opt := range stringOptions {
		str, err := Config.String(opt.section, opt.name)
		if Config.HasOption(opt.section, opt.name) && err != nil {
			messages = append(messages, fmt.Sprintf(parseErrorFmt, opt.section, opt.name, err))
			continue
		}
		if str == "" && opt.required {
			messages = append(messages, fmt.Sprintf(missingErrorFmt, opt.section, opt.name))
		}
		*opt.target = str
	}
	// Load boolean config options
	boolOptions := []struct {
		section  string
		name     string
		target   *bool
		required bool
	}{
		{"smtp", "store.messages", &smtpConfig.StoreMessages, true},
		{"web", "template.cache", &webConfig.TemplateCache, true},
		{"web", "monitor.visible", &webConfig.MonitorVisible, true},
	}
	for _, opt := range boolOptions {
		if Config.HasOption(opt.section, opt.name) {
			flag, err := Config.Bool(opt.section, opt.name)
			if err != nil {
				messages = append(messages, fmt.Sprintf(parseErrorFmt, opt.section, opt.name, err))
			}
			*opt.target = flag
		} else {
			if opt.required {
				messages = append(messages, fmt.Sprintf(missingErrorFmt, opt.section, opt.name))
			}
		}
	}
	// Load integer config options
	intOptions := []struct {
		section  string
		name     string
		target   *int
		required bool
	}{
		{"smtp", "ip4.port", &smtpConfig.IP4port, true},
		{"smtp", "max.recipients", &smtpConfig.MaxRecipients, true},
		{"smtp", "max.idle.seconds", &smtpConfig.MaxIdleSeconds, true},
		{"smtp", "max.message.bytes", &smtpConfig.MaxMessageBytes, true},
		{"pop3", "ip4.port", &pop3Config.IP4port, true},
		{"pop3", "max.idle.seconds", &pop3Config.MaxIdleSeconds, true},
		{"web", "ip4.port", &webConfig.IP4port, true},
		{"web", "monitor.history", &webConfig.MonitorHistory, true},
		{"datastore", "retention.minutes", &dataStoreConfig.RetentionMinutes, true},
		{"datastore", "retention.sleep.millis", &dataStoreConfig.RetentionSleep, true},
		{"datastore", "mailbox.message.cap", &dataStoreConfig.MailboxMsgCap, true},
	}
	for _, opt := range intOptions {
		if Config.HasOption(opt.section, opt.name) {
			num, err := Config.Int(opt.section, opt.name)
			if err != nil {
				messages = append(messages, fmt.Sprintf(parseErrorFmt, opt.section, opt.name, err))
			}
			*opt.target = num
		} else {
			if opt.required {
				messages = append(messages, fmt.Sprintf(missingErrorFmt, opt.section, opt.name))
			}
		}
	}
	// Load IP address config options
	ipOptions := []struct {
		section  string
		name     string
		target   *net.IP
		required bool
	}{
		{"smtp", "ip4.address", &smtpConfig.IP4address, true},
		{"pop3", "ip4.address", &pop3Config.IP4address, true},
		{"web", "ip4.address", &webConfig.IP4address, true},
	}
	for _, opt := range ipOptions {
		if Config.HasOption(opt.section, opt.name) {
			str, err := Config.String(opt.section, opt.name)
			if err != nil {
				messages = append(messages, fmt.Sprintf(parseErrorFmt, opt.section, opt.name, err))
				continue
			}
			addr := net.ParseIP(str)
			if addr == nil {
				messages = append(messages,
					fmt.Sprintf("Failed to parse IP [%v]%v: %q", opt.section, opt.name, str))
				continue
			}
			addr = addr.To4()
			if addr == nil {
				messages = append(messages,
					fmt.Sprintf("Failed to parse IP [%v]%v: %q not IPv4!",
						opt.section, opt.name, str))
			}
			*opt.target = addr
		} else {
			if opt.required {
				messages = append(messages, fmt.Sprintf(missingErrorFmt, opt.section, opt.name))
			}
		}
	}
	// Validate log level
	switch strings.ToUpper(logLevel) {
	case "":
		// Missing was already reported
	case "TRACE", "INFO", "WARN", "ERROR":
	default:
		messages = append(messages,
			fmt.Sprintf("Invalid value provided for [logging]level: %q", logLevel))
	}
	// Print messages and return error if any validations failed
	if len(messages) > 0 {
		fmt.Fprintln(os.Stderr, "Error(s) validating configuration:")
		sort.Strings(messages)
		for _, m := range messages {
			fmt.Fprintln(os.Stderr, " -", m)
		}
		return fmt.Errorf("Failed to validate configuration")
	}
	return nil
}
