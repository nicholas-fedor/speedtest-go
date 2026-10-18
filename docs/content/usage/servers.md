---
title: Choosing servers
description: Pick the speedtest.net servers a test uses, by ID, location, country, or keyword
weight: 2
---

By default, a test uses the server near you with the lowest latency.

| To | Use |
| --- | --- |
| Test specific servers | `--server` |
| Test a server speedtest.net does not list | `--custom-url` |
| Spread each test across several servers | `--multi` |
| Keep only servers in some countries | `--filter-cc` |
| Choose servers near another place | `--city` or `--location` |
| Narrow the servers by a keyword | `--search` |

## Server IDs

`list` shows the available servers with their IDs, distances, and latencies:

```bash
$ speedtest-go list
Testing From IP: 203.0.113.7 (Fujitsu) [34.9769, 138.3831]
[6691]     9.03km   32.3365ms  Shizuoka (Japan) by sudosan
[6087]   120.55km   51.7453ms  Fussa-shi (Japan) by Allied Telesis Capital Corporation
[6508]   125.44km   54.6683ms  Yokohama (Japan) by at2wn
...
```

Select servers by ID, repeated or separated by commas:

```bash
speedtest-go --server 6691 --server 6087
speedtest-go --server 6691,6087
```

An ID that speedtest.net does not list is an error, and the error names every such ID.

## Multi-server tests

With `-m` or `--multi`, each download and upload test spreads its connections across several servers at once. The servers selected with `--server` or `--custom-url` are the pool it uses.

```bash
speedtest-go --multi
```

## Countries

`--filter-cc` keeps only the servers in the given countries, named by their two-letter ISO codes. It takes several codes, repeated or separated by commas.

`auto` stands for your own country:

- The country of `--city`, when one is set
- Otherwise, the country speedtest.net reports for your connection

```bash
speedtest-go --filter-cc auto
speedtest-go list --filter-cc gb,ie
```

Rules:

- It applies to the server the test picks and to `list`.
- A filter that leaves no servers is an error.
- It cannot be combined with `--server` or `--custom-url`, which already choose the servers.

## Another location

`--city` and `--location` choose servers near another place, and `--search` narrows the servers by a keyword. They work for both the speed test and `list`.

| Flag | Takes |
| --- | --- |
| `--city` | A predefined city label, such as `capetown` |
| `--location` | Coordinates, as `lat,lon` |
| `--search` | A keyword |

`cities` lists the city labels:

```bash
$ speedtest-go cities
Available city labels (case insensitive):
  CC             CityLabel       Location
(za)                capetown    [-33.9391993, 18.4316716]
(pl)                  warsaw    [52.2396659, 21.0129345]
(sg)                  yishun    [1.4230218, 103.8404728]
...
```

```bash
speedtest-go --city capetown
speedtest-go --location=60,-110

# Or list the servers there first, then pick one
speedtest-go list --city capetown
speedtest-go --server <server_id_from_list>
```

Rules:

- Use one of `--city` and `--location`, not both.
- Leave all three out when `--server` or `--custom-url` already chooses the servers.
- An unknown city label or malformed coordinates are an invalid setting.
