---
title: Go
description: Install speedtest-go with go install on any platform Go supports
weight: 3
---

```bash
go install github.com/nicholas-fedor/speedtest-go/v2@latest
```

The binary lands in `$GOBIN`, or in `$HOME/go/bin` when `GOBIN` is not set. Run the same command to update it, and delete the binary to remove it.

To use speedtest-go as a library in your own program instead, see [Go Library](/go-library/).
