// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package k8s

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/limit"
	"github.com/steadybit/steadybit-debug/output"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/exec"
	"slices"
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
	// customization is what the ephemeral containers inherit from the container they target
	customization *debugCustomization
	// platformTls and extensionTls are the TLS settings the target container uses for those two destinations
	platformTls  []string
	extensionTls []string
	// hasAgentKey is set when the target container is given the key the agent authenticates with
	hasAgentKey bool
	// authProvider is how the agent authenticates, which decides what the platform accepts
	authProvider string

	mutex      sync.Mutex
	containers map[string]*sharedContainer
}

type sharedContainer struct {
	once sync.Once
	name string
	err  error
}

func NewConnectionTester(cfg *config.Config, pod *v1.Pod, targetContainer string) *ConnectionTester {
	return &ConnectionTester{
		config:          cfg,
		namespace:       pod.Namespace,
		pod:             pod.Name,
		targetContainer: targetContainer,
		customization:   customizationForContainer(pod, targetContainer),
		platformTls:     platformTlsArgs(pod, targetContainer),
		extensionTls:    extensionTlsArgs(pod, targetContainer),
		hasAgentKey:     authenticatesWithAgentKey(pod, targetContainer),
		authProvider:    literalEnv(pod, targetContainer)[envAuthProvider],
		containers:      make(map[string]*sharedContainer),
	}
}

// debugCustomization is the partial container spec that `kubectl debug --custom` merges into the ephemeral
// container. A debug container inherits nothing from the container it targets, which is why it cannot start in a
// pod that demands a non-root user: the kubelet has no numeric UID to verify it against. Handing it the target's
// own security context solves that, and the target's volume mounts let the test read the same certificates the
// container under test uses.
type debugCustomization struct {
	SecurityContext *v1.SecurityContext `json:"securityContext,omitempty"`
	VolumeMounts    []v1.VolumeMount    `json:"volumeMounts,omitempty"`
	// Env is passed through as it is written in the pod, so a value coming from a secret is resolved by Kubernetes
	// inside the container and never has to be read here
	Env []v1.EnvVar `json:"env,omitempty"`
}

// customizationForContainer returns nil when there is nothing to inherit, so that no --custom is passed at all.
func customizationForContainer(pod *v1.Pod, containerName string) *debugCustomization {
	for _, container := range pod.Spec.Containers {
		if container.Name != containerName {
			continue
		}

		customization := &debugCustomization{VolumeMounts: container.VolumeMounts}
		for _, variable := range container.Env {
			if variable.Name == envAgentKey {
				customization.Env = append(customization.Env, variable)
			}
		}
		if container.SecurityContext != nil {
			securityContext := container.SecurityContext.DeepCopy()
			// the tools in the debug container may need to write, and a read-only root filesystem is not something
			// the pod demands of them - unlike the user they have to run as
			securityContext.ReadOnlyRootFilesystem = nil
			customization.SecurityContext = securityContext
		}
		if customization.SecurityContext == nil && len(customization.VolumeMounts) == 0 && len(customization.Env) == 0 {
			return nil
		}
		return customization
	}
	return nil
}

// The agent is told about the TLS it uses through these variables, holding paths inside its own container - which
// is where the tests run too, with the target's volume mounts inherited, so the same paths resolve for them.
// Mapping them onto curl is what makes a test reproduce what the agent actually does instead of an anonymous
// request that an extension expecting a client certificate rejects.
const (
	envExtensionClientCertChain = "STEADYBIT_AGENT_EXTENSIONS_CLIENT_CERT_CHAIN_FILE"
	envExtensionClientCertKey   = "STEADYBIT_AGENT_EXTENSIONS_CLIENT_CERT_KEY_FILE"
	envExtensionClientCertPass  = "STEADYBIT_AGENT_EXTENSIONS_CLIENT_CERT_PASSWORD"
	envExtensionServerCert      = "STEADYBIT_AGENT_EXTENSIONS_SERVER_CERT"
	envExtensionInsecure        = "STEADYBIT_AGENT_EXTENSIONS_INSECURE_SKIP_VERIFY"
	envPlatformInsecure         = "STEADYBIT_AGENT_TLS_INSECURE_SKIP_VERIFY"
	// envAgentKey holds what the agent authenticates itself with, sent as the X-Agent-Key header
	envAgentKey = "STEADYBIT_AGENT_KEY"
	// envAuthProvider decides whether that key is what the platform expects at all. The agent defaults to AGENT_KEY
	// and can be switched to OAUTH2, where it presents a token fetched from an issuer instead - the key may still be
	// set in that case, but the platform rejects it, and reporting that as a rejected key would be misleading.
	envAuthProvider    = "STEADYBIT_AGENT_AUTH_PROVIDER"
	authProviderOAuth2 = "OAUTH2"
)

