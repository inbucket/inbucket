# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]
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

## [1.0] - 2014-04-14
### Added
- Add new configuration option `mailbox.message.cap` to prevent individual
  mailboxes from growing too large.
- Add Link button to messages, allows for directing another person to a
  specific message.

[Unreleased]: https://github.com/jhillyerd/inbucket/compare/1.0...HEAD
[1.0]:        https://github.com/jhillyerd/inbucket/compare/1.0-rc1...1.0

See http://keepachangelog.com/ for instructions on how to update this file.
