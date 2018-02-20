#!/usr/bin/env bash

set -eu

set -o pipefail

ENVIRONMENT_NAME="${ENVIRONMENT_NAME:?}"
CF_USERNAME="${CF_USERNAME:?}"
CF_PASSWORD="${CF_PASSWORD:?}"

api_endpoint="https://api.${ENVIRONMENT_NAME}.perm.cf-app.com"

cf api "${api_endpoint}" --skip-ssl-validation

cf auth "${CF_USERNAME}" "${CF_PASSWORD}"
authorization_token="$(cat ~/.cf/config.json | jq -r .AccessToken)"
ab -n300 -s6000 -H "Authorization: ${authorization_token}" -c1 -k "${api_endpoint}/v3/apps"

cf auth "${CF_USERNAME}" "${CF_PASSWORD}"
authorization_token="$(cat ~/.cf/config.json | jq -r .AccessToken)"
ab -n1000 -s6000 -H "Authorization: ${authorization_token}" -c10 -k "${api_endpoint}/v3/apps"
