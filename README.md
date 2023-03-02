[![CI](https://github.com/steadybit/steadybit-debug/actions/workflows/test.yml/badge.svg)](https://github.com/steadybit/steadybit-debug/actions/workflows/test.yml)
[![Release](https://github.com/steadybit/steadybit-debug/actions/workflows/release.yml/badge.svg)](https://github.com/steadybit/steadybit-debug/actions/workflows/release.yml)

# steadybit-debug

steadybit-debug collects data from installed Steadybit platforms and
agents to aid in customer support. It helps shorten feedback cycles and
avoids frequent back and forth between Steadybit and its customers.

## Prerequisites

- `kubectl` needs to be available on the `$PATH` and configured with the correct context.
- `curl` needs to be available on the `$PATH`.


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

## MTLS Support for extensions
If you configured your extensions to use mTLS between agent and extension, you need to provide the cert and key files to steadybit-debug. You can do this by adding the following to your `steadybit-debug.yml` file:

```yaml
platform:
  namespace: chaos-eng
  deployment: platform
agent:
  namespace: chaos-eng
tls:
   certChainFile: /Path/to/tls.crt
   certKeyFile: /Path/to/tls.key
```
or provide the cert and key via command line arguments:

```
steadybit-debug --platform-deployment platform \
   --platform-namespace platform \
   --agent-namespace steadybit-agent-to-prod \
   --cert-chain-file /Path/to/tls.crt \
   --cert-key-file= /Path/to/tls.key
```
## Execution

You execute the tool via `steadybit-debug`. Once executed, you will find that the
command collects debugging information within the current working directory.
Please send the generated .tar.gz file to your Steadybit contacts.

![Image showing the execution of the steadybit-debug command on a terminal. Log lines are giving an overview about the expected behavior of the tool.](./example-execution.png)

## Collected Information

This tool gathers data from your Kubernetes server and the admin endpoints of
the Steadybit platform, agents and extensions. The following listing shows an overview of the
generated files (as of 2022-07-28). Feel free to take a closer look at the data
it collected for your installation!

```
.
├── agent
│         ├── config.yaml
│         ├── description.txt
│         └── pods
│             ├── steadybit-agent-gpznj
│             │         ├── actions_metadata.yml
│             │         ├── config.yml
│             │         ├── connection_stats.yml
│             │         ├── connections.yml
│             │         ├── description.txt
│             │         ├── discovery_info.yml
│             │         ├── env.yml
│             │         ├── health.yml
│             │         ├── info.yml
│             │         ├── logs.txt
│             │         ├── logs_previous.txt
│             │         ├── prometheus_metrics.0.txt
│             │         ├── prometheus_metrics.1.txt
│             │         ├── prometheus_metrics.2.txt
│             │         ├── prometheus_metrics.3.txt
│             │         ├── prometheus_metrics.4.txt
│             │         ├── prometheus_metrics.5.txt
│             │         ├── prometheus_metrics.6.txt
│             │         ├── prometheus_metrics.7.txt
│             │         ├── prometheus_metrics.8.txt
│             │         ├── prometheus_metrics.9.txt
│             │         ├── self_test.yml
│             │         ├── target_stats.yml
│             │         ├── target_type_description.yml
│             │         ├── targets.yml
│             │         ├── threaddump.yml
│             │         ├── top.0.txt
│             │         ├── top.1.txt
│             │         ├── top.2.txt
│             │         ├── top.3.txt
│             │         ├── top.4.txt
│             │         ├── top.5.txt
│             │         ├── top.6.txt
│             │         ├── top.7.txt
│             │         ├── top.8.txt
│             │         └── top.9.txt
│             ├── steadybit-agent-hv7wh
│             │         ├── actions_metadata.yml
│             │         ├── config.yml
│             │         ├── connection_stats.yml
│             │         ├── connections.yml
│             │         ├── description.txt
│             │         ├── discovery_info.yml
│             │         ├── env.yml
│             │         ├── health.yml
│             │         ├── info.yml
│             │         ├── logs.txt
│             │         ├── logs_previous.txt
│             │         ├── prometheus_metrics.0.txt
│             │         ├── prometheus_metrics.1.txt
│             │         ├── prometheus_metrics.2.txt
│             │         ├── prometheus_metrics.3.txt
│             │         ├── prometheus_metrics.4.txt
│             │         ├── prometheus_metrics.5.txt
│             │         ├── prometheus_metrics.6.txt
│             │         ├── prometheus_metrics.7.txt
│             │         ├── prometheus_metrics.8.txt
│             │         ├── prometheus_metrics.9.txt
│             │         ├── self_test.yml
│             │         ├── target_stats.yml
│             │         ├── target_type_description.yml
│             │         ├── targets.yml
│             │         ├── threaddump.yml
│             │         ├── top.0.txt
│             │         ├── top.1.txt
│             │         ├── top.2.txt
│             │         ├── top.3.txt
│             │         ├── top.4.txt
│             │         ├── top.5.txt
│             │         ├── top.6.txt
│             │         ├── top.7.txt
│             │         ├── top.8.txt
│             │         └── top.9.txt
│             └── steadybit-agent-nbl9x
│                 ├── actions_metadata.yml
│                 ├── config.yml
│                 ├── connection_stats.yml
│                 ├── connections.yml
│                 ├── description.txt
│                 ├── discovery_info.yml
│                 ├── env.yml
│                 ├── health.yml
│                 ├── info.yml
│                 ├── logs.txt
│                 ├── logs_previous.txt
│                 ├── prometheus_metrics.0.txt
│                 ├── prometheus_metrics.1.txt
│                 ├── prometheus_metrics.2.txt
│                 ├── prometheus_metrics.3.txt
│                 ├── prometheus_metrics.4.txt
│                 ├── prometheus_metrics.5.txt
│                 ├── prometheus_metrics.6.txt
│                 ├── prometheus_metrics.7.txt
│                 ├── prometheus_metrics.8.txt
│                 ├── prometheus_metrics.9.txt
│                 ├── self_test.yml
│                 ├── target_stats.yml
│                 ├── target_type_description.yml
│                 ├── targets.yml
│                 ├── threaddump.yml
│                 ├── top.0.txt
│                 ├── top.1.txt
│                 ├── top.2.txt
│                 ├── top.3.txt
│                 ├── top.4.txt
│                 ├── top.5.txt
│                 ├── top.6.txt
│                 ├── top.7.txt
│                 ├── top.8.txt
│                 └── top.9.txt
├── debugging_config.yaml
├── extensions
│         └── steadybit-extension
│             ├── steadybit-extension-aws
│             │         ├── config.yaml
│             │         ├── description.txt
│             │         └── pods
│             │             └── steadybit-extension-aws-544bdd77f9-8vrlj
│             │                 ├── config.yml
│             │                 ├── description.txt
│             │                 ├── http
│             │                 │         ├── GET__.yml
│             │                 │         ├── GET__common_discovery_attribute-descriptions.yml
│             │                 │         ├── GET__rds_instance_discovery.yml
│             │                 │         ├── GET__rds_instance_discovery_attribute-descriptions.yml
│             │                 │         ├── GET__rds_instance_discovery_discovered-targets.yml
│             │                 │         └── GET__rds_instance_discovery_target-description.yml
│             │                 ├── logs.txt
│             │                 ├── logs_previous.txt
│             │                 ├── top.0.txt
│             │                 ├── top.1.txt
│             │                 ├── top.2.txt
│             │                 ├── top.3.txt
│             │                 ├── top.4.txt
│             │                 ├── top.5.txt
│             │                 ├── top.6.txt
│             │                 ├── top.7.txt
│             │                 ├── top.8.txt
│             │                 └── top.9.txt
│             ├── steadybit-extension-kubernetes
│             │         ├── config.yaml
│             │         ├── description.txt
│             │         └── pods
│             │             └── steadybit-extension-kubernetes-697886cd66-56bzk
│             │                 ├── config.yml
│             │                 ├── description.txt
│             │                 ├── http
│             │                 │         ├── GET__.yml
│             │                 │         ├── GET__deployment_attack_rollout-restart.yml
│             │                 │         └── GET__deployment_check_rollout-status.yml
│             │                 ├── logs.txt
│             │                 ├── logs_previous.txt
│             │                 ├── top.0.txt
│             │                 ├── top.1.txt
│             │                 ├── top.2.txt
│             │                 ├── top.3.txt
│             │                 ├── top.4.txt
│             │                 ├── top.5.txt
│             │                 ├── top.6.txt
│             │                 ├── top.7.txt
│             │                 ├── top.8.txt
│             │                 └── top.9.txt
│             ├── steadybit-extension-postman
│             │         ├── config.yaml
│             │         ├── description.txt
│             │         └── pods
│             │             └── steadybit-extension-postman-7797cc5b7d-cntrf
│             │                 ├── config.yml
│             │                 ├── description.txt
│             │                 ├── http
│             │                 │         ├── GET__.yml
│             │                 │         └── GET__postman_collection_run.yml
│             │                 ├── logs.txt
│             │                 ├── logs_previous.txt
│             │                 ├── top.0.txt
│             │                 ├── top.1.txt
│             │                 ├── top.2.txt
│             │                 ├── top.3.txt
│             │                 ├── top.4.txt
│             │                 ├── top.5.txt
│             │                 ├── top.6.txt
│             │                 ├── top.7.txt
│             │                 ├── top.8.txt
│             │                 └── top.9.txt
│             └── steadybit-extension-prometheus
│                 ├── config.yaml
│                 ├── description.txt
│                 └── pods
│                     └── steadybit-extension-prometheus-6c977768d6-pk9ff
│                         ├── config.yml
│                         ├── description.txt
│                         ├── http
│                         │         ├── GET__.yml
│                         │         ├── GET__prometheus_instance_discovery.yml
│                         │         ├── GET__prometheus_instance_discovery_attribute-descriptions.yml
│                         │         ├── GET__prometheus_instance_discovery_discovered-targets.yml
│                         │         ├── GET__prometheus_instance_discovery_target-description.yml
│                         │         └── GET__prometheus_metrics.yml
│                         ├── logs.txt
│                         ├── logs_previous.txt
│                         ├── top.0.txt
│                         ├── top.1.txt
│                         ├── top.2.txt
│                         ├── top.3.txt
│                         ├── top.4.txt
│                         ├── top.5.txt
│                         ├── top.6.txt
│                         ├── top.7.txt
│                         ├── top.8.txt
│                         └── top.9.txt
├── nodes
│         ├── fargate-ip-10-10-102-215.eu-central-1.compute.internal
│         │         ├── config.yaml
│         │         └── description.txt
│         ├── fargate-ip-10-10-85-138.eu-central-1.compute.internal
│         │         ├── config.yaml
│         │         └── description.txt
│         ├── ip-10-10-84-18.eu-central-1.compute.internal
│         │         ├── config.yaml
│         │         └── description.txt
│         ├── ip-10-10-93-233.eu-central-1.compute.internal
│         │         ├── config.yaml
│         │         └── description.txt
│         └── ip-10-10-94-179.eu-central-1.compute.internal
│             ├── config.yaml
│             └── description.txt
└── platform
    ├── config.yaml
    ├── database.zip
    ├── database.zip.log
    ├── description.txt
    └── pods
        ├── platform-695665d875-6pdm9
        │         ├── config.yml
        │         ├── configprops.yml
        │         ├── description.txt
        │         ├── env.yml
        │         ├── health.yml
        │         ├── info.yml
        │         ├── logs.txt
        │         ├── logs_previous.txt
        │         ├── prometheus_metrics.0.txt
        │         ├── prometheus_metrics.1.txt
        │         ├── prometheus_metrics.2.txt
        │         ├── prometheus_metrics.3.txt
        │         ├── prometheus_metrics.4.txt
        │         ├── prometheus_metrics.5.txt
        │         ├── prometheus_metrics.6.txt
        │         ├── prometheus_metrics.7.txt
        │         ├── prometheus_metrics.8.txt
        │         ├── prometheus_metrics.9.txt
        │         ├── target_stats.yml
        │         ├── threaddump.yml
        │         ├── top.0.txt
        │         ├── top.1.txt
        │         ├── top.2.txt
        │         ├── top.3.txt
        │         ├── top.4.txt
        │         ├── top.5.txt
        │         ├── top.6.txt
        │         ├── top.7.txt
        │         ├── top.8.txt
        │         └── top.9.txt
        └── platform-695665d875-gt4v5
            ├── config.yml
            ├── configprops.yml
            ├── description.txt
            ├── env.yml
            ├── health.yml
            ├── info.yml
            ├── logs.txt
            ├── logs_previous.txt
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

34 directories, 266 files
```
