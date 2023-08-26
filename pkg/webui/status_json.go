package webui

type jsonServerConfig struct {
	Version       string            `json:"version"`
	BuildDate     string            `json:"build-date"`
	POP3Listener  string            `json:"pop3-listener"`
	WebListener   string            `json:"web-listener"`
	SMTPConfig    jsonSMTPConfig    `json:"smtp-config"`
	StorageConfig jsonStorageConfig `json:"storage-config"`
}

type jsonSMTPConfig struct {
	Addr                string   `json:"addr"`
	DefaultAccept       bool     `json:"default-accept"`
	AcceptDomains       []string `json:"accept-domains"`
	RejectDomains       []string `json:"reject-domains"`
	DefaultStore        bool     `json:"default-store"`
	StoreDomains        []string `json:"store-domains"`
	DiscardDomains      []string `json:"discard-domains"`
	RejectOriginDomains []string `json:"reject-origin-domains"`
}

type jsonStorageConfig struct {
	MailboxMsgCap   int    `json:"mailbox-msg-cap"`
	StoreType       string `json:"store-type"`
	RetentionPeriod string `json:"retention-period"`
}
