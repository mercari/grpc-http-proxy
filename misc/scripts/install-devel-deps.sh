#!/bin/bash

set -e

go get -v -u github.com/golang/dep/cmd/dep
go get -v -u github.com/golang/lint/golint
go get -v -u github.com/haya14busa/goverage
go get -v -u github.com/haya14busa/reviewdog/cmd/reviewdog
