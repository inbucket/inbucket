#!/bin/sh
# start-inbucket.sh
# description: start inbucket (runs within a docker container)

INBUCKET_HOME="/opt/inbucket"

set -eo pipefail

exec "$INBUCKET_HOME/bin/inbucket" $* 
