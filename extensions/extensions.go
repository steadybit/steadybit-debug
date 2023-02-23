// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extensions

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
	"strconv"
	"strings"
)

func AddExtentionsDebuggingInformation(cfg *config.Config) {
	for _, namespace := range cfg.Extensions.Namespaces {
		services, err := k8s.FindExtensionsServices(cfg, namespace)
		if err != nil {
			log.Warn().Msgf("Failed to find services set '%s': %s", cfg.Extensions.Namespaces, err)
			return
		}

		for _, service := range services {
			pathForExtension := filepath.Join(cfg.OutputPath, "extensions", service.Name)
			k8s.AddDescription(cfg, filepath.Join(pathForExtension, "description.txt"), "service", service.Namespace, service.Name)
			k8s.AddConfig(cfg, filepath.Join(pathForExtension, "config.yaml"), "service", service.Namespace, service.Name)

			k8s.ForEachPodViaMapSelector(cfg, service.Namespace, service.Spec.Selector, func(pod *v1.Pod) {
				pathForPod := filepath.Join(pathForExtension, "pods", pod.Name)
				port := identifyPodPort(pod)

				k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
				k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
				k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
				k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
				k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name)

				k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
					Config:       cfg,
					OutputPath:   filepath.Join(pathForPod, "http.yml"),
					PodNamespace: pod.Namespace,
					PodName:      pod.Name,
					Url:          fmt.Sprintf("http://localhost:%d/", port),
				})
			})
		}
	}

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

	// try the default extension port
	return 8088
}
