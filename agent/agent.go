// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package agent

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func AddAgentDebuggingInformation(cfg *config.Config) {
	daemonSet, err := k8s.FindDaemonSet(cfg, cfg.Agent.Namespace, cfg.Agent.DaemonSet)
	if err != nil {
		log.Warn().Msgf("Failed to find daemon set '%s' in '%s': %s", cfg.Agent.DaemonSet, cfg.Agent.Namespace, err)
		return
	}

	pathForAgent := filepath.Join(cfg.OutputPath, "agent")
	k8s.AddDescription(cfg, filepath.Join(pathForAgent, "description.txt"), "daemonset", daemonSet.Namespace, daemonSet.Name)
	k8s.AddConfig(cfg, filepath.Join(pathForAgent, "config.yaml"), "daemonset", daemonSet.Namespace, daemonSet.Name)

	k8s.ForEachPod(cfg, daemonSet.Namespace, daemonSet.Spec.Selector, func(pod *v1.Pod, _ int) {
		pathForPod := filepath.Join(pathForAgent, "pods", pod.Name)
		port := identifyPodPort(pod)
		delay := time.Millisecond * 500

		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
		k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
		k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name)

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
					},
				},
			})
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
