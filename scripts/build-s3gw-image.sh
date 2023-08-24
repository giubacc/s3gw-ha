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

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
IMAGE_TAG=${IMAGE_TAG:-"$(git describe --tags --always)"}
imageS3GW="quay.io/s3gw/s3gw"

echo "--- Building s3gw-builder image ---"
docker build -t s3gw-builder:latest -f dockerfiles/Dockerfile.s3gw-builder .
echo "Building s3gw-builder image Completed ✔️"

echo "--- Building radosgw ---"
mkdir -p ${SCRIPT_DIR}/../build.ccache
docker run -v ${SCRIPT_DIR}/../build.ccache:/build.ccache -v ${SCRIPT_DIR}/../ceph:/ceph s3gw-builder:latest
echo "Building radosgw Completed ✔️"

echo "--- Creating final s3gw image ---"
docker build -t "${imageS3GW}:v${IMAGE_TAG}" -t "${imageS3GW}:latest" -f dockerfiles/Dockerfile.s3gw .
echo "Creating s3gw image Completed ✔️"
