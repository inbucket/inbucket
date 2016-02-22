swaks-tests
===========

[Swaks](http://www.jetmore.org/john/code/swaks/) - Swiss Army Knife for SMTP

Swaks gives us an easy way to generate mail to send into Inbucket.  You will need to
install Swaks before you can use the provided scripts.

## Usage

To deliver a batch of test email to the `swaks` mailbox, assuming Inbucket SMTP is listening
on localhost:2500:

    ./run-tests.sh

To deliver a batch of test email to the `james` mailbox:

    ./run-tests.sh james

You may also pass swaks options to deliver to a alternate host/port:

    ./run-tests --server inbucket.mydomain.com:25

To specify the mailbox with an alternate server, use `--to` with a local and host part:

    ./run-tests --server inbucket.mydomain.com:25 --to james@mydomain.com

## To Do

Replace Swaks with a native Go solution.
