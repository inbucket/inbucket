# Inbucket Configuration

Inbucket is configured via environment variables.  Most options have a
reasonable default, but it is likely you will need to change some to suite your
desired use cases.

Running `inbucket -help` will yield a condensed summary of the environment
variables it supports:

    KEY                                 DEFAULT             DESCRIPTION
    INBUCKET_LOGLEVEL                   info                debug, info, warn, or error
    INBUCKET_LUA_PATH                   inbucket.lua        Lua script path
    INBUCKET_MAILBOXNAMING              local               Use local, full, or domain addressing
    INBUCKET_SMTP_ADDR                  0.0.0.0:2500        SMTP server IP4 host:port
    INBUCKET_SMTP_DOMAIN                inbucket            HELO domain
    INBUCKET_SMTP_MAXRECIPIENTS         200                 Maximum RCPT TO per message
    INBUCKET_SMTP_MAXMESSAGEBYTES       10240000            Maximum message size
    INBUCKET_SMTP_DEFAULTACCEPT         true                Accept all mail by default?
    INBUCKET_SMTP_ACCEPTDOMAINS                             Domains to accept mail for
    INBUCKET_SMTP_REJECTDOMAINS                             Domains to reject mail for
    INBUCKET_SMTP_REJECTORIGINDOMAINS                       Domains to reject mail from
    INBUCKET_SMTP_DEFAULTSTORE          true                Store all mail by default?
    INBUCKET_SMTP_STOREDOMAINS                              Domains to store mail for
    INBUCKET_SMTP_DISCARDDOMAINS                            Domains to discard mail for
    INBUCKET_SMTP_TIMEOUT               300s                Idle network timeout
    INBUCKET_SMTP_TLSENABLED            false               Enable STARTTLS option
    INBUCKET_SMTP_TLSPRIVKEY            cert.key            X509 Private Key file for TLS Support
    INBUCKET_SMTP_TLSCERT               cert.crt            X509 Public Certificate file for TLS Support
    INBUCKET_POP3_ADDR                  0.0.0.0:1100        POP3 server IP4 host:port
    INBUCKET_POP3_DOMAIN                inbucket            HELLO domain
    INBUCKET_POP3_TIMEOUT               600s                Idle network timeout
    INBUCKET_WEB_ADDR                   0.0.0.0:9000        Web server IP4 host:port
    INBUCKET_WEB_BASEPATH                                   Base path prefix for UI and API URLs
    INBUCKET_WEB_UIDIR                  ui/dist             User interface dir
    INBUCKET_WEB_GREETINGFILE           ui/greeting.html    Home page greeting HTML
    INBUCKET_WEB_MONITORVISIBLE         true                Show monitor tab in UI?
    INBUCKET_WEB_MONITORHISTORY         30                  Monitor remembered messages
    INBUCKET_WEB_PPROF                  false               Expose profiling tools on /debug/pprof
    INBUCKET_STORAGE_TYPE               memory              Storage impl: file or memory
    INBUCKET_STORAGE_PARAMS                                 Storage impl parameters, see docs.
    INBUCKET_STORAGE_RETENTIONPERIOD    24h                 Duration to retain messages
    INBUCKET_STORAGE_RETENTIONSLEEP     50ms                Duration to sleep between mailboxes
    INBUCKET_STORAGE_MAILBOXMSGCAP      500                 Maximum messages per mailbox

The following documentation will describe each of these in more detail.


## Global

### Log Level

`INBUCKET_LOGLEVEL`

This setting controls the verbosity of log output.  A small desktop installation
should probably select `info`, but a busy shared installation would be better
off with `warn` or `error`.

- Default: `info`
- Values: one of `debug`, `info`, `warn`, or `error`

### Lua Script

`INBUCKET_LUA_PATH`

This is the path to the (optional) Inbucket Lua script.  If the specified file
is present, Inbucket will load it during startup.  Ignored if the file is not
found, or the setting is empty.

- Default: `inbucket.lua`

### Mailbox Naming

`INBUCKET_MAILBOXNAMING`

