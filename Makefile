PACKAGES := $(shell go list ./...)

.PHONY: build
build: dep
	@GO111MODULE=on go build -o build/proxy cmd/proxy/main.go

.PHONY: dep
dep:
	@GO111MODULE=on go mod vendor

.PHONY: test
test:
	@GO111MODULE=on go test -v -race ./...

.PHONY: coverage
coverage:
	@GO111MODULE=on go test -race -coverpkg=./... -coverprofile=coverage.txt ./...

.PHONY: reviewdog
reviewdog: devel-deps
	reviewdog -reporter="github-pr-review"

.PHONY: devel-deps
devel-deps:
	@GO111MODULE=on ./misc/scripts/install-devel-deps.sh
