Change Log
==========

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

[Unreleased]
------------

### Added
- Storage of To: header in messages (likely breaks existing datastores)

### Fixed
- We should no longer run out of file handles when dealing with a large number
  of recipients on a single message.

[1.1.0] - 2016-09-03
--------------------

### Added
- Homebrew inbucket.conf and formula (see README)

### Fixed
- Log and continue when unable to delete oldest message during cap enforcement

[1.1.0-rc2] - 2016-03-06
------------------------

### Added
- Message Cap to status page
- Search-while-you-type message list filter

### Fixed
- Shutdown hang in retention scanner
- Display empty subject as `(No Subject)`

[1.1.0-rc1] - 2016-03-04
------------------------

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

[1.0] - 2014-04-14
------------------

### Added
- Add new configuration option `mailbox.message.cap` to prevent individual
  mailboxes from growing too large.
- Add Link button to messages, allows for directing another person to a
  specific message.

[Unreleased]: https://github.com/jhillyerd/inbucket/compare/master...develop
[1.1.0]:      https://github.com/jhillyerd/inbucket/compare/1.1.0-rc2...1.1.0
[1.1.0-rc2]:  https://github.com/jhillyerd/inbucket/compare/1.1.0-rc1...1.1.0-rc2
[1.1.0-rc1]:  https://github.com/jhillyerd/inbucket/compare/1.0...1.1.0-rc1
[1.0]:        https://github.com/jhillyerd/inbucket/compare/1.0-rc1...1.0


Release Checklist
-----------------

1.  Create release branch: `git flow release start 1.x.0`
2.  Update CHANGELOG.md:
  - Ensure *Unreleased* section is up to date
  - Rename *Unreleased* section to release name and date.
  - Add new GitHub `/compare` link
3.  Update goxc version info: `goxc -wc -pv=1.x.0 -pr=snapshot`
4.  Run: `goxc interpolate-source` to update VERSION var
5.  Run tests
6.  Test cross-compile: `goxc`
7.  Commit changes and merge release: `git flow release finish 1.x.0`
8.  Upload to bintray: `goxc bintray`
9.  Update `binary_versions` option in `inbucket-site/_config.yml`

See http://keepachangelog.com/ for additional instructions on how to update this file.
