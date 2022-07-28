package output

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
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
}

func AddCommandOutput(opts AddCommandOutputOptions) {
	start := time.Now()

	if opts.Executions < 1 {
		opts.Executions = 1
	}

	if opts.DelayBetweenExecutions == nil {
		delay := time.Second
		opts.DelayBetweenExecutions = &delay
	}

	content := ""

	for i := 0; i < opts.Executions; i++ {
		log.Debug().Msgf("Executing: %s %s", opts.CommandName, strings.Join(opts.CommandArgs, " "))
		content = fmt.Sprintf("%s\n\n\n# Executed command (execution %d): %s %s", content, i+1, opts.CommandName, strings.Join(opts.CommandArgs, " "))

		cmd := exec.Command(opts.CommandName, opts.CommandArgs...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
		}
		content = fmt.Sprintf("%s\n\n%s", content, out)

		time.Sleep(*opts.DelayBetweenExecutions)
	}

	totalTime := time.Now().Sub(start)
	content = fmt.Sprintf("%s\n\n# Total execution time: %d millis", content, totalTime.Milliseconds())

	WriteToFile(opts.OutputPath, []byte(strings.TrimSpace(content)))
}
