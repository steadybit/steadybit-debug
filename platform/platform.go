// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package platform

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
	"sync"
	"time"
)

func AddPlatformDebuggingInformation(cfg *config.Config) {
	deployment, err := k8s.FindDeployment(cfg, cfg.Platform.Namespace, cfg.Platform.Deployment)
	if err != nil {
		log.Warn().Msgf("Failed to find platform deployment '%s' in '%s': %s", cfg.Platform.Deployment, cfg.Platform.Namespace, err)
		return
	}

	pathForPlatform := filepath.Join(cfg.OutputPath, "platform")
	k8s.AddDescription(cfg, filepath.Join(pathForPlatform, "description.txt"), "deployment", deployment.Namespace, deployment.Name)
	k8s.AddConfig(cfg, filepath.Join(pathForPlatform, "config.yaml"), "deployment", deployment.Namespace, deployment.Name)

	k8s.ForEachPod(cfg, deployment.Namespace, deployment.Spec.Selector, func(pod *v1.Pod, idx int) {
		pathForPod := filepath.Join(pathForPlatform, "pods", pod.Name)
		var wg sync.WaitGroup
		if idx == 0 && cfg.Platform.ExportDatabase {
			wg.Add(1)
			go func() {
				log.Debug().Msgf("Downloading database export for platform %s", pod.Name)
				defer wg.Done()
				//Download Database export
				k8s.DownloadFromPod(k8s.AddDownloadOutputOptions{
					PodNamespace: pod.Namespace,
					PodName:      pod.Name,
					Config:       cfg,
					Url:          "http://localhost:9090/actuator/database/export",
					OutputPath:   filepath.Join(pathForPlatform, "database.zip"),
					Method:       "GET",
				})
			}()

		}

		delay := time.Millisecond * 500

		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
		k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
		k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name)

		k8s.AddPodHttpMultipleEndpointOutput(
			k8s.AddPodHttpEndpointsOutputOptions{
				SharedPort: 9090,
				PodConfig: k8s.PodConfig{
					PodNamespace: pod.Namespace,
					PodName:      pod.Name,
					Config:       cfg,
				},
				EndpointOptions: []k8s.EndpointsOutputOptions{
					{
						OutputPath: filepath.Join(pathForPod, "env.yml"),
						Url:        "http://localhost:9090/actuator/env",
					},
					{
						OutputPath: filepath.Join(pathForPod, "configprops.yml"),
						Url:        "http://localhost:9090/actuator/configprops",
					},
					{
						OutputPath: filepath.Join(pathForPod, "health.yml"),
						Url:        "http://localhost:9090/actuator/health",
					}, {
						OutputPath:             filepath.Join(pathForPod, "prometheus_metrics.%d.txt"),
						Url:                    "http://localhost:9090/actuator/prometheus",
						Executions:             10,
						DelayBetweenExecutions: &delay,
					},
					{
						OutputPath: filepath.Join(pathForPod, "threaddump.yml"),
						Url:        "http://localhost:9090/actuator/threaddump",
					}, {
						OutputPath: filepath.Join(pathForPod, "info.yml"),
						Url:        "http://localhost:9090/actuator/info",
					}, {
						OutputPath: filepath.Join(pathForPod, "target_stats.yml"),
						Url:        "http://localhost:9090/actuator/targetstats",
					},
				},
			})
		wg.Wait()
	})
}
