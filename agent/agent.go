// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package agent

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	"github.com/steadybit/steadybit-debug/limit"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func AddAgentDebuggingInformation(cfg *config.Config) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		statefulSet, err := k8s.FindStatefulSet(cfg, cfg.Agent.Namespace, cfg.Agent.StatefulSet)
		if err != nil {
			log.Warn().Msgf("Failed to find agent stateful set '%s' in '%s': %s", cfg.Agent.StatefulSet, cfg.Agent.Namespace, err)
		} else {
			addAgentDebuggingData(cfg, filepath.Join(cfg.OutputPath, "agent"), statefulSet.Namespace, statefulSet.Name, "statefulset", statefulSet.Spec.Selector)
		}
	}()

	wg.Wait()
}

func addAgentDebuggingData(cfg *config.Config, outputPath string, namespace string, name string, kind string, selector *metav1.LabelSelector) {
	pathForAgent := outputPath
	k8s.AddDescription(cfg, filepath.Join(pathForAgent, "description.txt"), kind, namespace, name)
	k8s.AddConfig(cfg, filepath.Join(pathForAgent, "config.yaml"), kind, namespace, name)

	k8s.ForEachPod(cfg, namespace, selector, func(pod *v1.Pod, _ int) {
		pathForPod := filepath.Join(pathForAgent, "pods", pod.Name)
		port := identifyPodPort(pod)
		delay := time.Millisecond * 500
		platformUrl := identifyPlatformUrl(pod)

		k8s.AddDescription(cfg, filepath.Join(pathForPod, "description.txt"), "pod", pod.Namespace, pod.Name)
		k8s.AddConfig(cfg, filepath.Join(pathForPod, "config.yml"), "pod", pod.Namespace, pod.Name)
		k8s.AddLogs(cfg, filepath.Join(pathForPod, "logs.txt"), pod.Namespace, pod.Name)
		k8s.AddPreviousLogs(cfg, filepath.Join(pathForPod, "logs_previous.txt"), pod.Namespace, pod.Name)
		k8s.AddResourceUsage(cfg, filepath.Join(pathForPod, "top.%d.txt"), pod.Namespace, pod.Name, 10)

		tester := k8s.NewConnectionTester(cfg, pod.Namespace, pod.Name, pod.Spec.Containers[0].Name)
		platformConnectionTests := []func(){
			func() {
				tester.AddHttpConnectionTest(filepath.Join(pathForPod, "platform_connection_test.txt"), platformUrl+"/agent")
			},
			func() {
				tester.AddWebsocketCurlHttp1ConnectionTest(filepath.Join(pathForPod, "platform_websocket_http1_connection_test.txt"), platformUrl)
			},
			func() {
				tester.AddWebsocketCurlHttp2ConnectionTest(filepath.Join(pathForPod, "platform_websocket_http2_connection_test.txt"), platformUrl)
			},
			func() {
				tester.AddWebsocketWebsocatConnectionTest(filepath.Join(pathForPod, "platform_websocat_connection_test.txt"), platformUrl)
			},
		}
		if parsedPlatformUrl, err := url.Parse(platformUrl); err != nil {
			log.Err(err).Msgf("Failed to parse platform url '%s'", platformUrl)
		} else {
			platformConnectionTests = append(platformConnectionTests, func() {
				tester.AddTracerouteConnectionTest(filepath.Join(pathForPod, "platform_traceroute_test.txt"), parsedPlatformUrl.Host)
			})
		}
		runConnectionTests(cfg, platformConnectionTests)

		k8s.AddPodHttpMultipleEndpointOutput(
			k8s.AddPodHttpEndpointsOutputOptions{
				SharedPort: port,
				PodConfig: k8s.PodConfig{
					PodNamespace: pod.Namespace,
					PodName:      pod.Name,
					Config:       cfg,
				},
				EndpointOptions: []k8s.EndpointsOutputOptions{
					{
						OutputPath: filepath.Join(pathForPod, "env.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/env", port),
					},
					{
						OutputPath: filepath.Join(pathForPod, "health.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/health", port),
					},
					{
						OutputPath:             filepath.Join(pathForPod, "prometheus_metrics.%d.txt"),
						Url:                    fmt.Sprintf("http://localhost:%d/prometheus", port),
						Executions:             10,
						DelayBetweenExecutions: &delay,
					}, {
						OutputPath: filepath.Join(pathForPod, "threaddump.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/threaddump", port),
					},
					{
						OutputPath: filepath.Join(pathForPod, "info.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/info", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "discovery_info.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/info", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "targets.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/targets", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "target_stats.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/discovery/targets/stats", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "target_type_description.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/targetType/description", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "actions_metadata.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/actions/metadata", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "advice_definition.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/advice/definition", port),
					}, {
						OutputPath: filepath.Join(pathForPod, "enrichtment_rules.yml"),
						Url:        fmt.Sprintf("http://localhost:%d/targetEnrichment/rules", port),
					},
				},
			})

		extensionConnections := k8s.GetExtensionConnections(port, k8s.PodConfig{
			PodNamespace: pod.Namespace,
			PodName:      pod.Name,
			Config:       cfg,
		}, cfg)
		extensionConnectionTests := make([]func(), 0, len(extensionConnections))
		for idx, extensionConnection := range extensionConnections {
			outputPath := filepath.Join(pathForPod, fmt.Sprintf("extension_connection_test_%d.txt", idx))
			connectionUrl := extensionConnection.Url
			extensionConnectionTests = append(extensionConnectionTests, func() {
				tester.AddHttpConnectionTest(outputPath, connectionUrl)
			})
		}
		runConnectionTests(cfg, extensionConnectionTests)
	})
}

// maxParallelConnectionTests bounds how many tests are executed in one pod at the same time. They share the
// ephemeral container they run in, so this is about not putting too many processes into one pod at once.
const maxParallelConnectionTests = 8

// runConnectionTests runs the connection tests of one pod in parallel. Each of them waits for a connection
// attempt that only tells us something once it has run into its timeout, so running them one after another adds
// minutes per pod for no reason.
func runConnectionTests(cfg *config.Config, tests []func()) {
	if cfg.SkipConnectionTests {
		log.Debug().Msgf("Skipping %d connection tests", len(tests))
		return
	}

	bound := limit.New(maxParallelConnectionTests)

	var wg sync.WaitGroup
	for _, test := range tests {
		wg.Add(1)
		go func(test func()) {
			defer wg.Done()
			release := bound.Acquire()
			defer release()
			test()
		}(test)
	}
	wg.Wait()
}

func identifyPodPort(pod *v1.Pod) int {
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.ToUpper(env.Name) == "SERVER_PORT" {
				configuredPort, err := strconv.Atoi(env.Value)
				if err == nil {
					return configuredPort
				}
			}
		}
	}

	// try the default agent port
	return 42899
}

func identifyPlatformUrl(pod *v1.Pod) string {
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if strings.ToUpper(env.Name) == "STEADYBIT_AGENT_REGISTER_URL" {
				return env.Value
			}
		}
	}

	// try the default saas url
	return "https://platform.steadybit.com"
}
