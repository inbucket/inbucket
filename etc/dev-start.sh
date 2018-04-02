#!/bin/sh
# dev-start.sh
# description: Developer friendly Inbucket configuration

export INBUCKET_LOGLEVEL="debug"
export INBUCKET_SMTP_DISCARDDOMAINS="bitbucket.local"
export INBUCKET_WEB_TEMPLATECACHE="false"
export INBUCKET_WEB_COOKIEAUTHKEY="not-secret"
export INBUCKET_STORAGE_TYPE="file"
export INBUCKET_STORAGE_PARAMS="path:/tmp/inbucket"
export INBUCKET_STORAGE_RETENTIONPERIOD="15m"

if ! test -x ./inbucket; then
  echo "$PWD/inbucket not found/executable!" >&2
  echo "Run this script from the inbucket root directory after running make" >&2
  exit 1
fi

exec ./inbucket $*
