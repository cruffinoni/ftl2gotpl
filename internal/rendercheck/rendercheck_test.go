package rendercheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderConvertedTemplate(t *testing.T) {
	root := t.TempDir()
	samplePath := filepath.Join(root, "sample.json")
	require.NoError(t, os.WriteFile(samplePath, []byte(`{"name":"Ada"}`), 0o644))

	status, htmlOut, err := RenderConvertedTemplate("tpl", `Hello {{.name}}`, samplePath)
	require.NoError(t, err)
	require.Equal(t, StatusRendered, status)
	require.Equal(t, "Hello Ada", htmlOut)
}

func TestRenderConvertedTemplateMissingSample(t *testing.T) {
	status, htmlOut, err := RenderConvertedTemplate("tpl", `Hello {{.name}}`, filepath.Join(t.TempDir(), "missing.json"))
	require.NoError(t, err)
	require.Equal(t, StatusNoSample, status)
	require.Empty(t, htmlOut)
}
