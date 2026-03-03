package rendercheck

import (
	"encoding/json"
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

func TestRenderConvertedTemplateNormalizesNumericJSONTypes(t *testing.T) {
	root := t.TempDir()
	samplePath := filepath.Join(root, "sample.json")
	require.NoError(t, os.WriteFile(samplePath, []byte(`{"attachmentsCounter":1.0}`), 0o644))

	status, htmlOut, err := RenderConvertedTemplate("tpl", `{{if eq .attachmentsCounter 1}}ok{{else}}ko{{end}}`, samplePath)
	require.NoError(t, err)
	require.Equal(t, StatusRendered, status)
	require.Equal(t, "ok", htmlOut)
}

func TestNormalizeJSONNumbersNestedStructures(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"int_like_float": json.Number("42.0"),
		"nested": []any{
			json.Number("3"),
			map[string]any{
				"float": json.Number("3.25"),
			},
		},
		"bad_number": json.Number("not-a-number"),
	}

	got := normalizeJSONNumbers(input).(map[string]any)

	require.Equal(t, int64(42), got["int_like_float"])

	nestedSlice := got["nested"].([]any)
	require.Equal(t, int64(3), nestedSlice[0])

	nestedMap := nestedSlice[1].(map[string]any)
	require.Equal(t, 3.25, nestedMap["float"])

	require.Equal(t, json.Number("not-a-number"), got["bad_number"])
}
