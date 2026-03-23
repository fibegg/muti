package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration parsed from CLI flags.
type Config struct {
	Dirs          []string
	Extensions    []string
	Lang          string
	Rounds        int
	Mutations     int // 0 means random 1-5
	MRU           int // mutations-random-upto: random 1..N mutations per round
	Operator      string
	SkipOperators []string
	TestMode      bool
	Forever       bool
	Tool          string
	OutputDir     string
	Verbose       bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Dirs:   []string{"."},
		Rounds: 10,
		Tool:   "echo",
	}
}

// ApplyEnv reads MUTI_* environment variables and applies them as defaults.
// CLI flags take precedence — call this BEFORE flag parsing sets values.
func (c *Config) ApplyEnv() {
	if v := os.Getenv("MUTI_DIRS"); v != "" {
		c.Dirs = strings.Split(v, ",")
	}
	if v := os.Getenv("MUTI_EXTENSIONS"); v != "" {
		c.Extensions = strings.Split(v, ",")
	}
	if v := os.Getenv("MUTI_LANG"); v != "" {
		c.Lang = v
	}
	if v := os.Getenv("MUTI_ROUNDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Rounds = n
		}
	}
	if v := os.Getenv("MUTI_MUTATIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Mutations = n
		}
	}
	if v := os.Getenv("MUTI_MRU"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MRU = n
		}
	}
	if v := os.Getenv("MUTI_OPERATOR"); v != "" {
		c.Operator = v
	}
	if v := os.Getenv("MUTI_SKIP_OPERATORS"); v != "" {
		c.SkipOperators = strings.Split(v, ",")
	}
	if v := os.Getenv("MUTI_TOOL"); v != "" {
		c.Tool = v
	}
	if v := os.Getenv("MUTI_OUTPUT_DIR"); v != "" {
		c.OutputDir = v
	}
	if v := os.Getenv("MUTI_FOREVER"); v == "1" || v == "true" {
		c.Forever = true
	}
	if v := os.Getenv("MUTI_TEST"); v == "1" || v == "true" {
		c.TestMode = true
	}
	if v := os.Getenv("MUTI_VERBOSE"); v == "1" || v == "true" {
		c.Verbose = true
	}
}
