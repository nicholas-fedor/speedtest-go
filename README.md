<!-- markdownlint-disable -->
<div align="center">

# speedtest-go

<img src=".github/assets/speedtest-go.svg" alt="speedtest-go logo" width="128"/>
<!-- markdownlint-enable -->
</div>

This is a fork of [showwin/speedtest-go](https://github.com/showwin/speedtest-go) that has been updated to use the latest Go version and dependencies.

The project includes both a CLI and a public API that enables internet speed testing  using [speedtest.net](http://www.speedtest.net/).

## CLI

### Installation

#### Linux and macOS

The install script downloads a release from [GitHub Releases](https://github.com/nicholas-fedor/speedtest-go/releases), verifies its checksum, and installs it:

```bash
tmp=$(mktemp)
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/scripts/install.sh -o "$tmp" && sh "$tmp"
```

Run `sh "$tmp" update` to update to the latest release, and `sh "$tmp" uninstall` to remove it.

On Debian, Fedora, Alpine, Arch Linux, and their derivatives, the script installs the native package when it runs as root or can use `sudo`. Otherwise, it extracts the release archive into `$HOME/go/bin`. Either way, it verifies the signature of the checksums when [cosign](https://github.com/sigstore/cosign) is installed.

These environment variables change what the script installs:

| Variable  | Description                                       | Default                                                          |
| --------- | ------------------------------------------------- | ---------------------------------------------------------------- |
| `VERSION` | Release tag to install, such as `v2.0.0`          | The latest release                                               |
| `PREFIX`  | Directory for archive installs                    | `$HOME/go/bin`, or the directory of the previous archive install |
| `METHOD`  | `auto`, `package`, or `archive`                   | `auto`                                                           |

For example, to install the archive into `~/.local/bin`:

```bash
METHOD=archive PREFIX="$HOME/.local/bin" sh "$tmp"
```

Releases before v2.0.0 publish their packages under different names, so the script installs them from the archive. `METHOD=auto` falls back to the archive, and `METHOD=package` fails.

#### Windows

Download the `.zip` for your architecture from [GitHub Releases](https://github.com/nicholas-fedor/speedtest-go/releases) and extract `speedtest-go.exe`.

#### Docker Image

The image is published to two registries:

- Docker Hub: `nickfedor/speedtest-go`
- GitHub Container Registry: `ghcr.io/nicholas-fedor/speedtest-go`

```bash
docker pull nickfedor/speedtest-go:latest
docker pull ghcr.io/nicholas-fedor/speedtest-go:latest
```

Tags:

| Tag | Image |
| --- | --- |
| `latest` | The newest release |
| `1.8.4` | That release |
| `1.8` | The newest `1.8.x` release |
| `1` | The newest `1.x.x` release |
| `nightly` | Built each night from the latest changes |

Each tag covers every platform. To pull a single platform, put its prefix in front of the tag, such as `arm64v8-latest` or `amd64-1.8.4`:

| Platform | Prefix |
| --- | --- |
| `linux/amd64` | `amd64` |
| `linux/386` | `i386` |
| `linux/arm/v6` | `armhf` |
| `linux/arm64` | `arm64v8` |
| `linux/riscv64` | `riscv64` |

#### Running the Container

##### Docker

The image runs `speedtest-go` as its entrypoint and contains nothing else, not even a shell, so flags go straight after the image name.

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

Settings also come from environment variables and a settings file, as described in [Settings file and environment variables](#settings-file-and-environment-variables). The image runs as user 65534 with no home directory, so mount a settings file and name it with `SPEEDTEST_GO_CONFIG`. The file must be readable by that user.

```bash
# Set settings with environment variables
docker run --rm -e SPEEDTEST_GO_PING_MODE=tcp -e SPEEDTEST_GO_SERVER=6691,6087 nickfedor/speedtest-go:latest

# Read settings from a mounted file
docker run --rm -v "$PWD/config.yaml:/config.yaml:ro" -e SPEEDTEST_GO_CONFIG=/config.yaml nickfedor/speedtest-go:latest
```

The [Docker example](example/docker) runs the image with Docker Compose and a commented settings file.

##### Kubernetes

The image runs in Kubernetes as a pod or a CronJob, with its flags in `args`, since its entrypoint is `speedtest-go`. The [Kubernetes example](example/kubernetes) runs it both ways, with its settings in a ConfigMap and a hardened security context.

### Usage

```bash
speedtest-go --help
A command-line tool to test internet download and upload speeds using speedtest.net servers.

Usage:
  speedtest-go [flags]
  speedtest-go [command]

Available Commands:
  cities      List predefined city labels
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        List available speedtest servers
  version     Print version information

Flags:
      --base-url string     Set the speedtest API base URL (default https://www.speedtest.net).
      --city string         Change the location with a predefined city label.
      --config string       settings file (default speedtest-go/config.yaml in the user config directory)
      --custom-url string   Specify the url of the server instead of fetching from speedtest.net.
      --debug               Enable debug mode.
      --dl-size int         Set the download block size, from 350 to 4000 (default 1000).
      --dns-bind-source     DNS request binding source (experimental).
      --dscp int            Mark connections with a DSCP code point from 0 to 63 (Linux, macOS and BSD).
      --filter-cc strings   Only use servers in these countries (two-letter codes, or auto for your own).
  -h, --help                help for speedtest-go
      --json                Output results in json format.
      --jsonl               Output results in jsonl format (one json object per line).
      --location string     Change the location with a precise coordinate (format: lat,lon).
      --mask-isp            Mask your IP address, coordinates, and distances to servers in all output.
  -m, --multi               Enable multi-server mode.
      --no-download         Disable download test.
      --no-upload           Disable upload test.
      --ping-mode string    Select a method for Ping (support icmp/tcp/http). (default "http")
      --protocol string     Select the download and upload protocol (http or tcp). (default "http")
      --proxy string        Set a proxy(http[s] or socks) for the speedtest.
      --saving-mode         Test with few resources, though low accuracy (especially > 30Mbps).
      --search string       Fuzzy search servers by a keyword.
  -s, --server ints         Select server id to run speedtest.
      --source string       Send from a source IP address or network interface name.
  -t, --thread int          Set the number of concurrent connections.
      --ua string           Set the user-agent header for the speedtest.
      --ul-size int         Set the upload block size in kB, from 100 to 4000 (default 1000, or 800 with --multi).
  -u, --unit string         Set human-readable and auto-scaled rate units for output (options: decimal-bits/decimal-bytes/binary-bits/binary-bytes).
      --unix                Output results in unix like format.

Use "speedtest-go [command] --help" for more information about a command.
```

#### Settings file and environment variables

Every flag except `--help` can also be set with an environment variable, and every flag except `--help` and `--config` in a settings file. A flag on the command line takes precedence over the variable, the variable over the settings file, and the settings file over the default.

The settings file is YAML at `$XDG_CONFIG_HOME/speedtest-go/config.yaml`, which is `~/.config/speedtest-go/config.yaml` on Linux, `~/Library/Application Support/speedtest-go/config.yaml` on macOS, and `%LOCALAPPDATA%\speedtest-go\config.yaml` on Windows. Its keys are the flag names in camelCase, such as `pingMode` for `--ping-mode`, `dlSize` for `--dl-size`, and `filterCC` for `--filter-cc`. A missing settings file means the defaults. `--config` or `SPEEDTEST_GO_CONFIG` names another file, which must exist. An unknown key, a repeated key, or a value of the wrong type is an error, so a typo never passes silently.

```yaml
pingMode: tcp
unit: binary-bits
server: [6691, 6087]
filterCC: [gb, ie]
maskISP: true
```

The environment variable of a flag is `SPEEDTEST_GO_` followed by the flag name in upper case, with hyphens as underscores. A list holds its entries separated by commas.

```bash
SPEEDTEST_GO_PING_MODE=tcp SPEEDTEST_GO_DNS_BIND_SOURCE=true speedtest-go
SPEEDTEST_GO_SERVER=6691,6087 speedtest-go
```

#### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | The command succeeded. |
| `1` | The command failed, such as a network error or a `--server` ID that speedtest.net does not list. Errors go to stderr. |
| `2` | A setting is invalid, such as an unknown `--ping-mode` or `--unit`, a negative `--thread`, more than one of `--json`, `--jsonl` and `--unix`, or a settings file that cannot be read or decoded. |
| `130` | The run was interrupted with Ctrl-C, and no report was written. A second Ctrl-C ends a run that does not stop. |

#### Version

```bash
speedtest-go version
# speedtest-go v1.8.0 (1559e47, 2026-05-02) go1.26.5 linux/amd64
```

#### Test Internet Speed

Simply use `speedtest-go` command. The closest server is selected by default. Use the `-m` flag to enable multi-measurement mode (recommended)

In a terminal, each running test shows its live progress, and each result replaces it as a line. A download or upload also shows a progress bar and an estimate of the time left, which tracks when the test expects its rate to settle, and each server ends with a panel of its headline results. When the output is not a terminal, such as a pipe or a file, or with `--debug`, the results are plain lines without the progress, as with `--unix`.

```bash
## unix like format output
# speedtest-go --unix
speedtest-go

    speedtest-go v1.8.0 (1559e47, 2026-05-02)

✓ ISP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
✓ Found 20 Public Servers

✓ Test Server: [6691] 9.03km Shizuoka (Japan) by sudosan
✓ Latency: 4.452963ms Jitter: 41.271µs Min: 4.395179ms Max: 4.517576ms
✓ Packet Loss Analyzer: Running in background (<= 30 Secs)
✓ Download: 115.52 Mbps (Used: 135.75MB) (Latency: 4ms Jitter: 0ms Min: 4ms Max: 4ms)
✓ Upload: 4.02 Mbps (Used: 6.85MB) (Latency: 4ms Jitter: 1ms Min: 3ms Max: 8ms)
✓ Packet Loss: 8.82% (Sent: 217/Dup: 0/Max: 237)
```

#### Test with Other Servers

If you want to select other servers to test, you can see the available server list.

```bash
$ speedtest-go list
Testing From IP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
[6691]     9.03km   32.3365ms  Shizuoka (Japan) by sudosan
[6087]   120.55km   51.7453ms  Fussa-shi (Japan) by Allied Telesis Capital Corporation
[6508]   125.44km   54.6683ms  Yokohama (Japan) by at2wn
[6424]   148.23km   61.4724ms  Tokyo (Japan) by Cordeos Corp.
...
```

and select them by id.

```bash
$ speedtest-go --server 6691 --server 6087

    speedtest-go v1.8.0 (1559e47, 2026-05-02)

✓ ISP: 124.27.199.165 (Fujitsu) [34.9769, 138.3831]
✓ Found 2 Specified Public Server(s)

✓ Test Server: [6691] 9.03km Shizuoka (Japan) by sudosan
✓ Latency: 21.424ms Jitter: 1.644ms Min: 19.142ms Max: 23.926ms
✓ Packet Loss Analyzer: Running in background (<= 30 Sec)
✓ Download: 65.82Mbps (Used: 75.48MB) (Latency: 22ms Jitter: 2ms Min: 17ms Max: 24ms)
✓ Upload: 27.00Mbps (Used: 36.33MB) (Latency: 23ms Jitter: 2ms Min: 18ms Max: 25ms)
✓ Packet Loss: 0.00% (Sent: 321/Dup: 0/Max: 320)

✓ Test Server: [6087] 120.55km Fussa-shi (Japan) by Allied Telesis Capital Corporation
✓ Latency: 38.694699ms Jitter: 2.724ms Min: 36.443ms Max: 39.953ms
✓ Packet Loss Analyzer: Running in background (<= 30 Sec)
✓ Download: 72.24Mbps (Used: 83.72MB) (Latency: 37ms Jitter: 3ms Min: 36ms Max: 40ms)
✓ Upload: 29.56Mbps (Used: 47.64MB) (Latency: 38ms Jitter: 3ms Min: 37ms Max: 41ms)
✓ Packet Loss: 0.00% (Sent: 343/Dup: 0/Max: 342)
```

An ID that speedtest.net does not list is an error, and the error names every such ID. With `-m`, the servers selected with `--server` or `--custom-url` are the pool that each multi-server test spreads its connections across.

#### Test through a specific interface

On a host with several network connections, `--source` picks the one to test. It takes an IP address of this host or a network interface name.

```bash
speedtest-go --source 192.168.1.101
speedtest-go --source eth0
```

On Linux, an interface name binds the API, ping, download, upload and packet loss connections to the interface with `SO_BINDTODEVICE`, so their traffic leaves through it whatever the routing table prefers, and each one uses the interface's address of the matching IP version. Linux kernels before 5.7 allow that binding only with the `CAP_NET_RAW` capability. On other platforms, those connections use the interface's IPv4 address, or its first IPv6 address that is not link-local, as their source.

DNS lookups use the system resolver unless `--dns-bind-source` is set, which sends them from the source as well. An interface without an IPv4 address or an IPv6 address that is not link-local is an invalid setting, as is any value that is neither an IP address nor an interface.

#### Mark traffic with DSCP

On a network that applies quality of service, `--dscp` marks the API, ping, download, upload and packet loss connections with a Differentiated Services code point from 0 to 63, so the test measures the treatment that class of traffic receives. For example, 46 is Expedited Forwarding, often used for voice, and 34 is Assured Forwarding 41.

```bash
speedtest-go --dscp 46
```

The code point is set on IPv4 and IPv6 sockets on Linux, macOS and the BSDs, and on DNS sockets too when `--dns-bind-source` is set. Other platforms cannot set it, so a nonzero value there is an invalid setting, as is a value outside 0 to 63. A connection that cannot be marked fails rather than running unmarked.

#### Test over TCP

By default, downloads and uploads use HTTP. With `--protocol tcp`, they use the plain TCP protocol that speedtest.net servers also serve, normally on port 8080, which avoids HTTP overhead.

```bash
speedtest-go --protocol tcp
```

The ping method is chosen separately with `--ping-mode`, and the packet loss test works the same way in both modes. TCP connections cannot go through an HTTP proxy, so `--protocol tcp` together with `--proxy` is an invalid setting.

#### Choose transfer sizes

Each download and upload request moves one block of data. Larger blocks spend less of the test on starting requests, which can matter on fast links, while smaller blocks suit slow ones. `--dl-size` and `--ul-size` pick a block size from these tables:

| Flag | Sizes | Default |
| --- | --- | --- |
| `--dl-size` | 350, 500, 750, 1000, 1500, 2000, 2500, 3000, 3500, 4000 | 1000 |
| `--ul-size` | 100, 300, 500, 800, 1000, 1500, 2500, 3000, 3500, 4000 | 1000, or 800 with `--multi` |

Over HTTP, a download size is the side of the requested test image, so 2000 requests a 2000x2000 image, and an upload size is in kB. With `--protocol tcp`, both sizes are in kB. Any other value is an invalid setting.

```bash
speedtest-go --dl-size 2000 --ul-size 1500
```

#### Filter servers by country

`--filter-cc` keeps only the servers in the given countries, named by their two-letter ISO codes. It applies to the server the test picks and to `list`, and takes several codes, repeated or separated by commas. `auto` stands for your own country: the country of `--city` when one is set, and otherwise the country speedtest.net reports for your connection.

```bash
speedtest-go --filter-cc auto
speedtest-go list --filter-cc gb,ie
```

A filter that leaves no servers is an error. `--filter-cc` cannot be combined with `--server` or `--custom-url`, which already choose the servers.

#### Hide your address in the output

`--mask-isp` replaces the letters and digits of your IP address and coordinates with `*` wherever they appear: on the `ISP:` line, in `--unix` output, and in the `--json` and `--jsonl` reports. Separators stay, so `203.0.113.7` becomes `***.*.***.*`. Distances to servers are hidden too, since distances to servers at known coordinates would reveal where you are: they are masked on the `Test Server:` line and in `list`, and left out of the JSON reports. Your ISP name and country are still shown.

```bash
speedtest-go --mask-isp --json
speedtest-go list --mask-isp
```

#### Test with a virtual location

`--city` and `--location` choose servers near another place, by a predefined city label or by coordinates, and `--search` narrows the servers by a keyword. They work for both the speed test and `list`. Use one of `--city` and `--location`, and leave all three out when `--server` or `--custom-url` already chooses the servers. An unknown city label or malformed coordinates are an invalid setting.

```bash
$ speedtest-go cities
Available city labels (case insensitive):
  CC             CityLabel       Location
(za)                capetown    [-33.9391993, 18.4316716]
(pl)                  warsaw    [52.2396659, 21.0129345]
(sg)                  yishun    [1.4230218, 103.8404728]
...

$ speedtest-go --city=capetown

# Or with coordinates
$ speedtest-go --location=60,-110

# Or list the servers there first, then pick one
$ speedtest-go list --city=capetown
$ speedtest-go --server <server_id_from_list>
```

#### Memory Saving Mode

With `--saving-mode` option, it can be executed even in an insufficient memory environment like IoT devices.
The memory usage can be reduced to 1/10, about 10MB of memory is used.

However, please be careful that the accuracy is particularly low, especially in an environment of 30 Mbps or higher.
To get more accurate results, run multiple times and average.

For more details, please see [saving mode experimental result](https://github.com/nicholas-fedor/speedtest-go/blob/master/docs/saving_mode_experimental_result.md).

⚠️This feature has been deprecated > v1.4.0, because speedtest-go can always run with less than 10MBytes of memory now. Even so, `--saving-mode` is still a good way to reduce computation.

## Go API

```bash
go get github.com/nicholas-fedor/speedtest-go/v2
```

Upgrading from v1? The [migration guide](docs/v2-migration.md) maps each v1 function, method, and field to its v2 replacement.

### API Usage

The code below, based on the [naive example](https://github.com/nicholas-fedor/speedtest-go/blob/master/example/naive/main.go), finds the speedtest server with the lowest latency and tests the latency, download and upload speeds, and packet loss.

```go
package main

import (
 "context"
 "fmt"
 "log"
 "time"

 "github.com/nicholas-fedor/speedtest-go/v2/speedtest"
)

func main() {
 // Options configure the client, and New reports every invalid one. For example:
 //  - speedtest.WithProxy("socks5://127.0.0.1:7890") uses a proxy.
 //  - speedtest.WithSource("eth0") sends from a source IP address or network interface name.
 //  - speedtest.WithLocation(tokyo), with tokyo from speedtest.GetLocation("tokyo"), chooses servers near a city
 //    label instead of your own location. speedtest.ParseLocation("osaka", "34.6952,135.5006") takes coordinates.
 speedtestClient, err := speedtest.New()
 if err != nil {
  log.Fatal(err)
 }

 ctx := context.Background()

 // Get user's network information
 // user, _ := speedtestClient.FetchUserInfo(ctx)

 // Search server using serverID.
 // eg: fetch server with ID 28910.
 // speedtest.ErrServerNotFound will be returned if the server cannot be found.
 // server, err := speedtestClient.FetchServerByID(ctx, "28910")

 serverList, err := speedtestClient.FetchServers(ctx)
 if err != nil {
  log.Fatal(err)
 }

 // Without server IDs, FindServer returns the server with the lowest latency.
 targets, err := serverList.FindServer([]int{})
 if err != nil {
  log.Fatal(err)
 }

 for _, s := range targets {
  ping, err := speedtestClient.Ping(ctx, s)
  if err != nil {
   log.Fatal(err)
  }

  download, err := speedtestClient.Download(ctx, s)
  if err != nil {
   log.Fatal(err)
  }

  upload, err := speedtestClient.Upload(ctx, s)
  if err != nil {
   log.Fatal(err)
  }

  // Note: Rate is in bytes per second, a float64 that prints in Mbps. Each test reports its own results.
  fmt.Printf("Latency: %s, Download: %s, Upload: %s\n", ping.Latency, download.Rate, upload.Rate)

  // PacketLoss measures until its context ends, and a failed measurement still returns what it saw.
  lossCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
  packetLoss, err := speedtestClient.PacketLoss(lossCtx, s)
  cancel()

  if err != nil {
   log.Println(err)
  }

  fmt.Println(packetLoss)
 }
}
```

Each test returns its own result. `Ping` returns `PingStats` with the mean latency, jitter, minimum, and maximum, and `Download` and `Upload` return a `Transfer` with the rate, the bytes counted, and the results of each connection. `DownloadMulti` and `UploadMulti` spread one test across a main server and the other servers in a list, as the [multi-server example](https://github.com/nicholas-fedor/speedtest-go/blob/master/example/multi/main.go) shows. `speedtest.JSON` and `speedtest.JSONL` write results in the CLI's JSON format.

### Packet Loss

`Client.PacketLoss` measures the packet loss to one server until its context ends, as the example above does with a 30-second timeout.

The [code](https://github.com/nicholas-fedor/speedtest-go/blob/master/example/packet_loss/main.go) below uses the packet loss analyzer instead, which reports each sample as it arrives and can combine several servers.

```go
package main

import (
 "context"
 "fmt"
 "github.com/nicholas-fedor/speedtest-go/v2/speedtest"
 "log"
)

func checkError(err error) {
 if err != nil {
  log.Fatal(err)
 }
}

// Note: The current packet loss analyzer does not support udp over http.
// This means we cannot get packet loss through a proxy.
func main() {
 // Retrieve the server with the lowest latency
 speedtestClient, err := speedtest.New()
 checkError(err)
 serverList, _ := speedtestClient.FetchServers(context.Background())
 targets, _ := serverList.FindServer([]int{})

 // Create a packet loss analyzer that connects the way the client does, with default options
 analyzer := speedtestClient.NewPacketLossAnalyzer(nil)

 // Perform packet loss analysis on the target server
 for _, server := range targets {
  err := analyzer.Run(server.Host, func(packetLoss *speedtest.PacketLossStats) {
   fmt.Println(packetLoss, server.Host, server.Name)
   // fmt.Println(packetLoss.Loss())
  })
  checkError(err)
 }

 // or test several servers at the same time.
 packetLoss, err := analyzer.RunMulti(serverList.Available().Hosts())
 checkError(err)
 fmt.Println(packetLoss)
}
```

## Summary of Experimental Results

Speedtest-go is a great tool because of the following five reasons:

- Cross-platform available.
- Low memory environment.
- We are the first **FULL-FEATURED** open source speed testing project based on speedtest.net, including down/up rates, jitter and packet loss, etc.
- Testing time is the **SHORTEST** compare to [speedtest.net](http://www.speedtest.net/) and [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli), especially about 2x faster than [speedtest.net](http://www.speedtest.net/).
- Result is **MORE CLOSE** to [speedtest.net](http://www.speedtest.net/) than [speedtest-cli](https://github.com/sivel/speedtest-cli).

The following data is summarized. If you got interested, please see [more details](https://github.com/nicholas-fedor/speedtest-go/blob/master/docs/experimental_result.md).

### Download (Mbps)

distance = distance to testing server

- 0 - 1000(km) ≒ domestic
- 1000 - 8000(km) ≒ same region
- 8000 - 20000(km) ≒ really far!
- 20000km is half of the circumference of our planet.

| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
|:-------------:|:-------------:|:------------:|:-------------:|
|   0 - 1000    |     92.12     |  **91.21**   |     70.27     |
|  1000 - 8000  |     66.45     |  **65.51**   |     56.56     |
| 8000 - 20000  |     11.84     |     9.43     |   **11.87**   |

### Upload (Mbps)

| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
|:-------------:|:-------------:|:------------:|:-------------:|
|   0 - 1000    |     65.56     |  **47.58**   |     36.16     |
|  1000 - 8000  |     58.02     |  **54.74**   |     26.78     |
| 8000 - 20000  |     5.20      |   **8.32**   |     2.58      |

### Testing Time (sec)

| distance (km) | speedtest.net | speedtest-go | speedtest-cli |
|:-------------:|:-------------:|:------------:|:-------------:|
|   0 - 1000    |     45.03     |  **22.84**   |     24.46     |
|  1000 - 8000  |     44.89     |  **24.45**   |     28.52     |
| 8000 - 20000  |     49.64     |  **34.08**   |     41.26     |

## Contributors

See [Contributors](https://github.com/nicholas-fedor/speedtest-go/graphs/contributors), PRs are welcome! The [contributing guide](CONTRIBUTING.md) covers the development tools, tasks, and conventions.

## Issues

You can find or report issues in the [Issue Tracker](https://github.com/nicholas-fedor/speedtest-go/issues).

## LICENSE

[MIT](https://github.com/nicholas-fedor/speedtest-go/blob/master/LICENSE)
