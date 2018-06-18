#!/bin/bash

cd "$( dirname "${BASH_SOURCE[0]}" )"
_pwd="$( pwd )"

docker run --rm -it \
  --volume /var/run/docker.sock:/var/run/docker.sock:ro \
  --volume "$_pwd:$_pwd" --workdir "$_pwd" \
  docker/compose:1.21.2 --project-name defermon --file defermon.yml \
  "$@"
