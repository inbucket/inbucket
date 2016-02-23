#!/bin/sh
# install.sh
# description: Build, test, and install Inbucket. Should be executed inside a Docker container.

set -eo pipefail

installdir="${INBUCKET_HOME}"
srcdir="${INBUCKET_SRC}"
bindir="$installdir/bin"

# Setup
export GOBIN="$bindir"
builddate="$(date -Iseconds)"
cd "$srcdir"
go clean

# Build
echo "### Fetching Dependencies"
go get -d -t -v ./...

echo "### Testing Inbucket"
go test ./...

echo "### Building Inbucket"
go build -o inbucket -ldflags "-X 'main.BUILDDATE=$builddate'" -v .

echo "### Installing Inbucket"
mkdir -p "$bindir"
mkdir -p "/etc/opt"
mv inbucket "$bindir"
install etc/docker/inbucket.conf /etc/opt/inbucket.conf
install etc/docker/greeting.html /etc/opt/inbucket-greeting.html
cp -r themes "$installdir/"
