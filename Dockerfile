# syntax=docker/dockerfile:1@sha256:2780b5c3bab67f1f76c781860de469442999ed1a0d7992a5efdf2cffc0e3d769

# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166 AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN GOOS=$(echo $TARGETPLATFORM | cut -d/ -f1) \
    GOARCH=$(echo $TARGETPLATFORM | cut -d/ -f2) \
    go build -o speedtest-go

# Final stage
FROM alpine:latest@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

# Install bash
RUN apk add --no-cache bash

# Create non-root user
RUN adduser -D -h /home/speedtest speedtest

WORKDIR /home/speedtest

# Copy the binary from builder
COPY --from=builder /app/speedtest-go /usr/local/bin/

# Switch to non-root user
USER speedtest

# Set default shell
SHELL ["/bin/bash", "-c"]

# Set the entrypoint to bash, we do this rather than using the speedtest command directly
# such that you can also use this container in an interactive way to run speedtests.
# see the README for more info and examples.
CMD ["/bin/bash"]
