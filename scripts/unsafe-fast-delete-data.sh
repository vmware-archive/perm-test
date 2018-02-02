#!/bin/bash

set -eu

echo "Deleting organizations"
cf curl /v2/organizations | jq -r ".resources[].metadata.url" | xargs -I {} cf curl -X DELETE "{}?recursive=true&async=true"

echo "Deleting users"
cf curl /v2/users | jq -r ".resources[].metadata.url" | xargs -I {} cf curl -X DELETE {}
