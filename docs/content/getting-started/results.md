---
title: Reading the Results
description: What speedtest-go's live progress, plain lines, and JSON reports show, and the units of each JSON field
weight: 3
---

| Output | When |
| --- | --- |
| [Live progress](#live-progress) | In a terminal |
| [Plain lines](#plain-lines) | With `--unix` or `--debug`, or when the output is not a terminal |
| [JSON](#json-reports) | With `--json` or `--jsonl` |

## Live progress

In a terminal, each test shows its progress as it runs:

- Latency and packet loss show their measurements so far.
- A download or upload shows its rate, a progress bar, and an estimate of the time left.
- Each finished test becomes a line, and each server ends with a panel of its headline results.

## Plain lines

```text
$ speedtest-go --unix

    speedtest-go v1.8.0 (1559e47, 2026-05-02)

✓ ISP: 203.0.113.7 (Fujitsu) [34.9769, 138.3831]
✓ Found 20 Public Servers

✓ Test Server: [6691] 9.03km Shizuoka (Japan) by sudosan
✓ Latency: 4.452963ms Jitter: 41.271µs Min: 4.395179ms Max: 4.517576ms
✓ Packet Loss Analyzer: Running in background (<= 30 Secs)
✓ Download: 115.52 Mbps (Used: 135.75MB) (Latency: 4ms Jitter: 0ms Min: 4ms Max: 4ms)
✓ Upload: 4.02 Mbps (Used: 6.85MB) (Latency: 4ms Jitter: 1ms Min: 3ms Max: 8ms)
✓ Packet Loss: 8.82% (Sent: 217/Dup: 0/Max: 237)
```

| Line | Shows |
| --- | --- |
| `ISP` | Your IP address, ISP, and coordinates |
| `Found` | How many servers were found |
| `Test Server` | The server's ID, distance, location, and sponsor |
| `Latency` | The mean round trip, jitter, and the shortest and longest round trips |
| `Download`, `Upload` | The rate, the data moved, and the latency during the transfer |
| `Packet Loss` | The loss, and the counts it comes from |

[How a Test Works](../how-it-works/) explains each measurement, and [Output and privacy](/usage/output/) covers the rate units and hiding your address.

## JSON reports

`--json` writes one document for the run, and `--jsonl` writes one line per server, each with a `server` object in place of the `servers` list.

```json
{
  "timestamp": "2026-10-13 08:30:15.123",
  "userInfo": {
    "ip": "203.0.113.7",
    "lat": "35.6812",
    "lon": "139.7671",
    "isp": "Example ISP",
    "country": "JP"
  },
  "servers": [
    {
      "url": "http://speed.example/speedtest/upload.php",
      "lat": "35.6895",
      "lon": "139.6917",
      "name": "Tokyo",
      "country": "Japan",
      "cc": "JP",
      "sponsor": "Example Networks",
      "id": "1001",
      "host": "speed.example:8080",
      "distance": 6.95,
      "latency": 4500000,
      "maxLatency": 5000000,
      "minLatency": 4000000,
      "jitter": 300000,
      "dlSpeed": 12500000,
      "ulSpeed": 2500000,
      "testDuration": {
        "ping": 2000000000,
        "download": 10000000000,
        "upload": 10000000000,
        "total": 22000000000
      },
      "packetLoss": {
        "sent": 300,
        "dup": 1,
        "max": 301
      }
    }
  ]
}
```

| Field | Unit or format |
| --- | --- |
| `timestamp` | Local time, as `YYYY-MM-DD hh:mm:ss.mmm` |
| `userInfo` | Your IP address, coordinates, ISP, and country code. All but the country are masked with `--mask-isp` |
| `lat`, `lon` | Degrees, as strings |
| `distance` | Kilometres. Left out with `--mask-isp` |
| `latency`, `minLatency`, `maxLatency`, `jitter` | Nanoseconds |
| `dlSpeed`, `ulSpeed` | Bytes per second, or `-1` when the test did not run |
| `testDuration` | Nanoseconds for each test, or `null` when it did not run |
| `packetLoss` | The counts the loss comes from. See [Packet loss](../how-it-works/#packet-loss) |

In the example, `dlSpeed` of `12500000` bytes per second is 100 Mbps.
