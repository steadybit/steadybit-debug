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

The fan-out is bounded by the semaphores in `limit/limit.go` (`--max-concurrency`), because every collected pod
spawns a handful of `kubectl`/`curl` child processes and an unbounded fan-out over a large cluster is what made
the tool exhaust the memory of the machine it runs on. Acquisition order is `Namespaces` → `Items`/`Nodes` (pod
and node callbacks) → `Commands` (one per external execution); acquiring an earlier level while holding a later
one deadlocks, and each level is acquired in exactly one place. Nodes have their own semaphore because
semaphores queue FIFO and a big cluster's node goroutines would otherwise delay every other collector.
`limit.Configure` is called from `main` before the collectors start. `kubectl port-forward` is deliberately
unbounded — see the note on `k8s.PreparePortforwarding`.

Anything that can block forever needs a bound, since it now occupies a slot: `AddCommandOutputOptions.Timeout`
for child processes (started only after the slot is acquired, so queueing does not eat the budget — pass a
timeout instead of a deadline-carrying context), `stallTimeout` for HTTP responses, and the tools inside
`kubectl debug` ephemeral containers carry their own limits (`k8s.curlArgs`) so their diagnostic output finishes
before the outer `ephemeralContainerTimeout` kills it.

### Configuration (`config/config.go`)

One `Config` struct is the single source of truth for defaults, the `steadybit-debug.yml` file format
(`yaml:` tags) and the CLI flags (`long:`/`short:` tags parsed by `jessevdk/go-flags`). Layering: hardcoded
defaults in `newConfig()` → `steadybit-debug.yml` in the CWD → CLI flags. Adding an option means adding one
field with both tag sets — nothing else to register.

`cfg.Kubernetes.Client()` prefers in-cluster config and falls back to the kubeconfig path. It is called
repeatedly by the collectors but creates the clientset only once (package-level cache guarded by a mutex, only
successful creations are cached) — hence the raised `QPS`/`Burst`, since all collectors share one rate limiter.

### Output layer (`output/`)

Everything written into the archive goes through one of these:

- `AddCommandOutput` (`cmd.go`) — runs an external command with stdout and stderr going straight into the output
  file (never buffered — pod logs do not fit in memory). With `Executions > 1` the `OutputPath` **must** contain
  a `%d` for the execution index (used for repeated `kubectl top` / prometheus samples); the whole series holds
  one `limit.Commands` slot, otherwise `DelayBetweenExecutions` would no longer be the interval the samples were
  taken at. `Timeout` bounds a single execution.
- `AddHttpOutput` / `DoHttp` (`http.go`) — direct HTTP client with mTLS support from `cfg.Tls` (built once and
  cached), and an automatic retry over HTTPS when a response indicates "Client sent an HTTP request to an HTTPS
  server" (`errHttpsRequired`) — that indicator arrives with a `400`, so `doHttpRequest` checks the body before
  turning a status into an error. `doHttpRequest` owns the request lifecycle and passes the body to a callback;
  the `progressReader` cancels a request that stops making progress for `stallTimeout`, which bounds a dead
  port-forward without limiting how long a large response may take. `AddHttpOutput` streams the body into the
  file and only pretty-prints JSON below `maxBytesForJsonFormatting`; `DoHttp` (for callers that parse the
  response) errors out beyond `maxBytesForInMemoryResponse` rather than returning a truncated body. Keep it that
  way — discovery responses reach hundreds of megabytes.
- `DownloadOutput` (`download.go`) — `curl` download for binary payloads (platform DB export), writes a
  sibling `.log`.
- `AddJsonOutput` (`json.go`), `WriteToFile` (`output.go`).

The first three share `addOutputFile` (`output.go`), which owns the file format (header with the executed
command and start time, then the payload, then any error and the total execution time) and hands out the
`*os.File` the payload is streamed into. Add new collector output on top of it rather than assembling a file
yourself — and acquire `limit.Commands` in the entry point, not around the individual file.

### Talking to pods

`k8s/k8s.go` is the shared toolbox. `PreparePortforwarding` starts `kubectl port-forward` with a random local
port and scrapes the chosen port out of stdout; callers must `defer KillProcess(cmd, podConfig)`. Prefer
`AddPodHttpMultipleEndpointOutput` when hitting several endpoints on the same port so one forward is reused.
Connectivity tests (`ConnectionTester` in `connectiontest.go`) run *inside* the target pod via a `kubectl debug`
ephemeral container using `agent.curlImage`, and are executed in it with `kubectl exec` — one container serves
every test of a pod, because Kubernetes never removes an ephemeral container from a pod again. They spend nearly
all their time waiting, so `agent.runConnectionTests` runs them in parallel, bounded per pod. `ephemeralContainerName`
must stay unique: `ephemeralContainers` is patched with a merge key on the name, so a reused name would address an
earlier container instead of adding a new one.

A debug container inherits **nothing** from the container it targets, which is why it cannot even start in a pod
demanding a non-root user — the kubelet has no numeric UID to verify. `customizationForContainer` therefore hands
kubectl the target's own security context, volume mounts and agent-key env entry via `--custom` (minus
`readOnlyRootFilesystem`, since the tools may need to write), and `platformTlsArgs`/`extensionTlsArgs` turn the
agent's own TLS environment into curl flags so a test presents what the agent presents. The target is
`agent.identifyAgentContainer`, **not** `Containers[0]` — that one is the autoregistration sidecar, which has none
of this.

Credentials must never reach the archive, which records every executed command: only env entries with a literal
value are read (the client key password comes from a secret), and the agent key is referenced by name for the
container's shell to expand, with its request printing only a status code because `curl -v` prints the headers it
sends.

Two tests were removed rather than left failing: traceroute needs `CAP_NET_RAW`, which the chart's agents drop,
and websocat is only published for amd64 — curl speaks `wss://` itself, so it does that test now.

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
