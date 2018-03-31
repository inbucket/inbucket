# Inbucket Configuration

Inbucket is configured via environment variables.  Most options have a
reasonable default, but it is likely you will need to change some to suite your
desired use cases.

Running `inbucket -help` will yield a condensed summary of the environment
variables it supports:

    KEY                                 DEFAULT             DESCRIPTION
    INBUCKET_LOGLEVEL                   INFO                DEBUG, INFO, WARN, or ERROR
    INBUCKET_SMTP_ADDR                  0.0.0.0:2500        SMTP server IP4 host:port
    INBUCKET_SMTP_DOMAIN                inbucket            HELO domain
    INBUCKET_SMTP_DOMAINNOSTORE                             Load testing domain
    INBUCKET_SMTP_MAXRECIPIENTS         200                 Maximum RCPT TO per message
    INBUCKET_SMTP_MAXMESSAGEBYTES       10240000            Maximum message size
    INBUCKET_SMTP_STOREMESSAGES         true                Store incoming mail?
    INBUCKET_SMTP_TIMEOUT               300s                Idle network timeout
    INBUCKET_POP3_ADDR                  0.0.0.0:1100        POP3 server IP4 host:port
    INBUCKET_POP3_DOMAIN                inbucket            HELLO domain
    INBUCKET_POP3_TIMEOUT               600s                Idle network timeout
    INBUCKET_WEB_ADDR                   0.0.0.0:9000        Web server IP4 host:port
    INBUCKET_WEB_UIDIR                  ui                  User interface dir
    INBUCKET_WEB_GREETINGFILE           ui/greeting.html    Home page greeting HTML
    INBUCKET_WEB_TEMPLATECACHE          true                Cache templates after first use?
    INBUCKET_WEB_MAILBOXPROMPT          @inbucket           Prompt next to mailbox input
    INBUCKET_WEB_COOKIEAUTHKEY                              Session cipher key (text)
    INBUCKET_WEB_MONITORVISIBLE         true                Show monitor tab in UI?
    INBUCKET_WEB_MONITORHISTORY         30                  Monitor remembered messages
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
should probably select INFO, but a busy shared installation would be better off
with WARN or ERROR.

- Default: `INFO`
- Values: one of `DEBUG`, `INFO`, `WARN`, or `ERROR`


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

### Load Testing/No Store Domain

`INBUCKET_SMTP_DOMAINNOSTORE`

Mail sent to this domain will not be stored by Inbucket.  This is helpful if you
are load or soak testing a service, and do not plan to inspect the resulting
emails.  Messages sent to a domain other than this will be stored normally.

- Default: None
- Example: `bitbucket.local`

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

### Store Messages

`INBUCKET_SMTP_STOREMESSAGES`

This option can be used to disable mail storage entirely.  Useful for load
testing, or turning Inbucket into a black hole that will consume our entire
solar system.

- Default: `true`
- Values: `true` or `false`

### Network Idle Timeout

`INBUCKET_SMTP_TIMEOUT`

Delay before closing an idle SMTP connection.  The SMTP RFC recommends 300
seconds.  Consider reducing this *significantly* if you plan to expose Inbucket
to the public internet.

- Default: `300s`
- Values: Duration ending in `s` for seconds, `m` for minutes


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

### UI Directory

`INBUCKET_WEB_UIDIR`

This directory contains the templates and static assets for the web user
interface.  You will need to change this if the current working directory
doesn't contain the `ui` directory at startup.

Inbucket will load templates from the `templates` sub-directory, and serve
static assets from the `static` sub-directory.

- Default: `ui`
- Values: Operating system specific path syntax

### Greeting HTML File

`INBUCKET_WEB_GREETINGFILE`

The content of the greeting file will be injected into the front page of
Inbucket.  It can be used to instruct users on how to send mail into your
Inbucket installation, as well as link to REST documentation, etc.

- Default: `ui/greeting.html`

### Template Caching

`INBUCKET_WEB_TEMPLATECACHE`

Tells Inbucket to cache parsed template files.  This should be left as default
unless you are a developer working on the Inbucket web interface.

- Default: `true`
- Values: `true` or `false`

### Mailbox Prompt

`INBUCKET_WEB_MAILBOXPROMPT`

Text prompt displayed to the right of the mailbox name input field in the web
interface.  Can be used to nudge your users into typing just the mailbox name
instead of an entire email address.

Set to an empty string to hide the prompt.

- Default: `@inbucket`

### Cookie Authentication Key

`INBUCKET_WEB_COOKIEAUTHKEY`

Inbucket stores session information in an encrypted browser cookie.  Unless
specified, Inbucket generates a random key at startup.  The only notable data
stored in a user session is the list of recently accessed mailboxes.

- Default: None
- Value: Text string, no particular format required

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
- Examples: `maxkb=10240` or `path=/tmp/inbucket`

#### file parameters

- `path`: Operating system specific path to the directory where mail should be
  stored.

#### memory parameters

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
