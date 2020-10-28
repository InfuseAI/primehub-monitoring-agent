# Image URL to use all building/pushing image targets
IMG ?= monitoring:latest

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
	go build -o primehub-monitoring-agent main.go

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
	docker build . -t ${IMG}

# Build dev image, the agent could be running in the container
dev-docker-build: test
	docker build . -t ${IMG} --build-arg BASE_IMAGE=ubuntu:18.04