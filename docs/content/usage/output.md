---
title: Output and privacy
description: Choose speedtest-go's output format and rate units, and hide your address and location in the output
weight: 5
---

## Output formats

| Output | Flag | Use it for |
| --- | --- | --- |
| Live progress | none | Watching a test in a terminal |
| Plain lines | `--unix` | Logs, and terminals that cannot redraw |
| One JSON document | `--json` | Scripts that read the whole result |
| One JSON line per server | `--jsonl` | Scripts that process servers one at a time |

Things to know:

- Use at most one of `--json`, `--jsonl`, and `--unix`. More than one is an invalid setting.
- When the output is not a terminal, such as a pipe or a file, or with `--debug`, the results are plain lines without the progress, as with `--unix`.
- With `--json` or `--jsonl`, nothing but the report goes to standard output, and errors go to standard error.

Plain lines look like this:

```bash
$ speedtest-go --unix

    speedtest-go v<version> (<commit>, <date>)

✓ ISP: 203.0.113.7 (Fujitsu) [34.9769, 138.3831]
✓ Found 20 Public Servers

✓ Test Server: [6691] 9.03km Shizuoka (Japan) by sudosan
✓ Latency: 4.452963ms Jitter: 41.271µs Min: 4.395179ms Max: 4.517576ms
✓ Packet Loss Analyzer: Running in background (<= 30 Secs)
✓ Download: 115.52 Mbps (Used: 135.75MB) (Latency: 4ms Jitter: 0ms Min: 4ms Max: 4ms)
✓ Upload: 4.02 Mbps (Used: 6.85MB) (Latency: 4ms Jitter: 1ms Min: 3ms Max: 8ms)
✓ Packet Loss: 8.82% (Sent: 217/Dup: 0/Max: 237)
```

## Rate units

Rates are shown in Mbps by default. `--unit` chooses auto-scaled units instead:

| Unit | Shows |
| --- | --- |
| `decimal-bits` | bps, Kbps, Mbps, Gbps |
| `decimal-bytes` | B/s, KB/s, MB/s, GB/s |
| `binary-bits` | Kibps, Mibps, Gibps |
| `binary-bytes` | KiB/s, MiB/s, GiB/s |

```bash
speedtest-go --unit binary-bits
```

## Hide your address

`--mask-isp` replaces the letters and digits of your IP address, ISP name, and coordinates with `*`, so `203.0.113.7` becomes `***.*.***.*`.

| Output | What is hidden |
| --- | --- |
| The `ISP:` line and `--unix` output | Your address, ISP name, and coordinates |
| `--json` and `--jsonl` reports | Your address, ISP name, and coordinates |
| The `Test Server:` line and `list` | Distances to servers |
| `--json` and `--jsonl` reports | Distances to servers, which are left out |

Distances are hidden too, since distances to servers at known coordinates would reveal where you are. Your country is still shown.

```bash
speedtest-go --mask-isp --json
speedtest-go list --mask-isp
```
