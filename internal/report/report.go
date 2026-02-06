package report

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/cruffinoni/ftl2gotpl/internal/diagnostics"
)

// FileStatus is the per-template processing status used in reports.
type FileStatus string

const (
	StatusConverted       FileStatus = "converted"
	StatusConvertedNoData FileStatus = "converted_no_sample"
	StatusConversionError FileStatus = "failed_conversion"
	StatusParseError      FileStatus = "failed_parse"
	StatusRenderError     FileStatus = "failed_render"
)

// DiagnosticItem is the report-friendly representation of one error/diagnostic.
type DiagnosticItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// FileItem describes conversion and validation for one template file.
type FileItem struct {
	File             string           `json:"file"`
	Status           FileStatus       `json:"status"`
	Diagnostics      []DiagnosticItem `json:"diagnostics,omitempty"`
	FeaturesDetected []string         `json:"features_detected,omitempty"`
	HelpersRequired  []string         `json:"helpers_required,omitempty"`
	RenderChecked    bool             `json:"render_checked"`
	SamplePath       string           `json:"sample_path,omitempty"`
}

// Summary contains aggregate counters for a conversion run.
type Summary struct {
	Discovered       int      `json:"discovered"`
	Converted        int      `json:"converted"`
	ConversionFailed int      `json:"conversion_failed"`
	ParseFailed      int      `json:"parse_failed"`
	RenderFailed     int      `json:"render_failed"`
	NoSample         int      `json:"no_sample"`
	HelpersNeeded    []string `json:"helpers_needed,omitempty"`
}

// JSONReport is the structured report persisted by --report-json.
type JSONReport struct {
	GeneratedAt string     `json:"generated_at"`
	Summary     Summary    `json:"summary"`
	Files       []FileItem `json:"files"`
}

// NewJSONReport builds a report payload with RFC3339 generation timestamp.
func NewJSONReport(summary Summary, files []FileItem) JSONReport {
	return JSONReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:     summary,
		Files:       files,
	}
}

// ToDiagnosticItem converts an error to a typed report diagnostic.
func ToDiagnosticItem(file string, err error) DiagnosticItem {
	if d, ok := err.(diagnostics.Diagnostic); ok {
		return DiagnosticItem{
			Code:    d.Code,
			Message: d.Message,
			File:    d.File,
			Line:    d.Line,
			Column:  d.Column,
			Snippet: d.Snippet,
		}
	}
	return DiagnosticItem{
		Code:    "ERROR",
		Message: err.Error(),
		File:    file,
	}
}

// WriteJSON writes the full JSON report if path is non-empty.
func WriteJSON(path string, report JSONReport) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func intToString(v int) string {
	return strconv.Itoa(v)
}

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// WriteCSV writes the flattened CSV report if path is non-empty.
func WriteCSV(path string, files []FileItem) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()

	w := csv.NewWriter(fh)
	defer w.Flush()

	header := []string{
		"file",
		"status",
		"diagnostics_count",
		"helpers_count",
		"features_count",
		"render_checked",
		"sample_path",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	copied := append([]FileItem(nil), files...)
	sort.Slice(copied, func(i, j int) bool { return copied[i].File < copied[j].File })

	for _, item := range copied {
		row := []string{
			item.File,
			string(item.Status),
			intToString(len(item.Diagnostics)),
			intToString(len(item.HelpersRequired)),
			intToString(len(item.FeaturesDetected)),
			boolToString(item.RenderChecked),
			item.SamplePath,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Error()
}
