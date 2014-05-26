Inbucket [![Build Status](https://travis-ci.org/jhillyerd/inbucket.png?branch=master)](https://travis-ci.org/jhillyerd/inbucket)
========

Inbucket is an email testing service; it will accept messages for any email
address and make them available via web, REST and POP3.  Once compiled,
Inbucket does not have an external dependencies (HTTP, SMTP, POP3 and storage
are all built in).

Read more at the [Inbucket website][Inbucket]

Development Status
------------------

Inbucket is currently production quality: it is being used for real work.

Please check the [issues list][Issues]
for more details.

Building from Source
------------------------

You will need a functioning [Go installation][Golang] for this to work.

Grab the Inbucket source code and compile the daemon:

    go get -v github.com/jhillyerd/inbucket

Edit etc/inbucket.conf and tailor to your environment.  It should work on most
Unix and OS X machines as is.  Launch the daemon:

    $GOPATH/bin/inbucket $GOPATH/src/github.com/jhillyerd/inbucket/etc/inbucket.conf

By default the SMTP server will be listening on localhost port 2500 and
the web interface will be available at [localhost:9000](http://localhost:9000/).

The Inbucket website has a more complete guide to
[installing from source][From Source]

About
-----

Inbucket is written in [Google Go][Golang].

Inbucket is open source software released under the MIT License.  The latest
version can be found at https://github.com/jhillyerd/inbucket

[Inbucket]: http://www.inbucket.org/
[Issues]: https://github.com/jhillyerd/inbucket/issues?state=open
[From Source]: http://www.inbucket.org/installation/from-source.html
[Golang]: http://golang.org/
