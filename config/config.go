package config

import (
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

type Config struct {
	OutputPath string           `yaml:"outputPath"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Platform   PlatformConfig   `yaml:"platform"`
	Agent      AgentConfig      `yaml:"agent"`
}

type PlatformConfig struct {
	Deployment string `yaml:"deployment"`
	Namespace  string `yaml:"namespace"`
}

type AgentConfig struct {
	DaemonSet string `yaml:"daemonSet"`
	Namespace string `yaml:"namespace"`
}

type KubernetesConfig struct {
	KubeConfigPath string `yaml:"kubeConfigPath"`
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

	return Config{
		OutputPath: os.TempDir(),
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
	}
}

func LoadConfig() Config {
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
