// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package extensions

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const ExtensionAutoDiscoveryAnnotation = "steadybit.com/extension-auto-discovery"

func AddExtensionDebuggingInformation(cfg *config.Config) {
	var wg sync.WaitGroup
	namespaces, err := getAllNamespaces(cfg)
	if err != nil {
		log.Warn().Msgf("Failed to find extensions - looking up namespaces: %s", err)
		return
	}
	if len(namespaces) == 0 {
		log.Warn().Msgf("No namespaces found")
		return
	}
	for _, namespace := range namespaces {
		wg.Add(1)
		go func(namespace string) {
			defer wg.Done()
			findDebugInformationInNamespace(namespace, cfg)
		}(namespace)

	}
	wg.Wait()
}

func findDebugInformationInNamespace(namespace string, cfg *config.Config) {
	var wg sync.WaitGroup

	services, err := findExtensionsServices(cfg, namespace)
	if err != nil {
		log.Warn().Msgf("Failed to find services set '%s': %s", namespace, err)
		return
	}
	for _, service := range services {
		wg.Add(1)
		go func(service v1.Service) {
			defer wg.Done()
			forEachPod(cfg, "service", service.Namespace, service.Name, service.Spec.Selector, func(pod *v1.Pod) []podPort {
				return identifyPodPorts(pod, service.Annotations)
			})
		}(service)
	}

	daemonsets, err := findExtensionDaemonsets(cfg, namespace)
	if err != nil {
		log.Warn().Msgf("Failed to find daemonsets set '%s': %s", namespace, err)
		return
	}
	for _, daemonset := range daemonsets {
		wg.Add(1)
		go func(daemonset appsv1.DaemonSet) {
			defer wg.Done()
			forEachPod(cfg, "daemonset", daemonset.Namespace, daemonset.Name, daemonset.Spec.Selector.MatchLabels, func(pod *v1.Pod) []podPort {
				return identifyPodPorts(pod, pod.Annotations)
			})
		}(daemonset)
	}

	wg.Wait()
}

type identifyPorts func(pod *v1.Pod) []podPort

func forEachPod(cfg *config.Config, kind string, namespace string, name string, selector map[string]string, portsFn identifyPorts) {
	pathForExtension := filepath.Join(cfg.OutputPath, "extensions", namespace, name)
	k8s.AddDescription(cfg, filepath.Join(pathForExtension, "description.txt"), kind, namespace, name)
	k8s.AddConfig(cfg, filepath.Join(pathForExtension, "config.yaml"), kind, namespace, name)

	k8s.ForEachPodViaMapSelector(cfg, namespace, selector, func(pod *v1.Pod, _ int) {
		pathForPod := filepath.Join(pathForExtension, "pods", pod.Name)

		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
		k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
		k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name, 3)

		ports := portsFn(pod)
		for _, port := range ports {
			folderName := "http"
			if port.tls {
				folderName = "https"
			}
			TraverseExtensionEndpoints(TraverseExtensionEndpointsOptions{
				Config:       cfg,
				PodNamespace: pod.Namespace,
				PodName:      pod.Name,
				PathForPod:   filepath.Join(pathForPod, folderName),
				Port:         port.port,
				UseHttps:     port.tls,
			})
		}
	})
}

type extensionAutoDiscoveryExtensionTls struct {
	Server any `json:"server"`
	Client any `json:"client"`
}
type extensionAutoDiscoveryExtension struct {
	Port int                                `json:"port"`
	Tls  extensionAutoDiscoveryExtensionTls `json:"tls"`
}

type extensionAutoDiscovery struct {
	Extensions []extensionAutoDiscoveryExtension `json:"extensions"`
}

type podPort struct {
	port int
	tls  bool
}

func identifyPodPorts(pod *v1.Pod, annotations map[string]string) []podPort {
	//try to find the port via annotations
	extensionAutoDiscoveryString, ok := annotations[ExtensionAutoDiscoveryAnnotation]
	var defaultPort = podPort{
		port: 8080,
		tls:  false,
	}

	if ok {
		extensionAutoDiscoveryStruct := extensionAutoDiscovery{}
		err := json.Unmarshal([]byte(extensionAutoDiscoveryString), &extensionAutoDiscoveryStruct)
		if err != nil {
			log.Warn().Msgf("Failed to parse extension auto discovery annotation: %s", err)
			return []podPort{defaultPort}
		}
		ret := make([]podPort, 0, len(extensionAutoDiscoveryStruct.Extensions))
		for _, extension := range extensionAutoDiscoveryStruct.Extensions {
			useHttps := extension.Tls.Client != nil || extension.Tls.Server != nil
			ret = append(ret, podPort{
				port: extension.Port,
				tls:  useHttps,
			})
		}
		return ret
	}

	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.ToUpper(env.Name) == "STEADYBIT_EXTENSION_PORT" {
				configuredPort, err := strconv.Atoi(env.Value)
				if err == nil {
					return []podPort{{
						port: configuredPort,
						tls:  false,
					},
					}
				}
			}
		}
	}

	// try the default extension port
	return []podPort{defaultPort}
}

func getAllNamespaces(cfg *config.Config) (namespaces []string, err error) {
	client, err := cfg.Kubernetes.Client()
	if err != nil {
		return nil, err
	}
	list, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	namespaces = make([]string, 0, len(list.Items))
	for _, namespace := range list.Items {
		namespaces = append(namespaces, namespace.Name)
	}
	return namespaces, nil
}
func findExtensionsServices(cfg *config.Config, namespace string) ([]v1.Service, error) {
	client, err := cfg.Kubernetes.Client()
	if err != nil {
		return nil, err
	}

	listOfServices, err := client.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]v1.Service, 0, len(listOfServices.Items))
	for _, service := range listOfServices.Items {
		_, ok := service.Annotations[ExtensionAutoDiscoveryAnnotation]
		if ok {
			result = append(result, service)
		}
	}

	return result, nil
}

func findExtensionDaemonsets(cfg *config.Config, namespace string) ([]appsv1.DaemonSet, error) {
	client, err := cfg.Kubernetes.Client()
	if err != nil {
		return nil, err
	}

	listOfDaemonsets, err := client.AppsV1().DaemonSets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]appsv1.DaemonSet, 0, len(listOfDaemonsets.Items))
	for _, daemonset := range listOfDaemonsets.Items {
		_, ok := daemonset.Spec.Template.Annotations[ExtensionAutoDiscoveryAnnotation]
		if ok {
			result = append(result, daemonset)
		}
	}

	return result, nil
}