The mailbox naming setting determines the name of a mailbox for an incoming
message, and thus where it must be retrieved from later.

#### `local` ensures the domain is removed, such that:

- `james@inbucket.org` is stored in `james`
- `james+spam@inbucket.org` is stored in `james`

#### `full` retains the domain as part of the name, such that:

- `james@inbucket.org` is stored in `james@inbucket.org`
- `james+spam@inbucket.org` is stored in `james@inbucket.org`

Prior to the addition of the mailbox naming setting, Inbucket always operated in
local mode.  Regardless of this setting, the `+` wildcard/extension is not
incorporated into the mailbox name.

#### `domain` ensures the local-part is removed, such that:

- `james@inbucket.org` is stored in `inbucket.org`
- `matt@inbucket.org` is stored in `inbucket.org`
- `matt@noinbucket.com` is stored in `notinbucket.com`

- Default: `local`
- Values: one of `local` or `full` or `domain`


## SMTP

### Address and Port

`INBUCKET_SMTP_ADDR`

The IPv4 address and TCP port number the SMTP server should listen on, separated
by a colon.  Some operating systems may prevent Inbucket from listening on port
25 without escalated privileges.  Using an IP address of 0.0.0.0 will cause
Inbucket to listen on all available network interfaces.

- Default: `0.0.0.0:2500`

### Greeting Domain

`INBUCKET_SMTP_DOMAIN`

The domain used in the SMTP greeting:

    220 domain Inbucket SMTP ready

Most SMTP clients appear to ignore this value.

- Default: `inbucket`

### Maximum Recipients

`INBUCKET_SMTP_MAXRECIPIENTS`

Maximum number of recipients allowed (SMTP `RCPT TO` phase).  If you are testing
a mailing list server, you may need to increase this value.  For comparison, the
Postfix SMTP server uses a default of 1000, it would be unwise to exceed this.

- Default: `200`

### Maximum Message Size

`INBUCKET_SMTP_MAXMESSAGEBYTES`

Maximum allowable size of a message (including headers) in bytes.  Messages
exceeding this size will be rejected during the SMTP `DATA` phase.

- Default: `10240000` (10MB)

### Default Recipient Accept Policy

`INBUCKET_SMTP_DEFAULTACCEPT`

If true, Inbucket will accept mail to any domain unless present in the reject
domains list.  If false, recipients will be rejected unless their domain is
present in the accept domains list.

- Default: `true`
- Values: `true` or `false`

### Accepted Recipient Domain List

`INBUCKET_SMTP_ACCEPTDOMAINS`

List of domains to accept mail for when `INBUCKET_SMTP_DEFAULTACCEPT` is false;
has no effect when true.

- Default: None
- Values: Comma separated list of recipient domains
- Example: `localhost,mysite.org`

### Rejected Recipient Domain List

`INBUCKET_SMTP_REJECTDOMAINS`

List of domains to reject mail for when `INBUCKET_SMTP_DEFAULTACCEPT` is true;
has no effect when false.

- Default: None
- Values: Comma separated list of recipient domains
- Example: `reject.com,gmail.com`

### Rejected Origin Domain List

`INBUCKET_SMTP_REJECTORIGINDOMAINS`

List of domains to reject mail from.  This list is enforced regardless of the
`INBUCKET_SMTP_DEFAULTACCEPT` value.

Enforcement takes place during evalation of the `MAIL FROM` SMTP command, the
origin domain is extracted from the address presented and compared against the
list.  It does not take email headers into account.

- Default: None
- Values: Comma separated list of origin domains
- Example: `reject.com,gmail.com`

### Default Recipient Store Policy

`INBUCKET_SMTP_DEFAULTSTORE`

If true, Inbucket will store mail sent to any domain unless present in the
discard domains list.  If false, messages will be discarded unless their domain
is present in the store domains list.

- Default: `true`
- Values: `true` or `false`

### Stored Recipient Domain List

`INBUCKET_SMTP_STOREDOMAINS`

List of domains to store mail for when `INBUCKET_SMTP_DEFAULTSTORE` is false;
has no effect when true.

