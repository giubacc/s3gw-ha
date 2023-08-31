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

tidy:
	go mod tidy

########################################################################
# Build

s3gw-clean-ccache:
	sudo rm -rf build.ccache

s3gw-clean-build:
	sudo rm -rf ceph/build

s3gw-cmake:
	@./scripts/run-cmake.sh

s3gw-build:
	@./scripts/build-s3gw-image.sh

s3gw-push-image:
	docker push ghcr.io/giubacc/s3gw:latest

probe-build:
	go build -o probe/bin/probe probe/main.go
	docker build -t ghcr.io/giubacc/s3gw-probe:latest -f dockerfiles/Dockerfile.s3gw-probe .

probe-push-image:
	docker push ghcr.io/giubacc/s3gw-probe:latest

########################################################################
# k3d cluster Create/Delete/Prepare

k3d-cluster-start:
	@./scripts/cluster-create.sh
	k3d kubeconfig merge -ad
	kubectl config use-context k3d-s3gw-ha
	@./scripts/cluster-prepare.sh

k3d-cluster-delete:
	k3d cluster delete s3gw-ha
	@if test -f /usr/local/bin/rke2-uninstall.sh; then sudo sh /usr/local/bin/rke2-uninstall.sh; fi

########################################################################
# k3d s3gw Deploy/Undeploy

k3d-s3gw-deploy:
	@./scripts/cluster-s3gw-deploy.sh

k3d-s3gw-undeploy:
	helm uninstall -n s3gw-ha s3gw

########################################################################
# k3d probe Deploy/Undeploy

k3d-probe-deploy:
	@./scripts/cluster-probe-deploy.sh

k3d-probe-undeploy:
	helm uninstall -n s3gw-ha s3gw-probe
