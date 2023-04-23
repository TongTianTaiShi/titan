#!/bin/bash

echo "node-id: $1"
echo "area-id: $2"


titan-edge key import --path /root/.titanedge/private.key
titan-edge config set --node-id=$1 --area-id=$2

export LOCATOR_API_INFO=$3

titan-edge run