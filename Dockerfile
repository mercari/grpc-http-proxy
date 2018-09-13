FROM golang:1.11-alpine AS build-env

WORKDIR /go/src/github.com/mercari/grpc-http-proxy

ADD . /go/src/github.com/mercari/grpc-http-proxy
RUN apk --update add curl git make
RUN make build

FROM alpine:latest

COPY --from=build-env /go/src/github.com/mercari/grpc-http-proxy/build/proxy /proxy
RUN chmod a+x /proxy

EXPOSE 3000
CMD ["/proxy"]
