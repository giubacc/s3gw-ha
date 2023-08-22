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

imageS3GW="quay.io/s3gw/s3gw"
IMAGE_TAG=${IMAGE_TAG:-$(git describe --tags --always)}
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

source "${SCRIPT_DIR}/helpers.sh"

prepare_system_domain

# IMAGE_TAG is the one built from the 'make build-images'
echo "using s3gw image-tag    : $IMAGE_TAG"

k3d image import -c s3gw-ha "${imageS3GW}:v${IMAGE_TAG}"
echo "Importing s3gw image Completed ✔️"

function deploy_s3gw_latest_released {
  helm upgrade --wait --install -n s3gw-ha --create-namespace s3gw s3gw/s3gw  \
    --set publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set ui.publicDomain="$S3GW_SYSTEM_DOMAIN" \
    --set imageTag=v"${IMAGE_TAG}"
}

echo "Deploying s3gw"
deploy_s3gw_latest_released

echo
echo "Done deploying s3gw! ✔️"
