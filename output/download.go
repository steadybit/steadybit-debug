/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package output

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/limit"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type DownloadOptions struct {
	Config     *config.Config
	Method     string
	URL        url.URL
	OutputPath string
}

func DownloadOutput(opts DownloadOptions) {
	release := limit.Commands.Acquire()
	defer release()

	commandArgs := getCommandArgs(opts, false)

	addOutputFile(opts.OutputPath+".log", "curl "+strings.Join(commandArgs, " "), func(out *os.File) error {
		// curl writes the payload itself, only its diagnostics end up in the log
		result, err := doCurl(commandArgs)
		if bytes.Contains(result, []byte(httpsRequiredIndicator)) {
			// keep what the first attempt reported, it may have left a partial file behind
			_, _ = out.Write(result)
			if err != nil {
				_, _ = fmt.Fprintf(out, "\n# First attempt resulted in error: %s\n", err)
			}
			result, err = doCurl(getCommandArgs(opts, true))
		}
		_, _ = out.Write(result)
		return err
	})
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
