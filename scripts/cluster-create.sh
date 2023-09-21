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
NETWORK_NAME=s3gw-ha
CLUSTER_NAME=s3gw-ha
K3S_IMAGE=${K3S_IMAGE:-rancher/k3s:v1.25.10-k3s1}
AGENT_NODES=${AGENT_NODES:-0}
export KUBECONFIG=$SCRIPT_DIR/../tmp/ha-kubeconfig

check_deps() {
  if ! command -v k3d &> /dev/null
  then
      echo "k3d could not be found"
      exit
  fi
}

existingCluster() {
  k3d cluster list | grep ${CLUSTER_NAME}
}

if [[ "$(existingCluster)" != "" ]]; then
  echo "Cluster already exists, skipping creation."
  exit 0
fi

echo "Ensuring a network"
docker network create $NETWORK_NAME || echo "Network already exists"

#kind create cluster --name $CLUSTER_NAME --image $K3S_IMAGE

echo "Creating a new one named $CLUSTER_NAME"
if [ -z ${EXPOSE_CLUSTER_PORTS+x} ]; then
  # Without exposing ports on the host:
  k3d cluster create $CLUSTER_NAME \
    -v /dev/mapper:/dev/mapper \
    --network $NETWORK_NAME \
    --agents $AGENT_NODES \
    --image "$K3S_IMAGE"
else
  # Exposing ports on the host:
  k3d cluster create $CLUSTER_NAME \
    -v /dev/mapper:/dev/mapper \
    --network $NETWORK_NAME \
    --agents $AGENT_NODES \
    --image "$K3S_IMAGE" \
    -p '80:80@server:0' -p '443:443@server:0'
fi
k3d kubeconfig get $CLUSTER_NAME > $KUBECONFIG

echo "Waiting for node to be ready"
nodeName=$(kubectl get nodes -o name | head -n 1)
kubectl wait --for=condition=Ready "$nodeName"

date
echo "Waiting for the deployments of the foundational configurations to be ready"
# 1200s = 20 min, to handle even a horrendously slow setup. Regular is 10 to 30 seconds.
kubectl wait --for=condition=Available --namespace kube-system deployment/metrics-server --timeout=1200s
kubectl wait --for=condition=Available --namespace kube-system deployment/coredns --timeout=1200s
kubectl wait --for=condition=Available --namespace kube-system deployment/local-path-provisioner --timeout=1200s
date

echo
echo "Done! The cluster is ready. ✔️"