- Default: None
- Values: Comma separated list of recipient domains
- Example: `localhost,mysite.org`

### Discarded Recipient Domain List

`INBUCKET_SMTP_DISCARDDOMAINS`

Mail sent to these domains will not be stored by Inbucket.  This is helpful if
you are load or soak testing a service, and do not plan to inspect the resulting
emails.  Messages sent to a domain other than this will be stored normally.
Only has an effect when `INBUCKET_SMTP_DEFAULTSTORE` is true.

- Default: None
- Values: Comma separated list of recipient domains
- Example: `recycle.com,loadtest.org`

### Network Idle Timeout

`INBUCKET_SMTP_TIMEOUT`

Delay before closing an idle SMTP connection.  The SMTP RFC recommends 300
seconds.  Consider reducing this *significantly* if you plan to expose Inbucket
to the public internet.

- Default: `300s`
- Values: Duration ending in `s` for seconds, `m` for minutes

### TLS Support Availability

`INBUCKET_SMTP_TLSENABLED`

Enable the STARTTLS option for opportunistic TLS support

- Default: `false`
- Values: `true` or `false`

### TLS Private Key File

`INBUCKET_SMTP_TLSPRIVKEY`

Specify the x509 Private key file to be used for TLS negotiation.
This option is only valid when INBUCKET_SMTP_TLSENABLED is enabled.

- Default: `cert.key`
- Values: filename or path to private key
- Example: `server.privkey`

### TLS Public Certificate File

`INBUCKET_SMTP_TLSCERT`

Specify the x509 Certificate file to be used for TLS negotiation.
This option is only valid when INBUCKET_SMTP_TLSENABLED is enabled.

- Default: `cert.crt`
- Values: filename or path to the certificate key
- Example: `server.crt`

## POP3

### Address and Port

`INBUCKET_POP3_ADDR`

The IPv4 address and TCP port number the POP3 server should listen on, separated
by a colon.  Some operating systems may prevent Inbucket from listening on port
110 without escalated privileges.  Using an IP address of 0.0.0.0 will cause
Inbucket to listen on all available network interfaces.

- Default: `0.0.0.0:1100`

### Greeting Domain

`INBUCKET_POP3_DOMAIN`

The domain used in the POP3 greeting:

    +OK Inbucket POP3 server ready <26641.1522000423@domain>

Most POP3 clients appear to ignore this value.

- Default: `inbucket`

### Network Idle Timeout

`INBUCKET_POP3_TIMEOUT`

Delay before closing an idle POP3 connection.  The POP3 RFC recommends 600
seconds.  Consider reducing this *significantly* if you plan to expose Inbucket
to the public internet.

- Default: `600s`
- Values: Duration ending in `s` for seconds, `m` for minutes


## Web

### Address and Port

`INBUCKET_WEB_ADDR`

The IPv4 address and TCP port number the HTTP server should listen on, separated
by a colon.  Some operating systems may prevent Inbucket from listening on port
80 without escalated privileges.  Using an IP address of 0.0.0.0 will cause
Inbucket to listen on all available network interfaces.

- Default: `0.0.0.0:9000`

### Base Path

`INBUCKET_WEB_BASEPATH`

Base path prefix for UI and API URLs.  This option is used when you wish to
root all Inbucket URLs to a specific path when placing it behind a
reverse-proxy.

For example, setting the base path to `prefix` will move:
- the Inbucket status page from `/status` to `/prefix/status`,
- Bob's mailbox from `/m/bob` to `/prefix/m/bob`, and
- the REST API from `/api/v1/*` to `/prefix/api/v1/*`.

*Note:* This setting will not work correctly when running Inbucket via the npm
development server.

- Default: None

### UI Directory

`INBUCKET_WEB_UIDIR`

This directory contains the templates and static assets for the web user
interface.  You will need to change this if the current working directory
doesn't contain the `ui` directory at startup.

Inbucket will load templates from the `templates` sub-directory, and serve
static assets from the `static` sub-directory.

- Default: `ui/dist`
- Values: Operating system specific path syntax

### Greeting HTML File

`INBUCKET_WEB_GREETINGFILE`