// authenticatesWithAgentKey reports whether presenting the agent key says anything about the agent's credentials.
func authenticatesWithAgentKey(pod *v1.Pod, containerName string) bool {
	env := literalEnv(pod, containerName)
	if _, set := env[envAgentKey]; !set {
		return false
	}
	return !strings.EqualFold(env[envAuthProvider], authProviderOAuth2)
}

func platformTlsArgs(pod *v1.Pod, containerName string) []string {
	env := literalEnv(pod, containerName)

	var args []string
	if env[envPlatformInsecure] == "true" {
		args = append(args, "--insecure")
	}
	return args
}

func extensionTlsArgs(pod *v1.Pod, containerName string) []string {
	env := literalEnv(pod, containerName)

	var args []string
	if chain := env[envExtensionClientCertChain]; chain != "" {
		if _, protected := env[envExtensionClientCertPass]; protected {
			// the password is only ever delivered from a secret, and the command being executed is written into the
			// archive - so the key stays unused rather than putting its password in front of support
			log.Warn().Msgf("The client certificate of '%s' in namespace '%s' is password protected, so the connection tests cannot present it", pod.Name, pod.Namespace)
		} else {
			args = append(args, "--cert", chain)
			if key := env[envExtensionClientCertKey]; key != "" {
				args = append(args, "--key", key)
			}
		}
	}
	if serverCert := env[envExtensionServerCert]; serverCert != "" {
		args = append(args, "--cacert", serverCert)
	}
	if env[envExtensionInsecure] == "true" {
		args = append(args, "--insecure")
	}
	return args
}

// literalEnv collects the environment of the target container, skipping everything that is not spelled out in the
// pod: a value taken from a secret is deliberately not resolved, so that it cannot end up on a command line that
// is written into the archive.
func literalEnv(pod *v1.Pod, containerName string) map[string]string {
	env := make(map[string]string)
	for _, container := range pod.Spec.Containers {
		if container.Name != containerName {
			continue
		}
		for _, variable := range container.Env {
			if variable.ValueFrom == nil {
				env[variable.Name] = variable.Value
			} else {
				// recorded without its value, so a caller can tell it is set
				env[variable.Name] = ""
			}
		}
	}
	return env
}

// writeTo stores the customization where kubectl can read it, and returns the function removing it again.
func (c *debugCustomization) writeTo(directory string) (string, func(), error) {
	file, err := os.CreateTemp(directory, "steadybit-debug-container-*.json")
	if err != nil {
		return "", func() {}, err
	}
	remove := func() {
		_ = os.Remove(file.Name())
	}

	content, err := json.Marshal(c)
	if err == nil {
		_, err = file.Write(content)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		remove()
		return "", func() {}, err
	}
	return file.Name(), remove, nil
}

// AddPlatformConnectionTest reaches the platform the way the agent reaches it. Without the agent key the platform
// answers 401 to everything, so the test asks twice: once verbosely and anonymously, for the connection itself, and
// once with the key to find out whether it is accepted.
func (t *ConnectionTester) AddPlatformConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding platform connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)

	if !t.hasAgentKey {
		var notes []string
		if strings.EqualFold(t.authProvider, authProviderOAuth2) {
			notes = []string{
				"This agent authenticates with a token it fetches from an OAuth2 issuer, which this test cannot",
				"reproduce, so the request below carries no credentials and 401 is the expected answer.",
			}
		}
		t.execWithNotes(outputPath, t.config.Agent.CurlImage, notes, curlArgs(t.platformTls, "-v", url)...)
		return
	}

	notes := []string{
		"The second request presents the agent key. A 401 for it means the key was rejected;",
		"the platform answers an accepted key with 404 or 405, because it authenticates before it routes.",
	}
	t.execWithNotes(outputPath, t.config.Agent.CurlImage, notes, "sh", "-c", authenticatedPlatformScript(t.platformTls, url))
}

// authenticatedPlatformScript keeps the key out of the archive twice over: the header is written for the shell in
// the container to expand, so only the variable name is recorded, and the request carrying it prints nothing but
// its status code - curl would otherwise print the header it sent.
func authenticatedPlatformScript(tls []string, url string) string {
	return fmt.Sprintf("%s\necho\n%s\n",
		shellJoin(curlArgs(tls, "-v", url)),
		shellJoin(curlArgs(tls, "-s", "-o", "/dev/null", "-w", "authenticated request: HTTP %{http_code}\\n"))+
			" -H \"X-Agent-Key: $"+envAgentKey+"\" "+shellQuote(url))
}

