package platform

import "github.com/steadybit/steadybit_debug/config"

func AddPlatformDebuggingInformation(cfg *config.Config) {
	addPlatformDeploymentDescription(cfg)
	addPlatformDeploymentConfig(cfg)
}
