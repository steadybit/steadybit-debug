/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extensions

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/k8s"
	"github.com/steadybit/steadybit-debug/output"
	"net/url"
	"strings"
	"sync"
)

type TraverseExtensionEndpointsOptions struct {
	Config       *config.Config
	PodNamespace string
	PodName      string
	Port         int
	PathForPod   string
	UseHttps     bool
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
	baseUrl := fmt.Sprintf("http://localhost:%d/", options.Port)

	podUrl, err := url.Parse(baseUrl)
	if err != nil {
		log.Error().Msgf("Failed to parse URL '%s'", baseUrl)
		return
	}
	forwardingHostWithPort, cmd, err := k8s.PreparePortforwarding(k8s.PodConfig{
		PodNamespace: options.PodNamespace,
		PodName:      options.PodName,
		Config:       options.Config,
	}, options.Port)
	if err != nil {
		log.Error().Msgf("Failed to prepare port forwarding. Got error: %s", err)
		return
	}

	defer func() {
		k8s.KillProcess(cmd, k8s.PodConfig{
			PodNamespace: options.PodNamespace,
			PodName:      options.PodName,
			Config:       options.Config,
		})
	}()

	podUrl.Host = forwardingHostWithPort
	body, err := output.DoHttp(output.HttpOptions{
		Config:     options.Config,
		Method:     "GET",
		URL:        *podUrl,
		UseHttps:   options.UseHttps,
		FormatJson: false,
	})
	if err != nil {
		if strings.Contains(err.Error(), "remote error: tls: bad certificate") {
			log.Error().Msgf("Please provide proper TLS certificates for %s ", options.PodNamespace+"/"+options.PodName)
		}
		log.Error().Msgf("Failed to get '%s'", podUrl.String())
		return
	}

	extensionListResponse := extensionListResponse{}

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
		findDiscoveredTargetsUrl(options.Config, discovery.Method, discovery.Path, podUrl, options.UseHttps, &urlsToCurlSlice)
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

	var wg sync.WaitGroup
	for _, urlToCurl := range urlsToCurlSlice {
		outputPath := getOutputPath(options.PathForPod, urlToCurl)
		fullUrl := podUrl.JoinPath(urlToCurl.Path)
		wg.Add(1)
		urlToCurl := urlToCurl
		go func() {
			defer wg.Done()
			output.AddHttpOutput(output.AddHttpOutputOptions{
				Config:     options.Config,
				URL:        *fullUrl,
				Method:     urlToCurl.Method,
				OutputPath: outputPath,
				FormatJson: true,
				UseHttps:   options.UseHttps,
			})
		}()
	}
	wg.Wait()
}

func getOutputPath(pathForPod string, urlToCurl urlsToCurl) string {
	filename := strings.ReplaceAll(urlToCurl.Path, "/", "_")
	filename = fmt.Sprintf("%s_%s.yml", urlToCurl.Method, filename)
	outputPath := fmt.Sprintf("%s/%s", pathForPod, filename)
	return outputPath
}

func findDiscoveredTargetsUrl(cfg *config.Config, method discovery_kit_api.ReadHttpMethod, path string, podUrl *url.URL, useHttps bool, urlsToCurlSlicePtr *[]urlsToCurl) {
	fullUrl := podUrl.JoinPath(path)
	body, err := output.DoHttp(output.HttpOptions{
		Config:     cfg,
		Method:     string(method),
		URL:        *fullUrl,
		UseHttps:   useHttps,
		FormatJson: false,
	})
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
