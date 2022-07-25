package agent

import (
	"github.com/steadybit/steadybit_debug/config"
	"github.com/steadybit/steadybit_debug/output"
)

func addAgentDaemonSetDescription(config *config.Config) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      config,
		CommandName: "kubectl",
		CommandArgs: []string{"describe", "daemonset", "-n", config.Agent.Namespace, config.Agent.DaemonSet},
		OutputPath:  []string{"agent", "k8s", "description.txt"},
	})
}
