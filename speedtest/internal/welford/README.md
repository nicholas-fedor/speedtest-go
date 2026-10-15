# Welford's algorithm for test stability

This package implements the stability check and rate estimate from [showwin/speedtest-go#192](https://github.com/showwin/speedtest-go/issues/192).

<img width="1172" alt="SpeedTest-Go (1)" src="https://github.com/showwin/speedtest-go/assets/30739857/ecced0e9-830c-42d6-aa8b-e3dcf8d124d6">

1. Welford's algorithm computes the mean and standard deviation of the cumulative average rate quickly, one sample at a time.
2. The statistics cover a moving window of recent samples, so the volatile start of a test drops out of them.
3. The coefficient of variation (c.v.), the standard deviation divided by the mean, measures how settled the rate is.
4. An update counts as stable when the window is full and its c.v. is below 0.03. Each stable update adds one to a count and each other update takes one away, never below zero. The test is stable once the count reaches a third of the window and more than twice the window's samples have been seen.
5. The rate estimate is the average of an exponentially weighted moving average of the per-period samples, with smoothing factor 2/(N+1) for an N-sample window, and the window's mean sample.
6. The rate estimate is the live result of the test.
7. When the rate settles quickly, the test stops early, which shortens it on stable networks and devices.
