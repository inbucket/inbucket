#!/usr/bin/env bash
# run-tests.sh
# description: Generate test emails for Inbucket

set -eo pipefail
[ $TRACE ] && set -x

# We need to be in swaks-tests directory
cmdpath="$(dirname "$0")"
if [ "$cmdpath" != "." ]; then
  cd "$cmdpath"
fi

case "$1" in
  "")
    to="swaks"
    ;;
  --*)
    to="swaks"
    ;;
  *)
    to="$1"
    shift
    ;;
esac

export SWAKS_OPT_server="${SWAKS_OPT_server:-127.0.0.1:2500}"
export SWAKS_OPT_to="$to@inbucket.local"

# Basic test
swaks $* --h-Subject: "Swaks Plain Text" --body text.txt

# Multi-recipient test
swaks $* --to="$to@inbucket.local,alternate@inbucket.local" --h-Subject: "Swaks Multi-Recipient" \
  --body text.txt

# HTML test
swaks $* --h-Subject: "Swaks HTML" --data mime-html.raw

# Top level HTML test
swaks $* --h-Subject: "Swaks Top Level HTML" --data nonmime-html.raw

# Attachment test
swaks $* --h-Subject: "Swaks Attachment" --attach-type image/png --attach favicon.png \
  --body text.txt

# Encoded subject line test
swaks $* --data utf8-subject.raw

# Gmail test
swaks $* --data gmail.raw

# Outlook test
swaks $* --data outlook.raw

# Non-mime responsive HTML test
swaks $* --data nonmime-html-responsive.raw
swaks $* --data nonmime-html-inlined.raw

# Incorrect charset, malformed final boundary
swaks $* --data mime-errors.raw

# IP RCPT domain
swaks $* --to="swaks@[127.0.0.1]" --h-Subject: "IPv4 RCPT Address" --body text.txt
swaks $* --to="swaks@[IPv6:2001:db8:aaaa:1::100]" --h-Subject: "IPv6 RCPT Address" --body text.txt

# Inline attachment test
swaks $* --data mime-inline.raw
