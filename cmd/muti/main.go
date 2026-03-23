package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fibegg/muti/internal/config"
	"github.com/fibegg/muti/internal/language"
	"github.com/fibegg/muti/internal/mutation"
	"github.com/fibegg/muti/internal/runner"
)

var version = "dev"

func main() {
	cfg := config.DefaultConfig()
	cfg.ApplyEnv() // ENV defaults first, CLI flags override below
	var probeArgs []string

	rootCmd := &cobra.Command{
		Use:   "muti [flags] [-- PROBE_COMMAND...]",
		Short: "Language-agnostic mutation testing engine",
		Long: `muti - Language-agnostic mutation testing engine powered by tree-sitter.

Mutates source code in target directories and runs a probe command (test suite)
to determine if mutations are detected (killed) or survive (indicating poor coverage).

Mutations are piped as JSON to the configured --tool (default: echo to stdout).

All flags can also be set via environment variables with MUTI_ prefix:
  MUTI_DIRS, MUTI_EXTENSIONS, MUTI_LANG, MUTI_ROUNDS, MUTI_MUTATIONS,
  MUTI_MRU, MUTI_OPERATOR, MUTI_SKIP_OPERATORS,
  MUTI_TOOL, MUTI_OUTPUT_DIR, MUTI_FOREVER, MUTI_TEST, MUTI_VERBOSE
CLI flags take precedence over environment variables.`,
		Example: `  # Run forever with random 1-5 mutations per round
  muti --forever --mru 5 --dirs app -- bundle exec rspec --fail-fast

  # Mutate current dir, echo mutations to terminal
  muti -- ./run_tests.sh

  # Test mode: apply mutations, see what changed, no cleanup
  muti --test --dirs src --rounds 1

  # Use ENV variables
  MUTI_DIRS=app,lib MUTI_FOREVER=1 muti -- make test`,
		DisableFlagsInUseLine: true,
		SilenceUsage:          true,
		SilenceErrors:         true,
		Args:                  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Everything after "--" is the probe command
			probeCmd := strings.Join(probeArgs, " ")
			if probeCmd == "" && !cfg.TestMode {
				return fmt.Errorf("no probe command specified. Use: muti [flags] -- COMMAND\nOr use --test mode to apply mutations without running tests")
			}

			// MRU sets Mutations to random value each round (handled in runner)
			if cfg.MRU > 0 && cfg.Mutations > 0 {
				return fmt.Errorf("--mutations and --mru are incompatible (use one or the other)")
			}

			// Filter operators
			operators, err := mutation.FilterOperators(cfg.Operator, cfg.SkipOperators)
			if err != nil {
				return err
			}

			// Print banner
			printBanner(cfg, operators, probeCmd)

			return runner.Run(cfg, operators, probeCmd)
		},
	}

	// Parse -- separator manually
	rootCmd.TraverseChildren = true
	rawArgs := os.Args[1:]
	var flagArgs []string
	for i, arg := range rawArgs {
		if arg == "--" {
			probeArgs = rawArgs[i+1:]
			flagArgs = rawArgs[:i]
			break
		}
	}
	if flagArgs == nil {
		flagArgs = rawArgs
	}

	// Flags (defaults come from cfg which already has ENV applied)
	f := rootCmd.Flags()
	f.StringSliceVarP(&cfg.Dirs, "dirs", "d", cfg.Dirs, "Directories to mutate (env: MUTI_DIRS)")
	f.StringSliceVarP(&cfg.Extensions, "extensions", "e", cfg.Extensions, "File extensions (env: MUTI_EXTENSIONS)")
	f.StringVarP(&cfg.Lang, "lang", "l", cfg.Lang, "Force language (env: MUTI_LANG)")
	f.IntVarP(&cfg.Rounds, "rounds", "r", cfg.Rounds, "Mutation rounds (env: MUTI_ROUNDS)")
	f.IntVarP(&cfg.Mutations, "mutations", "m", cfg.Mutations, "Fixed mutations/round (env: MUTI_MUTATIONS)")
	f.IntVar(&cfg.MRU, "mutations-random-upto", cfg.MRU, "Random 1..N mutations/round (env: MUTI_MRU)")
	f.StringVarP(&cfg.Operator, "operator", "o", cfg.Operator, "Use only this operator (env: MUTI_OPERATOR)")
	f.StringSliceVarP(&cfg.SkipOperators, "skip-operators", "s", nil, "Operators to skip (env: MUTI_SKIP_OPERATORS)")
	f.BoolVar(&cfg.TestMode, "test", cfg.TestMode, "Test mode (env: MUTI_TEST)")
	f.BoolVar(&cfg.Forever, "forever", cfg.Forever, "Run forever until interrupted (env: MUTI_FOREVER)")
	f.StringVarP(&cfg.Tool, "tool", "t", cfg.Tool, "Pipe mutations to tool (env: MUTI_TOOL)")
	f.StringVar(&cfg.OutputDir, "output-dir", cfg.OutputDir, "Save mutations to dir (env: MUTI_OUTPUT_DIR)")
	f.BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Debug output (env: MUTI_VERBOSE)")

	// --mru shorthand: manually handle in PreRun by scanning rawArgs
	// This avoids pflag complexity — if user passes --mru N, we parse it ourselves
	for i, arg := range flagArgs {
		if arg == "--mru" && i+1 < len(flagArgs) {
			if n, err := fmt.Sscanf(flagArgs[i+1], "%d", &cfg.MRU); n == 1 && err == nil {
				// Remove --mru N from flagArgs so cobra doesn't complain
				flagArgs = append(flagArgs[:i], flagArgs[i+2:]...)
			}
			break
		}
		if strings.HasPrefix(arg, "--mru=") {
			if n, err := fmt.Sscanf(arg[6:], "%d", &cfg.MRU); n == 1 && err == nil {
				flagArgs = append(flagArgs[:i], flagArgs[i+1:]...)
			}
			break
		}
	}

	// PreRun validation
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cfg.Forever && cmd.Flags().Changed("rounds") {
			return fmt.Errorf("--forever and --rounds are incompatible")
		}
		if cfg.MRU > 0 && cfg.Mutations > 0 {
			return fmt.Errorf("--mutations and --mru are incompatible")
		}
		if cfg.Rounds < 1 {
			cfg.Rounds = 1
		}
		return nil
	}

	// Subcommands for listing
	rootCmd.AddCommand(&cobra.Command{
		Use:   "list-operators",
		Short: "List all available mutation operators",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available mutation operators:")
			for _, name := range mutation.AllNames() {
				fmt.Printf("  %s\n", name)
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "list-languages",
		Short: "List all supported languages",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Supported languages:")
			langs := language.ListAll()
			names := make([]string, 0, len(langs))
			for name := range langs {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				cfg := langs[name]
				fmt.Printf("  %-12s  extensions: %s\n", name, strings.Join(cfg.Extensions, ", "))
			}
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("muti %s\n", version)
		},
	})

	rootCmd.SetArgs(flagArgs)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printBanner(cfg *config.Config, operators []mutation.Operator, probeCmd string) {
	total := len(mutation.All())
	active := len(operators)

	fmt.Fprintln(os.Stderr, "╔══════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stderr, "║         🧬 MUTI — MUTATION TESTING 🧬       ║")
	fmt.Fprintln(os.Stderr, "╠══════════════════════════════════════════════╣")
	if cfg.Forever {
		fmt.Fprintf(os.Stderr, "║  Rounds: %-35s║\n", "∞ (forever)")
	} else {
		fmt.Fprintf(os.Stderr, "║  Rounds: %-35d║\n", cfg.Rounds)
	}
	mpr := "random 1-5"
	if cfg.MRU > 0 {
		mpr = fmt.Sprintf("random 1-%d", cfg.MRU)
	} else if cfg.Mutations > 0 {
		mpr = fmt.Sprintf("%d", cfg.Mutations)
	}
	fmt.Fprintf(os.Stderr, "║  Mutations/round: %-26s║\n", mpr)
	fmt.Fprintf(os.Stderr, "║  Operators: %-32s║\n", fmt.Sprintf("%d/%d", active, total))
	mode := "full"
	if cfg.TestMode {
		mode = "TEST (no probe, no reset)"
	}
	fmt.Fprintf(os.Stderr, "║  Mode: %-37s║\n", mode)
	probeDisplay := probeCmd
	if len(probeDisplay) > 42 {
		probeDisplay = probeDisplay[:39] + "..."
	}
	if probeDisplay == "" {
		probeDisplay = "(none — test mode)"
	}
	fmt.Fprintf(os.Stderr, "║  Probe: %-36s║\n", probeDisplay)
	dirsDisplay := strings.Join(cfg.Dirs, ", ")
	if len(dirsDisplay) > 36 {
		dirsDisplay = dirsDisplay[:33] + "..."
	}
	fmt.Fprintf(os.Stderr, "║  Target: %-35s║\n", dirsDisplay)
	toolDisplay := cfg.Tool
	if len(toolDisplay) > 37 {
		toolDisplay = toolDisplay[:34] + "..."
	}
	fmt.Fprintf(os.Stderr, "║  Tool: %-37s║\n", toolDisplay)
	fmt.Fprintln(os.Stderr, "╚══════════════════════════════════════════════╝")
	fmt.Fprintln(os.Stderr)
}
