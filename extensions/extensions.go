// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extensions

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
	"strconv"
	"strings"
)

const ExtensionAutoDiscoveryAnnotation = "steadybit.com/extension-auto-discovery"

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

				k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
				k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
				k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
				k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
				k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name)

				ports := identifyPodPorts(pod, service)
				for _, port := range ports {
					TraverseExtensionEndpoints(TraverseExtensionEndpointsOptions{
						Config:       cfg,
						PodNamespace: pod.Namespace,
						PodName:      pod.Name,
						PathForPod:   filepath.Join(pathForPod, "http"),
						BaseUrl:      fmt.Sprintf("http://localhost:%d/", port),
					})
				}
			})
		}
	}

}

type extensionAutoDiscoveryExtension struct {
	Port int `json:"port"`
}

type extensionAutoDiscovery struct {
	Extensions []extensionAutoDiscoveryExtension `json:"extensions"`
}

func identifyPodPorts(pod *v1.Pod, service *v1.Service) []int {
	//try to find the port via annotations
	extensionAutoDiscoveryString, ok := service.Annotations[ExtensionAutoDiscoveryAnnotation]
	const defaultPort = 8080
	if ok {
		extensionAutoDiscoveryStruct := extensionAutoDiscovery{}
		err := json.Unmarshal([]byte(extensionAutoDiscoveryString), &extensionAutoDiscoveryStruct)
		if err != nil {
			log.Warn().Msgf("Failed to parse extension auto discovery annotation: %s", err)
			return []int{defaultPort}
		}
		ret := make([]int, 0, len(extensionAutoDiscoveryStruct.Extensions))
		for _, extension := range extensionAutoDiscoveryStruct.Extensions {
			log.Debug().Msgf("Found extension port: %d", extension.Port)
			ret = append(ret, extension.Port)
		}
		return ret
	}

	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.ToUpper(env.Name) == "STEADYBIT_EXTENSION_PORT" {
				configuredPort, err := strconv.Atoi(env.Value)
				if err == nil {
					return []int{configuredPort}
				}
			}
		}
	}

	// try the default extension port
	return []int{defaultPort}
}