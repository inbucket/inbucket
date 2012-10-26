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

Screenshots
-----------
![An Email](http://cloud.github.com/downloads/jhillyerd/inbucket/inbucket-ss1.png)
*Viewing an email in Inbucket.*

![Metrics](http://cloud.github.com/downloads/jhillyerd/inbucket/inbucket-ss2.png)
*Watching metrics while Inbucket handles over 4,000 messages per minute.*

Development Status
------------------
Inbucket is currently alpha quality: it works but is not well tested.

It can:

 * Receive SMTP and ESMTP messages and store them to disk
 * List subject, sender and date of messages for a particular mailbox
 * Parse MIME multipart emails
 * Display the content of a particular message
 * Display the source of a message (headers + body text)
 * Display the HTML version of a message (in a new window)
 * Delete a message

It does not yet:

 * Display or download attachments

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

About
-----
Inbucket is written in [Google Go][1].

Inbucket is open source software released under the MIT License.  The latest
version can be found at https://github.com/jhillyerd/inbucket

[1]: http://golang.org/
