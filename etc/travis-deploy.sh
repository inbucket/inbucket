#!/bin/bash
# travis-deploy.sh
# description: Trigger goreleaser deployment

set -eo pipefail
set -x

# downloading deps probably added to go.mod and go.sum, goreleaser will fail.
git reset --hard
git clean -dfx

# build release.
curl -sL https://git.io/goreleaser | bash
