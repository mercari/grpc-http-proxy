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
