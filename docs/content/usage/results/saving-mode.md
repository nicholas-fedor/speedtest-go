---
title: Saving Mode Results
description: Rates and memory use of speedtest-go's saving mode compared with its normal mode
weight: 1
---

These measurements come from the [upstream project](https://github.com/showwin/speedtest-go), using an earlier version. They compare `--saving-mode` with the normal mode. [Saving mode](/usage/protocols/#saving-mode) describes what it changes.

| Column prefix | Mode |
| --- | --- |
| `N` | Normal mode |
| `S` | `--saving-mode` |

| exp No. | N - Download |  N - Upload  | N - RAM  | S - Download |  S - Upload  | S- RAM |
|:--------|:------------:|:------------:|:--------:|:------------:|:------------:|:------:|
| 1       | 54.13 Mbit/s | 61.07 Mbit/s | 85.82MB  | 50.65 Mbit/s | 29.25 Mbit/s | 5.39MB |
| 2       | 47.65 Mbit/s | 57.94 Mbit/s | 93.84MB  | 48.39 Mbit/s | 44.04 Mbit/s | 9.39MB |
| 3       | 52.58 Mbit/s | 56.43 Mbit/s | 109.86MB | 24.34 Mbit/s | 41.68 Mbit/s | 9.39MB |
| 4       | 43.73 Mbit/s | 49.46 Mbit/s | 109.86MB | 32.10 Mbit/s | 36.51 Mbit/s | 9.39MB |
| 5       | 54.66 Mbit/s | 42.81 Mbit/s | 101.84MB | 40.08 Mbit/s | 10.37 Mbit/s | 5.39MB |
