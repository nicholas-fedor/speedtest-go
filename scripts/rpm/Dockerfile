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


FROM centos:7 AS packager

# CentOS 7 reached EOL June 2024; switch to vault mirrors
RUN sed -i \
    -e 's|mirrorlist|#mirrorlist|g' \
    -e 's|#baseurl=http://mirror.centos.org|baseurl=http://vault.centos.org|g' \
    /etc/yum.repos.d/CentOS-*.repo

RUN yum install -y rpm-build && yum clean all

ARG VERSION=dev
ARG RELEASE=1

WORKDIR /rpmbuild
RUN mkdir -p BUILD BUILDROOT RPMS SOURCES SPECS SRPMS

COPY --from=builder /app/speedtest-go BUILD/speedtest-go
COPY scripts/rpm/speedtest-go.spec SPECS/speedtest-go.spec

RUN rpmbuild \
    --define "_topdir /rpmbuild" \
    --define "version ${VERSION}" \
    --define "release ${RELEASE}" \
    -bb SPECS/speedtest-go.spec

RUN mkdir -p /packages && find /rpmbuild/RPMS -name "*.rpm" -exec cp {} /packages/ \;


FROM scratch AS export
COPY --from=packager /packages/ /
