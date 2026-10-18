---
title: Kubernetes
description: Run speedtest-go in Kubernetes as a pod or a CronJob, with its settings in a ConfigMap
weight: 5
---

The [Kubernetes example](https://github.com/nicholas-fedor/speedtest-go/tree/master/example/kubernetes) runs the [Docker image](/installation/docker/) as a pod or a CronJob, with its settings in a ConfigMap and a hardened security context.

| File | Holds |
| --- | --- |
| `configmap.yaml` | The settings file, as the `speedtest-settings` ConfigMap |
| `pod.yaml` | A pod that runs one speed test |
| `cronjob.yaml` | A CronJob that runs a speed test every six hours |

## Run it

1. Create the settings:

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/example/kubernetes/configmap.yaml
    ```

2. Run a speed test once, and read its JSON results once the pod has finished:

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/example/kubernetes/pod.yaml
    kubectl wait --for=jsonpath='{.status.phase}'=Succeeded pod/speedtest --timeout=3m
    kubectl logs speedtest
    ```

3. Or run speed tests on a schedule:

    ```bash
    kubectl apply -f https://raw.githubusercontent.com/nicholas-fedor/speedtest-go/master/example/kubernetes/cronjob.yaml
    ```

    To try the CronJob without waiting for its schedule, start a job from it:

    ```bash
    kubectl create job speedtest-now --from=cronjob/speedtest
    ```

## Settings

| To | Do |
| --- | --- |
| Change the defaults | Edit the ConfigMap and apply it again. The next run reads the new settings |
| Override a setting with a flag | Add it to the container's `args`, such as `args: ["--server", "6691"]` |
| Override a setting with a variable | Add it to the container's `env`, such as `SPEEDTEST_GO_SERVER` with the value `6691,6087` |

Flags go in `args`, since the image's entrypoint is `speedtest-go`. ICMP pings do not work in the image, so use `--ping-mode http` or `tcp`.
