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

type AddCurlOutputOptions struct {
	Config     *config.Config
	Method     string
	URL        url.URL
	OutputPath string
}

func AddCurlOutput(opts AddCurlOutputOptions) {
	start := time.Now()
	outputPath := opts.OutputPath

	commandArgs := getCommandArgs(opts, false)

	content := fmt.Sprintf("# Executed command: %s %s", "curl", strings.Join(commandArgs, " "))
	content = fmt.Sprintf("%s\n# Started at: %s", content, time.Now().Format(time.RFC3339))

	out, err := doCurl(commandArgs)
	if err != nil {
		content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
	}
	if strings.Contains(string(out), "Client sent an HTTP request to an HTTPS server") {
		commandArgs := getCommandArgs(opts, true)
		out, err = doCurl(commandArgs)
		if err != nil {
			content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
		}
	}
	content = fmt.Sprintf("%s\n\n%s", content, out)

	totalTime := time.Now().Sub(start)
	content = fmt.Sprintf("%s\n\n# Total execution time: %d millis", content, totalTime.Milliseconds())

	WriteToFile(outputPath, []byte(strings.TrimSpace(content)))
}

func getCommandArgs(opts AddCurlOutputOptions, insecure bool) []string {
	commandArgs := []string{
		"-X", opts.Method,
		"-s", opts.URL.String(),
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
