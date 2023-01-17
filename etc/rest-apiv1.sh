#!/usr/bin/env bash
# rest-apiv1.sh
# description: Script to access Inbucket REST API version 1

API_HOST="localhost"
URL_ROOT="http://$API_HOST:9000/api/v1"

set -eo pipefail
[ $TRACE ] && set -x

usage() {
  echo "Usage: $0 <command> [argument1 [argument2 [..]]]"               >&2
  echo                                                                  >&2
  echo "Options:"                                                       >&2
  echo "  -h                       - show this help"                    >&2
  echo "  -i                       - show HTTP headers"                 >&2
  echo                                                                  >&2
  echo "Commands:"                                                      >&2
  echo "  list <mailbox>           - list mailbox contents"             >&2
  echo "  body <mailbox> <id>      - print message body"                >&2
  echo "  source <mailbox> <id>    - print message source"              >&2
  echo "  delete <mailbox> <id>    - delete message"                    >&2
  echo "  purge <mailbox>          - delete all messages in mailbox"    >&2
}

arg_check() {
  declare command="$1" expected="$2" received="$3"
  if [ $expected != $received ]; then
    echo "Error: Command '$command' requires $expected arguments, but received $received" >&2
    echo >&2
    usage
    exit 1
  fi
}

main() {
  # Process options
  local curl_opts=""
  local pretty="true"
  for arg in $*; do
    if [[ $arg == -* ]]; then
      case "$arg" in
        -h)
          usage
          exit
          ;;
        -i)
          curl_opts="$curl_opts -i"
          pretty=""
          ;;
        **)
          echo "Unknown option: $arg" >&2
          echo
          usage
          exit 1
          ;;
      esac
      shift
    else
      break
    fi
  done

  # Store command
  declare command="$1"
  shift

  local url=""
  local method="GET"
  local is_json=""

  case "$command" in
    body)
      arg_check "$command" 2 $#
      url="$URL_ROOT/mailbox/$1/$2"
      is_json="true"
      ;;
    delete)
      arg_check "$command" 2 $#
      method=DELETE
      url="$URL_ROOT/mailbox/$1/$2"
      ;;
    list)
      arg_check "$command" 1 $#
      url="$URL_ROOT/mailbox/$1"
      is_json="true"
      ;;
    purge)
      arg_check "$command" 1 $#
      method=DELETE
      url="$URL_ROOT/mailbox/$1"
      ;;
    source)
      arg_check "$command" 2 $#
      url="$URL_ROOT/mailbox/$1/$2/source"
      ;;
    *)
      echo "Unknown command $command" >&2
      echo >&2
      usage
      exit 1
      ;;
  esac

  # Use jq to pretty-print if installed and we are expecting JSON output
  if [ $pretty ] && [ $is_json ] && type -P jq >/dev/null; then
    curl -s $curl_opts -H "Accept: application/json" --noproxy "$API_HOST" -X "$method" "$url" | jq .
  else
    curl -s $curl_opts -H "Accept: application/json" --noproxy "$API_HOST" -X "$method" "$url"
  fi
}

if [ $# -lt 1 ]; then
  usage
  exit 1
fi

main $*
