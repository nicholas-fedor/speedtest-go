# Docker Example

This example runs the `speedtest-go` container image with Docker Compose, reading its settings from a mounted settings file.

## Files

### compose.yaml

- Runs one speed test with the `nickfedor/speedtest-go:latest` image.
- Mounts `config.yaml` read-only and names it with `SPEEDTEST_GO_CONFIG`.
- Runs the container with a read-only file system and no capabilities, since a speed test needs neither.

### config.yaml

- Holds the settings, with the flag names in camelCase as keys.
- Measures latency over TCP and writes the results as JSON.
- Lists other common settings, commented out.

## How to Run

1. Navigate to the example directory:

   ```bash
   cd example/docker
   ```

2. Run a speed test:

   ```bash
   docker compose run --rm speedtest
   ```

3. Save the JSON results to a file. `-T` keeps Docker from adding a terminal, so only the JSON reaches the file:

   ```bash
   docker compose run --rm -T speedtest > result.json
   ```

## Changing the Settings

Edit `config.yaml`, or override a setting for one run with a flag or an environment variable, which take precedence over the file:

```bash
# A flag after the service name
docker compose run --rm speedtest --server 6691

# An environment variable
docker compose run --rm -e SPEEDTEST_GO_SERVER=6691,6087 speedtest
```

`json: true` in `config.yaml` writes JSON. To see the plain result lines instead, remove it, or override it for one run:

```bash
docker compose run --rm -e SPEEDTEST_GO_JSON=false speedtest
```

## Notes

- The image runs as user 65534, so `config.yaml` must be readable by that user, as files created with the usual permissions are.
- A settings file with an unknown key, a repeated key, or a value of the wrong type stops the run with exit code 2, and the error names the file and key.
- Set `SPEEDTEST_GO_IMAGE` to use another image, such as `ghcr.io/nicholas-fedor/speedtest-go:latest`, a pinned version, or a local build.
- `--ping-mode icmp` does not work in the image, since ICMP needs raw sockets, which its unprivileged user cannot open.
