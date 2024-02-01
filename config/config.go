// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package config

import (
	"errors"
	"flag"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	OutputPath           string                     `yaml:"outputPath" short:"o" long:"output" description:"Path to output directory that will contain the debugging information"`
	NoCleanup            bool                       `yaml:"noCleanup" long:"no-cleanup" description:"Skip output directory deletion on command completion?"`
	Kubernetes           KubernetesConfig           `yaml:"kubernetes"`
	Platform             PlatformConfig             `yaml:"platform"`
	PlatformPortSplitter PlatformportSplitterConfig `yaml:"platform-port-splitter"`
	Outpost              OutpostConfig              `yaml:"outpost"`
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

type OutpostConfig struct {
	StatefulSet     string `yaml:"statefulSet" long:"outpost-stateful-set" description:"Kubernetes stateful set name of the Steadybit outpost"`
	Namespace       string `yaml:"namespace" long:"outpost-namespace" description:"Kubernetes namespace name of the Steadybit outpost"`
	CurlImage       string `yaml:"curlImage" long:"outpost-curl-image" description:"Image to use for connection testing with curl installed"`
	WebsocatImage   string `yaml:"websocatImage" long:"outpost-websocat-image" description:"Image to use for connection testing with websocat installed"`
	TracerouteImage string `yaml:"tracerouteImage" long:"outpost-traceroute-image" description:"Image to use for connection testing with traceroute installed"`
}

type Tls struct {
	CertChainFile string `yaml:"certChainFile" long:"cert-chain-file" description:"Path to the certificate chain file"`
	CertKeyFile   string `yaml:"certKeyFile" long:"cert-key-file" description:"Path to the certificate key file"`
}

type KubernetesConfig struct {
	KubeConfigPath string `yaml:"kubeConfigPath" long:"kube-config" description:"Path to Kubernetes config"`
}

func (c KubernetesConfig) Client() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Info().Msgf("Steadybit-Debug is running inside a cluster, config found")
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
		log.Error().Err(err).Msgf("Could not find kubernetes config")
		return nil, err
	}

	config.UserAgent = "steadybit-debug"
	config.Timeout = time.Second * 10
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error().Err(err).Msgf("Could not create kubernetes client")
		return nil, err
	}

	info, err := clientset.ServerVersion()
	if err != nil {
		log.Error().Err(err).Msgf("Could not fetch server version.")
		return nil, err
	}

	log.Info().Msgf("Cluster connected! Kubernetes Server Version %+v", info)

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
		OutputPath: outputPath,
		NoCleanup:  false,
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
		Outpost: OutpostConfig{
			Namespace:       "steadybit-outpost",
			StatefulSet:     "steadybit-outpost",
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
