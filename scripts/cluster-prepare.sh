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

# IMAGE_TAG is the one built from the 'make build-images'
IMAGE_TAG=${IMAGE_TAG:-$(git describe --tags --always)}
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "${SCRIPT_DIR}/helpers.sh"

# Ensure we have a value for --system-domain
prepare_system_domain

echo "Preparing k3d environment"

#Install the cert-manager
set +e
kubectl create namespace cert-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager --namespace cert-manager jetstack/cert-manager \
    --set installCRDs=true \
    --set extraArgs[0]=--enable-certificate-owner-ref=true \
    --version 1.10 \
    --wait
set -e

# Dump non-static properties
dump_scenario_properties

# Add the s3gw repo
helm repo add s3gw https://aquarist-labs.github.io/s3gw-charts/
helm repo update

echo
echo "Done preparing k3d environment! ✔️"
