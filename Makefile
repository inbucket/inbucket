SHELL = /bin/sh

SRC := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
PKGS := $(shell go list ./... | grep -v /vendor/)

.PHONY: all build clean fmt lint reflex simplify test

commands = client inbucket

all: clean test lint build

$(commands): %: cmd/% $(SRC)
	go build ./$<

clean:
	go clean $(PKGS)
	rm -f $(commands)
	rm -rf dist

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
	@echo "gofmt check..."
	@test -z "$(shell gofmt -l . | tee /dev/stderr)" || echo "[WARN] Fix formatting issues with 'make fmt'"
	@echo "golint check..."
	@golint -set_exit_status $(PKGS)
	@echo "go vet check..."
	@go vet $(PKGS)

reflex:
	reflex -r '\.go$$' -- sh -c 'echo; date; echo; go test ./... && echo ALL PASS'
