PKG := inbucket
SHELL := /bin/sh

SRC := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
PKGS := $$(go list ./... | grep -v /vendor/)

.PHONY: all build clean fmt install lint simplify test

all: test lint build

clean:
	go clean

deps:
	go get -t ./...

build: clean deps
	go build

install: build
	go install

test: clean deps
	go test -race ./...

fmt:
	@gofmt -l -w $(SRC)

simplify:
	@gofmt -s -l -w $(SRC)

lint:
	@test -z "$(shell gofmt -l . | tee /dev/stderr)" || echo "[WARN] Fix formatting issues with 'make fmt'"
	@golint -set_exit_status $${PKGS}
	@go vet $${PKGS}
