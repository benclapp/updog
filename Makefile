# Go parameters
VERSION=$(shell cat VERSION)
VERSION_MINOR=$(shell cat VERSION_MINOR)
GOCMD=go
GOBUILD=$(GOCMD) build
GOFLAGS=-ldflags "-X main.version=$(VERSION)"
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get

all: build-all docker
build: deps
		$(GOBUILD) $(GOFLAGS) -v
clean:
		$(GOCLEAN)
		rm -rf bin/*
		rm -rf updog-*
run:
		$(GORUN) $(GOFLAGS) updog.go
deps:
		$(GOGET) -d -v ./...
build-all: deps
		for OS in linux darwin windows ; do \
			env GOOS=$$OS GOARCH=amd64 $(GOBUILD) $(GOFLAGS) ; \
			mkdir updog-$(VERSION)-$$OS-amd64 ; \
			if [ -f updog ] ; then mv updog updog-$(VERSION)-$$OS-amd64 ; fi ; \
			if [ -f updog.exe ] ; then mv updog.exe updog-$(VERSION)-$$OS-amd64 ; fi ; \
			cp updog.yaml updog-$(VERSION)-$$OS-amd64 ; \
			tar -czf updog-$(VERSION)-$$OS-amd64.tar.gz updog-$(VERSION)-$$OS-amd64 ; \
			mv updog-$(VERSION)-$$OS-amd64.tar.gz bin/ ; \
			rm -rf updog-$(VERSION)-$$OS-amd64 ; \
		done

docker:
		docker build -t benclapp/updog:$(VERSION) -t benclapp/updog:$(VERSION_MINOR) -t benclapp/updog:latest .
		docker push benclapp/updog:$(VERSION)
		docker push benclapp/updog:$(VERSION_MINOR)
		docker push benclapp/updog:latest

