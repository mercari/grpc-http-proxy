PACKAGES := $(shell go list ./...)

.PHONY: build
build: dep
	@go build -o build/proxy cmd/proxy/main.go

.PHONY: dep
dep:
	dep version | go get -u github.com/golang/dep
	@dep ensure -v

.PHONY: test
test:
	@go test -v -race ./...

.PHONY: coverage
coverage: devel-deps
	goverage -v -covermode=atomic -coverprofile=coverage.txt $(PACKAGES)

.PHONY: reviewdog
reviewdog: devel-deps
	reviewdog -reporter="github-pr-review"

.PHONY: devel-deps
devel-deps:
	@./misc/scripts/install-devel-deps.sh