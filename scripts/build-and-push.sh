#!/bin/sh

set -e

_imageName=${1:-'daohoangson/go-deferred'}
_gitHead=`git rev-parse HEAD`
_tagged="$_imageName:$_gitHead"

echo "Building image $_tagged"
docker build . -t "$_imageName" -t "$_tagged"

_push=$PUSH
if [ -z "$_push" ]; then
  while true
  do
    read -p "Push image? [yN]" yn
    case $yn in
      [Yy]* ) break;;
      * ) exit;;
    esac
  done
elif [ "x$_push" != 'xyes' ]; then
  exit
fi

docker push "$_imageName:latest"
docker push "$_tagged"
