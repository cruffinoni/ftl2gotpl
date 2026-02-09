package rendercheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
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

func normalizeJSONNumbers(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for k, item := range v {
			v[k] = normalizeJSONNumbers(item)
		}
		return v
	case []any:
		for i, item := range v {
			v[i] = normalizeJSONNumbers(item)
		}
		return v
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		f, err := v.Float64()
		if err != nil {
			return v
		}
		if f == math.Trunc(f) && f >= math.MinInt64 && f <= math.MaxInt64 {
			return int64(f)
		}
		return f
	default:
		return value
	}
}

// RenderConvertedTemplate parses and executes converted content.
func RenderConvertedTemplate(name string, content string, samplePath string) (Status, string, error) {
	raw, err := os.ReadFile(samplePath)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusNoSample, "", nil
		}
		return StatusNoSample, "", fmt.Errorf("read sample %q: %w", samplePath, err)
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var payload any
	if err := dec.Decode(&payload); err != nil {
		return StatusNoSample, "", fmt.Errorf("decode sample JSON %q: %w", samplePath, err)
	}
	payload = normalizeJSONNumbers(payload)

	t, err := template.New(name).Funcs(convert.StubFuncMap()).Parse(content)
	if err != nil {
		return StatusNoSample, "", fmt.Errorf("parse converted template %q before render: %w", name, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, payload); err != nil {
		return StatusNoSample, "", fmt.Errorf("render template %q with sample %q: %w", name, samplePath, err)
	}

	return StatusRendered, buf.String(), nil
}
