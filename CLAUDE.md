# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`steadybit-debug` is a single-binary Go CLI that collects support/diagnostic data from an installed Steadybit
setup (platform, platform-port-splitter, agent, extensions, Kubernetes nodes) into a timestamped directory and
packs it into a `.tar.gz` for the customer to send to Steadybit support.

It is a *read-only* collector: it shells out to `kubectl` and `curl`, port-forwards to pods and hits their admin
endpoints. Both binaries must be on `$PATH` at runtime.

## Commands

```bash
go build .                # build the binary
go test ./...             # run tests (no test files exist yet)
go fmt ./...              # required before committing
go mod download

./steadybit-debug --help  # all CLI flags are derived from the Config struct tags
```

There is no linter configured; CI (`.github/workflows/test.yml`) runs `go build .`, `go test ./...` and a Snyk
scan. Releases are built by goreleaser on tag (`.goreleaser.yaml`), published to GitHub releases and a Homebrew
tap.

## Architecture

Flow in `main.go`: `config.GetConfig()` → `output.AddOutputDirectory` (creates
`steadybit-debug-<unix-ts>/`) → tee logging into `log.txt` → dump effective config →
`debugrun.GatherInformation` → `output.ZipOutputDirectory` → delete the directory unless `--no-cleanup`.

`debugrun/run.go` fans out into five concurrent collectors, one per subsystem:

| Package | Collects from |
| --- | --- |
| `platform` | platform Deployment + Spring Boot actuator endpoints on port 9090; optional DB export |
| `platform` (`platformPortSplitter.go`) | platform-port-splitter Deployment |
| `agent` | agent StatefulSet, its admin endpoints, and connectivity tests towards the platform |
| `extensions` | every namespace, discovering extensions by annotation |
| `k8s` (`nodes.go`) | all cluster nodes (describe + yaml) |

Concurrency is `sync.WaitGroup` everywhere; `k8s.ForEachPod`, `ForEachPodViaMapSelector` and `ForEachNode` run
their callback in parallel per item. Collectors never abort the run — failures are logged (usually at
`Debug`/`Warn`) and the missing data simply doesn't appear in the archive.

### Configuration (`config/config.go`)

One `Config` struct is the single source of truth for defaults, the `steadybit-debug.yml` file format
(`yaml:` tags) and the CLI flags (`long:`/`short:` tags parsed by `jessevdk/go-flags`). Layering: hardcoded
defaults in `newConfig()` → `steadybit-debug.yml` in the CWD → CLI flags. Adding an option means adding one
field with both tag sets — nothing else to register.

`cfg.Kubernetes.Client()` prefers in-cluster config and falls back to the kubeconfig path; it is called
repeatedly by collectors rather than cached.

### Output layer (`output/`)

Everything written into the archive goes through one of these, and each prepends a header with the executed
command, start time, any error, and total execution time:

- `AddCommandOutput` (`cmd.go`) — runs an external command, captures combined output. With `Executions > 1`
  the `OutputPath` **must** contain a `%d` for the execution index (used for repeated `kubectl top` /
  prometheus samples).
- `AddHttpOutput` / `DoHttp` (`http.go`) — direct HTTP client with mTLS support from `cfg.Tls`, and an
  automatic retry over HTTPS when a response indicates "Client sent an HTTP request to an HTTPS server".
- `DownloadOutput` (`download.go`) — `curl` download for binary payloads (platform DB export), writes a
  sibling `.log`.
- `AddJsonOutput` (`json.go`), `WriteToFile` (`output.go`).

### Talking to pods

`k8s/k8s.go` is the shared toolbox. `PreparePortforwarding` starts `kubectl port-forward` with a random local
port and scrapes the chosen port out of stdout; callers must `defer KillProcess(cmd, podConfig)`. Prefer
`AddPodHttpMultipleEndpointOutput` when hitting several endpoints on the same port so one forward is reused.
Connectivity tests (`AddHttpConnectionTest`, `AddTraceroute…`, `AddWebsocket…`) run *inside* the target pod via
`kubectl debug` ephemeral containers using the images configured under `agent.*Image`.

### Extension discovery (`extensions/`)

Extensions are found by scanning all namespaces for Services and DaemonSets carrying the
`steadybit.com/extension-auto-registration` annotation (legacy: `…/extension-auto-discovery`). Ports and TLS
come from that annotation's JSON, else the `STEADYBIT_EXTENSION_PORT` env var, else `8080`.
`extension.go:TraverseExtensionEndpoints` then crawls the extension: `GET /` is parsed as a combined
ActionKit/DiscoveryKit/EventKit list response and every advertised action/discovery/target-type/event-listener
path is fetched, with discovery descriptions followed one level deeper to also grab the `discover` endpoint.
Response shapes come from the `action-kit`, `discovery-kit` and `event-kit` Go API modules — bump those in
`go.mod` when new endpoint kinds need to be crawled.

## Conventions

- Go files carry either the SPDX header pair or the `Copyright <year> steadybit GmbH` block comment; match the
  neighbouring files when adding a file.
- Logging is `zerolog` via the package-level `log`. `Info` goes to the console *and* `log.txt`, `Debug` only to
  `log.txt`. Expected-to-fail collection steps log at `Debug`; pass `LogError: true` to
  `AddCommandOutputOptions` only for steps whose failure is genuinely noteworthy.
- When the README's "Collected Information" tree or config docs go stale after adding a collector, update
  `README.md` too — it's the customer-facing reference.
