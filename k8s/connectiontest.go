// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package k8s

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/limit"
	"github.com/steadybit/steadybit-debug/output"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// ephemeralContainerStartTimeout covers pulling the image on a node that does not have it yet. A container that
	// cannot start at all does not need it: the kubelet reports why within a second or two and the wait ends there.
	ephemeralContainerStartTimeout = 60 * time.Second
	// connectionTestTimeout is the last resort for a test that hangs - the tools below bound themselves.
	connectionTestTimeout = 30 * time.Second
	// keepAlive is how long the shared container waits around for tests to be executed in it. It has to outlast the
	// whole sequence of a pod, which is not just the tests: the admin endpoints and the extension connections are
	// collected between the two batches of them, and every test also queues for an execution slot with the rest of
	// the run. Once it ends, the tests that are left report that their container is gone, so this errs on the
	// generous side - the container is idling either way, and Kubernetes never removes it from the pod regardless.
	keepAlive = "600"

	// curl reports an unreachable target itself instead of being killed halfway through its output. Everything
	// worth collecting - the connection attempt, the TLS handshake, the response and its headers - either arrived
	// within milliseconds or is not going to arrive at all. A websocket upgrade in particular is answered with
	// '101 Switching Protocols' and then stays silent until a peer sends a frame, which curl would sit and wait for.
	curlConnectTimeout = "5"
	curlMaxTime        = "5"
)

// terminalWaitingReasons are the reasons the kubelet reports for a container that is never going to start, so
// that a test gives up on it right away instead of waiting for the start timeout. A hardened pod rejecting the
// image of a debug container is reported as CreateContainerConfigError.
var terminalWaitingReasons = map[string]bool{
	"CreateContainerConfigError": true,
	"CreateContainerError":       true,
	"ErrImagePull":               true,
	"ImagePullBackOff":           true,
	"InvalidImageName":           true,
	"RunContainerError":          true,
}

var ephemeralContainers atomic.Uint64

// ephemeralContainerName has to be unique for the lifetime of the pod, because ephemeral containers are never
// removed from it again: reusing a name makes `kubectl debug` refer to an earlier container instead of adding a
// new one. A timestamp alone is not enough, several containers are added within the same second.
func ephemeralContainerName() string {
	return fmt.Sprintf("steadybit-debug-%d-%d", time.Now().Unix(), ephemeralContainers.Add(1))
}

// ConnectionTester runs the connection tests of one pod from inside that pod. Tests sharing an image also share
// the ephemeral container they are executed in: starting one costs seconds, and since Kubernetes never removes it
// from the pod again, one container per test would both slow the run down and leave a long trail behind.
type ConnectionTester struct {
	config          *config.Config
	namespace       string
	pod             string
	targetContainer string

	mutex      sync.Mutex
	containers map[string]*sharedContainer
}

type sharedContainer struct {
	once sync.Once
	name string
	err  error
}

func NewConnectionTester(cfg *config.Config, namespace string, pod string, targetContainer string) *ConnectionTester {
	return &ConnectionTester{
		config:          cfg,
		namespace:       namespace,
		pod:             pod,
		targetContainer: targetContainer,
		containers:      make(map[string]*sharedContainer),
	}
}

func (t *ConnectionTester) AddHttpConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding http connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	t.exec(outputPath, t.config.Agent.CurlImage, curlArgs("-v", url)...)
}

func (t *ConnectionTester) AddWebsocketCurlHttp1ConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding curl http1 connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	t.exec(outputPath, t.config.Agent.CurlImage, curlArgs(append(websocketArgs(url), "--http1.1")...)...)
}

func (t *ConnectionTester) AddWebsocketCurlHttp2ConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding curl http2 connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	t.exec(outputPath, t.config.Agent.CurlImage, curlArgs(websocketArgs(url)...)...)
}

func (t *ConnectionTester) AddTracerouteConnectionTest(outputPath string, host string) {
	log.Debug().Msgf("Adding traceroute connection test for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	// one probe per hop with a one second wait, so an unreachable host is reported well within connectionTestTimeout
	t.exec(outputPath, t.config.Agent.TracerouteImage, "traceroute", "-m", "15", "-w", "1", "-q", "1", host)
}

// AddWebsocketWebsocatConnectionTest gets an ephemeral container of its own: the websocat image has no shell to
// keep a shared container alive with, so the test has to be the container's command.
func (t *ConnectionTester) AddWebsocketWebsocatConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding websocat connection test for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	wsUrl := strings.ReplaceAll(url, "https://", "wss://")
	wsUrl = strings.ReplaceAll(wsUrl, "http://", "ws://")

	name := ephemeralContainerName()

	// the test is the container's command, so there is nothing to wait for before running it - instead the
	// container is watched while it runs and the command is given up on as soon as the kubelet says it will never
	// start, with that reason reported instead of the kill signal. Having terminated is not a failure here: this
	// container is meant to run the test and exit, and aborting on that would cut its output short.
	ctx, abort := context.WithCancelCause(context.Background())
	defer abort(nil)
	watch, stopWatching := context.WithTimeout(context.Background(), ephemeralContainerStartTimeout)
	defer stopWatching()
	go func() {
		if _, err := t.awaitEphemeralContainer(watch, name); err != nil && watch.Err() == nil {
			abort(err)
		}
	}()

	args := append(t.debugArgs(name, t.config.Agent.WebsocatImage, "-it"), "websocat", wsUrl+"/ws", "-v")
	output.AddCommandOutput(ctx, output.AddCommandOutputOptions{
		Config:           t.config,
		CommandName:      "kubectl",
		CommandArgs:      args,
		OutputPath:       outputPath,
		Stdin:            strings.NewReader(" "),
		Timeout:          ephemeralContainerStartTimeout + connectionTestTimeout,
		ExecutionContext: t.executionContext(),
	})
}

