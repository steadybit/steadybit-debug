package output

import (
	"fmt"
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
	if opts.Executions < 1 {
		opts.Executions = 1
	}

	if opts.DelayBetweenExecutions == nil {
		delay := time.Second
		opts.DelayBetweenExecutions = &delay
	}

	content := ""

	for i := 0; i < opts.Executions; i++ {
		content = fmt.Sprintf("%s\n\n\n# Executed command (execution %d): %s %s", content, i+1, opts.CommandName, strings.Join(opts.CommandArgs, " "))

		cmd := exec.Command(opts.CommandName, opts.CommandArgs...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
		}
		content = fmt.Sprintf("%s\n\n%s", content, out)

		time.Sleep(*opts.DelayBetweenExecutions)
	}

	WriteToFile(opts.OutputPath, []byte(strings.TrimSpace(content)))
}
