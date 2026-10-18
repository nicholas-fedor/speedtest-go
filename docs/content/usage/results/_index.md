---
title: Experimental Results
description: How speedtest-go's rates and testing time compare with speedtest.net and speedtest-cli, by distance to the server
weight: 8
---

These measurements come from the [upstream project](https://github.com/showwin/speedtest-go), which compared speedtest-go with [speedtest.net](https://www.speedtest.net/) and [sivel/speedtest-cli](https://github.com/sivel/speedtest-cli) using an earlier version.

## Summary

| Finding | Detail |
| --- | --- |
| Testing time | About half as long as speedtest.net, and shorter than speedtest-cli |
| Accuracy | Closer to speedtest.net than speedtest-cli is |

Distances group the servers by how far away they are:

| Distance (km) | Meaning |
| --- | --- |
| 0 to 1000 | Domestic |
| 1000 to 8000 | The same region |
| 8000 to 20000 | Very far. 20000 km is half the circumference of the Earth |

### Average download (Mbps)

| Distance (km) | speedtest.net | speedtest-go | speedtest-cli |
| :-: | :-: | :-: | :-: |
| 0 to 1000 | 92.12 | **91.21** | 81.99 |
| 1000 to 8000 | 66.45 | **65.51** | 56.56 |
| 8000 to 20000 | 11.84 | 9.43 | **11.87** |

### Average upload (Mbps)

| Distance (km) | speedtest.net | speedtest-go | speedtest-cli |
| :-: | :-: | :-: | :-: |
| 0 to 1000 | 65.56 | **47.58** | 36.16 |
| 1000 to 8000 | 58.02 | **54.74** | 26.78 |
| 8000 to 20000 | 6.94 | **8.32** | 2.58 |

### Average testing time (seconds)

| Distance (km) | speedtest.net | speedtest-go | speedtest-cli |
| :-: | :-: | :-: | :-: |
| 0 to 1000 | 45.03 | **22.84** | 24.46 |
| 1000 to 8000 | 44.89 | **24.45** | 28.52 |
| 8000 to 20000 | 49.64 | **34.08** | 41.26 |

The bold value in each row is the one closest to speedtest.net's, or the shortest time.

## Method

- 9 servers were chosen at random, 3 in each distance group.
- Each tool tested each server twice.
- The order of the tools alternated between servers, so no tool always ran first.

## Full results

### Download (Mbps)

| Distance (km)          | Server ID | speedtest.net (test1) | speedtest.net (test2) | speedtest-go (test1) | speedtest-go (test2) | speedtest-cli (test1) | speedtest-cli (test2) |
|:-----------------------|:---------:|:---------------------:|:---------------------:|:--------------------:|:--------------------:|:---------------------:|:---------------------:|
| 0 - 1000 (9km)         |   6691    |         93.34         |         93.51         |        89.13         |        88.40         |         88.21         |         83.34         |
| 0 - 1000 (194km)       |   6368    |         93.45         |         94.20         |        92.93         |        92.26         |         88.45         |         77.07         |
| 0 - 1000 (865km)       |   6842    |         88.31         |         89.91         |        91.26         |        93.30         |         67.33         |         87.52         |
| 1000 - 8000 (1982km)   |   2589    |         90.24         |         94.05         |        79.61         |        79.52         |         79.16         |         70.36         |
| 1000 - 8000 (4801km)   |   3296    |         75.61         |         52.83         |        69.89         |        69.86         |         58.19         |         55.93         |
| 1000 - 8000 (6627km)   |   1718    |         42.04         |         43.92         |        39.81         |        54.35         |         44.47         |         31.27         |
| 8000 - 20000 (8972km)  |   3409    |         22.11         |         22.74         |        28.21         |        21.82         |         24.78         |         30.59         |
| 8000 - 20000 (13781km) |   3162    |         15.78         |         7.73          |         1.94         |         1.02         |         3.32          |         2.40          |
| 8000 - 20000 (17805km) |   4256    |         1.15          |         1.50          |         1.79         |         1.79         |         6.41          |         3.69          |

### Upload (Mbps)

| Distance (km)          | Server ID | speedtest.net (test1) | speedtest.net (test2) | speedtest-go (test1) | speedtest-go (test2) | speedtest-cli (test1) | speedtest-cli (test2) |
|:-----------------------|:---------:|:---------------------:|:---------------------:|:--------------------:|:--------------------:|:---------------------:|:---------------------:|
| 0 - 1000 (9km)         |   6691    |         52.03         |         48.82         |        40.72         |        43.84         |         36.03         |         36.2          |
| 0 - 1000 (194km)       |   6368    |         78.30         |         66.92         |        53.98         |        44.07         |         37.26         |         46.98         |
| 0 - 1000 (865km)       |   6842    |         65.34         |         81.96         |        50.27         |        52.57         |         29.42         |         31.08         |
| 1000 - 8000 (1982km)   |   2589    |         89.29         |         58.69         |        80.68         |        56.59         |         42.66         |         44.34         |
| 1000 - 8000 (4801km)   |   3296    |         50.16         |         50.78         |        54.94         |        54.18         |         20.59         |         20.23         |
| 1000 - 8000 (6627km)   |   1718    |         48.48         |         50.71         |        39.81         |        42.21         |         16.16         |         16.72         |
| 8000 - 20000 (8972km)  |   3409    |         18.07         |         20.30         |        20.35         |        24.92         |         5.87          |         3.37          |
| 8000 - 20000 (13781km) |   3162    |         1.45          |         0.33          |         1.78         |         0.77         |         1.08          |         1.34          |
| 8000 - 20000 (17805km) |   4256    |         1.16          |         0.30          |         1.00         |         1.11         |         2.37          |         1.42          |

### Testing Time (sec)

| Distance (km)          | Server ID | speedtest.net (test1) | speedtest.net (test2) | speedtest-go (test1) | speedtest-go (test2) | speedtest-cli (test1) | speedtest-cli (test2) |
|:-----------------------|:---------:|:---------------------:|:---------------------:|:--------------------:|:--------------------:|:---------------------:|:---------------------:|
| 0 - 1000 (9km)         |   6691    |         44.87         |         45.76         |        24.45         |        22.40         |         20.82         |         16.51         |
| 0 - 1000 (194km)       |   6368    |         43.85         |         42.97         |        19.60         |        22.36         |         22.25         |         24.01         |
| 0 - 1000 (865km)       |   6842    |         46.62         |         46.12         |        21.83         |        26.40         |         39.07         |         24.10         |
| 1000 - 8000 (1982km)   |   2589    |         45.38         |         44.18         |        18.71         |        21.35         |         26.15         |         25.71         |
| 1000 - 8000 (4801km)   |   3296    |         47.92         |         50.52         |        24.71         |        23.47         |         32.07         |         28.53         |
| 1000 - 8000 (6627km)   |   1718    |         40.15         |         41.16         |        28.05         |        30.42         |         31.01         |         27.65         |
| 8000 - 20000 (8972km)  |   3409    |         51.53         |         56.89         |        38.13         |        41.55         |         36.87         |         34.47         |
| 8000 - 20000 (13781km) |   3162    |         52.75         |         58.10         |        28.78         |        29.91         |         40.75         |         46.88         |
| 8000 - 20000 (17805km) |   4256    |         37.73         |         40.82         |        33.33         |        32.79         |         37.82         |         50.78         |
