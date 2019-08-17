Inbucket
=============================================================================
[![Build Status](https://travis-ci.org/inbucket/inbucket.png?branch=master)][Build Status]

Inbucket is an email testing service; it will accept messages for any email
address and make them available via web, REST and POP3.  Once compiled,
Inbucket does not have any external dependencies (HTTP, SMTP, POP3 and storage
are all built in).

A Go client for the REST API is available in
`github.com/inbucket/inbucket/pkg/rest/client` - [Go API docs]

Read more at the [Inbucket Website]

![Screenshot](http://www.inbucket.org/images/inbucket-ss1.png "Viewing a message")

## Development Status

Inbucket is currently production quality: it is being used for real work.

Please see the [Change Log] and [Issues List] for more details.  If you'd like
to contribute code to the project check out [CONTRIBUTING.md].


## Docker

Inbucket has automated [Docker Image] builds via Docker Hub.  The `stable` tag
tracks our `master` branch (releases), `latest` tracks our unstable
`development` branch.


## Homebrew Tap

(currently broken, being tracked in [issue
#68](https://github.com/inbucket/inbucket/issues/68))

Inbucket has an OS X [Homebrew] tap available as [jhillyerd/inbucket][Homebrew Tap],
see the `README.md` there for installation instructions.


## Building from Source

You will need functioning [Go] and [Node.js] installations for this to work.

```sh
git clone https://github.com/inbucket/inbucket.git
cd inbucket/ui
npm i
npm run build
cd ..
go build ./cmd/inbucket
```

_Note:_ You may also use the included Makefile to build and test the Go binaries.

Inbucket reads its configuration from environment variables, but comes with
built in sane defaults.  It should work on most Unix and OS X machines as is.
Launch the daemon:

```sh
./inbucket
```

By default the SMTP server will be listening on localhost port 2500 and
the web interface will be available at [localhost:9000](http://localhost:9000/).

See doc/[config.md] for more information on configuring Inbucket, but you will
likely find the [Configurator] tool easier to use.


## About

Inbucket is written in [Go]

Inbucket is open source software released under the MIT License.  The latest
version can be found at https://github.com/inbucket/inbucket

[Build Status]:     https://travis-ci.org/inbucket/inbucket
[Change Log]:       https://github.com/inbucket/inbucket/blob/master/CHANGELOG.md
[config.md]:        https://github.com/inbucket/inbucket/blob/master/doc/config.md
[Configurator]:     https://www.inbucket.org/configurator/
[CONTRIBUTING.md]:  https://github.com/inbucket/inbucket/blob/develop/CONTRIBUTING.md
[Docker Image]:     https://www.inbucket.org/binaries/docker.html
[From Source]:      https://www.inbucket.org/installation/from-source.html
[Go]:               https://golang.org/
[Go API docs]:      https://godoc.org/github.com/inbucket/inbucket/pkg/rest/client
[Homebrew]:         http://brew.sh/
[Homebrew Tap]:     https://github.com/inbucket/homebrew-inbucket
[Inbucket Website]: https://www.inbucket.org/
[Issues List]:      https://github.com/inbucket/inbucket/issues?state=open
[Node.js]:          https://nodejs.org/en/
