package k8s

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit_debug/config"
	"github.com/steadybit/steadybit_debug/output"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"path/filepath"
)

func FindDeployment(cfg *config.Config, namespace string, name string) (*appsv1.Deployment, error) {
	client, err := cfg.Kubernetes.Client()
	if err != nil {
		return nil, err
	}

	return client.
		AppsV1().
		Deployments(namespace).
		Get(context.Background(), name, metav1.GetOptions{})
}

func AddKubernetesDeploymentOutput(cfg *config.Config, path string, deployment *appsv1.Deployment) {
	addDescription(cfg, filepath.Join(path, "description.txt"), "deployment", deployment.Namespace, deployment.Name)
	addConfig(cfg, filepath.Join(path, "config.yaml"), "deployment", deployment.Namespace, deployment.Name)
	addPodOutput(cfg, filepath.Join(path, "pods"), deployment.Namespace, deployment.Spec.Selector)
}

func addDescription(config *config.Config, outputPath string, kind string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      config,
		CommandName: "kubectl",
		CommandArgs: []string{"describe", kind, "-n", namespace, name},
		OutputPath:  outputPath,
	})
}

func addConfig(config *config.Config, outputPath string, kind string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      config,
		CommandName: "kubectl",
		CommandArgs: []string{"get", kind, "-n", namespace, "-o", "yaml", name},
		OutputPath:  outputPath,
	})
}

func addPodOutput(cfg *config.Config, path string, namespace string, selector *metav1.LabelSelector) {
	podList, err := findPods(cfg, namespace, selector)
	if err != nil {
		log.Info().Msgf("Failed to find pods in namespace '%s' for selector '%s'. Got error: %s", namespace, selector.String(), err)
		return
	}

	for _, pod := range podList.Items {
		pathForPod := filepath.Join(path, pod.Name)
		addDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		addConfig(cfg, filepath.Join(pathForPod, "config.yaml"), "pod", pod.Namespace, pod.Name)
		addLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
	}
}

func findPods(cfg *config.Config,
	namespace string,
	selector *metav1.LabelSelector) (*v1.PodList, error) {
	client, err := cfg.Kubernetes.Client()
	if err != nil {
		return nil, err
	}

	selectorMap, err := metav1.LabelSelectorAsMap(selector)
	if err != nil {
		return nil, err
	}

	return client.
		CoreV1().
		Pods(namespace).
		List(context.Background(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(selectorMap).String(),
		})
}

func addLogs(cfg *config.Config, path string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      cfg,
		CommandName: "kubectl",
		CommandArgs: []string{"logs", "-n", namespace, "--all-containers", name},
		OutputPath:  path,
	})
}
