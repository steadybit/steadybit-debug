package config

import (
	"os"
)

type Config struct {
	OutputPath string         `json:"outputPath"`
	Platform   PlatformConfig `json:"platform"`
	Agent      AgentConfig    `json:"agent"`
}

type PlatformConfig struct {
	Deployment string `json:"deployment"`
	Namespace  string `json:"namespace"`
}

type AgentConfig struct {
	DaemonSet string `json:"daemonSet"`
	Namespace string `json:"namespace"`
}

func NewConfig() Config {
	return Config{
		OutputPath: os.TempDir(),
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
