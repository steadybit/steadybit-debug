// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package config

import (
	"errors"
	"flag"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/limit"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Config struct {
	OutputPath           string                     `yaml:"outputPath" short:"o" long:"output" description:"Path to output directory that will contain the debugging information"`
	NoCleanup            bool                       `yaml:"noCleanup" long:"no-cleanup" description:"Skip output directory deletion on command completion?"`
	MaxConcurrency       int                        `yaml:"maxConcurrency" long:"max-concurrency" description:"Maximum number of pods/nodes collected in parallel. Lower it to reduce the memory and CPU footprint on large clusters, 0 disables the limit"`
	SkipConnectionTests  bool                       `yaml:"skipConnectionTests" long:"skip-connection-tests" description:"Skip the connectivity tests that run inside the agent pod. They need an ephemeral container, which a pod with a restrictive security context refuses to start"`
	Kubernetes           KubernetesConfig           `yaml:"kubernetes"`
	Platform             PlatformConfig             `yaml:"platform"`
	PlatformPortSplitter PlatformportSplitterConfig `yaml:"platform-port-splitter"`
	Agent                AgentConfig                `yaml:"agent"`
	Tls                  Tls                        `yaml:"tls"`
}

type PlatformConfig struct {
	Deployment     string `yaml:"deployment" long:"platform-deployment" description:"Kubernetes deployment name of the Steadybit platform"`
	Namespace      string `yaml:"namespace" long:"platform-namespace" description:"Kubernetes namespace name of the Steadybit platform"`
	ExportDatabase bool   `yaml:"exportDatabase" long:"export-database" description:"Export database?"`
}

type PlatformportSplitterConfig struct {
	Deployment string `yaml:"deployment" long:"platform-splitter-deployment" description:"Kubernetes deployment name of the Steadybit platform splitter"`
	Namespace  string `yaml:"namespace" long:"platform-splitter-namespace" description:"Kubernetes namespace name of the Steadybit platform splitter"`
}

type AgentConfig struct {
	StatefulSet     string `yaml:"statefulSet" long:"agent-stateful-set" description:"Kubernetes stateful set name of the Steadybit agent"`
	Namespace       string `yaml:"namespace" long:"agent-namespace" description:"Kubernetes namespace name of the Steadybit agent"`
	CurlImage       string `yaml:"curlImage" long:"agent-curl-image" description:"Image to use for connection testing with curl installed"`
	WebsocatImage   string `yaml:"websocatImage" long:"agent-websocat-image" description:"Image to use for connection testing with websocat installed"`
	TracerouteImage string `yaml:"tracerouteImage" long:"agent-traceroute-image" description:"Image to use for connection testing with traceroute installed"`
}

type Tls struct {
	CertChainFile string `yaml:"certChainFile" long:"cert-chain-file" description:"Path to the certificate chain file"`
	CertKeyFile   string `yaml:"certKeyFile" long:"cert-key-file" description:"Path to the certificate key file"`
}

type KubernetesConfig struct {
	KubeConfigPath string `yaml:"kubeConfigPath" long:"kube-config" description:"Path to Kubernetes config"`
}

var (
	clientMutex  sync.Mutex
	cachedClient *kubernetes.Clientset
)

// Client returns the shared Kubernetes client. The client is created once and reused by all collectors - a
// client per call keeps a connection pool and a rate limiter of its own, which adds up to a significant amount
// of memory when the collectors run against a large cluster.
func (c KubernetesConfig) Client() (*kubernetes.Clientset, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if cachedClient != nil {
		return cachedClient, nil
	}

	clientset, err := c.newClient()
	if err != nil {
		return nil, err
	}

	cachedClient = clientset
	return cachedClient, nil
}

func (c KubernetesConfig) newClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Debug().Msgf("Steadybit-Debug is running inside a cluster, config found")
	} else if errors.Is(err, rest.ErrNotInCluster) {
		log.Debug().Msgf("Steadybit-Debug is not running inside a cluster, try local .kube config")
		var kubeconfig *string
		// use the current context in kubeconfig
		if home := homedir.HomeDir(); home != "" {
			config, err = clientcmd.BuildConfigFromFlags("", c.KubeConfigPath)
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
			flag.Parse()
			config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		}
	}

	if err != nil {
		log.Debug().Err(err).Msgf("Could not find kubernetes config")
		return nil, err
	}

	config.UserAgent = "steadybit-debug"
	config.Timeout = time.Second * 10
	// the shared client uses a single rate limiter for all collectors, the client-go default of 5 requests per
	// second is not enough to list the resources of a large cluster within the configured timeout
	config.QPS = 50
	config.Burst = 100
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Debug().Err(err).Msgf("Could not create kubernetes client")
		return nil, err
	}

	info, err := clientset.ServerVersion()
	if err != nil {
		log.Debug().Err(err).Msgf("Could not fetch server version.")
		return nil, err
	}

	log.Debug().Msgf("Cluster connected! Kubernetes Server Version %+v", info)

	return clientset, nil
}

func newConfig() Config {
	var kubeConfigPath string
	if home := homedir.HomeDir(); home != "" {
		kubeConfigPath = filepath.Join(home, ".kube", "config")
	}

	outputPath := os.TempDir()
	workingDirectory, err := os.Getwd()
	if err == nil {
		outputPath = workingDirectory
	}

	return Config{
		OutputPath:          outputPath,
		NoCleanup:           false,
		MaxConcurrency:      limit.DefaultMaxConcurrency,
		SkipConnectionTests: false,
		Kubernetes: KubernetesConfig{
			KubeConfigPath: kubeConfigPath,
		},
		Platform: PlatformConfig{
			Namespace:      "steadybit-platform",
			Deployment:     "steadybit-platform",
			ExportDatabase: false,
		},
		PlatformPortSplitter: PlatformportSplitterConfig{
			Namespace:  "steadybit-platform",
			Deployment: "platform-port-splitter",
		},
		Agent: AgentConfig{
			Namespace:       "steadybit-agent",
			StatefulSet:     "steadybit-agent",
			CurlImage:       "curlimages/curl",
			WebsocatImage:   "mtilson/websocat",
			TracerouteImage: "alpine",
		},
		Tls: Tls{
			CertChainFile: "",
			CertKeyFile:   "",
		},
	}
}

func loadConfig() Config {
	config := newConfig()

	path := "steadybit-debug.yml"
	fileContent, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Msgf("No steadybit-debug configuration file found at path '%s'. Will continue with default configuration.", path)
			return config
		} else {
			log.Err(err).Msgf("Failed to load steadybit-debug configuration file from path '%s'", path)
			os.Exit(1)
		}
	}

	err = yaml.Unmarshal(fileContent, &config)
	if err != nil {
		log.Err(err).Msgf("Failed to parse steadybit-debug configuration from path '%s' as YAML", path)
		os.Exit(1)
	}

	return config
}

func GetConfig() Config {
	config := loadConfig()

	_, err := flags.Parse(&config)
	if err != nil {
		os.Exit(1)
	}

	return config
}
