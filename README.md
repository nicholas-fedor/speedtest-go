<!-- markdownlint-disable MD033 MD041 -->
<div align="center">

<img src=".github/assets/speedtest-go.svg" alt="speedtest-go logo" width="128"/>

# speedtest-go

Test internet download and upload speeds with [speedtest.net](https://www.speedtest.net/)

[![GitHub Release](https://img.shields.io/github/v/release/nicholas-fedor/speedtest-go)](https://github.com/nicholas-fedor/speedtest-go/releases/latest) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/nicholas-fedor/speedtest-go) [![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

[Documentation](https://speedtest-go.nickfedor.com/) · [Getting Started](https://speedtest-go.nickfedor.com/getting-started/) · [CLI Reference](https://speedtest-go.nickfedor.com/usage/cli-reference/) · [Go Library](https://speedtest-go.nickfedor.com/go-library/)

<img src="docs/static/images/demo.gif" alt="A speed test against a San Jose server, showing the live progress bars, each result, and the summary panel" width="800"/>

</div>
<!-- markdownlint-enable MD033 MD041 -->

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Go library](#go-library)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## Install

On Linux and macOS, the install script downloads the latest release, verifies its checksum, and installs it:

```bash
tmp=$(mktemp)
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/scripts/install.sh -o "$tmp" && sh "$tmp"
```

| Other methods | Command or source |
| --- | --- |
| Go | `go install github.com/nicholas-fedor/speedtest-go/v2@latest` |
| Docker | `docker pull nickfedor/speedtest-go:latest` |
| Binaries for Linux, macOS, and Windows | [GitHub Releases](https://github.com/nicholas-fedor/speedtest-go/releases), as described in [Binaries](https://speedtest-go.nickfedor.com/installation/binaries/) |

See [Installation](https://speedtest-go.nickfedor.com/installation/) for the script's options, updating and uninstalling, the image tags, and Kubernetes.

## Usage

| To | Run |
| --- | --- |
| Test the server with the lowest latency | `speedtest-go` |
| List the servers near you | `speedtest-go list` |
| Test specific servers | `speedtest-go --server 6691 --server 6087` |
| Write the results as JSON | `speedtest-go --json` |
| Test over TCP | `speedtest-go --protocol tcp --ping-mode tcp` |

Every flag except `--help` can also be set with a `SPEEDTEST_GO_` environment variable, and every flag except `--help` and `--config` in a settings file. See [Configuration](https://speedtest-go.nickfedor.com/usage/configuration/).

## Go library

```bash
go get github.com/nicholas-fedor/speedtest-go/v2
```

```go
client, err := speedtest.New()
if err != nil {
    log.Fatal(err)
}

servers, err := client.FetchServers(ctx)
if err != nil {
    log.Fatal(err)
}

targets, err := servers.FindServer(nil)
if err != nil {
    log.Fatal(err)
}

download, err := client.Download(ctx, targets[0])
if err != nil {
    log.Fatal(err)
}

fmt.Println(download.Rate)
```

See [Go Library](https://speedtest-go.nickfedor.com/go-library/) for a complete example, and the [migration guide](https://speedtest-go.nickfedor.com/go-library/migration/) when upgrading from v1.

## Documentation

| Page | Covers |
| --- | --- |
| [Getting Started](https://speedtest-go.nickfedor.com/getting-started/) | A first test, its output, and choosing servers |
| [Installation](https://speedtest-go.nickfedor.com/installation/) | The install script, binaries for Linux, macOS, and Windows, Go, Docker, and Kubernetes |
| [Configuration](https://speedtest-go.nickfedor.com/usage/configuration/) | The settings file, environment variables, and every setting |
| [Usage](https://speedtest-go.nickfedor.com/usage/) | Servers, network paths, test protocols, output, and privacy |
| [Go Library](https://speedtest-go.nickfedor.com/go-library/) | Measuring from a Go program, and migrating from v1 |
| [Troubleshooting](https://speedtest-go.nickfedor.com/usage/troubleshooting/) | Exit codes, and what each error means |
| [CLI Reference](https://speedtest-go.nickfedor.com/usage/cli-reference/) | Every command and flag |
| [Experimental Results](https://speedtest-go.nickfedor.com/usage/results/) | Comparisons with speedtest.net and speedtest-cli |

## Contributing

Contributions are welcome. Please read the [contributing guide](CONTRIBUTING.md) before opening a pull request. It covers the development setup, the tasks, and the project's conventions.

To report a bug or request a feature, open an issue in the [issue tracker](https://github.com/nicholas-fedor/speedtest-go/issues).

Thanks to everyone who has [contributed](https://github.com/nicholas-fedor/speedtest-go/graphs/contributors).

## License

speedtest-go is a fork of [showwin/speedtest-go](https://github.com/showwin/speedtest-go) and is released under the [MIT License](LICENSE).
