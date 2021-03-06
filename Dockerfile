# We need the cp command to copy files to shared volumes
ARG BASE_IMAGE=busybox:1.32

FROM golang:1.13 as builder
ARG LDFLAGS

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY monitoring/ monitoring/
COPY misc/ misc/

# Build
RUN echo "LDFLAGS=${LDFLAGS}"
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o primehub-monitoring-agent -ldflags "${LDFLAGS}" main.go
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o gonvml-example misc/example.go

FROM ${BASE_IMAGE}
COPY --from=builder /workspace/primehub-monitoring-agent /
COPY --from=builder /workspace/gonvml-example /

