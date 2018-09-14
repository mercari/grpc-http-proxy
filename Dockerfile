FROM golang:1.11 AS build-env

ARG VERSION

WORKDIR /go/src/github.com/mercari/grpc-http-proxy

ADD . /go/src/github.com/mercari/grpc-http-proxy

RUN go get github.com/golang/dep/cmd/dep

RUN dep ensure -v
RUN CGO_ENABLED=0 GOOS=linux go install -v \
    -ldflags="-w -s" \
    -ldflags "-X main.version=${VERSION}" \
    github.com/mercari/grpc-http-proxy/cmd/proxy

FROM alpine:latest

COPY --from=build-env /go/bin/proxy /proxy
RUN chmod a+x /proxy

EXPOSE 3000
CMD ["/proxy"]
