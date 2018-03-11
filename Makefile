SHELL = /bin/sh

SRC := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
PKGS := $(shell go list ./... | grep -v /vendor/)

.PHONY: all build clean fmt lint simplify test

commands = client inbucket

all: clean test lint build

$(commands): %: cmd/%
	go build ./$<

clean:
	go clean $(PKGS)
	rm -f $(commands)

deps:
	go get -t ./...

build: $(commands)

test:
	go test -race ./...

fmt:
	@gofmt -l -w $(SRC)

simplify:
	@gofmt -s -l -w $(SRC)

lint:
	@test -z "$(shell gofmt -l . | tee /dev/stderr)" || echo "[WARN] Fix formatting issues with 'make fmt'"
	@golint -set_exit_status $(PKGS)
	@go vet $(PKGS)
