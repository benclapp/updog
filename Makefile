# Go parameters
VERSION=$(shell cat VERSION)
VERSION_MINOR=$(shell cat VERSION_MINOR)
GOCMD=go
GOBUILD=$(GOCMD) build
GOFLAGS=-ldflags "-X main.version=$(VERSION)"
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get

all: help


release: ## Build binaries for Linux/Darwin/Windows, build and pushes docker images
	build-all 
	docker
	docker-push

build: ## build updog
		$(GOBUILD) $(GOFLAGS) -v

clean: ## go clean, and removes bin directory and temp files
		$(GOCLEAN)
		rm -rf bin
		rm -rf updog-*

run: ## Run's updog with some build flags
		$(GORUN) $(GOFLAGS) updog.go

build-all: ## Build updog for Linux/Darwin/Windows, compresses to .tar.gz
		mkdir -p bin
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

docker: ## Build docker image with 'x.y', 'x.y.z', and 'latest' tags
		docker build --pull -t benclapp/updog:$(VERSION) -t benclapp/updog:$(VERSION_MINOR) -t benclapp/updog:latest .

docker-push: ## Pushes previously build docker images. Only Ben can run this
		docker push benclapp/updog:$(VERSION)
		docker push benclapp/updog:$(VERSION_MINOR)
		docker push benclapp/updog:latest

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
