Change Log
==========

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]


## [v3.1.1] - 2025-12-06

### Fixed
- Go version update for CVE-2025-47907
- Removed broken Windows arm7 build (#589)


## [v3.1.0] - 2025-07-27

### Added
- Note in logs that a missing Lua script is not an error (#575)

### Fixed
- Accept and handle emails sent with an empty 821.From (#561)


## [v3.1.0-beta3] - 2024-11-02

### Added
- Lua scripting additions:
  - Add `SMTPSession` and `BeforeRcptToAccepted` event (#541)
  - Add `SMTPResponse` type for extensions (#539)
  - Add `RemoteAddr` to `SMTPSession` (#548)
- Context support for REST client (#496)

### Fixed
- Rename Lua `BeforeMailAccepted`, change args (#547)
- pop3: Prevent STLS cmd triggered crashes (#516)
- ui: date-format version, fixes yarn build (#508)
- rework client example to omit `log.Fatal`, breaks defer (#489)
- Rest Client: Allow relative URLs (#477)


## [v3.1.0-beta2] - 2024-02-05

### Added
- Reject mail by origin domain: `INBUCKET_SMTP_REJECTORIGINDOMAINS` (#375)
  - Wildcard support (#412)
- Version flag for `inbucket` cmd (#385)
- STLS support for POP3 (#384)
- ForceTLS flag for SMTP (#402)
- Lua scripting additions:
  - `logger` API for Lua (#407)
  - `before.message_stored` handler (#417, #418)
- `$` is replaced with `:` in filestore paths, for `D:\...` syntax (#449)
- REST Client `transport` support (#463)

### Fixed
- UI & Storage paths in systemd service file (#393)
- Web UI will redirect from `prefix` to `prefix/` (#397)
- Include inlines when listing attachments (#398)
- Fail Inbucket startup if unable to create storage dir (#448)
- Close directory file handles immediately, fixes Windows locking (#457)


## [v3.1.0-beta1] - 2023-02-28

### Added
- Monitor tab updates when messages are deleted (#337)
- Initial framework for extensions
- Initial Lua scripting implementation, supporting events:
  - `after.message_deleted`
  - `after.message_stored`
  - `before.mail_accepted`
- Provide `http` and `json` modules for Lua scripts

### Fixed
- Support for IP address as domain in RCPT TO (#285)


## [v3.0.4] - 2022-10-02

### Fixed
- More flexible support of `AUTH=<>` FROM parameter (#291)


## [v3.0.3] - 2022-08-07

### Fixed
- Support for `AUTH=<>` FROM parameter (#284)


## [v3.0.2] - 2022-07-04

Note: We had to abandon the 3.0.1 release, see the blog post [What happened to
3.0?](https://www.inbucket.org/news/2022/05/whathappenedtothree.html) for
details.

### Changed
- arm Docker builds now rely on amd64 frontend build stage
- Frontend build migrated from npm+webpack to yarn+parcel, node 16


## [v3.0.1-rc2] - 2022-01-23

### Added
- Builds for arm7 and arm64 platforms

### Changed
- Abandoned git-flow process, the `master` branch renamed to `main`


## [v3.0.1-rc1] - 2022-01-17

### Fixed
- GitHub built packages (rpm, deb, tarball) no longer missing UI files (#250)

### Changed
- Update Go dependencies
- Update NPM dependencies


## [v3.0.0] - 2021-09-19

Unchanged from rc4.


## [v3.0.0-rc4] - 2021-08-22

### Fixed
- Various MIME header decoding improvements

### Changed
- Bump Go version to 1.17 (#233)


## v3.0.0-rc3 - 2021-08-01

Unchanaged from 3.0.0-rc2.  This release is to update our build automation and
tags for Docker Hub and ghcr.io.


## [v3.0.0-rc2] - 2021-07-31

### Added
- Support for SMTP AUTH (#197, thanks makarchuk)
- Dark mode support (#218, thanks nerones)

### Fixed
- Prevent potential click jacking (#190, thanks stuartskelton)
- Error on 8 character long SMTP commands (#221)
- Allow empty username and password during AUTH (#225)


## [v3.0.0-rc1] - 2020-09-24

### Added
- Refresh button to reload mailbox contents
- Improved keyboard (tab) focus highlights

### Changed
- The UI now includes the Open Sans webfont instead of relying on browser/OS
  fonts


## [v3.0.0-beta3] - 2020-09-04

### Added
- Docker `HEALTHCHECK`
- Mouse-out delay to improve pop-up menu navigation
- Support for configurable URL base path with `INBUCKET_WEB_BASEPATH`

### Changed
- Updated frontend and backend dependencies, Docker image base

### Fixed
- Improved layout on mobile and wide displays
- Prevent unexpected input for modal dialogs
- Allow empty SMTP `MAIL FROM:<>`


## [v3.0.0-beta2] - 2019-08-17

### Added
- Ability to name mailboxes after domain of email recipient, set via
  `INBUCKET_MAILBOXNAMING`, thanks MatthewJohn.

### Changed
- Updated JavaScript dependencies.
- Updated Go dependencies.
- Updated Docker build: Go to 1.12, and Alpine Linux to 3.10

### Fixed
- URLs to view/download attachments from REST API, #138
- Support for late EHLO, #141


## [v3.0.0-beta1] - 2019-03-14

### Added
- `posix-millis` field to REST message and header responses for easier date
  parsing.

### Changed
- Rewrote the user interface from scratch, it's now an Elm powered single page
  application.
- Moved the Inbucket repository to its own GitHub organization.
- Update to enmime v0.5.0


## v2.1.0 - 2018-12-15

No change from beta1.


## [v2.1.0-beta1] - 2018-10-31

### Added
- Use Go 1.11 modules for reproducible builds.
- SMTP TLS support (thanks kingforaday.)
- `INBUCKET_WEB_PPROF` configuration option for performance profiling.
- Godoc example for the REST API client.

### Changed
- Docker build now uses Go 1.11 and Alpine 3.8

### Fixed
- Render UTF-8 addresses correctly in both REST API and Web UI.
- Memory storage now correctly returns the newest message when asked for ID
  `latest`.


## [v2.0.0] - 2018-05-05

### Changed
- Corrected docs for INBUCKET_STORAGE_PARAMS (thanks evilmrburns.)
- Disabled color log output on Windows, doesn't work there.


## [v2.0.0-rc1] - 2018-04-07

### Added
- Inbucket is now configured using environment variables instead of a config
  file.
- In-memory storage option, best for small installations and desktops.  Will be
  used by default.
- Storage type is now displayed on Status page.
- Store size is now calculated during retention scan and displayed on the Status
  page.
- Debian `.deb` package generation to release process.
- RedHat `.rpm` package generation to release process.
- Message seen flag in REST and Web UI so you can see which messages have
  already been read.
- Recipient domain accept policy; Inbucket can now reject mail to specific
  domains.
- Configurable support for identifying a mailbox by full email address instead
  of just the local part (username).
- Friendly URL support: `<inbucket-url>/<mailbox>` will redirect your browser to
  that mailbox.

### Changed
- Massive refactor of back-end code.  Inbucket should now be both easier and
  more enjoyable to work on.
- Changes to file storage format, will require pre-2.0 mail store directories to
  be deleted.
- Renamed `themes` directory to `ui` and eliminated the intermediate `bootstrap`
  directory.
- Docker build:
  - Uses the same default ports as other builds; smtp:2500 http:9000 pop3:1100
  - Uses volume `/config` for `greeting.html`
  - Uses volume `/storage` for mail storage
- Log output is now structured, and will be output as JSON with the `-logjson`
  flag; which is enabled by default for the Docker container.
- SMTP and POP3 network tracing is no longer logged regardless of level, but can
  be sent to stdout via `-netdebug` flag.
- Replaced store/nostore config variables with a storage policy that mirrors the
  domain accept policy.

### Removed
- No longer support SIGHUP or log file rotation.


## [v1.3.1] - 2018-03-10

### Fixed
- Adding additional locking during message delivery to prevent race condition
  that could lose messages.


## [v1.3.0] - 2018-02-28

### Added
- Button to purge mailbox contents from the UI.
- Simple HTML/CSS sanitization; `Safe HTML` and `Plain Text` UI tabs.

### Changed
- Reverse message display sort order in the UI; now newest first.


## [v1.2.0] - 2017-12-27

### Changed
- No significant code changes from rc2

### Added
- `rest/client` types `MessageHeader` and `Message` with convenience methods;
  provides a more natural API
- Powerful command line REST
  [client](https://github.com/inbucket/inbucket/wiki/cmd-client)
- Allow use of `latest` as a message ID in REST calls

### Changed
- `rest/client.NewV1` renamed to `New`
- `rest/client` package now embeds the shared `rest/model` structs into its own
  types
- Fixed panic when `monitor.history` set to 0


## [v1.2.0-rc1] - 2017-01-29

### Added
- Storage of `To:` header in messages (likely breaks existing datastores)
- Attachment list to [GET message
  JSON](https://github.com/inbucket/inbucket/wiki/REST-GET-message)
- [Go client for REST
  API](https://godoc.org/github.com/inbucket/inbucket/rest/client)
- Monitor feature: lists messages as they arrive, regardless of their
  destination mailbox
- Make `@inbucket` mailbox prompt configurable
- Warnings and errors from MIME parser are displayed with message

### Fixed
- No longer run out of file handles when dealing with a large number of
  recipients for a single message.
- Empty intermediate directories are now removed when a mailbox is deleted,
  leaving less junk on your filesystem.

### Changed
- Build now requires Go 1.7 or later
- Removed legacy `integral` theme, as most new features only in `bootstrap`
- Removed old RESTful APIs, must use `/api/v1` base URI now
- Allow increased local-part length of 128 chars for Mailgun
- RedHat and Ubuntu now use systemd instead of legacy init systems


## [v1.1.0] - 2016-09-03

### Added
- Homebrew inbucket.conf and formula (see README)

### Fixed
- Log and continue when unable to delete oldest message during cap enforcement


## [v1.1.0-rc2] - 2016-03-06

### Added
- Message Cap to status page
- Search-while-you-type message list filter

### Fixed
- Shutdown hang in retention scanner
- Display empty subject as `(No Subject)`


## [v1.1.0-rc1] - 2016-03-04

### Added
- Inbucket now builds with Go 1.5 or 1.6
- Project can build & run inside a Docker container
- Add new default theme based on Bootstrap
- Your recently accessed mailboxes are listed in the GUI
- HTML-only messages now get down-converted to plain text automatically
- This change log

### Changed
- RESTful API moved to `/api/v1` base URI
- More graceful shutdown on Ctrl-C or when errors encountered


## [v1.0] - 2014-04-14

### Added
- Add new configuration option `mailbox.message.cap` to prevent individual
  mailboxes from growing too large.
- Add Link button to messages, allows for directing another person to a
  specific message.


[Unreleased]:   https://github.com/inbucket/inbucket/compare/v3.1.1...main
[v3.1.1]: https://github.com/inbucket/inbucket/compare/v3.1.0...v3.1.1
[v3.1.0]: https://github.com/inbucket/inbucket/compare/v3.1.0-beta3...v3.1.0
[v3.1.0-beta3]: https://github.com/inbucket/inbucket/compare/v3.1.0-beta2...v3.1.0-beta3
[v3.1.0-beta2]: https://github.com/inbucket/inbucket/compare/v3.1.0-beta1...v3.1.0-beta2
[v3.1.0-beta1]: https://github.com/inbucket/inbucket/compare/v3.0.4...v3.1.0-beta1
[v3.0.4]:       https://github.com/inbucket/inbucket/compare/v3.0.3...v3.0.4
[v3.0.3]:       https://github.com/inbucket/inbucket/compare/v3.0.2...v3.0.3
[v3.0.2]:       https://github.com/inbucket/inbucket/compare/v3.0.1-rc2...v3.0.2
[v3.0.1-rc2]:   https://github.com/inbucket/inbucket/compare/v3.0.1-rc1...v3.0.1-rc2
[v3.0.1-rc1]:   https://github.com/inbucket/inbucket/compare/v3.0.0...v3.0.1-rc1
[v3.0.0]:       https://github.com/inbucket/inbucket/compare/v3.0.0-rc4...v3.0.0
[v3.0.0-rc4]:   https://github.com/inbucket/inbucket/compare/v3.0.0-rc2...v3.0.0-rc4
[v3.0.0-rc2]:   https://github.com/inbucket/inbucket/compare/v3.0.0-rc1...v3.0.0-rc2
[v3.0.0-rc1]:   https://github.com/inbucket/inbucket/compare/v3.0.0-beta3...v3.0.0-rc1
[v3.0.0-beta3]: https://github.com/inbucket/inbucket/compare/v3.0.0-beta2...v3.0.0-beta3
[v3.0.0-beta2]: https://github.com/inbucket/inbucket/compare/v3.0.0-beta1...v3.0.0-beta2
[v3.0.0-beta1]: https://github.com/inbucket/inbucket/compare/v2.1.0...v3.0.0-beta1
[v2.1.0-beta1]: https://github.com/inbucket/inbucket/compare/v2.0.0...v2.1.0-beta1
[v2.0.0]:       https://github.com/inbucket/inbucket/compare/v2.0.0-rc1...v2.0.0
[v2.0.0-rc1]:   https://github.com/inbucket/inbucket/compare/v1.3.1...v2.0.0-rc1
[v1.3.1]:       https://github.com/inbucket/inbucket/compare/v1.3.0...v1.3.1
[v1.3.0]:       https://github.com/inbucket/inbucket/compare/v1.2.0...v1.3.0
[v1.2.0]:       https://github.com/inbucket/inbucket/compare/1.2.0-rc2...1.2.0
[v1.2.0-rc2]:   https://github.com/inbucket/inbucket/compare/1.2.0-rc1...1.2.0-rc2
[v1.2.0-rc1]:   https://github.com/inbucket/inbucket/compare/1.1.0...1.2.0-rc1
[v1.1.0]:       https://github.com/inbucket/inbucket/compare/1.1.0-rc2...1.1.0
[v1.1.0-rc2]:   https://github.com/inbucket/inbucket/compare/1.1.0-rc1...1.1.0-rc2
[v1.1.0-rc1]:   https://github.com/inbucket/inbucket/compare/1.0...1.1.0-rc1
[v1.0]:         https://github.com/inbucket/inbucket/compare/1.0-rc1...1.0


## Release Checklist

1.  Create a release branch
2.  Update CHANGELOG.md:
    - Ensure *Unreleased* section is up to date
    - Rename *Unreleased* section to release name and date
    - Add new GitHub `/compare` link
    - Update previous tag version for *Unreleased* 
3.  Run tests
4.  Update goreleaser, and then test cross-compile: `goreleaser release --snapshot --clean`
5.  Commit changes and merge release PR into main
6.  Create new release via GitHub, use CHANGELOG release notes, tag `vX.Y.Z`
7.  Push tags and wait for
    [GitHub actions](https://github.com/inbucket/inbucket/actions) to complete
    -- it will add compiled release assets

See http://keepachangelog.com/ for additional instructions on how to update this file.
