// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package k8s

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
)

// TestEphemeralContainerState covers what the two callers of it disagree about: a container that has terminated is
// broken when it was supposed to keep running for tests to be executed in it, but perfectly fine when the test
// itself was its command. Reporting that as a failure would abort a test that had already succeeded.
func TestEphemeralContainerState(t *testing.T) {
	tests := []struct {
		name          string
		status        v1.ContainerStatus
		expected      containerState
		errorContains string
	}{
		{
			name:     "running",
			status:   v1.ContainerStatus{Name: "test", State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
			expected: containerRunning,
		},
		{
			name: "terminated successfully is not a failure",
			status: v1.ContainerStatus{Name: "test", State: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{ExitCode: 0}}},
			expected: containerTerminated,
		},
		{
			name: "terminated with an error is not a failure to start either",
			status: v1.ContainerStatus{Name: "test", State: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{ExitCode: 255}}},
			expected: containerTerminated,
		},
		{
			name: "still being created",
			status: v1.ContainerStatus{Name: "test", State: v1.ContainerState{
				Waiting: &v1.ContainerStateWaiting{Reason: "ContainerCreating"}}},
			expected: containerPending,
		},
		{
			name: "rejected by a restrictive security context",
			status: v1.ContainerStatus{Name: "test", State: v1.ContainerState{
				Waiting: &v1.ContainerStateWaiting{
					Reason:  "CreateContainerConfigError",
					Message: "container has runAsNonRoot and image has non-numeric user (curl_user)"}}},
			expected:      containerFailed,
			errorContains: "runAsNonRoot",
		},
		{
			name: "image cannot be pulled",
			status: v1.ContainerStatus{Name: "test", State: v1.ContainerState{
				Waiting: &v1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: "back-off pulling image"}}},
			expected:      containerFailed,
			errorContains: "ImagePullBackOff",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := &v1.Pod{Status: v1.PodStatus{EphemeralContainerStatuses: []v1.ContainerStatus{test.status}}}

			state, err := ephemeralContainerState(pod, "test")

			if state != test.expected {
				t.Errorf("expected state %d, got %d", test.expected, state)
			}
			if test.errorContains == "" && err != nil {
				t.Errorf("expected no error, got: %s", err)
			}
			if test.errorContains != "" && (err == nil || !strings.Contains(err.Error(), test.errorContains)) {
				t.Errorf("expected an error containing %q, got: %v", test.errorContains, err)
			}
		})
	}
}

func TestEphemeralContainerStateIgnoresOtherContainers(t *testing.T) {
	pod := &v1.Pod{Status: v1.PodStatus{EphemeralContainerStatuses: []v1.ContainerStatus{
		{Name: "someone-elses-container", State: v1.ContainerState{
			Waiting: &v1.ContainerStateWaiting{Reason: "CreateContainerConfigError"}}},
	}}}

	state, err := ephemeralContainerState(pod, "test")

	if state != containerPending || err != nil {
		t.Errorf("expected the container to be reported as pending without error, got %d and %v", state, err)
	}
}

func TestEphemeralContainerNamesAreUnique(t *testing.T) {
	// they have to stay unique within the same second, which is when tests of one pod are started
	names := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name := ephemeralContainerName()
		if names[name] {
			t.Fatalf("'%s' was handed out twice", name)
		}
		names[name] = true
	}
}
