#!/bin/sh

_secret="$1"
if [ -z "$_secret" ]; then
  echo 'defermon requires a secret to run' >&2
  exit 1
fi

exec docker run --rm -p 8080:8080 xfrocks/go-deferred defermon 8080 "$_secret"
