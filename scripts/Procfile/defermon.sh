#!/bin/sh

_port=${DEFERMON_PORT:-'80'}
_secret=${DEFERMON_SECRET:-'s3cr3t'}

exec defermon "$_port" "$_secret"