func curlArgs(args ...string) []string {
	return append([]string{"curl", "--connect-timeout", curlConnectTimeout, "--max-time", curlMaxTime}, args...)
}

func websocketArgs(url string) []string {
	return []string{"-v", "--http1.1", url + "/ws",
		"-H", "upgrade: websocket", "-H", "connection: Upgrade",
		"-H", "sec-websocket-key: dummy", "-H", "sec-websocket-Version: 13", "-v"}
}

// exec runs one test in the shared container of imageName, adding that container on first use.
func (t *ConnectionTester) exec(outputPath string, imageName string, command ...string) {
	container, err := t.container(imageName)
	if err != nil {
		// the reason was reported once when the container failed, so this only says which test it costs
		log.Debug().Msgf("Skipping '%s' for '%s' in namespace '%s': %s", outputPath, t.pod, t.namespace, err)
		output.AddFailureOutput(outputPath, strings.Join(command, " "), err)
		return
	}

	args := []string{"exec", t.pod, "-n", t.namespace, "-c", container, "--"}
	args = append(args, command...)

	output.AddCommandOutput(context.Background(), output.AddCommandOutputOptions{
		Config:           t.config,
		CommandName:      "kubectl",
		CommandArgs:      args,
		OutputPath:       outputPath,
		Timeout:          connectionTestTimeout,
		ExecutionContext: t.executionContext(),
	})
}

// container returns the ephemeral container running imageName, starting it once for all tests that need it.
func (t *ConnectionTester) container(imageName string) (string, error) {
	t.mutex.Lock()
	shared, ok := t.containers[imageName]
	if !ok {
		shared = &sharedContainer{}
		t.containers[imageName] = shared
	}
	t.mutex.Unlock()

	shared.once.Do(func() {
		shared.name, shared.err = t.startContainer(imageName)
		if shared.err != nil {
			log.Warn().Msgf("No connection test can run in '%s' of '%s' in namespace '%s': %s",
				imageName, t.pod, t.namespace, shared.err)
		}
	})
	return shared.name, shared.err
}

func (t *ConnectionTester) startContainer(imageName string) (string, error) {
	name := ephemeralContainerName()

	ctx, cancel := context.WithTimeout(context.Background(), ephemeralContainerStartTimeout)
	defer cancel()

	release := limit.Commands.Acquire()
	cmd := exec.CommandContext(ctx, "kubectl", append(t.debugArgs(name, imageName), "sleep", keepAlive)...)
	log.Debug().Msgf("Executing: %s", cmd.String())
	out, err := cmd.CombinedOutput()
	release()
	if err != nil {
		return "", fmt.Errorf("failed to add ephemeral container: %s: %s", err, strings.TrimSpace(string(out)))
	}

	state, err := t.awaitEphemeralContainer(ctx, name)
	if err != nil {
		return "", err
	}
	if state != containerRunning {
		// the shared container only runs 'sleep', so anything else means it is not usable for the tests
		return "", fmt.Errorf("ephemeral container did not stay running")
	}
	return name, nil
}

func (t *ConnectionTester) debugArgs(name string, imageName string, extra ...string) []string {
	args := []string{"debug"}
	args = append(args, extra...)
	return append(args, t.pod, "-n", t.namespace, "--target", t.targetContainer, "--image", imageName, "-c", name, "--")
}

// awaitEphemeralContainer polls until the container has left the pending state and reports what it reached, or
// the kubelet's reason as an error as soon as it is clear that it will never start. Whether having terminated is
// a problem depends on what the container was asked to run, so that is left to the caller.
func (t *ConnectionTester) awaitEphemeralContainer(ctx context.Context, name string) (containerState, error) {
	client, err := t.config.Kubernetes.Client()
	if err != nil {
		return containerPending, err
	}

	for {
		pod, err := client.CoreV1().Pods(t.namespace).Get(ctx, t.pod, metav1.GetOptions{})
		if err != nil {
			log.Debug().Err(err).Msgf("Failed to read '%s' while waiting for its ephemeral container", t.pod)
		} else if state, err := ephemeralContainerState(pod, name); state != containerPending {
			return state, err
		}

		select {
		case <-ctx.Done():
			return containerPending, fmt.Errorf("ephemeral container did not start within %s", ephemeralContainerStartTimeout)
		case <-time.After(time.Second):
		}
	}
}

type containerState int

const (
	// containerPending is a container that has not started yet, but still might
	containerPending containerState = iota
	containerRunning
	// containerTerminated has run - which is what is expected of a container whose command is the test itself
	containerTerminated
	// containerFailed is never going to start, and comes with the reason the kubelet gave
	containerFailed
)

func ephemeralContainerState(pod *v1.Pod, name string) (containerState, error) {
	for _, status := range pod.Status.EphemeralContainerStatuses {
		if status.Name != name {
			continue
		}
		if status.State.Running != nil {
			return containerRunning, nil
		}
		if status.State.Terminated != nil {
			return containerTerminated, nil
		}
		if waiting := status.State.Waiting; waiting != nil && terminalWaitingReasons[waiting.Reason] {
			return containerFailed, fmt.Errorf("%s: %s", waiting.Reason, strings.TrimSpace(waiting.Message))
		}
	}
	return containerPending, nil
}

func (t *ConnectionTester) executionContext() string {
	return fmt.Sprintf("%s/%s", t.namespace, t.pod)
}
