# Orbital

Orbital is a lightweight Go tool that starts GitHub Actions self-hosted runners from your Docker image on a configured Docker context.

It uses a GitHub App to issue runner registration tokens, builds a runner image from your source image, and keeps the configured runner containers running until shutdown.

## Features

- **Docker context based operation**: Connects to the configured Docker context and starts runner-compatible containers there.
- **Runner image build orchestration**: Builds a runner image from a user-provided source image on the configured Docker context.
- **Dynamic runner registration**: Authenticates as a GitHub App and generates runner registration tokens on demand.
- **Platform-aware runner injection**: Resolves the GitHub Actions runner archive OS/architecture from the inspected source image metadata.
- **Runner container reconciliation**: Keeps the configured number of runner containers running and recreates them when necessary.
- **Graceful shutdown**: Stops and removes managed runner containers when Orbital receives a termination signal.

## Prerequisites

Before using Orbital, make sure you have:

- [Go](https://go.dev/) 1.26.1 or later
- [Docker](https://www.docker.com/)
- A reachable Docker context already configured via `docker context`
- A GitHub App installed on your organization with permission to manage self-hosted runners
- A source image that contains the tools and runtime environment your jobs need

The repository includes an example source image in:

- `example/Dockerfile`

Orbital builds the actual runner image itself by layering the GitHub Actions runner and its entrypoint on top of your configured source image.

## Configuration

Orbital uses a YAML configuration file.

Example:

```yaml
docker:
  context: default
  sourceImage: source:latest
  runnerImageName: orbital-runner:latest

github:
  org: ExampleOrg
  appId: 1234567
  installationId: 123456789
  pem: ~/.ssh/github-app.pem

runner:
  group: RunnerGroup
  labels:
    - linux
    - amd64
    - ubuntu24.04
  namePrefix: ubuntu
  count: 3

mount:
  source: /Volumes/cache
  target: /Volumes/cache

runtime:
  pollIntervalSeconds: 10
```

### Runner image contract

Orbital builds a runner image on startup using `docker.sourceImage` as the Dockerfile `FROM` image.
The generated image is tagged with `docker.runnerImageName` and is then used for `docker run`.

Your `docker.sourceImage` is responsible for providing the environment your jobs need.
Orbital is responsible for adding the GitHub Actions runner itself, installing the minimum runner download dependencies, and writing the startup entrypoint.

Current assumptions and limits:

- build and run always use the same `docker.context`
- Orbital currently targets Linux runner containers
- runner archive selection is resolved from the inspected OS/CPU architecture of `docker.sourceImage`
- Orbital currently supports source images whose package manager is one of: `apt-get`, `apk`, `dnf`, or `yum`
- distroless, scratch, and other images without a supported package manager are currently not supported
- `docker.sourceImage` should already be compatible with the GitHub Actions runner runtime requirements

Orbital determines the runner archive platform by running `docker image inspect` against
`docker.sourceImage` on the configured Docker context and uses that result when building the
final runner image.

Orbital passes the following environment variables to each runner container:

- `GITHUB_URL`
- `RUNNER_NAME`
- `RUNNER_TOKEN`

Orbital also passes the following optional environment variables when configured:

- `RUNNER_GROUP`
- `RUNNER_LABELS`

See `example/Dockerfile` for a reference implementation of a compatible
source image.

### Configuration fields

- `docker.context`: Name of the Docker context Orbital should use.
- `docker.sourceImage`: Existing Docker image used as the `FROM` image when Orbital builds the runner image.
- `docker.runnerImageName`: Tag name for the runner image built by Orbital and used for runner containers.
- `github.org`: GitHub organization name.
- `github.appId`: GitHub App ID.
- `github.installationId`: GitHub App installation ID.
- `github.pem`: Path to the GitHub App private key (`.pem`).
- `runner.group`: Optional runner group name. When set, Orbital passes it to the runner registration command.
- `runner.labels`: Optional custom labels assigned to the runner. When set, Orbital passes them to the runner registration command.
- `runner.namePrefix`: Prefix used for runner names.
- `runner.count`: Number of runner containers Orbital should keep running.
- `mount.source`: Host path mounted into the runner container.
- `mount.target`: Destination path inside the runner container.
- `runtime.pollIntervalSeconds`: Polling interval, in seconds, for runner reconciliation.

### GitHub App permissions

Your GitHub App must be installed on the target organization and be able to create self-hosted runner registration tokens for that organization. In practice, configure the app with the permissions required for organization self-hosted runner administration before using Orbital.

## Quick start

Build the binary:

```bash
go build -o ./build/orbital ./app
```

Create a configuration file, for example:

```bash
cp ./example/config.yaml ./config.yaml
```

Then start Orbital:

```bash
./build/orbital start -c ./config.yaml
```

## What happens when it starts

When Orbital starts, it will:

1. Load the configuration file.
2. Check whether the configured Docker context is reachable.
3. Generate GitHub runner registration tokens through your GitHub App.
4. Build the runner image from `docker.sourceImage`.
5. Start and maintain the configured number of runner containers.
6. Keep polling at the configured interval and recreate stopped containers if needed.
7. Stop and remove managed containers when the process is terminated, such as with `Ctrl+C`.

## Notes

- Managed container names are derived from a sanitized `docker.runnerImageName` and a 1-based index.
- Runner names are generated from `runner.namePrefix` and a 1-based index, for example `ubuntu-1`, `ubuntu-2`, and `ubuntu-3`.
- Avoid including a trailing hyphen in `runner.namePrefix`, otherwise generated runner names become harder to read, such as `ubuntu--1`.
- `runner.group` and `runner.labels` are optional. If omitted, Orbital registers the runner without those options.
- Orbital does not start container runtimes for you. Ensure the target Docker Engine behind the configured context is already running before starting Orbital.

## Motivation

I wanted to build an ephemeral GitHub Actions self-hosted runner environment for Android app builds on an arm64 macOS machine.
Since the officially distributed `aapt2` binary is x86, Rosetta 2 is required. The most straightforward option would be to run the self-hosted runner inside a macOS VM, but due to limitations of the Virtualization Framework, only up to two macOS VMs can be launched.

To work around this, I decided to use Lima/Colima to run a Linux VM that can forward execution to Rosetta 2, and then run x86 containers on top of it.
Based on that setup, I created Orbital as a way to launch and maintain GitHub Actions self-hosted runners on the Docker context provided by Lima/Colima.

