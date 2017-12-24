#!/bin/bash
# travis-deploy.sh
# description: Trigger goreleaser deployment in correct build scenarios

set -eo pipefail
set -x

if [[ "$TRAVIS_GO_VERSION" == "$DEPLOY_WITH_MAJOR."* ]]; then
  curl -sL https://git.io/goreleaser | bash
fi
