Inbucket
========

Inbucket is an email testing service; it will accept messages for any email
address and make them available to view via a web interface.

It allows web developers, software engineers and system administrators to
quickly see the emailed output of ther applications.  No per-account setup is
required! Mailboxes are created on the fly as mail is received for them, and
no password is required to browse the cotent of the mailboxes.

Inbucket has a built-in SMTP server and stores incoming mail as flat files on
disk - no external SMTP or database daemons required.

Status
------
Inbucket is currently in development.  It mostly works, but is not well
tested or documented.

About
-----
Inbucket is written in [Google Go][1], and utilizes the [Revel framework][2]
for its web interface.

Inbucket is open source software released under the MIT License.  The latest
version can be found at [github](https://github.com/jhillyerd/inbucket).

[1]: http://golang.org/
[2]: http://robfig.github.com/revel/ 
