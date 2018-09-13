# grpc-http-proxy

[![CircleCI](https://circleci.com/gh/mercari/grpc-http-proxy.svg?style=shield&circle-token=2a2be18757cc9a28dc396a3c30277c98ed060d33)](https://circleci.com/gh/mercari/grpc-http-proxy)
[![codecov](https://codecov.io/gh/mercari/grpc-http-proxy/branch/master/graph/badge.svg?token=aTIypBc4JX)](https://codecov.io/gh/mercari/grpc-http-proxy)

`grpc-http-proxy` is a proxy which converts HTTP calls to gRPC calls with little configuration.
It is designed to run in a Kubernetes cluster, and uses the Kubernetes API to find in-cluster servers that provide the desired gRPC service.

## How to run
Use Helm to deploy to a Kubernetes cluster.
```console
$ cd helm/grpc-http-proxy
$ helm install --values values.yaml --name my-proxy
```

## Configuration
### Enable gRPC reflection
`grpc-http-proxy` uses the [gRPC server reflection protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md) to obtain information from the gRPC service on the supported methods and the format of messages.
Enable server reflection by following the instructions found [here](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#known-implementations).

### Environment variables
The following environment variables are used for configuration:
- `PORT`: Port that `grpc-http-proxy` listens on. Defaults to 3000.
- `TOKEN`: Access token for  `grpc-http-proxy`. If set, the access token will be required to be in `X-Access-Token` field of the request header. Defaults to empty, which means no token is required to access `grpc-http-proxy`.
- `LOG_LEVEL`: The log level can be `INFO`, `DEBUG`, or `ERROR`. Defaults to `INFO`.

`LOG_LEVEL` and `TOKEN` are configurable through the Helm chart's `values.yaml`

### Cluster side settings for Kubernetes API service discovery
The service discovery works by looking for Kubernetes Services with a specific annotation. In order to have it pick up the Kubernetes Service in front of your gRPC server, do the following.
#### 1. Add the `grpc-service` annotation
Put the `grpc-http-proxy.alpha.mercari.com/grpc-service` annotation on the Service. If your gRPC service's fully qualified name is `my.package.MyService`, add the annotation `grpc-http-proxy.alpha.mercari.com/grpc-service: my.package.MyService`.

```diff
  kind: Service
  apiVersion: v1
  metadata:
    name: my-service
    annotation:
+     grpc-http-proxy.alpha.mercari.com/grpc-service: my.package.MyService
```


#### 2. Name your port with a name which begins with `grpc`.
`grpc-http-proxy` will send requests to the port whose name begins with `grpc`. If there are multiple matching ports, the first one will be selected.
This step may be skipped if the port you intend to use is the only port exposed on the Kubernetes Service.

Examples of valid port names are `grpc` and `grpc-foo`, 

#### 3. [optional] Add the `grpc-service-version` annotation
If you intend to call multiple versions of your gRPC server through `grpc-http-proxy`, put the `grpc-http-proxy.alpha.mercari.com/grpc-service-version` annotation on the Service.
The version can be any string you like.

```diff
    annotation:
+     translator/backend-version: pr-42
```


## TODOs
A non-exhaustive list of additional features that could be desired:
- Ability to find services though a static configuration file.

Contributions are welcomed :)

## Committers
Tomoya TABUCHI ([@tomoyat1](https://github.com/tomoyat1))

## Contribution
Please read the CLA below carefully before submitting your contribution.

https://www.mercari.com/cla/

## LICENSE
Copyright 2018 Mercari, Inc.

Licensed under the MIT License.