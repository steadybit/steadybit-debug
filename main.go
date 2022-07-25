package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit_debug/agent"
	"github.com/steadybit/steadybit_debug/config"
	"github.com/steadybit/steadybit_debug/output"
	"github.com/steadybit/steadybit_debug/platform"
	"os"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// TODO parse config
	cfg := config.NewConfig()
	cfg.Platform.Namespace = "platform"
	cfg.Agent.Namespace = "steadybit-agent-to-prod"
	output.AddOutputDirectory(&cfg)

	log.Info().Msgf("Debugging output will be written to %s", cfg.OutputPath)

	output.AddJsonOutput(output.AddJsonOutputOptions{
		Config:     &cfg,
		Content:    cfg,
		OutputPath: []string{"debugging_config.yaml"},
	})
	platform.AddPlatformDebuggingInformation(&cfg)
	agent.AddAgentDebuggingInformation(&cfg)
}
