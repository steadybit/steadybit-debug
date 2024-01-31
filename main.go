// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/debugrun"
	"github.com/steadybit/steadybit-debug/output"
	"os"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.GetConfig()
	output.AddOutputDirectory(&cfg)

	output.AddJsonOutput(output.AddJsonOutputOptions{
		Config:     &cfg,
		Content:    cfg,
		OutputPath: []string{"debugging_config.yaml"},
	})
	debugrun.GatherInformation(&cfg)
	output.ZipOutputDirectory(&cfg)

	if !cfg.NoCleanup {
		err := os.RemoveAll(cfg.OutputPath)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to remove output directory '%s' after completion", cfg.OutputPath)
		}
	}
}
