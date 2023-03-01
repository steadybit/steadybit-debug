// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package config

import (
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

type Config struct {
	OutputPath string           `yaml:"outputPath" short:"o" long:"output" description:"Path to output directory that will contain the debugging information"`
	NoCleanup  bool             `yaml:"noCleanup" long:"no-cleanup" description:"Skip output directory deletion on command completion?"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Platform   PlatformConfig   `yaml:"platform"`
	Agent      AgentConfig      `yaml:"agent"`
	Tls        Tls              `yaml:"tls"`
}

type PlatformConfig struct {
	Deployment string `yaml:"deployment" long:"platform-deployment" description:"Kubernetes deployment name of the Steadybit platform"`
	Namespace  string `yaml:"namespace" long:"platform-namespace" description:"Kubernetes namespace name of the Steadybit platform"`
}

type AgentConfig struct {
	DaemonSet string `yaml:"daemonSet" long:"agent-daemon-set" description:"Kubernetes daemon set name of the Steadybit agent"`
	Namespace string `yaml:"namespace" long:"agent-namespace" description:"Kubernetes namespace name of the Steadybit agent"`
}

type Tls struct {
	CertChainFile string `yaml:"certChainFile" long:"cert-chain-file" description:"Path to the certificate chain file"`
	CertKeyFile   string `yaml:"certKeyFile" long:"cert-key-file" description:"Path to the certificate key file"`
}

type KubernetesConfig struct {
	KubeConfigPath string `yaml:"kubeConfigPath" long:"kube-config" description:"Path to Kubernetes config"`
}

func (c KubernetesConfig) Client() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", c.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
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
			Namespace:  "steadybit-platform",
			Deployment: "steadybit-platform",
		},
		Agent: AgentConfig{
			Namespace: "steadybit-agent",
			DaemonSet: "steadybit-agent",
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
