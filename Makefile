APP_NAME?=$(shell basename $(PWD))
GCP_PROJECT=kvantic-one
REGISTRY?=eu.gcr.io/$(GCP_PROJECT)
IMAGE=$(REGISTRY)/$(APP_NAME)
VERSION?=$(shell scripts/get_version.sh)
GOCMD=CGO_ENABLED=0 GOOS=linux go
PKGS ?= ./...
DEFAULT_BIN_NAME=./cmd/unprotected-server
BINARIES := $(wildcard cmd/*/.)

dev-deps:
	GO111MODULE=on go get -u github.com/golangci/golangci-lint/cmd/golangci-lint@v1.16.0

lint:
	golangci-lint run --fix $(PKGS)

test: test_unit test_integration
	#pass

up:
	#docker-compose up -d

down:
	#docker-compose down -v

test_integration:
	go test $(PKGS) -count=1 -tags=integration

test_unit:
	go test $(PKGS) -count=1

run: build-all
	./$(DEFAULT_BIN_NAME)

.PHONY: build-all
build-all: $(BINARIES)

.PHONY: $(BINARIES)
$(BINARIES):
	$(GOCMD) build -ldflags "-X main.version=$(VERSION)" -o $(@D)-server ./$@

vendor: go.mod
	$(GOCMD) mod vendor

.PHONY: build_image
build_image: vendor
	@docker build --target build -t $(IMAGE)_build:latest .

.PHONY: image
image:
	@echo "Building $(IMAGE):$(VERSION)"
	@docker build --build-arg=VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

run-docker: image
	@docker run $(IMAGE):$(VERSION)

.PHONY: image
push: image
	@echo "Pushing $(IMAGE):$(VERSION)"
	@docker push $(IMAGE):$(VERSION)

clean:
	@$(GOCMD) clean
	@$(GOCMD) clean -modcache

distclean: clean
	@rm -rf vendor
