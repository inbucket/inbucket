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


## Homebrew Tap

(currently broken, being tracked in [issue
#68](https://github.com/inbucket/inbucket/issues/68))

Inbucket has an OS X [Homebrew] tap available as [jhillyerd/inbucket][Homebrew Tap],
see the `README.md` there for installation instructions.


## Building from Source

You will need a functioning [Go installation][Google Go] for this to work.

Grab the Inbucket source code and compile the daemon:

    go get -v github.com/inbucket/inbucket/cmd/inbucket

Edit etc/inbucket.conf and tailor to your environment.  It should work on most
Unix and OS X machines as is.  Launch the daemon:

    $GOPATH/bin/inbucket $GOPATH/src/github.com/inbucket/inbucket/etc/inbucket.conf

By default the SMTP server will be listening on localhost port 2500 and
the web interface will be available at [localhost:9000](http://localhost:9000/).

The Inbucket website has a more complete guide to
[installing from source][From Source]


## About

Inbucket is written in [Google Go]

Inbucket is open source software released under the MIT License.  The latest
version can be found at https://github.com/inbucket/inbucket

[Go API docs]:      https://godoc.org/github.com/inbucket/inbucket/pkg/rest/client
[Build Status]:     https://travis-ci.org/inbucket/inbucket
[Change Log]:       https://github.com/inbucket/inbucket/blob/master/CHANGELOG.md
[CONTRIBUTING.md]:  https://github.com/inbucket/inbucket/blob/develop/CONTRIBUTING.md
[From Source]:      http://www.inbucket.org/installation/from-source.html
[Google Go]:        http://golang.org/
[Homebrew]:         http://brew.sh/
[Homebrew Tap]:     https://github.com/inbucket/homebrew-inbucket
[Inbucket Website]: http://www.inbucket.org/
[Issues List]:      https://github.com/inbucket/inbucket/issues?state=open
