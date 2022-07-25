package config

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

type Config struct {
	OutputPath string           `json:"outputPath"`
	Kubernetes KubernetesConfig `json:"kubernetes"`
	Platform   PlatformConfig   `json:"platform"`
	Agent      AgentConfig      `json:"agent"`
}

type PlatformConfig struct {
	Deployment string `json:"deployment"`
	Namespace  string `json:"namespace"`
}

type AgentConfig struct {
	DaemonSet string `json:"daemonSet"`
	Namespace string `json:"namespace"`
}

type KubernetesConfig struct {
	KubeConfigPath string `json:"kubeConfigPath"`
}

func (c KubernetesConfig) Client() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", c.KubeConfigPath)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func NewConfig() Config {
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
			Deployment: "platform",
		},
		Agent: AgentConfig{
			Namespace: "steadybit-agent",
			DaemonSet: "steadybit-agent",
		},
	}
}
