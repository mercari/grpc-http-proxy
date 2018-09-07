#!/bin/bash

set -e

go get -v -u github.com/golang/dep/cmd/dep
go get -v github.com/golang/mock/gomock
go install github.com/golang/mock/mockgen
go get -v -u github.com/golang/lint/golint
go get -v -u github.com/haya14busa/reviewdog/cmd/reviewdog
