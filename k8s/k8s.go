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

func FindDaemonSet(cfg *config.Config, namespace string, name string) (*appsv1.DaemonSet, error) {
	client, err := cfg.Kubernetes.Client()
	if err != nil {
		return nil, err
	}

	return client.
		AppsV1().
		DaemonSets(namespace).
		Get(context.Background(), name, metav1.GetOptions{})
}

func AddDescription(config *config.Config, outputPath string, kind string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      config,
		CommandName: "kubectl",
		CommandArgs: []string{"describe", kind, "-n", namespace, name},
		OutputPath:  outputPath,
	})
}

func AddConfig(config *config.Config, outputPath string, kind string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      config,
		CommandName: "kubectl",
		CommandArgs: []string{"get", kind, "-n", namespace, "-o", "yaml", name},
		OutputPath:  outputPath,
	})
}

func ForEachPod(cfg *config.Config, namespace string, selector *metav1.LabelSelector, fn func(pod *v1.Pod)) {
	podList, err := findPods(cfg, namespace, selector)
	if err != nil {
		log.Info().Msgf("Failed to find pods in namespace '%s' for selector '%s'. Got error: %s", namespace, selector.String(), err)
		return
	}

	for _, pod := range podList.Items {
		log.Debug().Msgf("Gathering information for pod '%s' in namespace '%s'", pod.Name, pod.Namespace)
		fn(&pod)
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

func AddLogs(cfg *config.Config, path string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      cfg,
		CommandName: "kubectl",
		CommandArgs: []string{"logs", "-n", namespace, "--all-containers", name},
		OutputPath:  path,
	})
}

type AddPodHttpEndpointOutputOptions struct {
	Config       *config.Config
	OutputPath   string
	PodNamespace string
	PodName      string
	Url          string
}

func AddPodHttpEndpointOutput(options AddPodHttpEndpointOutputOptions) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      options.Config,
		CommandName: "kubectl",
		CommandArgs: []string{"exec", "-n", options.PodNamespace, options.PodName, "--", "curl", "-s", options.Url},
		OutputPath:  options.OutputPath,
	})
}
