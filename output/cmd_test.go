package output

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/steadybit/steadybit-debug/config"
)

const largeOutputSize = 32 * 1024 * 1024

func TestAddCommandOutputWritesHeaderOutputAndFooter(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "nested", "out.txt")

	AddCommandOutput(context.Background(), AddCommandOutputOptions{
		Config:      &config.Config{},
		CommandName: "echo",
		CommandArgs: []string{"hello"},
		OutputPath:  outputPath,
	})

	content := readFile(t, outputPath)
	for _, expected := range []string{"# Executed command: echo hello", "# Started at: ", "hello", "# Total execution time: "} {
		if !strings.Contains(content, expected) {
			t.Errorf("expected output to contain %q, got:\n%s", expected, content)
		}
	}
}

func TestAddCommandOutputReportsFailures(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "out.txt")

	AddCommandOutput(context.Background(), AddCommandOutputOptions{
		Config:      &config.Config{},
		CommandName: "sh",
		CommandArgs: []string{"-c", "echo boom >&2; exit 3"},
		OutputPath:  outputPath,
	})

	content := readFile(t, outputPath)
	if !strings.Contains(content, "boom") {
		t.Errorf("expected stderr to be captured, got:\n%s", content)
	}
	if !strings.Contains(content, "# Resulted in error: ") {
		t.Errorf("expected the exit code to be reported, got:\n%s", content)
	}
}

func TestAddCommandOutputWritesOneFilePerExecution(t *testing.T) {
	directory := t.TempDir()

	AddCommandOutput(context.Background(), AddCommandOutputOptions{
		Config:      &config.Config{},
		CommandName: "echo",
		CommandArgs: []string{"hello"},
		OutputPath:  filepath.Join(directory, "out.%d.txt"),
		Executions:  2,
	})

	for _, name := range []string{"out.0.txt", "out.1.txt"} {
		if _, err := os.Stat(filepath.Join(directory, name)); err != nil {
			t.Errorf("expected '%s' to exist: %s", name, err)
		}
	}
}

// TestAddCommandOutputDoesNotBufferLargeOutput guards the fix for the tool exhausting the machines memory: the
// output of a command like `kubectl logs` must end up in the file without being held in memory.
func TestAddCommandOutputDoesNotBufferLargeOutput(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "out.txt")

	allocatedBefore := totalAllocated()
	AddCommandOutput(context.Background(), AddCommandOutputOptions{
		Config:      &config.Config{},
		CommandName: "head",
		CommandArgs: []string{"-c", strconv.Itoa(largeOutputSize), "/dev/zero"},
		OutputPath:  outputPath,
	})
	allocated := totalAllocated() - allocatedBefore

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("expected output file: %s", err)
	}
	if info.Size() < largeOutputSize {
		t.Errorf("expected at least %d bytes of output, got %d", largeOutputSize, info.Size())
	}
	if allocated > largeOutputSize/4 {
		t.Errorf("expected the output to be streamed, but %d bytes were allocated for %d bytes of output", allocated, largeOutputSize)
	}
}

func totalAllocated() uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.TotalAlloc
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read '%s': %s", path, err)
	}
	return string(content)
}

func TestAddCommandOutputRecordsNotes(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "out.txt")

	AddCommandOutput(context.Background(), AddCommandOutputOptions{
		Config:      &config.Config{},
		CommandName: "echo",
		CommandArgs: []string{"hello"},
		OutputPath:  outputPath,
		Notes:       []string{"Connection requires authentication: true"},
	})

	// the note belongs in the header block, which is separated from the payload by an empty line
	header, payload, found := strings.Cut(readFile(t, outputPath), "\n\n")
	if !found {
		t.Fatalf("expected a header followed by the output, got:\n%s", header)
	}
	if !strings.Contains(header, "# Connection requires authentication: true") {
		t.Errorf("expected the note in the header, got:\n%s", header)
	}
	if !strings.Contains(payload, "hello") {
		t.Errorf("expected the command output below the header, got:\n%s", payload)
	}
}
