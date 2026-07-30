// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package agent

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

// TestIdentifyAgentContainer pins that the agent is picked over the autoregistration extension sharing its pod: the
// extension carries neither the certificates nor the environment the tests have to reproduce, and it comes first.
// Its name is also the longer one, so a prefix match would find the wrong container.
func TestIdentifyAgentContainer(t *testing.T) {
	pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{
		{Name: "steadybit-agent-kubernetes-autoregistration"},
		{Name: "steadybit-agent"},
	}}}

	if got := identifyAgentContainer(pod).Name; got != "steadybit-agent" {
		t.Errorf("expected the agent container, got '%s'", got)
	}
}

func TestIdentifyAgentContainerFallsBackToTheFirst(t *testing.T) {
	pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{Name: "only-one"}}}}

	if got := identifyAgentContainer(pod).Name; got != "only-one" {
		t.Errorf("expected the only container, got '%s'", got)
	}
}
