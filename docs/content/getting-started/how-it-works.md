---
title: How a Test Works
description: How speedtest-go chooses its servers, and how it measures latency, download and upload rates, and packet loss
weight: 2
---

A run has these steps:

1. [Find the servers](#finding-the-servers)
2. [Measure latency](#latency-and-jitter)
3. [Measure the download and upload rates](#download-and-upload)
4. [Measure packet loss](#packet-loss), in the background while the other tests run

With several servers, the tests run on each server in turn.

## Finding the servers

1. speedtest-go asks speedtest.net for your IP address, ISP, location, and country.
2. It fetches the servers speedtest.net lists near you, or near the place `--city` or `--location` names.
3. It chooses the servers to test:

| Settings | Servers tested |
| --- | --- |
| None | The server with the lowest latency |
| `--server` | The servers with those IDs |
| `--custom-url` | The server at that URL, which speedtest.net does not need to list |
| `--filter-cc` | The server with the lowest latency in those countries |

[Choosing servers](/usage/servers/) covers each setting.

## Latency and jitter

speedtest-go measures 10 round trips to the server:

| Result | Meaning |
| --- | --- |
| Latency | The mean round trip |
| Jitter | How much the round trips vary |
| Min, Max | The shortest and longest round trips |

`--ping-mode` chooses how each round trip is made: an HTTP request, the speedtest.net TCP protocol, or ICMP. [Test protocols](/usage/protocols/#ping-methods) compares them.

## Download and upload

Each transfer test moves data over several connections at once: by default one per CPU for downloads, and up to 8 for uploads.

| Aspect | Behavior |
| --- | --- |
| Duration | Up to 15 seconds, ending early once the rate settles |
| Rate | The bytes moved over the time they took |
| Loaded latency | Pings every half second during the transfer, which show how latency changes under load |
| Protocol | HTTP by default, or plain TCP with `--protocol tcp` |

In a terminal, the progress bar's estimate of the time left tracks how soon the rate is expected to settle.

`--thread`, `--dl-size`, and `--ul-size` tune the transfers, and `--no-download` and `--no-upload` skip them. [Test protocols](/usage/protocols/) covers them.

## Packet loss

Packet loss is measured over UDP, in the background while the other tests run, for up to 30 seconds. speedtest-go numbers each packet it sends, and the server reports what it received, so the result is the loss on the way up to the server.

| Result | Meaning |
| --- | --- |
| Sent | The packets the server received |
| Dup | The duplicates among them |
| Max | The highest packet number the server received. Packets are numbered from 0, so Max + 1 packets had been sent by then |

The loss is the share of those Max + 1 packets that never arrived: `1 - (Sent - Dup) / (Max + 1)`. Packet loss shows N/A when the server received none of them, such as through a proxy or a firewall that blocks UDP.
