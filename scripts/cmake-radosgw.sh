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

echo "--- Building radosgw ---"
mkdir -p ${SCRIPT_DIR}/../build.ccache.local

cd ${SCRIPT_DIR}/../ceph
./do_cmake.sh \
 -DENABLE_GIT_VERSION=ON\
 -DWITH_PYTHON3=3\
 -DWITH_CCACHE=ON\
 -DWITH_TESTS=OFF\
 -DCMAKE_BUILD_TYPE=Debug\
 -DWITH_RADOSGW_AMQP_ENDPOINT=OFF\
 -DWITH_RADOSGW_KAFKA_ENDPOINT=OFF\
 -DWITH_RADOSGW_SELECT_PARQUET=OFF\
 -DWITH_RADOSGW_MOTR=OFF\
 -DWITH_RADOSGW_DBSTORE=ON\
 -DWITH_RADOSGW_SFS=ON\
 -DWITH_RADOSGW_LUA_PACKAGES=OFF\
 -DWITH_MANPAGE=OFF\
 -DWITH_OPENLDAP=OFF\
 -DWITH_LTTNG=OFF\
 -DWITH_RDMA=OFF\
 -DWITH_SYSTEM_BOOST=OFF

echo "cmake radosgw Completed ✔️"
