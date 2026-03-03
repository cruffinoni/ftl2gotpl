package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inputDir := filepath.Join(root, "input")
	require.NoError(t, os.MkdirAll(inputDir, 0o755))

	inputFile := filepath.Join(root, "input.txt")
	require.NoError(t, os.WriteFile(inputFile, []byte("not a directory"), 0o644))

	tests := []struct {
		name        string
		cfg         Config
		wantErrText string
		assertValid func(t *testing.T, cfg Config)
	}{
		{
			name: "missing input directory",
			cfg: Config{
				In:  "   ",
				Out: filepath.Join(root, "out"),
			},
			wantErrText: "--in is required",
		},
		{
			name: "missing output directory",
			cfg: Config{
				In: inputDir,
			},
			wantErrText: "--out is required",
		},
		{
			name: "invalid extension",
			cfg: Config{
				In:  inputDir,
				Out: filepath.Join(root, "out"),
				Ext: "gotmpl",
			},
			wantErrText: "--ext must start with '.'",
		},
		{
			name: "input path must be a directory",
			cfg: Config{
				In:  inputFile,
				Out: filepath.Join(root, "out"),
			},
			wantErrText: "must be a directory",
		},
		{
			name: "defaults and path cleaning are applied",
			cfg: Config{
				In:          filepath.Join(inputDir, "."),
				Out:         filepath.Join(root, "out", "nested", ".."),
				Glob:        "   ",
				Ext:         " ",
				SamplesRoot: "\t",
			},
			assertValid: func(t *testing.T, cfg Config) {
				t.Helper()
				require.Equal(t, filepath.Clean(inputDir), cfg.In)
				require.Equal(t, filepath.Clean(filepath.Join(root, "out")), cfg.Out)
				require.Equal(t, DefaultGlob, cfg.Glob)
				require.Equal(t, DefaultOutputExt, cfg.Ext)
				require.Equal(t, filepath.Clean(DefaultSamplesDir), cfg.SamplesRoot)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := test.cfg
			err := cfg.Validate()

			if test.wantErrText != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErrText)
				return
			}

			require.NoError(t, err)
			if test.assertValid != nil {
				test.assertValid(t, cfg)
			}
		})
	}
}
