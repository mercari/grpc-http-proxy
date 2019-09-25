PACKAGES := $(shell go list ./...)

.PHONY: build
build:
	@go build -o build/proxy cmd/proxy/main.go

.PHONY: test
test:
	@go test -v -race ./...

.PHONY: coverage
coverage:
	@go test -race -coverpkg=./... -coverprofile=coverage.txt ./...

.PHONY: reviewdog
reviewdog: devel-deps
	reviewdog -reporter="github-pr-review"

.PHONY: devel-deps
devel-deps:
	@./misc/scripts/install-devel-deps.sh