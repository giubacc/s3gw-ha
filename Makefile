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

CGO_ENABLED ?= 0

tag:
	@git describe --tags --abbrev=0

lint:
	golangci-lint run

tidy:
	go mod tidy

fmt:
	go fmt ./...

########################################################################
# Build

clean-ccache:
	sudo rm -rf build.ccache

clean-build:
	sudo rm -rf ceph/build

cmake:
	@./scripts/run-cmake.sh

build:
	@./scripts/build-image.sh

########################################################################
# cluster Create/Delete/Prepare

cluster-start:
	@./scripts/cluster-create.sh
	k3d kubeconfig merge -ad
	kubectl config use-context k3d-s3gw-ha
	@./scripts/cluster-prepare.sh

cluster-delete:
	k3d cluster delete s3gw-ha
	@if test -f /usr/local/bin/rke2-uninstall.sh; then sudo sh /usr/local/bin/rke2-uninstall.sh; fi

########################################################################
# s3gw Deploy/Undeploy

deploy:
	@./scripts/cluster-s3gw-deploy.sh

undeploy:
	helm uninstall -n s3gw-ha s3gw
