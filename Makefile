# Go parameters
VERSION=$(shell cat VERSION)
GOCMD=go
GOBUILD=$(GOCMD) build
GOFLAGS=-ldflags "-X main.version=$(VERSION)"
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=updog
BINARY_UNIX=updog_unix

all: build-all docker
build: deps
		$(GOBUILD) $(GOFLAGS) -v
clean:
		# $(GOCLEAN)
		rm -rf bin/*
		rm updog
run:
		$(GORUN) $(GOFLAGS) updog.go
deps:
		$(GOGET) -d -v ./...
build-all: deps
		env GOOS=linux GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o bin/updog-linux-amd64
		env GOOS=darwin GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o bin/updog-darwin-amd64
		env GOOS=windows GOARCH=amd64 $(GOBUILD) $(GOFLAGS) -o bin/updog-windows-amd64.exe

docker:
		docker build -t benclapp/updog:$(VERSION) .