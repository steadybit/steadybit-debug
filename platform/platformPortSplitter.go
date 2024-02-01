// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package platform

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
)

func AddPlatformPortSplitterDebuggingInformation(cfg *config.Config) {
	deployment, err := k8s.FindDeployment(cfg, cfg.PlatformPortSplitter.Namespace, cfg.PlatformPortSplitter.Deployment)
	if err != nil {
		log.Warn().Msgf("Failed to find platform port splitter deployment '%s' in '%s': %s", cfg.PlatformPortSplitter.Deployment, cfg.PlatformPortSplitter.Namespace, err)
		return
	}

	pathForPlatformPortSplitter := filepath.Join(cfg.OutputPath, "platform-port-splitter")
	k8s.AddDescription(cfg, filepath.Join(pathForPlatformPortSplitter, "description.txt"), "deployment", deployment.Namespace, deployment.Name)
	k8s.AddConfig(cfg, filepath.Join(pathForPlatformPortSplitter, "config.yaml"), "deployment", deployment.Namespace, deployment.Name)

	k8s.ForEachPod(cfg, deployment.Namespace, deployment.Spec.Selector, func(pod *v1.Pod, idx int) {
		pathForPod := filepath.Join(pathForPlatformPortSplitter, "pods", pod.Name)
		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
		k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
		k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name, 5)
	})
}
