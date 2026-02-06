package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultGlob       = "**/*.ftl"
	DefaultOutputExt  = ".gotmpl"
	DefaultSamplesDir = "testdata/samples"
)

// Config stores runtime options for one conversion run.
type Config struct {
	In          string
	Out         string
	Glob        string
	Ext         string
	SamplesRoot string

	ReportJSON string
	ReportCSV  string

	RenderCheck bool
	Strict      bool
}

// Default returns baseline configuration values used by CLI flags.
func Default() Config {
	return Config{
		Glob:        DefaultGlob,
		Ext:         DefaultOutputExt,
		SamplesRoot: DefaultSamplesDir,
		Strict:      false,
	}
}

// Validate normalizes and checks the configuration before execution.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.In) == "" {
		return fmt.Errorf("--in is required")
	}
	if strings.TrimSpace(c.Out) == "" {
		return fmt.Errorf("--out is required")
	}

	if strings.TrimSpace(c.Glob) == "" {
		c.Glob = DefaultGlob
	}
	if strings.TrimSpace(c.Ext) == "" {
		c.Ext = DefaultOutputExt
	}
	if !strings.HasPrefix(c.Ext, ".") {
		return fmt.Errorf("--ext must start with '.', got %q", c.Ext)
	}
	if strings.TrimSpace(c.SamplesRoot) == "" {
		c.SamplesRoot = DefaultSamplesDir
	}

	c.In = filepath.Clean(c.In)
	c.Out = filepath.Clean(c.Out)
	c.SamplesRoot = filepath.Clean(c.SamplesRoot)

	info, err := os.Stat(c.In)
	if err != nil {
		return fmt.Errorf("input path %q is not accessible: %w", c.In, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("input path %q must be a directory", c.In)
	}

	return nil
}
