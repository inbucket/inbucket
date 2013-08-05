package config

import (
	"container/list"
	"fmt"
	"github.com/robfig/config"
	"net"
	"os"
	"strings"
)

// SmtpConfig houses the SMTP server configuration - not using pointers
// so that I can pass around copies of the object safely.
type SmtpConfig struct {
	Ip4address      net.IP
	Ip4port         int
	Domain          string
	DomainNoStore   string
	MaxRecipients   int
	MaxIdleSeconds  int
	MaxMessageBytes int
	StoreMessages   bool
}

type WebConfig struct {
	Ip4address    net.IP
	Ip4port       int
	TemplateDir   string
	TemplateCache bool
	PublicDir     string
}

type DataStoreConfig struct {
	Path             string
	RetentionMinutes int
	RetentionSleep   int
}

var (
	// Global goconfig object
	Config *config.Config

	// Parsed specific configs
	smtpConfig      *SmtpConfig
	webConfig       *WebConfig
	dataStoreConfig *DataStoreConfig
)

// GetSmtpConfig returns a copy of the SmtpConfig object
func GetSmtpConfig() SmtpConfig {
	return *smtpConfig
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
	requireOption(messages, "web", "ip4.address")
	requireOption(messages, "web", "ip4.port")
	requireOption(messages, "web", "template.dir")
	requireOption(messages, "web", "template.cache")
	requireOption(messages, "web", "public.dir")
	requireOption(messages, "datastore", "path")
	requireOption(messages, "datastore", "retention.minutes")
	requireOption(messages, "datastore", "retention.sleep.millis")

	// Return error if validations failed
	if messages.Len() > 0 {
		fmt.Fprintln(os.Stderr, "Error(s) validating configuration:")
		for e := messages.Front(); e != nil; e = e.Next() {
			fmt.Fprintln(os.Stderr, " -", e.Value.(string))
		}
		return fmt.Errorf("Failed to validate configuration")
	}

	if err = parseSmtpConfig(); err != nil {
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

// parseSmtpConfig trying to catch config errors early
func parseSmtpConfig() error {
	smtpConfig = new(SmtpConfig)
	section := "smtp"

	// Parse IP4 address only, error on IP6.
	option := "ip4.address"
	str, err := Config.String(section, option)
	if err != nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	addr := net.ParseIP(str)
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	addr = addr.To4()
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v' not IPv4!", section, option, err)
	}
	smtpConfig.Ip4address = addr

	option = "ip4.port"
	smtpConfig.Ip4port, err = Config.Int(section, option)
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
		return fmt.Errorf("Failed to parse [%v]%v: '%v'", section, option, err)
	}
	addr = addr.To4()
	if addr == nil {
		return fmt.Errorf("Failed to parse [%v]%v: '%v' not IPv4!", section, option, err)
	}
	webConfig.Ip4address = addr

	option = "ip4.port"
	webConfig.Ip4port, err = Config.Int(section, option)
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
