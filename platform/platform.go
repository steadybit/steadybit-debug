package platform

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
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

	k8s.ForEachPod(cfg, deployment.Namespace, deployment.Spec.Selector, func(pod *v1.Pod) {
		pathForPod := filepath.Join(pathForPlatform, "pods", pod.Name)

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
			Url:          "http://localhost:9090/actuator/env",
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "health.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          "http://localhost:9090/actuator/health",
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:                 cfg,
			OutputPath:             filepath.Join(pathForPod, "prometheus_metrics.%d.txt"),
			PodNamespace:           pod.Namespace,
			PodName:                pod.Name,
			Url:                    "http://localhost:9090/actuator/prometheus",
			Executions:             10,
			DelayBetweenExecutions: &delay,
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "threaddump.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          "http://localhost:9090/actuator/threaddump",
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "info.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          "http://localhost:9090/actuator/info",
		})
		k8s.AddPodHttpEndpointOutput(k8s.AddPodHttpEndpointOutputOptions{
			Config:       cfg,
			OutputPath:   filepath.Join(pathForPod, "target_stats.yml"),
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Url:          "http://localhost:9090/actuator/targetstats",
		})
	})
}
