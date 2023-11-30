#!/bin/sh
# docker-run.sh
# description: Launch Inbucket's docker image

# Docker Image Tag
IMAGE="inbucket/inbucket:edge"

# Ports exposed on host:
PORT_HTTP=9000
PORT_SMTP=2500
PORT_POP3=1100

# Volumes exposed on host:
VOL_CONFIG="/tmp/inbucket/config"
VOL_DATA="/tmp/inbucket/storage"

set -e

main() {
  local run_opts=""

  for arg in $*; do
    case "$arg" in
      -h)
        usage
        exit
        ;;
      -b)
        build
        ;;
      -r)
        reset
        ;;
      -d)
        run_opts="$run_opts -d"
        ;;
      *)
        usage
        exit 1
        ;;
    esac
  done

  set -x

  docker run $run_opts \
    -p $PORT_HTTP:9000 \
    -p $PORT_SMTP:2500 \
    -p $PORT_POP3:1100 \
    -v "$VOL_CONFIG:/config" \
    -v "$VOL_DATA:/storage" \
    "$IMAGE"
}

usage() {
  echo "$0 [options]" 2>&1
  echo "  -b    build - build image before starting" 2>&1
  echo "  -d    detach - detach and print container ID" 2>&1
  echo "  -r    reset - purge config and data before startup" 2>&1
  echo "  -h    help - print this message" 2>&1
}

build() {
  echo "Building $IMAGE"
  docker build . -t "$IMAGE"
  echo
}

reset() {
  rm -rf "$VOL_CONFIG"
  rm -rf "$VOL_DATA"
}

main $*
