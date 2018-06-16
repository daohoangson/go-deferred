#!/bin/sh

set -e

_imageName=${1:-'xfrocks/go-deferred'}
_gitHead=`git rev-parse HEAD`
_tagged="$_imageName:$_gitHead"

echo "Building image $_tagged"
docker build . -t "$_imageName" -t "$_tagged"

while true
do
  read -p "Push image? [yN]" yn
  case $yn in
    [Yy]* ) break;;
    * ) exit;;
  esac
done

docker push "$_imageName:latest"
docker push "$_tagged"
