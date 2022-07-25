package platform

import (
	"github.com/steadybit/steadybit_debug/config"
	"github.com/steadybit/steadybit_debug/output"
)

func addPlatformDeploymentDescription(config *config.Config) {
	output.AddCommandOutput(output.AddCommandOutputOptions{
		Config:      config,
		CommandName: "kubectl",
		CommandArgs: []string{"describe", "deployment", "-n", config.Platform.Namespace, config.Platform.Deployment},
		OutputPath:  []string{"platform", "k8s", "description.txt"},
	})
}
