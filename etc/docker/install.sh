#!/bin/sh
# install.sh
# description: Build, test, and install Inbucket. Should be executed inside a Docker container.

set -eo pipefail

installdir="$INBUCKET_HOME"
srcdir="$INBUCKET_SRC"
bindir="$installdir/bin"
defaultsdir="$installdir/defaults"
contextdir="/con/context"

echo "### Installing OS Build Dependencies"
apk add --no-cache --virtual .build-deps git

# Setup
export GOBIN="$bindir"
cd "$srcdir"
builddate="$(date -Iseconds)"
buildver="$(git describe --tags --always)"

# Build
go clean
echo "### Fetching Dependencies"
go get -t -v ./...

echo "### Testing Inbucket"
go test ./...

echo "### Building Inbucket"
go build -o inbucket -ldflags "-X 'main.version=$buildver' -X 'main.date=$builddate'" -v ./cmd/inbucket

echo "### Installing Inbucket"
set -x
mkdir -p "$bindir"
install inbucket "$bindir"
mkdir -p "$contextdir"
install etc/docker/defaults/start-inbucket.sh /
cp -r ui "$installdir/"
mkdir -p "$defaultsdir"
cp etc/docker/defaults/greeting.html "$defaultsdir"
set +x

echo "### Removing OS Build Dependencies"
apk del .build-deps

echo "### Removing $GOPATH"
rm -rf "$GOPATH"
