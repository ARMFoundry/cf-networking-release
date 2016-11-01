#!/bin/bash

set -e -u

THIS_DIR=$(cd $(dirname $0) && pwd)
cd $THIS_DIR

export CONFIG=/tmp/bosh-lite-integration-config.json
export APPS_DIR=../example-apps
export BASE_MANIFEST=../../bosh-lite/deployments/diego.yml
export UPGRADE_MANIFEST=../../bosh-lite/deployments/diego_with_netman.yml
#bosh vms 2>/dev/null |grep router |grep -v ha_proxy|awk '{print  $11}'
export ASG_TARGET_IP="10.244.0.22"

echo '
{
  "api": "api.bosh-lite.com",
  "admin_user": "admin",
  "admin_password": "admin",
  "apps_domain": "bosh-lite.com",
  "skip_ssl_validation": true,
  "use_http": true,
  "bosh_director_url":"https://192.168.50.4:25555",
  "bosh_admin_user":"admin",
  "bosh_admin_password":"admin",
  "bosh_deployment_name":"cf-warden-diego",
  "bosh_director_ca_cert":"../../../bosh-lite/ca/certs/ca.crt"
}
' > $CONFIG

../../scripts/generate-bosh-lite-manifests

ginkgo -v .
