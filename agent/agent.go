package agent

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit_debug/config"
	"github.com/steadybit/steadybit_debug/k8s"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
)

func AddAgentDebuggingInformation(cfg *config.Config) {
	daemonSet, err := k8s.FindDaemonSet(cfg, cfg.Agent.Namespace, cfg.Agent.DaemonSet)
	if err != nil {
		log.Info().Msgf("Failed to find daemon set '%s' in '%s': %s", cfg.Agent.DaemonSet, cfg.Agent.Namespace, err)
		return
	}

	pathForAgent := filepath.Join(cfg.OutputPath, "agent")
	k8s.AddDescription(cfg, filepath.Join(pathForAgent, "description.txt"), "daemonset", daemonSet.Namespace, daemonSet.Name)
	k8s.AddConfig(cfg, filepath.Join(pathForAgent, "config.yaml"), "daemonset", daemonSet.Namespace, daemonSet.Name)

	k8s.ForEachPod(cfg, daemonSet.Namespace, daemonSet.Spec.Selector, func(pod *v1.Pod) {
		pathForPod := filepath.Join(pathForAgent, "pods", pod.Name)
		// TODO this one can change!
		// STEADYBIT_HTTP_ENDPOINT_PORT?
		port := 42899

		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)

		// TODO does not work
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
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "prometheus_metrics.txt"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/prometheus", port),
		})
		// TODO does not work
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "threaddump.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/threaddump", port),
		})
		// TODO does not work
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "info.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          fmt.Sprintf("http://localhost:%d/info", port),
		})
	})
}
