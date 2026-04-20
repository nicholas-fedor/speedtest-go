FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w \
      -X github.com/nicholas-fedor/speedtest-go/speedtest.version=${VERSION} \
      -X github.com/nicholas-fedor/speedtest-go/internal/output.commit=${COMMIT} \
      -X github.com/nicholas-fedor/speedtest-go/internal/output.date=${DATE}" \
    -o speedtest-go .


FROM ubuntu:22.04 AS packager

ARG VERSION=dev
ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    dpkg-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

RUN mkdir -p pkg/usr/bin pkg/DEBIAN

COPY --from=builder /app/speedtest-go pkg/usr/bin/speedtest-go
RUN chmod 755 pkg/usr/bin/speedtest-go

COPY scripts/deb/control pkg/DEBIAN/control
RUN sed -i "s/VERSION_PLACEHOLDER/${VERSION}/" pkg/DEBIAN/control

RUN dpkg-deb --build pkg "speedtest-go_${VERSION}_amd64.deb"

RUN mkdir -p /packages && cp "speedtest-go_${VERSION}_amd64.deb" /packages/


FROM scratch AS export
COPY --from=packager /packages/ /
