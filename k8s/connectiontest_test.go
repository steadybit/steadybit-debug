// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package k8s

import (
	"os"
	"slices"
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

func TestCustomizationForContainer(t *testing.T) {
	readOnly := true
	user := int64(10000)

	tests := []struct {
		name      string
		container v1.Container
		assert    func(*testing.T, *debugCustomization)
	}{
		{
			name: "the security context is inherited, so the kubelet can verify the user",
			container: v1.Container{Name: "target", SecurityContext: &v1.SecurityContext{
				RunAsUser:              &user,
				ReadOnlyRootFilesystem: &readOnly,
			}},
			assert: func(t *testing.T, c *debugCustomization) {
				if c == nil || c.SecurityContext == nil {
					t.Fatal("expected the security context to be inherited")
				}
				if c.SecurityContext.RunAsUser == nil || *c.SecurityContext.RunAsUser != user {
					t.Error("expected runAsUser to be inherited")
				}
				if c.SecurityContext.ReadOnlyRootFilesystem != nil {
					t.Error("expected readOnlyRootFilesystem to be dropped, the tools may need to write")
				}
			},
		},
		{
			name: "volume mounts are inherited, so the test can read the same certificates",
			container: v1.Container{Name: "target", VolumeMounts: []v1.VolumeMount{
				{Name: "extensions-tls-client", MountPath: "/opt/steadybit/agent/etc/extensions/client"},
			}},
			assert: func(t *testing.T, c *debugCustomization) {
				if c == nil || len(c.VolumeMounts) != 1 {
					t.Fatalf("expected one volume mount, got %+v", c)
				}
				if c.VolumeMounts[0].MountPath != "/opt/steadybit/agent/etc/extensions/client" {
					t.Errorf("unexpected mount path %q", c.VolumeMounts[0].MountPath)
				}
			},
		},
		{
			name:      "nothing to inherit means no customization at all",
			container: v1.Container{Name: "target"},
			assert: func(t *testing.T, c *debugCustomization) {
				if c != nil {
					t.Errorf("expected no customization, got %+v", c)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{test.container}}}
			test.assert(t, customizationForContainer(pod, "target"))
		})
	}
}

func TestCustomizationForUnknownContainer(t *testing.T) {
	pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{Name: "someone-else"}}}}

	if c := customizationForContainer(pod, "target"); c != nil {
		t.Errorf("expected no customization for a container that is not in the pod, got %+v", c)
	}
}

func TestCustomizationIsWrittenAsAPartialContainerSpec(t *testing.T) {
	user := int64(10000)
	customization := &debugCustomization{
		SecurityContext: &v1.SecurityContext{RunAsUser: &user},
		VolumeMounts:    []v1.VolumeMount{{Name: "certs", MountPath: "/certs"}},
	}

	path, remove, err := customization.writeTo(t.TempDir())
	if err != nil {
		t.Fatalf("failed to write the customization: %s", err)
	}
	defer remove()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read it back: %s", err)
	}
	for _, expected := range []string{`"runAsUser":10000`, `"mountPath":"/certs"`} {
		if !strings.Contains(string(content), expected) {
			t.Errorf("expected %s in %s", expected, content)
		}
	}

	remove()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected the file to be removed, got %v", err)
	}
}

func TestTlsArgsFromTheAgentEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		env       []v1.EnvVar
		platform  string
		extension string
	}{
		{
			name:      "nothing configured",
			env:       nil,
			platform:  "",
			extension: "",
		},
		{
			name: "client and server certificate",
			env: []v1.EnvVar{
				{Name: envExtensionClientCertChain, Value: "/certs/tls.crt"},
				{Name: envExtensionClientCertKey, Value: "/certs/tls.key"},
				{Name: envExtensionServerCert, Value: "/certs/ca.crt"},
			},
			extension: "--cert /certs/tls.crt --key /certs/tls.key --cacert /certs/ca.crt",
		},
		{
			name: "a password protected key is not presented at all",
			env: []v1.EnvVar{
				{Name: envExtensionClientCertChain, Value: "/certs/tls.crt"},
				{Name: envExtensionClientCertKey, Value: "/certs/tls.key"},
				{Name: envExtensionClientCertPass, ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{Key: "extensionsClientKeyPassword"}}},
			},
			extension: "",
		},
		{
			name:      "verification disabled for extensions",
			env:       []v1.EnvVar{{Name: envExtensionInsecure, Value: "true"}},
			extension: "--insecure",
		},
		{
			name:     "verification disabled for the platform",
			env:      []v1.EnvVar{{Name: envPlatformInsecure, Value: "true"}},
			platform: "--insecure",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{Name: "agent", Env: test.env}}}}

			if got := strings.Join(platformTlsArgs(pod, "agent"), " "); got != test.platform {
				t.Errorf("platform: expected '%s', got '%s'", test.platform, got)
			}
			if got := strings.Join(extensionTlsArgs(pod, "agent"), " "); got != test.extension {
				t.Errorf("extension: expected '%s', got '%s'", test.extension, got)
			}
		})
	}
}

