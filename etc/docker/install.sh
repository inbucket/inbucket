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
git fetch -t
builddate="$(date -Iseconds)"
buildver="$(git describe --tags)"

# Build
go clean
echo "### Fetching Dependencies"
go get -t -v ./...

echo "### Testing Inbucket"
go test ./...

echo "### Building Inbucket"
go build -o inbucket -ldflags "-X 'main.version=$buildver' -X 'main.date=$builddate'" -v .

echo "### Installing Inbucket"
set -x
mkdir -p "$bindir"
install inbucket "$bindir"
mkdir -p "$contextdir"
install etc/docker/defaults/start-inbucket.sh "$contextdir"
cp -r themes "$installdir/"
mkdir -p "$defaultsdir"
cp etc/docker/defaults/inbucket.conf "$defaultsdir"
cp etc/docker/defaults/greeting.html "$defaultsdir"
set +x

echo "### Removing OS Build Dependencies"
apk del .build-deps

echo "### Removing $GOPATH"
rm -rf "$GOPATH"
