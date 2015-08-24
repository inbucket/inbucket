#!/bin/bash
# install.sh
# description: Build, test, and install Inbucket. Should be executed inside a Docker container.

set -eo pipefail

installdir="${INBUCKET_HOME}"
srcdir="${INBUCKET_SRC}"
bindir="$installdir/bin"

# Setup
export GOBIN="$bindir"
builddate="$(date --iso-8601=seconds)"
cd "$srcdir"
go clean

# Build
echo "### Fetching Dependencies"
go get -d -v ./...
go get -v github.com/stretchr/testify

echo "### Testing Inbucket"
go test ./...

echo "### Building Inbucket"
mkdir -p "$bindir"
go build -o inbucket -race -ldflags "-X 'main.BUILD_DATE=$builddate'" -v .

echo "### Installing Inbucket"
mv inbucket "$bindir"
install etc/docker/inbucket.conf /etc/opt/inbucket.conf
install etc/docker/greeting.html /etc/opt/inbucket-greeting.html
cp -r themes "$installdir/"
