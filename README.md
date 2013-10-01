Inbucket
========

Inbucket is an email testing service; it will accept messages for any email
address and make them available to view via a web interface.

It allows web developers, software engineers and system administrators to
quickly see the emailed output of ther applications.  No per-account setup is
required! Mailboxes are created on the fly as mail is received for them, and
no password is required to browse the content of the mailboxes.

Inbucket has a built-in SMTP server and stores incoming mail as flat files on
disk - no external SMTP or database daemons required.

There is also a built-in POP3 server, which allows message rendering to be
checked in multiple email programs or to verify message delivery as part of
an integration test.

Read more at the [Inbucket website](http://jhillyerd.github.io/inbucket/).

Development Status
------------------

Inbucket is currently beta quality: it works but is not well tested.

Please check the [issues list](https://github.com/jhillyerd/inbucket/issues?state=open)
for more details.

Installation from Source
------------------------

You will need a functioning [Go installation][1] for this to work.

Grab the Inbucket source code and compile the daemon:

    go get -v github.com/jhillyerd/inbucket

Edit etc/inbucket.conf and tailor to your environment.  It should work on most
Unix and OS X machines as is.  Launch the daemon:

    $GOPATH/bin/inbucket $GOPATH/src/github.com/jhillyerd/inbucket/etc/inbucket.conf

By default the SMTP server will be listening on localhost port 2500 and
the web interface will be available at [localhost:9000](http://localhost:9000/).

There are RedHat EL6 init, logrotate and httpd proxy configs provided.

About
-----

Inbucket is written in [Google Go][1].

Inbucket is open source software released under the MIT License.  The latest
version can be found at https://github.com/jhillyerd/inbucket

[1]: http://golang.org/
