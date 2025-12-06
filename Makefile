.PHONY: tools lint test

MODULE := $(shell go list -m)

tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.1
	go install go.uber.org/nilaway/cmd/nilaway@latest

lint:
	go vet ./...
	golangci-lint run
	nilaway -include-pkgs="$(MODULE)" ./...

test:
	go test -v -race ./...
