/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package debugrun

import (
	"github.com/steadybit/steadybit-debug/agent"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/extensions"
	"github.com/steadybit/steadybit-debug/k8s"
	"github.com/steadybit/steadybit-debug/platform"
	"sync"
)

func GatherInformation(cfg *config.Config) {
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		platform.AddPlatformDebuggingInformation(cfg)
	}()

	go func() {
		defer wg.Done()
		platform.AddPlatformPortSplitterDebuggingInformation(cfg)
	}()

	go func() {
		defer wg.Done()
		agent.AddAgentDebuggingInformation(cfg)
	}()

	go func() {
		defer wg.Done()
		k8s.AddKubernetesNodesInformation(cfg)
	}()

	go func() {
		defer wg.Done()
		extensions.AddExtensionDebuggingInformation(cfg)
	}()

	wg.Wait()
}