// shellJoin quotes every argument, so that only what this function adds itself is left for the shell to interpret.
func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

// AddExtensionConnectionTest reaches an extension the way the agent reaches it, which for a setup using mutual TLS
// means presenting the agent's client certificate - without it the extension rejects the request and the test says
// nothing about whether the agent itself can get through.
func (t *ConnectionTester) AddExtensionConnectionTest(outputPath string, connection Connection) {
	log.Debug().Msgf("Adding extension connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	// whether the connection expects credentials decides how to read a 401 here: as the endpoint being unreachable
	// or as it answering exactly as it should to a request that carries none
	notes := []string{fmt.Sprintf("Connection requires authentication: %t", connection.Auth)}
	t.execWithNotes(outputPath, t.config.Agent.CurlImage, notes, curlArgs(t.extensionTls, "-v", connection.Url)...)
}

func (t *ConnectionTester) AddWebsocketCurlHttp1ConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding curl http1 connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	t.exec(outputPath, t.config.Agent.CurlImage, curlArgs(t.platformTls, append(websocketArgs(url), "--http1.1")...)...)
}

func (t *ConnectionTester) AddWebsocketCurlHttp2ConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding curl http2 connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	t.exec(outputPath, t.config.Agent.CurlImage, curlArgs(t.platformTls, websocketArgs(url)...)...)
}

// AddWebsocketConnectionTest opens an actual websocket, which validates the handshake beyond the plain upgrade
// the tests above send: curl checks the server's sec-websocket-accept before switching protocols.
func (t *ConnectionTester) AddWebsocketConnectionTest(outputPath string, url string) {
	log.Debug().Msgf("Adding websocket connection test via curl for '%s' in namespace '%s' to '%s'", t.pod, t.namespace, outputPath)
	wsUrl := strings.ReplaceAll(url, "https://", "wss://")
	wsUrl = strings.ReplaceAll(wsUrl, "http://", "ws://")
	t.exec(outputPath, t.config.Agent.CurlImage, curlArgs(t.platformTls, "-v", wsUrl+"/ws")...)
}

// curlArgs assembles a curl invocation. It concatenates into a new slice on purpose: appending to the stored TLS
// arguments would let the tests running in parallel write into each other's arguments.
func curlArgs(tls []string, args ...string) []string {
	return slices.Concat([]string{"curl", "--connect-timeout", curlConnectTimeout, "--max-time", curlMaxTime}, tls, args)
}

func websocketArgs(url string) []string {
	return []string{"-v", "--http1.1", url + "/ws",
		"-H", "upgrade: websocket", "-H", "connection: Upgrade",
		"-H", "sec-websocket-key: dummy", "-H", "sec-websocket-Version: 13", "-v"}
}

// exec runs one test in the shared container of imageName, adding that container on first use.
func (t *ConnectionTester) exec(outputPath string, imageName string, command ...string) {
	t.execWithNotes(outputPath, imageName, nil, command...)
}

func (t *ConnectionTester) execWithNotes(outputPath string, imageName string, notes []string, command ...string) {
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
		Notes:            notes,
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

	out, err := t.addEphemeralContainer(ctx, name, imageName, t.customization)
	if err != nil && t.customization != nil && bytes.Contains(out, []byte("unknown flag: --custom")) {
		// kubectl learned --custom in 1.30; an older one can only add a container that inherits nothing, which is
		// still what it did before this was passed
		log.Debug().Msgf("This kubectl does not know 'kubectl debug --custom', the connection tests cannot inherit anything from '%s'", t.targetContainer)
		out, err = t.addEphemeralContainer(ctx, name, imageName, nil)
	}
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

func (t *ConnectionTester) addEphemeralContainer(ctx context.Context, name string, imageName string, customization *debugCustomization) ([]byte, error) {
	args := t.debugArgs(name, imageName)
	if customization != nil {
		path, remove, err := customization.writeTo(os.TempDir())
		if err != nil {
			return nil, err
		}
		defer remove()
		args = append(args, "--custom", path)
	}
	args = append(args, "--", "sleep", keepAlive)

	release := limit.Commands.Acquire()
	defer release()

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	log.Debug().Msgf("Executing: %s", cmd.String())
	return cmd.CombinedOutput()
}

func (t *ConnectionTester) debugArgs(name string, imageName string) []string {
	return []string{"debug", t.pod, "-n", t.namespace, "--target", t.targetContainer, "--image", imageName, "-c", name}
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
