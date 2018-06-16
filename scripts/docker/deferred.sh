#!/bin/sh

exec docker run --rm xfrocks/go-deferred deferred "$@"
