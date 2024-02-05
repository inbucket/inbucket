#!/bin/sh
# dev-start.sh
# description: Developer friendly Inbucket configuration

export INBUCKET_LOGLEVEL="debug"
#export INBUCKET_MAILBOXNAMING="domain"
export INBUCKET_SMTP_REJECTDOMAINS="bad-actors.local"
#export INBUCKET_SMTP_DEFAULTACCEPT="false"
export INBUCKET_SMTP_ACCEPTDOMAINS="good-actors.local"
export INBUCKET_SMTP_DISCARDDOMAINS="bitbucket.local"
#export INBUCKET_SMTP_DEFAULTSTORE="false"
export INBUCKET_SMTP_STOREDOMAINS="important.local"
export INBUCKET_WEB_TEMPLATECACHE="false"
export INBUCKET_WEB_COOKIEAUTHKEY="not-secret"
export INBUCKET_WEB_UIDIR="ui/dist"
#export INBUCKET_WEB_MONITORVISIBLE="false"
#export INBUCKET_WEB_BASEPATH="prefix"
export INBUCKET_STORAGE_TYPE="file"
export INBUCKET_STORAGE_PARAMS="path:/tmp/inbucket"
export INBUCKET_STORAGE_RETENTIONPERIOD="3h"
export INBUCKET_STORAGE_MAILBOXMSGCAP="300"

if ! test -x ./inbucket; then
  echo "$PWD/inbucket not found/executable!" >&2
  echo "Run this script from the inbucket root directory after running make." >&2
  exit 1
fi

index="$INBUCKET_WEB_UIDIR/index.html"
if ! test -f "$index"; then
  echo "$index does not exist!" >&2
  echo "Run 'yarn build' from the 'ui' directory." >&2
  exit 1
fi

exec ./inbucket $*
