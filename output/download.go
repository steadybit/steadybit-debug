/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package output

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

type DownloadOptions struct {
	Config     *config.Config
	Method     string
	URL        url.URL
	OutputPath string
}

func DownloadOutput(opts DownloadOptions) {
	start := time.Now()
	outputPathLog := opts.OutputPath + ".log"

	commandArgs := getCommandArgs(opts, false)

	logContent := fmt.Sprintf("# Executed command: %s %s", "curl", strings.Join(commandArgs, " "))
	logContent = fmt.Sprintf("%s\n# Started at: %s", logContent, time.Now().Format(time.RFC3339))

	out, err := doCurl(commandArgs)
	fmt.Println(string(out))
	if err != nil {
		logContent = fmt.Sprintf("%s\n# Resulted in error: %s", logContent, err)
	}
	if strings.Contains(string(out), "Client sent an HTTP request to an HTTPS server") {
		commandArgs := getCommandArgs(opts, true)
		out, err = doCurl(commandArgs)
		if err != nil {
			logContent = fmt.Sprintf("%s\n# Resulted in error: %s", logContent, err)
		}
	}
	totalTime := time.Now().Sub(start)
	logContent = fmt.Sprintf("%s\n\n# Total execution time: %d millis", logContent, totalTime.Milliseconds())

	WriteToFile(outputPathLog, []byte(strings.TrimSpace(logContent)))
}

func getCommandArgs(opts DownloadOptions, insecure bool) []string {
	commandArgs := []string{
		"-X", opts.Method,
		"-s", opts.URL.String(),
		"--output", opts.OutputPath,
	}
	if opts.URL.Scheme == "https" || insecure {
		commandArgs = append(commandArgs, "--insecure")
	}
	return commandArgs
}

func doCurl(commandArgs []string) ([]byte, error) {
	commandName := "curl"
	cmd := exec.Command(commandName, commandArgs...)
	log.Debug().Msgf("Executing: %s", cmd.String())
	out, err := cmd.CombinedOutput()
	return out, err
}
