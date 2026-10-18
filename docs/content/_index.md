---
title: speedtest-go
layout: hextra-home
description: Test internet download and upload speeds with speedtest.net servers, from the command line or from Go
---

<div class="st-hero not-prose">
<img src="images/logo.svg" alt="The speedtest-go gopher beside a progress bar" width="220" height="220" />

{{< hextra/hero-headline >}}
  speedtest-go
{{< /hextra/hero-headline >}}

{{< hextra/hero-subtitle >}}
  Test internet download and upload speeds with speedtest.net servers, from the command line or from Go
{{< /hextra/hero-subtitle >}}

<div class="st-hero-actions">
{{< hextra/hero-button text="Get started" link="getting-started/" >}}
{{< hextra/hero-button text="Install" link="installation/" style="background: transparent; color: inherit; box-shadow: inset 0 0 0 1px currentColor;" >}}
</div>
</div>

<div class="st-section">

{{< video name="demo" alt="A speed test against a San Jose server, showing the live progress bars, each result, and the summary panel" >}}

</div>

<div class="st-section">

## What it measures

{{< hextra/feature-grid cols="3" >}}
  {{< hextra/feature-card title="Latency and jitter" icon="clock" subtitle="Round trips over HTTP, TCP, or ICMP, with the minimum, maximum, and jitter" >}}
  {{< hextra/feature-card title="Download and upload" icon="switch-vertical" subtitle="Rates over HTTP or plain TCP, across one server or several at once" >}}
  {{< hextra/feature-card title="Packet loss" icon="chart-bar" subtitle="Measured over UDP in the background while the other tests run" >}}
{{< /hextra/feature-grid >}}

</div>

<div class="st-section">

## Quick start

```bash
tmp=$(mktemp)
curl -sSfL https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/scripts/install.sh -o "$tmp" && sh "$tmp"
speedtest-go
```

</div>

<div class="st-section">

## Documentation

{{< cards cols="2" >}}
  {{< card link="getting-started/" title="Getting Started" icon="play" subtitle="Run a first test and read its results" >}}
  {{< card link="installation/" title="Installation" icon="download" subtitle="The install script, binaries, Go, Docker, and Kubernetes" >}}
  {{< card link="usage/" title="Usage" icon="book-open" subtitle="Settings, guides, troubleshooting, and the CLI reference" >}}
  {{< card link="go-library/" title="Go Library" icon="code" subtitle="Measure from your own Go program" >}}
{{< /cards >}}

</div>
