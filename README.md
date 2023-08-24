# s3gw-ha

## Local setup

### Bootstrap

> **Before doing anything else**: ensure to execute the following command
> after the clone:

```shell
git submodule update --init --recursive
```

### Requirements

- Docker, Docker compose
- Helm
- k3d
- kubectl
- Go (1.20+)

### Build the s3gw backend image

Build the s3gw's image:

```shell
make s3gw-cmake
```

```shell
make s3gw-build
```

> **Be patient**: this will take long.

After the command completes successfully,
you will see the following images:

```shell
docker images
```

- `quay.io/s3gw/s3gw:{@TAG}`

Where `{@TAG}` is the evaluation of the following expression:

```bash
$(git describe --tags --always)
```

### Build the prober

```shell
make tidy
```

```shell
make probe-build
```

### Create the cluster

You create the `k3d-s3gw-ha` cluster with:

```shell
make cluster-start
```

> **WARNING**: the command updates your `.kube/config` with the credentials of
> the just created `k3d-s3gw-ha` cluster and sets its context as default.

### Delete the acceptance cluster

```shell
make cluster-delete
```

### Deploy the s3gw-ha/s3gw on the cluster

```shell
make deploy
```

### Undeploy the s3gw-ha/s3gw from the cluster

```shell
make undeploy
```

## License

Copyright (c) 2023 [SUSE, LLC](http://suse.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
