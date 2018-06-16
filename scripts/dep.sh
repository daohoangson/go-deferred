#!/bin/sh

set -e

if which dep >/dev/null; then
  echo 'OK' >/dev/null
else
  # https://github.com/golang/dep
  curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
fi

exec dep "$@"
