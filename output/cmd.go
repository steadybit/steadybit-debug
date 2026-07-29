package output

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"github.com/steadybit/steadybit-debug/limit"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type AddCommandOutputOptions struct {
	Config                 *config.Config
	CommandName            string
	CommandArgs            []string
	OutputPath             string
	Executions             int
	DelayBetweenExecutions *time.Duration
	Stdin                  io.Reader
	ExecutionContext       string
	LogError               bool
	// Timeout limits a single execution, starting once it acquired an execution slot. Zero means no limit.
	Timeout time.Duration
}

// AddCommandOutput opts.OutputPath must include a %d to replace the execution number when opts.Executions > 1
func AddCommandOutput(ctx context.Context, opts AddCommandOutputOptions) {
	if opts.Executions < 1 {
		opts.Executions = 1
	}

	if opts.DelayBetweenExecutions == nil {
		delay := time.Second
		opts.DelayBetweenExecutions = &delay
	}

	// one slot for the whole series: acquiring per execution would let a busy run stretch the delay between two
	// samples to minutes, and the samples of `kubectl top` and the prometheus endpoint are only comparable when
	// they are taken at the requested interval
	release := limit.Commands.Acquire()
	defer release()

	for i := 0; i < opts.Executions; i++ {
		filePath := opts.OutputPath

		if opts.Executions > 1 {
			filePath = fmt.Sprintf(filePath, i)
		}

		addCommandOutputWithoutLoop(ctx, opts, filePath)

		if i < opts.Executions-1 {
			time.Sleep(*opts.DelayBetweenExecutions)
		}
	}
}

func addCommandOutputWithoutLoop(ctx context.Context, opts AddCommandOutputOptions, outputPath string) {
	command := fmt.Sprintf("%s %s", opts.CommandName, strings.Join(opts.CommandArgs, " "))

	addOutputFile(outputPath, command, func(out *os.File) error {
		if opts.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
			defer cancel()
		}

		cmd := exec.CommandContext(ctx, opts.CommandName, opts.CommandArgs...)
		log.Debug().Msgf("Executing: %s", cmd.String())

		cmd.Stdin = opts.Stdin
		// the same file for both streams makes os/exec pass a single descriptor to the child, which keeps the
		// output interleaved in the order it was written - as it was with cmd.CombinedOutput()
		cmd.Stdout = out
		cmd.Stderr = out

		err := cmd.Run()
		if err != nil {
			// a caller that gave up on the command knows why, which is more useful than the kill signal the command
			// reports for it
			if cause := context.Cause(ctx); cause != nil && !errors.Is(cause, context.Canceled) {
				err = cause
			}
			event := log.Debug()
			if opts.LogError {
				event = log.Error()
			}
			event.Str("context", opts.ExecutionContext).Str("cmd", cmd.String()).Msgf("Error executing command")
		}
		return err
	})
}
