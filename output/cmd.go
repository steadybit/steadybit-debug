package output

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"io"
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
}

// AddCommandOutput opts.OutputPath must include a %d to replace the execution number when opts.Executions > 1
func AddCommandOutput(opts AddCommandOutputOptions) {
	if opts.Executions < 1 {
		opts.Executions = 1
	}

	if opts.DelayBetweenExecutions == nil {
		delay := time.Second
		opts.DelayBetweenExecutions = &delay
	}

	for i := 0; i < opts.Executions; i++ {
		filePath := opts.OutputPath

		if opts.Executions > 1 {
			filePath = fmt.Sprintf(filePath, i)
		}

		addCommandOutputWithoutLoop(opts, filePath)

		time.Sleep(*opts.DelayBetweenExecutions)
	}
}

func addCommandOutputWithoutLoop(opts AddCommandOutputOptions, outputPath string) {
	start := time.Now()

	content := fmt.Sprintf("# Executed command: %s %s", opts.CommandName, strings.Join(opts.CommandArgs, " "))
	content = fmt.Sprintf("%s\n# Started at: %s", content, time.Now().Format(time.RFC3339))

	cmd := exec.Command(opts.CommandName, opts.CommandArgs...)
	log.Debug().Msgf("Executing: %s", cmd.String())
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
	}
	content = fmt.Sprintf("%s\n\n%s", content, out)

	totalTime := time.Now().Sub(start)
	content = fmt.Sprintf("%s\n\n# Total execution time: %d millis", content, totalTime.Milliseconds())

	WriteToFile(outputPath, []byte(strings.TrimSpace(content)))
}
