#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/bosh-lite-integration-config.json
export APPS_DIR=../example-apps
export BASE_MANIFEST=../../bosh-lite/deployments/diego.yml
export UPGRADE_MANIFEST=../../bosh-lite/deployments/diego_with_netman.yml

echo '
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,
  "bosh_director_url":"lite",
  "bosh_admin_user":"admin",
  "bosh_admin_password":"admin",
  "bosh_deployment_name":"cf-warden-diego"
}
' > $CONFIG

../../scripts/generate-bosh-lite-manifests

ginkgo -v .
