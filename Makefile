APP_NAME := cryptospect-cli
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X main.version=$(VERSION)

.PHONY: build lint fmt vet test clean release

build:
	GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod-cache CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/cryptospect-cli

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

test:
	go test -race -cover ./...

clean:
	rm -rf bin/ dist/

release:
	goreleaser release --clean