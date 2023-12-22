// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package agent

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func AddAgentDebuggingInformation(cfg *config.Config) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		daemonSet, err := k8s.FindDaemonSet(cfg, cfg.Agent.Namespace, cfg.Agent.DaemonSet)
		if err != nil {
			log.Warn().Msgf("Failed to find agent daemon set '%s' in '%s': %s", cfg.Agent.DaemonSet, cfg.Agent.Namespace, err)
		} else {
			addAgentDebuggingData(cfg, filepath.Join(cfg.OutputPath, "agent"), daemonSet.Namespace, daemonSet.Name, "daemonset", daemonSet.Spec.Selector)
		}
	}()

	go func() {
		defer wg.Done()
		statefulSet, err := k8s.FindStatefulSet(cfg, cfg.Outpost.Namespace, cfg.Outpost.StatefulSet)
		if err != nil {
			log.Warn().Msgf("Failed to find outpost stateful set '%s' in '%s': %s", cfg.Outpost.StatefulSet, cfg.Outpost.Namespace, err)
		} else {
			addAgentDebuggingData(cfg, filepath.Join(cfg.OutputPath, "outpost"), statefulSet.Namespace, statefulSet.Name, "statefulset", statefulSet.Spec.Selector)
		}
	}()

	wg.Wait()
}

func addAgentDebuggingData(cfg *config.Config, outputPath string, namespace string, name string, kind string, selector *metav1.LabelSelector) {
	pathForAgent := outputPath
	k8s.AddDescription(cfg, filepath.Join(pathForAgent, "description.txt"), kind, namespace, name)
	k8s.AddConfig(cfg, filepath.Join(pathForAgent, "config.yaml"), kind, namespace, name)

	k8s.ForEachPod(cfg, namespace, selector, func(pod *v1.Pod, _ int) {
		pathForPod := filepath.Join(pathForAgent, "pods", pod.Name)
		port := identifyPodPort(pod)
		delay := time.Millisecond * 500
		platformUrl := identifyPlatformUrl(pod)

		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
		k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
		k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name)

		k8s.AddHttpConnectionTest(cfg, filepath.Join(pathForPod, "platform_connection_test.txt"), pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, platformUrl+"/agent")
		k8s.AddWebsocketCurlHttp1ConnectionTest(cfg, filepath.Join(pathForPod, "platform_websocket_http1_connection_test.txt"), pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, platformUrl)
		k8s.AddWebsocketCurlHttp2ConnectionTest(cfg, filepath.Join(pathForPod, "platform_websocket_http2_connection_test.txt"), pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, platformUrl)
		k8s.AddWebsocketWebsocatConnectionTest(cfg, filepath.Join(pathForPod, "platform_websocat_connection_test.txt"), pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, platformUrl)

		k8s.AddPodHttpMultipleEndpointOutput(
			k8s.AddPodHttpEndpointsOutputOptions{
				SharedPort: port,
				PodConfig: k8s.PodConfig{
					PodNamespace: pod.Namespace,
					PodName:      pod.Name,
					Config:       cfg,
				},
				EndpointOptions: []k8s.EndpointsOutputOptions{
					{
						OutputPath: filepath.Join(pathForPod, "env.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/env", port),
					},
					{
						OutputPath: filepath.Join(pathForPod, "health.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/health", port),
					},
					{
						OutputPath:             filepath.Join(pathForPod, "prometheus_metrics.%d.txt"),
						Url:                    fmt.Sprintf("http://localhost:%d/prometheus", port),
						Executions:             10,
						DelayBetweenExecutions: &delay,
					}, {
						OutputPath: filepath.Join(pathForPod, "threaddump.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/threaddump", port),
					},
					{
						OutputPath: filepath.Join(pathForPod, "info.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/info", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "self_test.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/self-test", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "discovery_info.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/info", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "targets.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/targets", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "target_stats.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/targets/stats", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "connections.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/connections", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "connection_stats.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/connections/stats", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "target_type_description.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/targetType/description", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "actions_metadata.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/actions/metadata", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "advice_definition.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/advice/definition", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "enrichtment_rules.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/targetEnrichment/rules", port),
					},
				},
			})

		extensionConnections := k8s.GetExtensionConnections(port, k8s.PodConfig{
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Config:       cfg,
		}, cfg)
		for idx, extensionConnection := range extensionConnections {
			k8s.AddHttpConnectionTest(cfg, filepath.Join(pathForPod, fmt.Sprintf("extension_connection_test_%d.txt", idx)), pod.Namespace, pod.Name, pod.Spec.Containers[0].Name, extensionConnection.Url)
		}
	})
}

func identifyPodPort(pod *v1.Pod) int {
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.ToUpper(env.Name) == "STEADYBIT_HTTP_ENDPOINT_PORT" {
				configuredPort, err := strconv.Atoi(env.Value)
				if err == nil {
					return configuredPort
				}
			}
		}
	}

	// try the default agent port
	return 42899
}

func identifyPlatformUrl(pod *v1.Pod) string {
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.ToUpper(env.Name) == "STEADYBIT_AGENT_REGISTER_URL" {
				return env.Value
			}
		}
	}

	// try the default saas url
	return "https://platform.steadybit.com"
}
