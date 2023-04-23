#!/bin/bash

echo "node-id: $1"
echo "area-id: $2"

titan-candidate key import --path /root/.titancandidate/private.key
titan-candidate config set --node-id=$1 --area-id=$2

export LOCATOR_API_INFO=$3

titan-candidate run