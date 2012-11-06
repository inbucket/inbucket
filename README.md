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

Features
--------
 * Receive and store SMTP & ESMTP
 * List messages in a mailbox
 * Display the text content of a particular message
 * Display the source of a message (headers + body text)
 * Display the HTML version of a message (in a new window)
 * List MIME attachments with buttons to display or download
 * Delete a message
 * Purge messages after a configurable amount of time
 * Optional load test mode; messages are never written to disk

It does not yet:

 * Display inline attachments within HTML email

Screenshots
-----------
![An Email](http://cloud.github.com/downloads/jhillyerd/inbucket/inbucket-ss1.png)
*Viewing an email in Inbucket.*

![Metrics](http://cloud.github.com/downloads/jhillyerd/inbucket/inbucket-ss2.png)
*Watching metrics while Inbucket recieves and stores over 4,000 messages per minute.*

Development Status
------------------
Inbucket is currently beta quality: it works but is not well tested.

Please check the [issues list](https://github.com/jhillyerd/inbucket/issues?state=open)
for more details.

Installation
------------
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
