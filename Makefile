# Image URL to use all building/pushing image targets
IMG ?= monitoring:latest
VERSION ?=
LDFLAGS =

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

LDFLAGS += -X primehub-monitoring-agent/monitoring.tagVersion=${VERSION}
LDFLAGS += -X primehub-monitoring-agent/monitoring.gitCommit=${GIT_COMMIT}
LDFLAGS += -X primehub-monitoring-agent/monitoring.gitTreeState=${GIT_DIRTY}
LDFLAGS += $(EXT_LDFLAGS)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: agent

# Run tests
test: fmt vet
	go test ./... -coverprofile cover.out

# Build primehub-monitoring-agent binary
agent: fmt vet
	go build -o primehub-monitoring-agent -ldflags '$(LDFLAGS)' main.go

# Run usage-agnet
run: fmt vet
	go run ./main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build release image
docker-build: test
	docker build --build-arg LDFLAGS='${LDFLAGS}' -t ${IMG} .

# Build dev image, the agent could be running in the container
dev-docker-build: test
	docker build --build-arg BASE_IMAGE=ubuntu:18.04 --build-arg LDFLAGS='${LDFLAGS}' -t ${IMG} .