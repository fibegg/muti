package output

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fibegg/muti/internal/mutation"
)

// Emitter sends mutation results to an external tool.
type Emitter struct {
	Tool      string
	OutputDir string
	Verbose   bool
	mu        sync.Mutex
	counter   atomic.Int64
}

// NewEmitter creates an emitter for the given tool command.
func NewEmitter(tool string, outputDir string, verbose bool) *Emitter {
	return &Emitter{
		Tool:      tool,
		OutputDir: outputDir,
		Verbose:   verbose,
	}
}

// Emit sends a mutation result to the configured tool.
// Thread-safe — can be called from multiple goroutines.
func (e *Emitter) Emit(result *mutation.MutationResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal mutation result: %w", err)
	}

	// Pipe JSON to the tool via stdin
	if e.Tool != "" {
		e.mu.Lock()
		err := e.pipeTo(data)
		e.mu.Unlock()
		if err != nil {
			return err
		}
	}

	// Also save to file if output dir is configured
	if e.OutputDir != "" {
		if err := e.saveToFile(data, result); err != nil {
			return err
		}
	}

	return nil
}

func (e *Emitter) pipeTo(data []byte) error {
	tool := strings.TrimSpace(e.Tool)
	if tool == "" {
		return nil
	}

	// Special case: if tool is "echo", just print to stdout
	if tool == "echo" {
		fmt.Println(string(data))
		return nil
	}

	// Execute via shell to support pipes, quotes, and complex commands
	cmd := exec.Command("sh", "-c", tool)
	cmd.Stdin = strings.NewReader(string(data))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if e.Verbose {
			fmt.Fprintf(os.Stderr, "  [WARN] tool %q failed: %v\n", e.Tool, err)
		}
		return nil // Don't fail the whole run if tool fails
	}
	return nil
}

func (e *Emitter) saveToFile(data []byte, result *mutation.MutationResult) error {
	if err := os.MkdirAll(e.OutputDir, 0o755); err != nil {
		return err
	}

	seq := e.counter.Add(1)
	filename := fmt.Sprintf("mutation-%04d-%s-%d-%d.json",
		seq,
		result.Operator,
		result.Line,
		result.Column,
	)
	path := filepath.Join(e.OutputDir, filename)
	return os.WriteFile(path, data, 0o644)
}
