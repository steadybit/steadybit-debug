package output

import (
	"fmt"
	"github.com/steadybit/steadybit_debug/config"
	"os"
	"path"
	"path/filepath"
	"time"
)

func AddOutputDirectory(cfg *config.Config) {
	directoryName := fmt.Sprintf("steadybit_debug_%d", time.Now().Unix())
	cfg.OutputPath = path.Join(cfg.OutputPath, directoryName)
	_ = os.Mkdir(cfg.OutputPath, os.ModePerm)
}

func WriteToFile(path string, content []byte) {
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	os.WriteFile(path, content, 0666)
}
