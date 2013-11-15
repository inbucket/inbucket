#!/bin/sh

export SWAKS_OPT_server="127.0.0.1:2500"
export SWAKS_OPT_to="swaks@inbucket.local"

# Basic test
swaks $* --h-Subject: "Swaks Plain Text" --body text.txt

# HTML test
swaks $* --h-Subject: "Swaks HTML" --data mime-html.raw

# Top level HTML test
swaks $* --h-Subject: "Swaks Top Level HTML" --data nonmime-html.raw

# Attachment test
swaks $* --h-Subject: "Swaks Attachment" --attach-type image/png --attach favicon.png --body text.txt

# Encoded subject line test
swaks $* --data utf8-subject.txt
