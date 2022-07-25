package platform

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit_debug/config"
	"github.com/steadybit/steadybit_debug/k8s"
	"path/filepath"
)

func AddPlatformDebuggingInformation(cfg *config.Config) {
	deployment, err := k8s.FindDeployment(cfg, cfg.Platform.Namespace, cfg.Platform.Deployment)
	if err != nil {
		log.Info().Msgf("Failed to find deployment '%s' in '%s': %s", cfg.Platform.Deployment, cfg.Platform.Namespace, err)
	} else {
		k8s.AddKubernetesDeploymentOutput(cfg, filepath.Join(cfg.OutputPath, "platform"), deployment)
	}
}
