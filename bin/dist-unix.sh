#!/bin/sh
# Compile and package inbucket dist for unix

if [ "${1}x" = "x" ]; then
  echo "Usage: $0 <version-label>" 1>&2
  exit 1
fi
label="$1"

# Bail on error
set -e

# Work directory
tmpdir=/tmp/inbucket-dist.$$
mkdir -p $tmpdir

# Figure out our build env/target
go env > $tmpdir/env
. $tmpdir/env
distname="inbucket-${label}-${GOOS}_$GOARCH"
distdir="$tmpdir/$distname"

echo "Building $distname..."
mkdir -p $distdir
go build -o $distdir/inbucket -a -v github.com/jhillyerd/inbucket

echo "Copying resources..."
cp LICENSE README.md $distdir/
cp -r etc $distdir/etc
cp -r themes $distdir/themes

echo "Tarballing..."
tarball="$HOME/$distname.tbz2"
cd $tmpdir
tar --owner=root --group=root -cjvf $tarball $distname

echo "Cleaning up..."
if [ "$tmpdir" != "/" ]; then
  rm -rf $tmpdir
fi

echo "Created $tarball"
