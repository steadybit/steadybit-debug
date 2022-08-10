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

	k8s.ForEachPod(cfg, daemonSet.Namespace, daemonSet.Spec.Selector, func(pod *v1.Pod) {
		pathForPod := filepath.Join(pathForAgent, "pods", pod.Name)
		port := identifyPodPort(pod)
		delay := time.Millisecond * 500

		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
		k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
		k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name)

		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "env.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/env", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "health.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/health", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:                 cfg,
			OutputPath:             filepath.Join(pathForPod, "prometheus_metrics.%d.txt"),
			PodNamespace:           pod.Namespace,
			PodName:                pod.Name,
			Url:                    fmt.Sprintf("http://localhost:%d/prometheus", port),
			Executions:             10,
			DelayBetweenExecutions: &delay,
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "threaddump.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/threaddump", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "info.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/info", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "self_test.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/self-test", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "discovery_info.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/discovery/info", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "targets.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/discovery/targets", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "target_stats.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/discovery/targets/stats", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "connections.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/discovery/connections", port),
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "connection_stats.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/discovery/connections/stats", port),
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