The content of the greeting file will be injected into the front page of
Inbucket.  It can be used to instruct users on how to send mail into your
Inbucket installation, as well as link to REST documentation, etc.

- Default: `ui/greeting.html`

### Monitor Visible

`INBUCKET_WEB_MONITORVISIBLE`

If true, the Monitor tab will be available, allowing users to observe all
messages received by Inbucket as they arrive.  Disabling the monitor facilitates
security through obscurity.

This setting has no impact on the availability of the underlying WebSocket,
which may be used by other parts of the Inbucket interface or continuous
integration tests.

- Default: `true`
- Values: `true` or `false`

### Monitor History

`INBUCKET_WEB_MONITORHISTORY`

The number of messages to remember on the *server* for new Monitor clients.
Does not impact the amount of *new* messages displayed by the Monitor.
Increasing this has no appreciable impact on memory use, but may slow down the
Monitor user interface.

This setting has the same effect on the amount of messages available via
WebSocket.

Setting to 0 will disable the monitor, but will probably break new mail
notifications in the web interface when I finally get around to implementing
them.

- Default: `30`
- Values: Integer greater than or equal to 0

### Performance Profiling & Debug Tools

`INBUCKET_WEB_PPROF`

If true, Go's pprof package will be installed to the `/debug/pprof` URI.  This
exposes detailed memory and CPU performance data for debugging Inbucket.  If you
enable this option, please make sure it is not exposed to the public internet,
as its use can significantly impact performance.

For example usage, see https://golang.org/pkg/net/http/pprof/

- Default: `false`
- Values: `true` or `false`


## Storage

### Type

`INBUCKET_STORAGE_TYPE`

Selects the storage implementation to use.  Currently Inbucket supports two:

- `file`: stores messages as individual files in a nested directory structure
  based on the hash of the mailbox name.  Each mailbox also includes an index
  file to speed up enumeration of the mailbox contents.
- `memory`: stores messages in RAM, they will be lost if Inbucket is restarted,
  or crashes, etc.

File storage is recommended for larger/shared installations.  Memory is better
suited to desktop or continuous integration test use cases.

- Default: `memory`
- Values: `file` or `memory`

### Parameters

`INBUCKET_STORAGE_PARAMS`

Parameters specific to the storage type selected.  Formatted as a comma
separated list of key:value pairs.

- Default: None
- Examples: `maxkb:10240` or `path:/tmp/inbucket`

#### `file` type parameters

- `path`: Operating system specific path to the directory where mail should be
  stored.  `$` characters will be replaced with `:` in the final path value,
  allowing Windows drive letters, i.e. `D$\inbucket`.

#### `memory` type parameters

- `maxkb`: Maximum size of the mail store in kilobytes.  The oldest messages in
  the store will be deleted to enforce the limit.  In-memory storage has some
  overhead, for now it is recommended to set this to half the total amount of
  memory you are willing to allocate to Inbucket.

### Retention Period

`INBUCKET_STORAGE_RETENTIONPERIOD`

If set, Inbucket will scan the contents of its mail store once per minute,
removing messages older than this.  This will be enforced regardless of the type
of storage configured.

- Default: `24h`
- Values: Duration ending in `m` for minutes, `h` for hours.  Should be
  significantly longer than one minute, or `0` to disable.

### Retention Sleep

`INBUCKET_STORAGE_RETENTIONSLEEP`

Duration to sleep between scanning each mailbox for expired messages.
Increasing this number will reduce disk thrashing, but extend the length of time
required to complete a scan of the entire mail store.

This delay is still enforced for memory stores, but could be reduced from the
default.  Setting to `0` may degrade performance of HTTP/SMTP/POP3 services.

- Default: `50ms`
- Values: Duration ending in `ms` for milliseconds, `s` for seconds

### Per Mailbox Message Cap

`INBUCKET_STORAGE_MAILBOXMSGCAP`

Maximum messages allowed in a single mailbox, exceeding this will cause older
messages to be deleted from the mailbox.

- Default: `500`
- Values: Positive integer, or `0` to disable
