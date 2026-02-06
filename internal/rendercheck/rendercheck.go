package rendercheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/cruffinoni/ftl2gotpl/internal/convert"
)

// Status reports the outcome of render validation for one converted template.
type Status string

const (
	StatusRendered Status = "rendered"
	StatusNoSample Status = "no_sample"
)

// SamplePath returns the sidecar JSON sample path for a template relative path.
func SamplePath(samplesRoot string, relTemplatePath string) string {
	return filepath.Join(samplesRoot, relTemplatePath+".json")
}

// RenderConvertedTemplate parses and executes converted content.
func RenderConvertedTemplate(name string, content string, samplePath string) (Status, error) {
	raw, err := os.ReadFile(samplePath)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusNoSample, nil
		}
		return StatusNoSample, fmt.Errorf("read sample %q: %w", samplePath, err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return StatusNoSample, fmt.Errorf("decode sample JSON %q: %w", samplePath, err)
	}

	t, err := template.New(name).Funcs(convert.StubFuncMap()).Parse(content)
	if err != nil {
		return StatusNoSample, fmt.Errorf("parse converted template %q before render: %w", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, payload); err != nil {
		return StatusNoSample, fmt.Errorf("render template %q with sample %q: %w", name, samplePath, err)
	}

	return StatusRendered, nil
}