// TestTlsArgsNeverCarryASecret is the point of only reading literal values: the executed command ends up in the
// archive, so a password taken from a secret must not be resolvable into it.
func TestTlsArgsNeverCarryASecret(t *testing.T) {
	pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{Name: "agent", Env: []v1.EnvVar{
		{Name: envExtensionClientCertChain, Value: "/certs/tls.crt"},
		{Name: envExtensionClientCertPass, Value: "hunter2"},
	}}}}}

	for _, arg := range extensionTlsArgs(pod, "agent") {
		if strings.Contains(arg, "hunter2") {
			t.Fatalf("the key password ended up in the arguments: %v", extensionTlsArgs(pod, "agent"))
		}
	}
}

// TestCurlArgsDoNotShareTheTlsSlice guards the parallel tests against writing into each other's arguments.
func TestCurlArgsDoNotShareTheTlsSlice(t *testing.T) {
	tls := make([]string, 0, 8)
	tls = append(tls, "--insecure")

	first := curlArgs(tls, "-v", "https://first")
	second := curlArgs(tls, "-v", "https://second")

	if slices.Contains(first, "https://second") || !slices.Contains(first, "https://first") {
		t.Errorf("the second invocation wrote into the first: %v", first)
	}
	if !slices.Contains(second, "https://second") {
		t.Errorf("unexpected second invocation: %v", second)
	}
}

// TestAuthenticatedPlatformScriptKeepsTheKeyOut is the whole point of going through a shell: the key is referenced
// by name so the container's shell expands it, and the request carrying it prints only a status code, because curl
// prints the headers it sends.
func TestAuthenticatedPlatformScriptKeepsTheKeyOut(t *testing.T) {
	script := authenticatedPlatformScript(nil, "https://platform/agent")

	if !strings.Contains(script, `"X-Agent-Key: $STEADYBIT_AGENT_KEY"`) {
		t.Errorf("expected the header to reference the variable, got:\n%s", script)
	}
	_, authenticated, found := strings.Cut(script, "\necho\n")
	if !found {
		t.Fatalf("expected the two requests to be separated, got:\n%s", script)
	}
	if strings.Contains(authenticated, "'-v'") {
		t.Errorf("the authenticated request must not be verbose, curl would print the header:\n%s", authenticated)
	}
	if !strings.Contains(script, "%{http_code}") {
		t.Errorf("expected the authenticated request to report its status code, got:\n%s", script)
	}
}

func TestAuthenticatedPlatformScriptQuotesItsArguments(t *testing.T) {
	script := authenticatedPlatformScript([]string{"--insecure"}, "https://platform/agent?a=b&c='d'")

	// everything but the deliberate variable reference is quoted, so the shell has nothing left to interpret
	if strings.Contains(script, "&c='d'") {
		t.Errorf("expected the url to be quoted, got:\n%s", script)
	}
	if !strings.Contains(script, `'https://platform/agent?a=b&c='\''d'\'''`) {
		t.Errorf("expected the embedded quotes to be escaped, got:\n%s", script)
	}
}

func TestCustomizationPassesTheAgentKeyThroughWithoutReadingIt(t *testing.T) {
	pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{Name: "agent", Env: []v1.EnvVar{
		{Name: envAgentKey, ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{Key: "key", LocalObjectReference: v1.LocalObjectReference{Name: "steadybit-agent"}}}},
		{Name: "UNRELATED", Value: "not inherited"},
	}}}}}

	customization := customizationForContainer(pod, "agent")

	if customization == nil || len(customization.Env) != 1 {
		t.Fatalf("expected exactly the agent key to be passed through, got %+v", customization)
	}
	if customization.Env[0].ValueFrom == nil || customization.Env[0].Value != "" {
		t.Error("expected the secret reference to be passed through rather than a value")
	}
	if !authenticatesWithAgentKey(pod, "agent") {
		t.Error("expected the agent key to be recognised")
	}
}

// TestAgentKeyIsNotProbedForAnOAuth2Agent guards against a misleading archive: such an agent may still be given a
// key, but it authenticates with a token instead, so the platform rejects the key and a 401 would read as if the
// agent's credentials were broken.
func TestAgentKeyIsNotProbedForAnOAuth2Agent(t *testing.T) {
	withProvider := func(provider string) *v1.Pod {
		env := []v1.EnvVar{{Name: envAgentKey, ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{Key: "key"}}}}
		if provider != "" {
			env = append(env, v1.EnvVar{Name: envAuthProvider, Value: provider})
		}
		return &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{Name: "agent", Env: env}}}}
	}

	if !authenticatesWithAgentKey(withProvider(""), "agent") {
		t.Error("expected the key to be probed when no provider is configured, which is the default")
	}
	if !authenticatesWithAgentKey(withProvider("AGENT_KEY"), "agent") {
		t.Error("expected the key to be probed for an agent that authenticates with it")
	}
	if authenticatesWithAgentKey(withProvider("OAUTH2"), "agent") {
		t.Error("expected the key not to be probed for an agent authenticating with a token")
	}
	if authenticatesWithAgentKey(withProvider("oauth2"), "agent") {
		t.Error("expected the provider to be recognised regardless of case")
	}
}
