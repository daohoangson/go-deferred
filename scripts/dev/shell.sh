#!/bin/sh

_path="/go/src/github.com/daohoangson/go-deferred"

exec docker run --rm -it -v "$PWD:$_path" -w "$_path" golang bash
