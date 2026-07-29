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
	file, err := CreateFile(path)
	if err == nil {
		defer func() {
			_ = file.Close()
		}()
		_, err = file.Write(content)
	}
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to write '%s'", path)
	}
}

// CreateFile creates the file and all missing parent directories.
func CreateFile(path string) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return nil, err
	}
	return os.Create(path)
}

// addOutputFile writes one entry of the archive: the header naming the executed command, the payload streamed
// into the file by write, and finally the error write reported plus the total execution time.
//
// The payload is never buffered in memory - pod logs and discovery responses are easily large enough to exhaust
// the available memory, the more so because collectors run in parallel. Handing out the *os.File also lets
// os/exec pass the file descriptor to a child process instead of copying its output through a pipe.
//
// Acquiring limit.Commands is up to the caller: a series of repeated executions has to hold one slot for the
// whole series, otherwise the delay between the samples is no longer the delay that was asked for.
func addOutputFile(outputPath string, command string, write func(out *os.File) error) {
	start := time.Now()

	file, err := CreateFile(outputPath)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to create '%s'", outputPath)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	_, _ = fmt.Fprintf(file, "# Executed command: %s\n# Started at: %s\n\n", command, start.Format(time.RFC3339))

	if err := write(file); err != nil {
		_, _ = fmt.Fprintf(file, "\n# Resulted in error: %s", err)
	}

	_, _ = fmt.Fprintf(file, "\n\n# Total execution time: %d millis\n", time.Since(start).Milliseconds())
}

// AddFailureOutput records a collection step that could not be executed at all, so that the archive says why
// instead of leaving the file empty or absent.
func AddFailureOutput(outputPath string, command string, err error) {
	addOutputFile(outputPath, command, func(*os.File) error {
		return err
	})
}
