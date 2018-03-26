Change Log
==========

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added
- Inbucket is now configured using environment variables instead of a config
  file.
- In-memory storage option, best for small installations and desktops.  Will be
  used by default.
- Storage type is now displayed on Status page.
- Store size is now calculated during retention scan and displayed on the Status
  page.
- Debian `.deb` package generation to release process.

### Changed
- Massive refactor of back-end code.  Inbucket should now be both easier and
  more enjoyable to work on.
- Renamed `themes` directory to `ui` and eliminated the intermediate `bootstrap`
  directory.
- Docker build:
  - Uses the same default ports as other builds; smtp:2500 http:9000 pop3:1100
  - Uses volume `/config` for `greeting.html`
  - Uses volume `/storage` for mail storage


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
  [client](https://github.com/jhillyerd/inbucket/wiki/cmd-client)
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
  JSON](https://github.com/jhillyerd/inbucket/wiki/REST-GET-message)
- [Go client for REST
  API](https://godoc.org/github.com/jhillyerd/inbucket/rest/client)
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

[Unreleased]:  https://github.com/jhillyerd/inbucket/compare/master...develop
[v1.3.1]:      https://github.com/jhillyerd/inbucket/compare/v1.3.0...v1.3.1
[v1.3.0]:      https://github.com/jhillyerd/inbucket/compare/v1.2.0...v1.3.0
[v1.2.0]:      https://github.com/jhillyerd/inbucket/compare/1.2.0-rc2...1.2.0
[v1.2.0-rc2]:  https://github.com/jhillyerd/inbucket/compare/1.2.0-rc1...1.2.0-rc2
[v1.2.0-rc1]:  https://github.com/jhillyerd/inbucket/compare/1.1.0...1.2.0-rc1
[v1.1.0]:      https://github.com/jhillyerd/inbucket/compare/1.1.0-rc2...1.1.0
[v1.1.0-rc2]:  https://github.com/jhillyerd/inbucket/compare/1.1.0-rc1...1.1.0-rc2
[v1.1.0-rc1]:  https://github.com/jhillyerd/inbucket/compare/1.0...1.1.0-rc1
[v1.0]:        https://github.com/jhillyerd/inbucket/compare/1.0-rc1...1.0


## Release Checklist

1.  Create release branch: `git flow release start 1.x.0`
2.  Update CHANGELOG.md:
    - Ensure *Unreleased* section is up to date
    - Rename *Unreleased* section to release name and date.
    - Add new GitHub `/compare` link
3.  Run tests
4.  Test cross-compile: `goreleaser --snapshot`
5.  Commit changes and merge release: `git flow release finish`
6.  Push tags and wait for https://travis-ci.org/jhillyerd/inbucket build to
    complete
7.  Update `binary_versions` option in `inbucket-site/_config.yml`

See http://keepachangelog.com/ for additional instructions on how to update this file.
