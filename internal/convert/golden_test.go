package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoldenFixtures(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("..", "..", "testdata", "fixtures", "*.ftl"))
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	converter := NewConverter()
	for _, inputPath := range fixtures {
		base := strings.TrimSuffix(inputPath, ".ftl")
		expectedPath := base + ".expected.gotmpl"

		inputRaw, err := os.ReadFile(inputPath)
		require.NoError(t, err)
		expectedRaw, err := os.ReadFile(expectedPath)
		require.NoError(t, err)

		got, err := converter.Convert(filepath.Base(inputPath), string(inputRaw))
		require.NoError(t, err)

		require.Equal(t, string(expectedRaw), got.Output)
	}
}
