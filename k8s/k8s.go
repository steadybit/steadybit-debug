// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package k8s

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/output"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sync"
	"time"
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

// ForEachPod note that the function fn will be executed in parallel for each pod
func ForEachPod(cfg *config.Config, namespace string, selector *metav1.LabelSelector, fn func(pod *v1.Pod)) {
	podList, err := findPods(cfg, namespace, selector)
	if err != nil {
		log.Info().Msgf("Failed to find pods in namespace '%s' for selector '%s'. Got error: %s", namespace, selector.String(), err)
		return
	}

	var wg sync.WaitGroup
	for _, pod := range podList.Items {
		wg.Add(1)

		podForAsyncFunction := pod
		go func(pod *v1.Pod) {
			defer wg.Done()
			fn(pod)
		}(&podForAsyncFunction)
	}
	wg.Wait()
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

// ForEachNode note that the function fn will be executed in parallel for each node
func ForEachNode(cfg *config.Config, fn func(node *v1.Node)) {
	client, err := cfg.Kubernetes.Client()
	if err != nil {
		log.Info().Msgf("Failed to create Kubernetes client while trying to find node information. Got error: %s", err)
		return
	}

	nodeList, err := client.
		CoreV1().
		Nodes().
		List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Info().Msgf("Failed to find nodes. Got error: %s", err)
		return
	}

	var wg sync.WaitGroup
	for _, node := range nodeList.Items {
		wg.Add(1)

		nodeForAsyncFunction := node
		go func(node *v1.Node) {
			defer wg.Done()
			fn(node)
		}(&nodeForAsyncFunction)
	}
	wg.Wait()
}

func AddLogs(cfg *config.Config, path string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      cfg,
		CommandName: "kubectl",
		CommandArgs: []string{"logs", "-n", namespace, "--all-containers", name},
		OutputPath:  path,
	})
}

func AddPreviousLogs(cfg *config.Config, path string, namespace string, name string) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      cfg,
		CommandName: "kubectl",
		CommandArgs: []string{"logs", "-n", namespace, "--previous", "--all-containers", name},
		OutputPath:  path,
	})
}

// AddResourceUsage path must include '%d' to replace the execution number within the file path
func AddResourceUsage(cfg *config.Config, path string, namespace string, name string) {
	delay := time.Millisecond * 500
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:                 cfg,
		CommandName:            "kubectl",
		CommandArgs:            []string{"top", "pod", "-n", namespace, "--containers", name},
		OutputPath:             path,
		Executions:             10,
		DelayBetweenExecutions: &delay,
	})
}

type AddPodHttpEndpointOutputOptions struct {
	Config                 *config.Config
	OutputPath             string
	PodNamespace           string
	PodName                string
	Url                    string
	Executions             int
	DelayBetweenExecutions *time.Duration
}

func AddPodHttpEndpointOutput(options AddPodHttpEndpointOutputOptions) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:                 options.Config,
		CommandName:            "kubectl",
		CommandArgs:            []string{"exec", "-n", options.PodNamespace, options.PodName, "--", "curl", "-s", options.Url},
		OutputPath:             options.OutputPath,
		Executions:             options.Executions,
		DelayBetweenExecutions: options.DelayBetweenExecutions,
	})
}
