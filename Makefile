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

export AWS_ACCESS_KEY = test
export AWS_SECRET_KEY = test

tidy:
	go mod tidy

########################################################################
# Probe

s3gw-clean-ccache:
	sudo rm -rf build.ccache

s3gw-clean-build:
	sudo rm -rf ceph/build

s3gw-cmake:
	sudo rm -rf ceph/build
	@./scripts/run-cmake.sh

s3gw-build:
	@./scripts/build-s3gw-image.sh

s3gw-push-image:
	docker push ghcr.io/giubacc/s3gw:latest

docker-probe-build:
	docker build -t ghcr.io/giubacc/s3gw-probe:latest -f dockerfiles/Dockerfile.s3gw-probe .

probe-push-image:
	docker push ghcr.io/giubacc/s3gw-probe:latest


########################################################################
# k3d cluster

k3d-start:
	@./scripts/cluster-create.sh
	k3d kubeconfig merge -ad
	kubectl config use-context k3d-s3gw-ha
	@./scripts/cluster-prepare.sh

k3d-delete:
	k3d cluster delete s3gw-ha
	@if test -f /usr/local/bin/rke2-uninstall.sh; then sudo sh /usr/local/bin/rke2-uninstall.sh; fi

########################################################################
# k3d s3gw Deploy/Undeploy

k3d-deploy:
	@./scripts/cluster-s3gw-deploy.sh

k3d-undeploy:
	helm uninstall -n s3gw-ha s3gw-ha
	helm uninstall -n s3gw-sd s3gw-sd
	helm uninstall -n s3gw-sd s3gw-probe

########################################################################
# local tests

local-setup:
	cd tests \
	&& python3 -m venv venv \
	&& source venv/bin/activate \
	&& pip install -r requirements.txt

watchdog:
	cd tests \
	&& python3 -m venv venv \
	&& source venv/bin/activate \
	&& python3 ./s3gw_watchdog.py

fio:
	cd tests \
	&& python3 -m venv venv \
	&& source venv/bin/activate \
	&& python3 ./fio_watchdog.py

saver:
	cd tests \
	&& python3 -m venv venv \
	&& source venv/bin/activate \
	&& python3 ./s3gw_data_saver.py

radosgw-cmake:
	sudo rm -rf ceph/build
	@./scripts/cmake-radosgw.sh

radosgw-compile:
	@./scripts/build-radosgw.sh

radosgw-build:
	sudo rm -rf ceph/build
	@./scripts/cmake-radosgw.sh
	@./scripts/build-radosgw.sh

clean-wd:
	sudo rm -rf wd/*

probe-build:
	go build -o probe/bin/probe probe/main.go

probe-run:
	probe/bin/probe -s3gw-endpoint http://localhost:7480 -wbtd 300

probe-fr-up-run:
	probe/bin/probe -s3gw-endpoint http://localhost:7480 -wbtd 300 -collectAt frontend-up
