#!/bin/bash

set -e

go get -v golang.org/x/lint/golint@latest
go get -v github.com/reviewdog/reviewdog/cmd/reviewdog@latest
