.PHONY: build test vet
build:
	go build -o bin/xnode-agent ./cmd/xnode-agent

test:
	go test ./...

vet:
	go vet ./...
