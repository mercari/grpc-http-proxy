FROM golang:1.13 AS build-env

ARG VERSION

ENV GO111MODULE=on

WORKDIR /go/src/github.com/mercari/grpc-http-proxy

ADD . /go/src/github.com/mercari/grpc-http-proxy

RUN CGO_ENABLED=0 GOOS=linux go install -v \
    -ldflags="-w -s" \
    -ldflags "-X main.version=${VERSION}" \
    github.com/mercari/grpc-http-proxy/cmd/proxy

FROM alpine:3.10.2

COPY --from=build-env /go/bin/proxy /proxy
RUN chmod a+x /proxy

EXPOSE 3000
CMD ["/proxy"]
