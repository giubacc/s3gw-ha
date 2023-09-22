#!/bin/bash
# Copyright © 2023 SUSE LLC
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

source "${SCRIPT_DIR}/helpers.sh"

prepare_system_domain

set +e
helm uninstall -n s3gw-sd s3gw-probe
set -e

k3d image import -c s3gw-ha ghcr.io/giubacc/s3gw-probe:latest
echo "Importing s3gw-probe image Completed ✔️"

function deploy_s3gw_probe {
  set +e
  helm upgrade --wait --install -n s3gw-sd --create-namespace s3gw-probe charts/s3gw-probe \
    --set probe.publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set s3gw.endpointIngress=http://s3gw-ha-s3gw-ha."$S3GW_SYSTEM_DOMAIN"
  set -e
}

echo "Deploying s3gw-probe"
deploy_s3gw_probe

echo
echo "Done deploying s3gw-probe! ✔️"
