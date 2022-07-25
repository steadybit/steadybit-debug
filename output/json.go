package output

import (
	"encoding/json"
	"fmt"
	"github.com/steadybit/steadybit_debug/config"
	"path/filepath"
)

type AddJsonOutputOptions struct {
	Config     *config.Config
	Content    any
	OutputPath []string
}

func AddJsonOutput(opts AddJsonOutputOptions) {
	json, err := json.MarshalIndent(opts.Content, "", "\t")

	content := ""
	if err != nil {
		content = fmt.Sprintf("# Failed to JSON serialize: %s", err)
	}
	content = fmt.Sprintf("%s\n\n%s", content, json)

	outputFilePath := filepath.Join(opts.Config.OutputPath, filepath.Join(opts.OutputPath...))
	WriteToFile(outputFilePath, []byte(content))
}
