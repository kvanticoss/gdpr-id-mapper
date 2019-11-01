#!/bin/bash

# Will echo [{maybe git tag}-]{short commit hash}
TAG=$(git describe --exact-match --tags $(git log -n1 --pretty='%h') 2> /dev/null)
if [ $? -eq 0 ] ; then
  TAG=$TAG-
fi
echo "$(git describe --exact-match --tags $(git log -n1 --pretty='%h') 2> /dev/null)$TAG$(git log -n1 --pretty='%h')"
