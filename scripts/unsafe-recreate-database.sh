#!/bin/bash

set -eu

ENVIRONMENT_NAME="${ENVIRONMENT_NAME:-bosh-cleopatra}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-cf}"
VARIABLE_NAME="${VARIABLE_NAME:-cf_mysql_mysql_admin_password}"

credhub_path="/${ENVIRONMENT_NAME}/${DEPLOYMENT_NAME}/${VARIABLE_NAME}"

mysql_admin_username=root
mysql_admin_password="$(https_proxy=$BOSH_ALL_PROXY credhub get -n "${credhub_path}" -j | jq -r .value)"

echo "Recreating cloud_controller database"
bosh -d "${DEPLOYMENT_NAME}" ssh database/0 \
  -c "/var/vcap/packages/mariadb/bin/mysql -u ${mysql_admin_username} --password=${mysql_admin_password} -e 'drop database cloud_controller'"
bosh -d "${DEPLOYMENT_NAME}" ssh database/0 \
  -c "/var/vcap/packages/mariadb/bin/mysql -u ${mysql_admin_username} --password=${mysql_admin_password} -e 'create database cloud_controller'"

echo "Recreating perm database"
bosh -d "${DEPLOYMENT_NAME}" ssh database/0 \
  -c "/var/vcap/packages/mariadb/bin/mysql -u ${mysql_admin_username} --password=${mysql_admin_password} -e 'drop database perm'"
bosh -d "${DEPLOYMENT_NAME}" ssh database/0 \
  -c "/var/vcap/packages/mariadb/bin/mysql -u ${mysql_admin_username} --password=${mysql_admin_password} -e 'create database perm'"

echo "Restarting CC"
bosh -d "${DEPLOYMENT_NAME}" -n restart api

echo "Restarting perm"
bosh -d "${DEPLOYMENT_NAME}" -n restart perm
