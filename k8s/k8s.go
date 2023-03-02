// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package k8s

import (
	"bufio"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/output"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
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
func ForEachPod(cfg *config.Config, namespace string, selector *metav1.LabelSelector, fn func(pod *v1.Pod, idx int)) {
	podList, err := findPods(cfg, namespace, selector)
	if err != nil {
		log.Info().Msgf("Failed to find pods in namespace '%s' for selector '%s'. Got error: %s", namespace, selector.String(), err)
		return
	}

	doWithPods(podList, fn)
}

// ForEachPodViaMapSelector note that the function fn will be executed in parallel for each pod
func ForEachPodViaMapSelector(cfg *config.Config, namespace string, selectorMap map[string]string, fn func(pod *v1.Pod, idx int)) {
	podList, err := findPodsBySelectorMap(cfg, namespace, selectorMap)
	if err != nil {
		log.Info().Msgf("Failed to find pods in namespace '%s' for selector '%s'. Got error: %s", namespace, selectorMap, err)
		return
	}

	doWithPods(podList, fn)
}

func doWithPods(podList *v1.PodList, fn func(pod *v1.Pod, idx int)) {
	var wg sync.WaitGroup
	for idx, pod := range podList.Items {
		wg.Add(1)

		podForAsyncFunction := pod
		idx := idx
		go func(pod *v1.Pod) {
			defer wg.Done()
			fn(pod, idx)
		}(&podForAsyncFunction)
	}
	wg.Wait()
}

func findPods(cfg *config.Config,
	namespace string,
	selector *metav1.LabelSelector) (*v1.PodList, error) {
	selectorMap, err := metav1.LabelSelectorAsMap(selector)
	if err != nil {
		return nil, err
	}

	return findPodsBySelectorMap(cfg, namespace, selectorMap)
}

func findPodsBySelectorMap(cfg *config.Config, namespace string, selectorMap map[string]string) (*v1.PodList, error) {
	client, err := cfg.Kubernetes.Client()
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

type EndpointsOutputOptions struct {
	OutputPath             string
	Url                    string
	Executions             int
	DelayBetweenExecutions *time.Duration
}

type PodConfig struct {
	PodNamespace string
	PodName      string
	Config       *config.Config
}
type AddPodHttpEndpointsOutputOptions struct {
	SharedPort      int
	PodConfig       PodConfig
	EndpointOptions []EndpointsOutputOptions
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

type AddDownloadOutputOptions struct {
	Config       *config.Config
	OutputPath   string
	PodNamespace string
	PodName      string
	Url          string
	Method       string
}

func AddPodHttpMultipleEndpointOutput(options AddPodHttpEndpointsOutputOptions) {
	forwardingHostWithPort, cmd, err := PreparePortforwarding(PodConfig{
		PodNamespace: options.PodConfig.PodNamespace,
		PodName:      options.PodConfig.PodName,
		Config:       options.PodConfig.Config,
	}, options.SharedPort)
	if err != nil {
		log.Error().Msgf("Failed to prepare port forwarding. Got error: %s", err)
		return
	}

	defer func() {
		KillProcess(cmd, options.PodConfig)
	}()

	var wg sync.WaitGroup
	for _, endpoint := range options.EndpointOptions {
		wg.Add(1)
		go func(endpoint EndpointsOutputOptions) {
			defer wg.Done()
			podUrl, err := url.Parse(endpoint.Url)
			if err != nil {
				log.Error().Msgf("Failed to parse URL '%s'", endpoint.Url)
				return
			}
			podUrl.Host = forwardingHostWithPort

			output.AddCommandOutput(output.AddCommandOutputOptions{
				Config:                 options.PodConfig.Config,
				OutputPath:             endpoint.OutputPath,
				Executions:             endpoint.Executions,
				DelayBetweenExecutions: endpoint.DelayBetweenExecutions,
				CommandName:            "curl",
				CommandArgs:            []string{"-s", podUrl.String()},
			})
		}(endpoint)
	}
	wg.Wait()

}
func AddPodHttpEndpointOutput(options AddPodHttpEndpointOutputOptions) {
	podUrl, err := url.Parse(options.Url)
	if err != nil {
		log.Error().Msgf("Failed to parse URL '%s'", options.Url)
		return
	}
	port, _ := strconv.Atoi(podUrl.Port())
	host, cmd, err := PreparePortforwarding(PodConfig{
		PodNamespace: options.PodNamespace,
		PodName:      options.PodName,
		Config:       options.Config,
	}, port)
	if err != nil {
		log.Error().Msgf("Failed to prepare port forwarding. Got error: %s", err)
		return
	}

	defer func() {
		KillProcess(cmd, PodConfig{
			PodNamespace: options.PodNamespace,
			PodName:      options.PodName,
			Config:       options.Config,
		})
	}()

	podUrl.Host = host

	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:                 options.Config,
		CommandName:            "curl",
		CommandArgs:            []string{"-s", podUrl.String()},
		OutputPath:             options.OutputPath,
		Executions:             options.Executions,
		DelayBetweenExecutions: options.DelayBetweenExecutions,
	})
}

func DownloadFromPod(options AddDownloadOutputOptions) {
	downloadUrl, err := url.Parse(options.Url)
	if err != nil {
		log.Error().Msgf("Failed to parse URL '%s'", options.Url)
		return
	}
	port, _ := strconv.Atoi(downloadUrl.Port())
	forwardingHostWithPort, cmd, err := PreparePortforwarding(PodConfig{
		PodNamespace: options.PodNamespace,
		PodName:      options.PodName,
		Config:       options.Config,
	}, port)
	if err != nil {
		log.Error().Msgf("Failed to prepare port forwarding. Got error: %s", err)
		return
	}

	defer func() {
		KillProcess(cmd, PodConfig{
			PodNamespace: options.PodNamespace,
			PodName:      options.PodName,
			Config:       options.Config,
		})
	}()

	downloadUrl.Host = forwardingHostWithPort
	output.DownloadOutput(output.DownloadOptions{
		Config:     options.Config,
		OutputPath: options.OutputPath,
		Method:     options.Method,
		URL:        *downloadUrl,
	})
}

func KillProcess(cmd *exec.Cmd, options PodConfig) {
	err := cmd.Process.Kill()
	if err != nil {
		log.Error().Msgf("Failed to stop port-forward for '%s' in namespace '%s", options.PodName, options.PodNamespace)
		return
	}
}

func PreparePortforwarding(options PodConfig, port int) (string, *exec.Cmd, error) {
	cmd := exec.Command("kubectl", "port-forward", "-n", options.PodNamespace, fmt.Sprintf("pod/%s", options.PodName), fmt.Sprintf(":%d", port))
	log.Debug().Msgf("Executing: %s", cmd.String())

	cmdOut, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		log.Error().Msgf("Failed to port-forward for '%s' in namespace '%s", options.PodName, options.PodNamespace)
		return "", nil, err
	}

	var result string
	var r = regexp.MustCompile("^Forwarding from .+:(\\d+) -> \\d+$")
	scanner := bufio.NewScanner(cmdOut)
	for scanner.Scan() {
		m := r.FindStringSubmatch(scanner.Text())
		if m != nil {
			result = fmt.Sprintf("localhost:%s", m[1])
			break
		}
	}
	return result, cmd, nil
}
