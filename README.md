# steadybit-debug

steadybit-debug collects data from installed Steadybit platforms and
agents to aid in customer support. It helps shorten feedback cycles and
avoids frequent back and forth between Steadybit and its customers.

## Prerequisites

- Kubectl needs to be available on the `$PATH` and configured.

## Installation

steadybit-debug is available on Linux, macOS and Windows platforms.

 - Binaries for Linux, Windows and macOS are available as tarballs in the [releases](https://github.com/steadybit/steadybit-debug/releases) page.
 - Via Homebrew for macOS or LinuxBrew for Linux
    ```
    brew tap steadybit/homebrew-steadybit-debug
    brew install steadybit-debug
    ``` 

## Configuration

steadybit-debug has sensible defaults that will work out-of-the-box for
users of our Helm charts that haven't renamed the namespaces, deployments
or daemon sets. If you made changes, you could configure steadybit-debug
to support your specific setup.

### Via Command-Line Arguments

Command-line arguments can be used to change the most common configuration
options. The following snippet show how to change the references to
Kubernetes workloads. Refer to `steadybit-debug --help` for more
information.

```
steadybit-debug --platform-deployment platform \
   --platform-namespace platform \
   --agent-namespace steadybit-agent-to-prod
```

### Via Configuration Files

Configuration is supported through a file called `steadybit-debug.yml`
existing within your current working directory.

```yaml
platform:
  namespace: chaos-eng
  deployment: platform
agent:
  namespace: chaos-eng
```

To learn more about all the available configuration options please inspect
the Go `Config` [struct definition](https://github.com/steadybit/steadybit-debug/blob/main/config/config.go#L11).

## Execution

You execute the tool via `steadybit-debug`. Once executed, you will find that the
command collects debugging information within the current working directory.
Please send the generated .tar.gz file to your Steadybit contacts.

![Image showing the execution of the steadybit-debug command on a terminal. Log lines are giving an overview about the expected behavior of the tool.](./example-execution.png)

## Collected Information

This tool gathers data from your Kubernetes server and the admin endpoints of
the Steadybit platform and agents. The following listing shows an overview of the
generated files (as of 2022-07-28). Feel free to take a closer look at the data
it collected for your installation!

```
.
├── agent
│   ├── config.yaml
│   ├── description.txt
│   └── pods
│       └── steadybit-agent-h94gl
│           ├── config.yml
│           ├── connection_stats.yml
│           ├── description.txt
│           ├── env.yml
│           ├── health.yml
│           ├── info.yml
│           ├── logs-previous.txt
│           ├── logs.txt
│           ├── prometheus_metrics.0.txt
│           ├── prometheus_metrics.1.txt
│           ├── prometheus_metrics.2.txt
│           ├── prometheus_metrics.3.txt
│           ├── prometheus_metrics.4.txt
│           ├── prometheus_metrics.5.txt
│           ├── prometheus_metrics.6.txt
│           ├── prometheus_metrics.7.txt
│           ├── prometheus_metrics.8.txt
│           ├── prometheus_metrics.9.txt
│           ├── self_test.yml
│           ├── target_stats.yml
│           ├── threaddump.yml
│           ├── top.0.txt
│           ├── top.1.txt
│           ├── top.2.txt
│           ├── top.3.txt
│           ├── top.4.txt
│           ├── top.5.txt
│           ├── top.6.txt
│           ├── top.7.txt
│           ├── top.8.txt
│           └── top.9.txt
├── debugging_config.yaml
└── platform
    ├── config.yaml
    ├── description.txt
    └── pods
        └── platform-77666d8ff9-pnncs
            ├── config.yml
            ├── description.txt
            ├── env.yml
            ├── health.yml
            ├── info.yml
            ├── logs-previous.txt
            ├── logs.txt
            ├── prometheus_metrics.0.txt
            ├── prometheus_metrics.1.txt
            ├── prometheus_metrics.2.txt
            ├── prometheus_metrics.3.txt
            ├── prometheus_metrics.4.txt
            ├── prometheus_metrics.5.txt
            ├── prometheus_metrics.6.txt
            ├── prometheus_metrics.7.txt
            ├── prometheus_metrics.8.txt
            ├── prometheus_metrics.9.txt
            ├── target_stats.yml
            ├── threaddump.yml
            ├── top.0.txt
            ├── top.1.txt
            ├── top.2.txt
            ├── top.3.txt
            ├── top.4.txt
            ├── top.5.txt
            ├── top.6.txt
            ├── top.7.txt
            ├── top.8.txt
            └── top.9.txt

6 directories, 65 files
```