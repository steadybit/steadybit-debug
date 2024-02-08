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
  exportDatabase: false
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
  exportDatabase: false
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

## Database Export support
If you need to export the database of your platform, you can do this by adding the following to your `steadybit-debug.yml` file:

```yaml
platform:
  namespace: chaos-eng
  deployment: platform
  exportDatabase: true
```
This is disabled by default.

## Execution

You execute the tool via `steadybit-debug`. Once executed, you will find that the
command collects debugging information within the current working directory.
Please send the generated .tar.gz file to your Steadybit contacts.

![Image showing the execution of the steadybit-debug command on a terminal. Log lines are giving an overview about the expected behavior of the tool.](./example-execution.png)

## Collected Information

This tool gathers data from your Kubernetes server and the admin endpoints of
the Steadybit platform, agents and extensions. The following listing shows an overview of the
generated files (as of 2024-02-08). Feel free to take a closer look at the data
it collected for your installation!

```
.
├── agent
│   ├── config.yaml
│   ├── description.txt
│   └── pods
│       └── steadybit-agent-0
│           ├── actions_metadata.yml
│           ├── advice_definition.yml
│           ├── config.yml
│           ├── description.txt
│           ├── discovery_info.yml
│           ├── enrichtment_rules.yml
│           ├── env.yml
│           ├── extension_connection_test_0.txt
│           ├── extension_connection_test_1.txt
│           ├── extension_connection_test_10.txt
│           ├── extension_connection_test_11.txt
│           ├── extension_connection_test_12.txt
│           ├── extension_connection_test_13.txt
│           ├── extension_connection_test_14.txt
│           ├── extension_connection_test_15.txt
│           ├── extension_connection_test_16.txt
│           ├── extension_connection_test_17.txt
│           ├── extension_connection_test_18.txt
│           ├── extension_connection_test_2.txt
│           ├── extension_connection_test_3.txt
│           ├── extension_connection_test_4.txt
│           ├── extension_connection_test_5.txt
│           ├── extension_connection_test_6.txt
│           ├── extension_connection_test_7.txt
│           ├── extension_connection_test_8.txt
│           ├── extension_connection_test_9.txt
│           ├── health.yml
│           ├── info.yml
│           ├── logs.txt
│           ├── logs_previous.txt
│           ├── platform_connection_test.txt
│           ├── platform_traceroute_test.txt
│           ├── platform_websocat_connection_test.txt
│           ├── platform_websocket_http1_connection_test.txt
│           ├── platform_websocket_http2_connection_test.txt
│           ├── prometheus_metrics.0.txt
│           ├── prometheus_metrics.1.txt
│           ├── prometheus_metrics.2.txt
│           ├── prometheus_metrics.3.txt
│           ├── prometheus_metrics.4.txt
│           ├── prometheus_metrics.5.txt
│           ├── prometheus_metrics.6.txt
│           ├── prometheus_metrics.7.txt
│           ├── prometheus_metrics.8.txt
│           ├── prometheus_metrics.9.txt
│           ├── target_stats.yml
│           ├── target_type_description.yml
│           ├── targets.yml
│           ├── threaddump.yml
│           ├── top.0.txt
│           ├── top.1.txt
│           ├── top.2.txt
│           ├── top.3.txt
│           ├── top.4.txt
│           ├── top.5.txt
│           ├── top.6.txt
│           ├── top.7.txt
│           ├── top.8.txt
│           └── top.9.txt
├── debugging_config.yaml
├── extensions
│   └── steadybit-agent
│       ├── steadybit-agent-extension-container
│       │   ├── config.yaml
│       │   ├── description.txt
│       │   └── pods
│       │       ├── steadybit-agent-extension-container-hdmb6
│       │       │   ├── config.yml
│       │       │   ├── description.txt
│       │       │   ├── http
│       │       │   │   ├── GET__.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.container_discovery.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.container_discovery_discovered-targets.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.container_discovery_target-description.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.fill_disk.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_bandwidth.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_blackhole.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_block_dns.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_delay.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_package_corruption.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_package_loss.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.pause.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stop.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stress_cpu.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stress_io.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stress_mem.yml
│       │       │   │   └── GET__discovery_attributes.yml
│       │       │   ├── logs.txt
│       │       │   ├── logs_previous.txt
│       │       │   ├── top.0.txt
│       │       │   ├── top.1.txt
│       │       │   └── top.2.txt
│       │       ├── steadybit-agent-extension-container-x6hq6
│       │       │   ├── config.yml
│       │       │   ├── description.txt
│       │       │   ├── http
│       │       │   │   ├── GET__.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.container_discovery.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.container_discovery_discovered-targets.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.container_discovery_target-description.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.fill_disk.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_bandwidth.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_blackhole.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_block_dns.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_delay.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_package_corruption.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.network_package_loss.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.pause.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stop.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stress_cpu.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stress_io.yml
│       │       │   │   ├── GET__com.steadybit.extension_container.stress_mem.yml
│       │       │   │   └── GET__discovery_attributes.yml
│       │       │   ├── logs.txt
│       │       │   ├── logs_previous.txt
│       │       │   ├── top.0.txt
│       │       │   ├── top.1.txt
│       │       │   └── top.2.txt
│       │       └── steadybit-agent-extension-container-xccpk
│       │           ├── config.yml
│       │           ├── description.txt
│       │           ├── http
│       │           │   ├── GET__.yml
│       │           │   ├── GET__com.steadybit.extension_container.container_discovery.yml
│       │           │   ├── GET__com.steadybit.extension_container.container_discovery_discovered-targets.yml
│       │           │   ├── GET__com.steadybit.extension_container.container_discovery_target-description.yml
│       │           │   ├── GET__com.steadybit.extension_container.fill_disk.yml
│       │           │   ├── GET__com.steadybit.extension_container.network_bandwidth.yml
│       │           │   ├── GET__com.steadybit.extension_container.network_blackhole.yml
│       │           │   ├── GET__com.steadybit.extension_container.network_block_dns.yml
│       │           │   ├── GET__com.steadybit.extension_container.network_delay.yml
│       │           │   ├── GET__com.steadybit.extension_container.network_package_corruption.yml
│       │           │   ├── GET__com.steadybit.extension_container.network_package_loss.yml
│       │           │   ├── GET__com.steadybit.extension_container.pause.yml
│       │           │   ├── GET__com.steadybit.extension_container.stop.yml
│       │           │   ├── GET__com.steadybit.extension_container.stress_cpu.yml
│       │           │   ├── GET__com.steadybit.extension_container.stress_io.yml
│       │           │   ├── GET__com.steadybit.extension_container.stress_mem.yml
│       │           │   └── GET__discovery_attributes.yml
│       │           ├── logs.txt
│       │           ├── logs_previous.txt
│       │           ├── top.0.txt
│       │           ├── top.1.txt
│       │           └── top.2.txt
│       ├── steadybit-agent-extension-host
│       │   ├── config.yaml
│       │   ├── description.txt
│       │   └── pods
│       │       ├── steadybit-agent-extension-host-74njx
│       │       │   ├── config.yml
│       │       │   ├── description.txt
│       │       │   ├── http
│       │       │   │   ├── GET__.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.fill_disk.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.host_discovery.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.host_discovery_discovered-targets.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.host_discovery_target-description.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_bandwidth.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_blackhole.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_block_dns.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_delay.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_package_corruption.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_package_loss.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.shutdown.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stop-process.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stress-cpu.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stress-io.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stress-mem.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.timetravel.yml
│       │       │   │   └── GET__discovery_attributes.yml
│       │       │   ├── logs.txt
│       │       │   ├── logs_previous.txt
│       │       │   ├── top.0.txt
│       │       │   ├── top.1.txt
│       │       │   └── top.2.txt
│       │       ├── steadybit-agent-extension-host-bg529
│       │       │   ├── config.yml
│       │       │   ├── description.txt
│       │       │   ├── http
│       │       │   │   ├── GET__.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.fill_disk.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.host_discovery.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.host_discovery_discovered-targets.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.host_discovery_target-description.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_bandwidth.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_blackhole.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_block_dns.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_delay.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_package_corruption.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.network_package_loss.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.shutdown.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stop-process.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stress-cpu.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stress-io.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.stress-mem.yml
│       │       │   │   ├── GET__com.steadybit.extension_host.timetravel.yml
│       │       │   │   └── GET__discovery_attributes.yml
│       │       │   ├── logs.txt
│       │       │   ├── logs_previous.txt
│       │       │   ├── top.0.txt
│       │       │   ├── top.1.txt
│       │       │   └── top.2.txt
│       │       └── steadybit-agent-extension-host-qph74
│       │           ├── config.yml
│       │           ├── description.txt
│       │           ├── http
│       │           │   ├── GET__.yml
│       │           │   ├── GET__com.steadybit.extension_host.fill_disk.yml
│       │           │   ├── GET__com.steadybit.extension_host.host_discovery.yml
│       │           │   ├── GET__com.steadybit.extension_host.host_discovery_discovered-targets.yml
│       │           │   ├── GET__com.steadybit.extension_host.host_discovery_target-description.yml
│       │           │   ├── GET__com.steadybit.extension_host.network_bandwidth.yml
│       │           │   ├── GET__com.steadybit.extension_host.network_blackhole.yml
│       │           │   ├── GET__com.steadybit.extension_host.network_block_dns.yml
│       │           │   ├── GET__com.steadybit.extension_host.network_delay.yml
│       │           │   ├── GET__com.steadybit.extension_host.network_package_corruption.yml
│       │           │   ├── GET__com.steadybit.extension_host.network_package_loss.yml
│       │           │   ├── GET__com.steadybit.extension_host.shutdown.yml
│       │           │   ├── GET__com.steadybit.extension_host.stop-process.yml
│       │           │   ├── GET__com.steadybit.extension_host.stress-cpu.yml
│       │           │   ├── GET__com.steadybit.extension_host.stress-io.yml
│       │           │   ├── GET__com.steadybit.extension_host.stress-mem.yml
│       │           │   ├── GET__com.steadybit.extension_host.timetravel.yml
│       │           │   └── GET__discovery_attributes.yml
│       │           ├── logs.txt
│       │           ├── logs_previous.txt
│       │           ├── top.0.txt
│       │           ├── top.1.txt
│       │           └── top.2.txt
│       ├── steadybit-agent-extension-http
│       │   ├── config.yaml
│       │   ├── description.txt
│       │   └── pods
│       │       └── steadybit-agent-extension-http-ddc796f6d-dwfkl
│       │           ├── config.yml
│       │           ├── description.txt
│       │           ├── http
│       │           │   ├── GET__.yml
│       │           │   ├── GET__com.steadybit.extension_http.check.fixed_amount.yml
│       │           │   └── GET__com.steadybit.extension_http.check.periodically.yml
│       │           ├── logs.txt
│       │           ├── logs_previous.txt
│       │           ├── top.0.txt
│       │           ├── top.1.txt
│       │           └── top.2.txt
│       └── steadybit-agent-extension-kubernetes
│           ├── config.yaml
│           ├── description.txt
│           └── pods
│               └── steadybit-agent-extension-kubernetes-7d694f8fff-256rh
│                   ├── config.yml
│                   ├── description.txt
│                   ├── http
│                   │   ├── GET__.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.crash_loop_pod.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.delete_pod.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.drain_node.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-cluster_discovery.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-cluster_discovery_discovered-targets.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-cluster_discovery_target-description.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-container_discovery.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-container_discovery_discovered-targets.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-daemonset_discovery.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-daemonset_discovery_discovered-targets.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-daemonset_discovery_target-description.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-deployment_discovery.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-deployment_discovery_discovered-targets.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-deployment_discovery_target-description.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-node_discovery.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-node_discovery_discovered-targets.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-node_discovery_target-description.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-pod_discovery.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-pod_discovery_discovered-targets.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-pod_discovery_target-description.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-statefulset_discovery.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-statefulset_discovery_discovered-targets.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes-statefulset_discovery_target-description.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.kubernetes_logs.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.node_count_check.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.pod_count_check.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.pod_count_metric.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.rollout-restart.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.rollout-status.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.scale_deployment.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.scale_statefulset.yml
│                   │   ├── GET__com.steadybit.extension_kubernetes.taint_node.yml
│                   │   └── GET__discovery_attributes.yml
│                   ├── logs.txt
│                   ├── logs_previous.txt
│                   ├── top.0.txt
│                   ├── top.1.txt
│                   └── top.2.txt
├── nodes
│   ├── fargate-ip-10-40-83-162.eu-central-1.compute.internal
│   │   ├── config.yaml
│   │   └── description.txt
│   ├── fargate-ip-10-40-96-134.eu-central-1.compute.internal
│   │   ├── config.yaml
│   │   └── description.txt
│   ├── ip-10-40-83-252.eu-central-1.compute.internal
│   │   ├── config.yaml
│   │   └── description.txt
│   ├── ip-10-40-92-38.eu-central-1.compute.internal
│   │   ├── config.yaml
│   │   └── description.txt
│   └── ip-10-40-94-131.eu-central-1.compute.internal
│       ├── config.yaml
│       └── description.txt
└── output.txt

xx directories, xxx files

```
