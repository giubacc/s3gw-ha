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

echo "--- Building s3gw-cmaker image ---"
docker build --target s3gw-base -t s3gw-base:latest -f dockerfiles/Dockerfile.s3gw-cmaker .
docker build -t s3gw-cmaker:latest -f dockerfiles/Dockerfile.s3gw-cmaker .
echo "Building s3gw-cmaker image Completed ✔️"

echo "--- run cmaker ---"
docker run -v ${SCRIPT_DIR}/../ceph:/ceph s3gw-cmaker:latest
echo "run cmaker Completed ✔️"
