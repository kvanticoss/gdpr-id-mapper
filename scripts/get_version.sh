#!/bin/bash

# Will echo [{maybe git tag}]_{short commit hash}
echo "$(git describe --exact-match --tags $(git log -n1 --pretty='%h') 2> /dev/null)$TAG$(git log -n1 --pretty='%h')"
