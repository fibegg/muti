package runner

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fibegg/muti/internal/config"
	"github.com/fibegg/muti/internal/engine"
	igit "github.com/fibegg/muti/internal/git"
	"github.com/fibegg/muti/internal/mutation"
	"github.com/fibegg/muti/internal/output"
)

const (
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

// WorkerResult holds aggregated results from a single worker.
type WorkerResult struct {
	Killed   int
	Survived int
	Errors   int
}

// Run executes the mutation testing session.
func Run(cfg *config.Config, operators []mutation.Operator, probeCmd string) error {
	// Git preconditions
	repoRoot, err := igit.EnsureGitReady(cfg.Dirs)
	if err != nil {
		return err
	}

	// Resolve dirs to repo-relative paths (handles absolute paths correctly)
	relDirs := make([]string, len(cfg.Dirs))
	for i, d := range cfg.Dirs {
		absDir, _ := filepath.Abs(d)
		rel, _ := filepath.Rel(repoRoot, absDir)
		relDirs[i] = rel
	}

	// Create emitter
	emitter := output.NewEmitter(cfg.Tool, cfg.OutputDir, cfg.Verbose)

	return runInPlace(cfg, repoRoot, relDirs, operators, probeCmd, emitter)
}

// runInPlace mutates files directly in the repo, runs probe, then resets via git checkout.
func runInPlace(cfg *config.Config, repoRoot string, relDirs []string, operators []mutation.Operator, probeCmd string, emitter *output.Emitter) error {
	// Signal handling for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\n🧹 Interrupted — resetting modified files...")
		igit.ResetDirs(repoRoot, relDirs)
		os.Exit(1)
	}()
	defer signal.Stop(sigCh)

	fmt.Fprintf(os.Stderr, "🔧 Running in-place (single worker)\n\n")

	result := WorkerResult{}
	totalRounds := cfg.Rounds

	var maxDuration time.Duration

	for round := 0; cfg.Forever || round < totalRounds; round++ {
		roundStart := time.Now()
		count := cfg.Mutations
		if count == 0 {
			if cfg.MRU > 0 {
				count = rand.Intn(cfg.MRU) + 1
			} else {
				count = rand.Intn(5) + 1
			}
		}

		roundLabel := fmt.Sprintf("%d", round+1)
		if cfg.Forever {
			fmt.Fprintf(os.Stderr, "── Round %s (∞) (%d mutation%s) ──\n", roundLabel, count, plural(count))
		} else {
			fmt.Fprintf(os.Stderr, "── Round %s/%d (%d mutation%s) ──\n", roundLabel, totalRounds, count, plural(count))
		}

		// Mutate files directly
		dirs := make([]string, len(relDirs))
		for i, d := range relDirs {
			dirs[i] = filepath.Join(repoRoot, d)
		}

		eng := engine.NewEngine(dirs, cfg.Extensions, cfg.Lang, operators, cfg.Verbose)
		mutations, err := eng.Mutate(count)
		if err != nil {
			result.Errors++
			fmt.Fprintf(os.Stderr, "  ✗ Round error: %v\n", err)
			continue
		}

		if len(mutations) == 0 {
			fmt.Fprintf(os.Stderr, "  ⚠ No mutations could be applied, skipping\n")
			continue
		}

		for _, m := range mutations {
			fmt.Fprintf(os.Stderr, "  • [%s] %s:%d: %s\n", m.Operator, filepath.Base(m.File), m.Line, m.Description)
		}

		if cfg.TestMode {
			fmt.Fprintln(os.Stderr, "  🧪 Test mode — mutations applied, skipping probe")
			for _, m := range mutations {
				m.ProbeResult = "test_mode"
				_ = emitter.Emit(m)
			}
			// Don't reset in test mode
			continue
		}

		// Run probe from repo root
		fmt.Fprintf(os.Stderr, "  ▶ Running: %s\n", probeCmd)
		cmd := exec.Command("sh", "-c", probeCmd)
		cmd.Dir = repoRoot
		
		probeStart := time.Now()
		probeErr := cmd.Start()
		if probeErr == nil {
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			ticker := time.NewTicker(100 * time.Millisecond)
			
			spinner := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
			spinIdx := 0

		WaitLoop:
			for {
				select {
				case probeErr = <-done:
					ticker.Stop()
					break WaitLoop
				case <-ticker.C:
					elapsed := time.Since(probeStart)
					spinChar := string(spinner[spinIdx%len(spinner)])
					spinIdx++

					if maxDuration == 0 {
						fmt.Fprintf(os.Stderr, "\r\033[K  %s Probe running... (%s)", spinChar, formatDuration(elapsed))
					} else {
						pct := float64(elapsed) / float64(maxDuration)
						if pct > 1.0 {
							pct = 1.0
						}
						
						barLen := 20
						filled := int(pct * float64(barLen))
						empty := barLen - filled
						
						bar := ""
						for i := 0; i < filled; i++ {
							bar += "█"
						}
						for i := 0; i < empty; i++ {
							bar += "░"
						}
						
						fmt.Fprintf(os.Stderr, "\r\033[K  %s [%s] %.0f%% (%s / %s)", spinChar, bar, pct*100, formatDuration(elapsed), formatDuration(maxDuration))
					}
				}
			}
			fmt.Fprintf(os.Stderr, "\r\033[K")
		}

		roundDur := time.Since(probeStart)
		if roundDur > maxDuration {
			maxDuration = roundDur
		}

		exitCode := 0
		if probeErr != nil {
			var exitErr *exec.ExitError
			if errors.As(probeErr, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				result.Errors++
				fmt.Fprintf(os.Stderr, "  ✗ Probe failed to execute: %v\n", probeErr)
				igit.ResetDirs(repoRoot, relDirs)
				continue
			}
		}

		if exitCode != 0 {
			result.Killed++
			fmt.Fprintf(os.Stderr, "  %s✗ Tests FAILED — mutation killed ✓%s\n", colorGreen, colorReset)
		} else {
			result.Survived++
			fmt.Fprintf(os.Stderr, "  %s✓ Tests PASSED — mutation survived (poor coverage)%s\n", colorRed, colorReset)
			for _, m := range mutations {
				m.ProbeResult = "survived"
				m.ProbeExit = 0
				_ = emitter.Emit(m)
			}
		}

		// Reset files after each round
		igit.ResetDirs(repoRoot, relDirs)
		fmt.Fprintf(os.Stderr, "  ⏱ %s\n\n", formatDuration(time.Since(roundStart)))
	}

	// Print summary
	fmt.Fprintln(os.Stderr, "═══════════════════════════════════════════════")
	fmt.Fprintf(os.Stderr, " Results: %s%d killed%s │ %s%d survived%s │ %d errors\n", colorGreen, result.Killed, colorReset, colorRed, result.Survived, colorReset, result.Errors)
	fmt.Fprintln(os.Stderr, "═══════════════════════════════════════════════")

	return nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}
