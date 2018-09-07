PACKAGES := $(shell go list ./...)

.PHONY: build
build: dep generate
	@go build -o build/proxy cmd/proxy/main.go

.PHONY: dep
dep:
	dep version | go get -u github.com/golang/dep
	@dep ensure -v

.PHONY: generate
generate: devel-deps
	@go generate ./...

.PHONY: test
test: generate
	@go test -v -race ./...

.PHONY: coverage
coverage: generate
	@go test -race -coverpkg=./... -coverprofile=coverage.txt ./...

.PHONY: reviewdog
reviewdog: devel-deps
	reviewdog -reporter="github-pr-review"

.PHONY: devel-deps
devel-deps:
	@./misc/scripts/install-devel-deps.sh