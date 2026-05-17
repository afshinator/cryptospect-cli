APP_NAME := cryptospect-cli
VERSION  := $(shell git describe --tags --dirty 2>/dev/null)
ifeq ($(VERSION),)
LDFLAGS  := -s -w
else
LDFLAGS  := -s -w \
	-X github.com/afshinator/cryptospect-cli/internal/version.Value=$(VERSION) \
	-X github.com/afshinator/cryptospect-cli/internal/version.tagged=true
endif

.PHONY: build lint fmt vet test clean release

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/cryptospect-cli

fmt:
	goimports -w cmd/ internal/
	gofumpt -w cmd/ internal/

lint:
	golangci-lint run ./...

vet:
	go vet ./...

test:
	GOMODCACHE=$(PWD)/.gomodcache GOCACHE=$(PWD)/.cache/go go test -race -cover ./...

clean:
	rm -rf bin/ dist/

release:
	goreleaser release --clean