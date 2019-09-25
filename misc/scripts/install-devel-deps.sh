#!/bin/bash

set -e

go get -v -u golang.org/x/lint
go get -v -u github.com/reviewdog/reviewdog/cmd/reviewdog
