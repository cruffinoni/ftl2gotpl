package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteJSONAndCSV(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "audit", "report.json")
	csvPath := filepath.Join(dir, "audit", "report.csv")

	files := []FileItem{
		{
			File:             "a.ftl",
			Status:           StatusConverted,
			FeaturesDetected: []string{"directive:if"},
			HelpersRequired:  []string{"trim"},
			RenderChecked:    true,
		},
		{
			File:          "b.ftl",
			Status:        StatusConversionError,
			Diagnostics:   []DiagnosticItem{{Code: "ERR", Message: "boom"}},
			RenderChecked: false,
		},
	}
	summary := Summary{
		Discovered:       2,
		Converted:        1,
		ConversionFailed: 1,
	}

	rep := NewJSONReport(summary, files)
	require.NoError(t, WriteJSON(jsonPath, rep))
	require.NoError(t, WriteCSV(csvPath, files))

	raw, err := os.ReadFile(jsonPath)
	require.NoError(t, err)
	var decoded JSONReport
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, 2, decoded.Summary.Discovered)

	_, err = os.Stat(csvPath)
	require.NoError(t, err)
}
