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

function prepare_system_domain {
  if [[ -z "${S3GW_SYSTEM_DOMAIN}" ]]; then
    echo -e "\e[32mS3GW_SYSTEM_DOMAIN not set. Trying to use a magic domain...\e[0m"
    S3GW_CLUSTER_IP=$(docker inspect k3d-s3gw-ha-server-0 | jq -r '.[0]["NetworkSettings"]["Networks"]["s3gw-ha"]["IPAddress"]')
    if [[ -z $S3GW_CLUSTER_IP ]]; then
      echo "Couldn't find the cluster's IP address"
      exit 1
    fi
    export S3GW_CLUSTER_IP="${S3GW_CLUSTER_IP}"
    export S3GW_SYSTEM_DOMAIN="${S3GW_CLUSTER_IP}.omg.howdoi.website"
  fi
  echo -e "Using \e[32m${S3GW_SYSTEM_DOMAIN}\e[0m for s3gw domain"
}

function dump_scenario_properties {
  echo IMAGE_TAG:$IMAGE_TAG
  echo S3GW_CLUSTER_IP:$S3GW_CLUSTER_IP
  echo S3GW_SYSTEM_DOMAIN:$S3GW_SYSTEM_DOMAIN
  echo RELEASE:$RELEASE
  echo NAMESPACE:$NAMESPACE

  cat > wd/properties.json << EOF
{
  "IMAGE_TAG": "$IMAGE_TAG",
  "S3GW_CLUSTER_IP": "$S3GW_CLUSTER_IP",
  "S3GW_SYSTEM_DOMAIN": "$S3GW_SYSTEM_DOMAIN",
  "RELEASE": "$RELEASE",
  "NAMESPACE": "$NAMESPACE"
}
EOF

  echo -e "dumped properties.json"
}
