package output

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/steadybit-debug/config"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

func AddOutputDirectory(cfg *config.Config) {
	directoryName := fmt.Sprintf("steadybit-debug-%d", time.Now().Unix())
	cfg.OutputPath = path.Join(cfg.OutputPath, directoryName)
	err := os.Mkdir(cfg.OutputPath, os.ModePerm)
	if err != nil {
		log.Error().Msgf("Failed create target directory '%s' for debugging information: %s", cfg.OutputPath, err)
		os.Exit(1)
	}

	hint := ""
	if !cfg.NoCleanup {
		hint = " (directory will be deleted on command completion)"
	}
	log.Info().Msgf("Debugging output will be collected at %s%s", cfg.OutputPath, hint)
}

func ZipOutputDirectory(cfg *config.Config) string {
	targetPath := fmt.Sprintf("%s.tar.gz", cfg.OutputPath)
	cwd := filepath.Join(cfg.OutputPath, "..")
	// Use relative paths for the last argument to `tar` so that the paths within tar are nice and short
	relativeOutputPath, _ := filepath.Rel(cwd, cfg.OutputPath)
	cmd := exec.Command("tar", "-czf", targetPath, relativeOutputPath)
	cmd.Dir = cwd
	err := cmd.Run()
	if err != nil {
		log.Error().Msgf("Failed turn target directory '%s' into tar archive at '%s'. Got error: %s", cfg.OutputPath, targetPath, err)
		os.Exit(1)
	}
	log.Info().Msgf("Debugging output collected at: %s", targetPath)
	return targetPath
}

func WriteToFile(path string, content []byte) {
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	os.WriteFile(path, content, 0666)
}
