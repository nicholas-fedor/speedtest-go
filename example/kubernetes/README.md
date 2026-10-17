# Kubernetes Example

This example runs the `speedtest-go` container image in Kubernetes, once as a pod and on a schedule as a CronJob, with its settings in a ConfigMap.

## Files

### configmap.yaml

- Holds the settings file as the `speedtest-settings` ConfigMap.
- Measures latency over TCP and writes the results as JSON.
- Uses the flag names in camelCase as keys, the same as a settings file anywhere else.

### pod.yaml

- Runs one speed test with the `nickfedor/speedtest-go:latest` image.
- Mounts the ConfigMap read-only at `/etc/speedtest-go` and names the file with `SPEEDTEST_GO_CONFIG`.
- Runs as a non-root user, with a read-only root file system, no capabilities, and the default seccomp profile, since a speed test needs nothing more.
- Takes flags in `args`. The image's entrypoint is `speedtest-go`, so setting `command` would replace it.

### cronjob.yaml

- Runs the same speed test every six hours.
- Never starts a run while the previous one is still going, and does not retry a failed run.

## How to Run

1. Create the settings:

   ```bash
   kubectl apply -f example/kubernetes/configmap.yaml
   ```

2. Run a speed test once, and read its JSON results once the pod has finished:

   ```bash
   kubectl apply -f example/kubernetes/pod.yaml
   kubectl wait --for=jsonpath='{.status.phase}'=Succeeded pod/speedtest --timeout=3m
   kubectl logs speedtest
   ```

3. Or run speed tests on a schedule:

   ```bash
   kubectl apply -f example/kubernetes/cronjob.yaml
   ```

   To try the CronJob without waiting for its schedule, start a job from it:

   ```bash
   kubectl create job speedtest-now --from=cronjob/speedtest
   ```

## Changing the Settings

- Edit `configmap.yaml` and apply it again. The next run reads the new settings.
- Override a setting for the pod or the CronJob with a flag in `args`, such as `args: ["--server", "6691"]`.
- Or set an environment variable in `env`, such as `SPEEDTEST_GO_SERVER` with the value `6691,6087`.

## Notes

- A settings file with an unknown key, a repeated key, or a value of the wrong type stops the run with exit code 2, and the pod's log names the file and key.
- `--ping-mode icmp` does not work in the image, since ICMP needs raw sockets, which its unprivileged user cannot open.
