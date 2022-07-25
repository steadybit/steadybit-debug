package output

import (
	"fmt"
	"github.com/steadybit/steadybit_debug/config"
	"os/exec"
	"strings"
)

type AddCommandOutputOptions struct {
	Config      *config.Config
	CommandName string
	CommandArgs []string
	OutputPath  string
}

func AddCommandOutput(opts AddCommandOutputOptions) {
	cmd := exec.Command(opts.CommandName, opts.CommandArgs...)
	out, err := cmd.CombinedOutput()

	content := fmt.Sprintf("# Executed command: %s %s", opts.CommandName, strings.Join(opts.CommandArgs, " "))
	if err != nil {
		content = fmt.Sprintf("%s\n# Resulted in error: %s", content, err)
	}
	content = fmt.Sprintf("%s\n\n%s", content, out)

	WriteToFile(opts.OutputPath, []byte(content))
}
