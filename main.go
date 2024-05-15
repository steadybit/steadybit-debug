// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/debugrun"
	"github.com/steadybit/steadybit-debug/output"
	"io"
	"os"
	"path/filepath"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.GetConfig()
	output.AddOutputDirectory(&cfg)
	addLoggingToFile(&cfg)

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

func addLoggingToFile(cfg *config.Config) *os.File {
	file, err := os.OpenFile(
		filepath.Join(cfg.OutputPath, "log.txt"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)

	writers := []io.Writer{
		&zerolog.FilteredLevelWriter{
			Writer: zerolog.LevelWriterAdapter{zerolog.ConsoleWriter{Out: os.Stderr}},
			Level:  zerolog.InfoLevel,
		},
		&zerolog.FilteredLevelWriter{
			Writer: zerolog.LevelWriterAdapter{file},
			Level:  zerolog.DebugLevel,
		},
	}
	writer := zerolog.MultiLevelWriter(writers...)
	log.Logger = zerolog.New(writer).Level(zerolog.DebugLevel).With().Timestamp().Logger()

	if err != nil {
		panic(err)
	}
	return file
}
