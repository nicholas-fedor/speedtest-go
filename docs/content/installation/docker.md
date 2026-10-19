---
title: Docker
description: Pull speedtest-go's Docker image, choose a tag and platform, and run it with flags or a settings file
weight: 4
---

The image runs `speedtest-go` as its entrypoint and contains nothing else, not even a shell.

## Registries

The image is published to two registries:

- Docker Hub: `nickfedor/speedtest-go`
- GitHub Container Registry: `ghcr.io/nicholas-fedor/speedtest-go`

```bash
docker pull nickfedor/speedtest-go:latest
docker pull ghcr.io/nicholas-fedor/speedtest-go:latest
```

## Tags

| Tag | Image |
| --- | --- |
| `latest` | The newest release |
| `<major>.<minor>.<patch>` | That release |
| `<major>.<minor>` | The newest release in that minor series |
| `<major>` | The newest release in that major series |
| `nightly` | Built each night from the latest changes |

Each tag covers every platform. To pull a single platform, put its prefix in front of the tag, such as `arm64v8-latest` or `amd64-<major>.<minor>.<patch>`:

| Platform | Prefix |
| --- | --- |
| `linux/amd64` | `amd64` |
| `linux/386` | `i386` |
| `linux/arm/v6` | `armhf` |
| `linux/arm64` | `arm64v8` |
| `linux/riscv64` | `riscv64` |

## Run it

Flags go straight after the image name:

```bash
# Run a speed test with the default settings
docker run --rm nickfedor/speedtest-go:latest

# Run with a specific server
docker run --rm nickfedor/speedtest-go:latest --server 6691

# Run with multiple servers and JSON output
docker run --rm nickfedor/speedtest-go:latest --server 6691 --server 6087 --json

# Run with a custom location
docker run --rm nickfedor/speedtest-go:latest --location=60,-110
```

ICMP pings need raw sockets, which the image cannot open, so use `--ping-mode http` or `tcp` in a container.

## Settings

Settings also come from environment variables and a settings file, as described in [Configuration](/usage/configuration/).

The image runs as user 65534 with no home directory, so it has no default settings file. To use one:

1. Mount the file into the container. It must be readable by user 65534.
2. Name it with `SPEEDTEST_GO_CONFIG`.

```bash
# Set settings with environment variables
docker run --rm -e SPEEDTEST_GO_PING_MODE=tcp -e SPEEDTEST_GO_SERVER=6691,6087 nickfedor/speedtest-go:latest

# Read settings from a mounted file
docker run --rm -v "$PWD/config.yaml:/config.yaml:ro" -e SPEEDTEST_GO_CONFIG=/config.yaml nickfedor/speedtest-go:latest
```

## Docker Compose

The [Docker Compose example](https://github.com/nicholas-fedor/speedtest-go/tree/master/example/docker) runs the image with a commented settings file, a read-only file system, and no capabilities.
