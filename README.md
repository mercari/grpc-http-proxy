# grpc-http-proxy

[![CircleCI](https://circleci.com/gh/mercari/grpc-http-proxy.svg?style=shield&circle-token=2a2be18757cc9a28dc396a3c30277c98ed060d33)](https://circleci.com/gh/mercari/grpc-http-proxy)
[![codecov](https://codecov.io/gh/mercari/grpc-http-proxy/branch/master/graph/badge.svg?token=aTIypBc4JX)](https://codecov.io/gh/mercari/grpc-http-proxy)


**:warning: This is not production ready**

grpc-http-proxy is a reverse proxy which converts JSON HTTP requests to gRPC calls without much configuration.
It is designed to run in a Kubernetes cluster, and uses the Kubernetes API to find in-cluster servers that provide the desired gRPC service using custom Kubernetes annotations.

![image](https://user-images.githubusercontent.com/1614811/45482670-a6b3bd80-b789-11e8-9243-70dac1a7fd41.png)

## Background
Although existing solutions, such as [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway),  generate reverse proxies that convert JSON HTTP requests to gRPC calls exist, they require the following to work:
- Custom annotations to the gRPC service definitions must be manually be added to define the HTTP endpoint to gRPC method mappings.
- The reverse proxy must be generated for each gRPC service.

As gRPC service definitions get larger, and more services are created, this can get unmanageable quickly.

grpc-http-proxy was created to be a single reverse proxy that works with all gRPC services, and without all the manual mapping. This enables gRPC services to be accessed through HTTP requests with less hassle than before.

## How it works
![image](https://user-images.githubusercontent.com/1614811/45482621-8e43a300-b789-11e8-96c9-dcba18f30aed.png)

A request to the grpc-http-proxy's endpoint `/v1/<service>/<method>` will make the proxy call the `<method>` method of the `<service>` gRPC service.

Given the service name and method name are known, the gRPC call is made in the following steps:
1. The [gRPC server reflection protocol](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md) is used to obtain information from the gRPC service on the supported methods and the format of messages. 
2. The JSON request body is converted, using the information obtained above, to the Protobuf message by following the [Proto3 to JSON mapping specification](https://developers.google.com/protocol-buffers/docs/proto3#json).
3. The gRPC call is made to the upstream service.
4. The response is converted to JSON, and returned to the caller

Metadata is passed to the upstream if it is put in the HTTP request's header, with the key prefixed with `Grpc-Metadata-`.

Also, grpc-http-proxy itself can be configured with an access token. If so, only requests with the specified access token in the `X-Access-Token` header are handled.

Mappings between gRPC service names and the upstream Kubernetes Services are defined by a custom annotation in the Kubernetes Service resource.
The Kubernetes API will be listened upon and the mapping will be kept up to date as Services with annotations are created, deleted, or updated.

## Installation
Use Helm to deploy to a Kubernetes cluster.

```console
$ git clone https://github.com/mercari/grpc-http-proxy && cd ./grpc-http-proxy
$ helm install --name grpc-http-proxy helm/grpc-http-proxy --namespace kube-system
```

This will deploy grpc-http-proxy without an access token to the `kube-system` namespace . To specify one, set the `accessToken` value as follows:

```console
$ helm install --set accessToken SUPER_SECRET --name grpc-http-proxy helm/grpc-http-proxy --namespace kube-system
```

Also, there is a `values.yaml` file for more in-depth configuration.

*note: RBAC is currently not supported by the Helm chart, so the default pod ServiceAccount should have access to all services within the cluster.*

## Configuration
After installing grpc-http-proxy, some configuration is required to make it find your services.

### Enable gRPC reflection
Enable server reflection in your gRPC servers by following the instructions found [here](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md#known-implementations).

### Cluster side settings for Kubernetes API service discovery
The service discovery works by looking for Kubernetes Services with a specific annotation. In order to have it pick up the Kubernetes Service in front of your gRPC server, do the following.
#### 1. Add the `grpc-service` annotation
Put the `grpc-http-proxy.alpha.mercari.com/grpc-service` annotation on the Service. If your gRPC service's fully qualified name is `my.package.MyService`, add the annotation `grpc-http-proxy.alpha.mercari.com/grpc-service: my.package.MyService`.

```diff
  kind: Service
  apiVersion: v1
  metadata:
    name: my-service
    annotations:
+     grpc-http-proxy.alpha.mercari.com/grpc-service: my.package.MyService
```

If your gRPC server exports multiple services, specify them in a list delimited by a comma (`,`).

```diff
  kind: Service
  apiVersion: v1
  metadata:
    name: my-service
    annotation:
+     grpc-http-proxy.alpha.mercari.com/grpc-service: my.package.MyService,my.anotherpackage.OtherService
```

#### 2. Name your port with a name which begins with `grpc`.
grpc-http-proxy will send requests to the port whose name begins with `grpc`. If there are multiple matching ports, the first one will be selected.
This step may be skipped if the port you intend to use is the only port exposed on the Kubernetes Service.

```diff
  kind: Service
  apiVersion: v1
  metadata:
    name: my-service
    annotations:
      grpc-http-proxy.alpha.mercari.com/grpc-service: my.package.MyService
  spec:
    ports:
-   - name: foo
+   - name: grpc-foo
      port: 5000
      protocol: TCP
      targetPort: 5000
```

#### 3. [optional] Add the `grpc-service-version` annotation
If you intend to call multiple versions of your gRPC server through grpc-http-proxy, put the `grpc-http-proxy.alpha.mercari.com/grpc-service-version` annotation on the Service.
The version can be any string you like.

```diff
    annotations:
+     translator/backend-version: pr-42
```

## Examples
In the following examples, grpc-http-proxy is running at `grpc-http-proxy.example.com`, and have the access token set to `foo`.
The gRPC service `Echo` is called, which is defined by the following `.proto` file:

```proto
syntax = "proto3";

package com.example;

service Echo {
    rpc Say(EchoMessage) returns (EchoMessage) {};
}

message EchoMessage {
    string message_body = 1;
}
```

### Single version service
The Kubernetes Service manifest will look like this:

```yaml
kind: Service
apiVersion: v1
metadata:
  name: echo-service
  annotations:
    grpc-http-proxy.alpha.mercari.com/grpc-service: com.example.Echo
spec:
  ports:
  - name: grpc-echo
    port: 5000
    protocol: TCP
    targetPort: 5000
```

The request to call `Say` through grpc-http-proxy, and its response would be:

```console
$ curl -H'X-Access-Token: foo' -XPOST -d'{"message_body":"Hello, World!"}' grpc-http-proxy.example.com/v1/com.example.Echo/Say
{"message_body":"Hello, World!"}
```

## Passing metadata to the gRPC service
To pass metadata with the key `somekey` to `Echo` service, add the metadata to the HTTP request like below:

```console
$ curl -H'X-Access-Token: foo' -H'Grpc-Metadata-somekey: value' -XPOST -d'{"message_body":"Hello, World!"}' grpc-http-proxy.example.com/v1/com.example.Echo/Say
{"message_body":"Hello, World!"}
```

### Multiple versions of a service
Let's say that you have a newer version of the `Echo` server in the same cluster that you would like to call . The Service manifest for the newer server should specify the version with an annotation, and would look like this:

```yaml
kind: Service
apiVersion: v1
metadata:
  name: newer-echo-service
  annotations:
    grpc-http-proxy.alpha.mercari.com/grpc-service: com.example.Echo
    grpc-http-proxy.alpha.mercari.com/grpc-service-version: newer-version
spec:
  ports:
  - name: grpc-echo
    port: 5000
    protocol: TCP
    targetPort: 5000
```

In order to choose the new version, the version name specified in the annotation should be added as a query parameter.
The request to call `Say` on the newer server, and its response would be:

```console
$ curl -H'X-Access-Token: foo' -XPOST -d'{"message_body":"Hello, World!"}' grpc-http-proxy.example.com/v1/com.example.Echo/Say?version=newer-version
{"message_body":"Hello, World!"}
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