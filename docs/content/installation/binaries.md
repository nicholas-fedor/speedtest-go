---
title: Binaries
description: Install speedtest-go on Linux, macOS, or Windows from a release archive, after checking it against the release's checksums
weight: 2
---

Each [release](https://github.com/nicholas-fedor/speedtest-go/releases) has an archive for every platform, holding the `speedtest-go` binary and its license. The archive names follow this pattern:

```text
speedtest-go_<os>_<architecture>_<version>.tar.gz
speedtest-go_windows_<architecture>_<version>.zip
```

The examples below use version 1.8.4. Set `VERSION` to the release you want, without its `v`, and `BASE` holds the release's download URL. Each one checks the archive against the release's `checksums.txt` before installing it.

## Linux

| Architecture | Archive |
| --- | --- |
| x86-64 | `speedtest-go_linux_amd64_<version>.tar.gz` |
| ARM64 | `speedtest-go_linux_arm64v8_<version>.tar.gz` |
| ARMv6 and ARMv7 | `speedtest-go_linux_armhf_<version>.tar.gz` |
| x86 | `speedtest-go_linux_i386_<version>.tar.gz` |
| RISC-V 64 | `speedtest-go_linux_riscv64_<version>.tar.gz` |

```bash
VERSION=1.8.4
BASE=https://github.com/nicholas-fedor/speedtest-go/releases/download/v${VERSION}
curl -sSfLO "${BASE}/speedtest-go_linux_amd64_${VERSION}.tar.gz"
curl -sSfLO "${BASE}/checksums.txt"
sha256sum --check --ignore-missing checksums.txt
tar -xzf "speedtest-go_linux_amd64_${VERSION}.tar.gz" speedtest-go
sudo install speedtest-go /usr/local/bin/
```

To install it for your user only, put it in a directory on your `PATH` instead, such as `~/.local/bin`.

The [install script](/installation/script/) does the same, and also installs the native package on Debian, Fedora, Alpine, and Arch Linux.

## macOS

| Mac | Archive |
| --- | --- |
| Apple silicon | `speedtest-go_macOS_arm64v8_<version>.tar.gz` |
| Intel | `speedtest-go_macOS_amd64_<version>.tar.gz` |

```bash
VERSION=1.8.4
BASE=https://github.com/nicholas-fedor/speedtest-go/releases/download/v${VERSION}
curl -sSfLO "${BASE}/speedtest-go_macOS_arm64v8_${VERSION}.tar.gz"
curl -sSfLO "${BASE}/checksums.txt"
shasum -a 256 --check --ignore-missing checksums.txt
tar -xzf "speedtest-go_macOS_arm64v8_${VERSION}.tar.gz" speedtest-go
sudo install speedtest-go /usr/local/bin/
```

The binary is not signed by Apple. An archive downloaded with a browser is quarantined, so macOS blocks the binary the first time it runs. Remove the quarantine to allow it:

```bash
xattr -d com.apple.quarantine speedtest-go
```

## Windows

| Architecture | Archive |
| --- | --- |
| x86-64 | `speedtest-go_windows_amd64_<version>.zip` |
| ARM64 | `speedtest-go_windows_arm64v8_<version>.zip` |
| x86 | `speedtest-go_windows_i386_<version>.zip` |

In PowerShell:

```powershell
$Version = "1.8.4"
$Base = "https://github.com/nicholas-fedor/speedtest-go/releases/download/v$Version"
$Dir = "$env:LOCALAPPDATA\speedtest-go"
$Zip = "speedtest-go_windows_amd64_$Version.zip"
Invoke-WebRequest "$Base/$Zip" -OutFile $Zip
Invoke-WebRequest "$Base/checksums.txt" -OutFile checksums.txt
$Expected = ((Select-String -Path checksums.txt -Pattern "\s$([regex]::Escape($Zip))$").Line -split "\s+")[0]
if ((Get-FileHash $Zip -Algorithm SHA256).Hash -ne $Expected) { throw "checksum mismatch for $Zip" }
Expand-Archive $Zip -DestinationPath $Dir
& "$Dir\speedtest-go.exe"
```

To run it as `speedtest-go` from any terminal, add its directory to your `PATH`:

```powershell
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", "$UserPath;$Dir", "User")
```

## Verify the signature

The steps above check the archive against `checksums.txt`. To also check that `checksums.txt` came from this project's release workflow, verify its [cosign](https://github.com/sigstore/cosign) signature, with `VERSION` and `BASE` set as above:

```bash
curl -sSfLO "${BASE}/checksums.txt.sig"
cosign verify-blob \
  --bundle checksums.txt.sig \
  --certificate-identity-regexp '^https://github.com/nicholas-fedor/speedtest-go/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

## Update and uninstall

| Action | Do |
| --- | --- |
| Update | Install the newer release the same way, over the old binary |
| Uninstall | Delete the binary, such as `/usr/local/bin/speedtest-go` or the `speedtest-go` directory in `%LOCALAPPDATA%` |
