/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extensions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/output"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

type TraverseExtensionEndpointsOptions struct {
	Config       *config.Config
	PodNamespace string
	PodName      string
	BaseUrl      string
	PathForPod   string
}

type urlsToCurl struct {
	Method string
	Path   string
}

type extensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
	event_kit_api.EventListenerList `json:",inline"`
}

func TraverseExtensionEndpoints(options TraverseExtensionEndpointsOptions) {
	podUrl, err := url.Parse(options.BaseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse URL '%s'", options.BaseUrl)
		return
	}

	cmd := exec.Command("kubectl", "port-forward", "-n", options.PodNamespace, fmt.Sprintf("pod/%s", options.PodName), fmt.Sprintf(":%s", podUrl.Port()))
	log.Debug().Msgf("Executing: %s", cmd.String())

	cmdOut, _ := cmd.StdoutPipe()
	err = cmd.Start()
	if err != nil {
		log.Error().Msgf("Failed to port-forward for '%s' in namespace '%s", options.PodName, options.PodNamespace)
		return
	}

	defer func() {
		err = cmd.Process.Kill()
		if err != nil {
			log.Error().Msgf("Failed to stop port-forward for '%s' in namespace '%s", options.PodName, options.PodNamespace)
			return
		}
	}()

	var r = regexp.MustCompile("^Forwarding from .+:(\\d+) -> \\d+$")
	scanner := bufio.NewScanner(cmdOut)
	for scanner.Scan() {
		m := r.FindStringSubmatch(scanner.Text())
		if m != nil {
			podUrl.Host = fmt.Sprintf("localhost:%s", m[1])
			break
		}
	}

	response, err := http.Get(podUrl.String())
	if err != nil {
		log.Error().Msgf("Failed to get '%s'", podUrl.String())
		return
	}

	defer closeResponse(response)

	extensionListResponse := extensionListResponse{}
	body, err := io.ReadAll(response.Body)
	if err := json.Unmarshal(body, &extensionListResponse); err != nil {
		log.Err(err).Msgf("Failed to parse response body: %s", string(body))
	}

	urlsToCurlSlice := make([]urlsToCurl, 0)
	urlsToCurlSlice = append(urlsToCurlSlice, urlsToCurl{Method: "GET", Path: "/"})

	for _, action := range extensionListResponse.Actions {
		urlsToCurlSlice = append(urlsToCurlSlice, urlsToCurl{Method: string(action.Method), Path: action.Path})
	}

	for _, discovery := range extensionListResponse.Discoveries {
		urlsToCurlSlice = append(urlsToCurlSlice, urlsToCurl{Method: string(discovery.Method), Path: discovery.Path})
		findDiscoveredTargetsUrl(discovery.Method, discovery.Path, podUrl, &urlsToCurlSlice)
	}

	for _, targetAttribute := range extensionListResponse.TargetAttributes {
		urlsToCurlSlice = append(urlsToCurlSlice, urlsToCurl{Method: string(targetAttribute.Method), Path: targetAttribute.Path})
	}

	for _, targetType := range extensionListResponse.TargetTypes {
		urlsToCurlSlice = append(urlsToCurlSlice, urlsToCurl{Method: string(targetType.Method), Path: targetType.Path})
	}

	for _, eventListener := range extensionListResponse.EventListeners {
		urlsToCurlSlice = append(urlsToCurlSlice, urlsToCurl{Method: string(eventListener.Method), Path: eventListener.Path})
	}

	for _, urlToCurl := range urlsToCurlSlice {
		filename := strings.ReplaceAll(urlToCurl.Path, "/", "_")
		outputPath := fmt.Sprintf("%s/%s.yml", options.PathForPod, filename)
		fullUrl := podUrl.JoinPath(urlToCurl.Path)
		output.AddCommandOutput(output.AddCommandOutputOptions{
			Config:      options.Config,
			CommandName: "curl",
			CommandArgs: []string{"-s", fullUrl.String(), "-X", strings.ToUpper(urlToCurl.Method)},
			OutputPath:  outputPath,
		})
	}
}

func findDiscoveredTargetsUrl(method discovery_kit_api.DescribingEndpointReferenceMethod, path string, podUrl *url.URL, urlsToCurlSlicePtr *[]urlsToCurl) {
	fullUrl := podUrl.JoinPath(path)
	response, err := doHttpRequest(fullUrl, string(method))
	if err != nil {
		log.Error().Msgf("Failed to get '%s'", fullUrl.String())
		return
	}

	defer closeResponse(response)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Msgf("Failed to read response body")
		return
	}

	discoveryDescriptionResponse := discovery_kit_api.DiscoveryDescription{}
	if err := json.Unmarshal(body, &discoveryDescriptionResponse); err != nil {
		log.Err(err).Msgf("Failed to parse response body: %s", string(body))
	}

	*urlsToCurlSlicePtr = append(*urlsToCurlSlicePtr, urlsToCurl{Method: string(discoveryDescriptionResponse.Discover.Method), Path: discoveryDescriptionResponse.Discover.Path})
}

func closeResponse(response *http.Response) {
	func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Msgf("Failed to close response body")
			return
		}
	}(response.Body)
}

func doHttpRequest(url *url.URL, method string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(strings.ToUpper(method), url.String(), nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}
