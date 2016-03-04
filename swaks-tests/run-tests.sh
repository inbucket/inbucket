#!/bin/sh
# run-tests.sh
# description: Generate test emails for Inbucket

set -eo pipefail

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
    ;;
esac

export SWAKS_OPT_server="127.0.0.1:2500"
export SWAKS_OPT_to="$to@inbucket.local"

# Basic test
swaks $* --h-Subject: "Swaks Plain Text" --body text.txt

# HTML test
swaks $* --h-Subject: "Swaks HTML" --data mime-html.raw

# Top level HTML test
swaks $* --h-Subject: "Swaks Top Level HTML" --data nonmime-html.raw

# Attachment test
swaks $* --h-Subject: "Swaks Attachment" --attach-type image/png --attach favicon.png --body text.txt

# Encoded subject line test
swaks $* --data utf8-subject.raw

# Gmail test
swaks $* --data gmail.raw

# Outlook test
swaks $* --data outlook.raw

# Nonemime responsive HTML test
swaks $* --data nonmime-html-responsive.raw
