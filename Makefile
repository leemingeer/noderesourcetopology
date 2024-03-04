.PHONY: all build build-%
GO_CMD ?= go
VERSION := v1#$(shell git describe --tags --dirty --always)
BUILD_FLAGS = -tags osusergo,netgo \
              -ldflags "-s -w -extldflags=-static -X github.com/leemingeer/noderesourcetopology/pkg/version.version=$(VERSION)"

BUILD_BINARIES := topology-updater
build-%:
	$(GO_CMD) build -v -o bin/ $(BUILD_FLAGS) ./cmd/$*
build:	$(foreach bin, $(BUILD_BINARIES), build-$(bin))


IMAGE_REGISTRY ?= localhost
IMAGE_NAME := topology-updater
IMAGE_REPO := $(IMAGE_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG_NAME ?= $(VERSION)
IMAGE_TAG := $(IMAGE_REPO):$(IMAGE_TAG_NAME)
IMAGE_EXTRA_TAG_NAMES ?=
IMAGE_EXTRA_TAGS := $(foreach tag,$(IMAGE_EXTRA_TAG_NAMES),$(IMAGE_REPO):$(tag))

BASE_IMAGE_FULL ?= debian:bookworm-slim
BUILDER_IMAGE ?= golang:1.21-bookworm
BASE_IMAGE_MINIMAL ?= scratch

IMAGE_BUILD_CMD ?= docker build
IMAGE_BUILD_ARGS = --build-arg VERSION=$(VERSION) \
                --build-arg HOSTMOUNT_PREFIX=$(CONTAINER_HOSTMOUNT_PREFIX) \
                --build-arg BUILDER_IMAGE=$(BUILDER_IMAGE) \
                --build-arg BASE_IMAGE_FULL=$(BASE_IMAGE_FULL) \
                --build-arg BASE_IMAGE_MINIMAL=$(BASE_IMAGE_MINIMAL)

IMAGE_BUILD_ARGS_FULL = --target full \
                        -t $(IMAGE_TAG)-full \
                        $(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)-full) \
                        $(IMAGE_BUILD_EXTRA_OPTS) ./

IMAGE_BUILD_ARGS_MINIMAL = --target minimal \
                           -t $(IMAGE_TAG) \
                           -t $(IMAGE_TAG)-minimal \
                           $(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag) -t $(tag)-minimal) \
                           $(IMAGE_BUILD_EXTRA_OPTS) ./

image:
	$(IMAGE_BUILD_CMD) $(IMAGE_BUILD_ARGS) $(IMAGE_BUILD_ARGS_FULL)
	$(IMAGE_BUILD_CMD) $(IMAGE_BUILD_ARGS) $(IMAGE_BUILD_ARGS_MINIMAL)

install-%:
	$(GO_CMD) install -v $(BUILD_FLAGS) ./cmd/$*

install:	$(foreach bin, $(BUILD_BINARIES), install-$(bin))