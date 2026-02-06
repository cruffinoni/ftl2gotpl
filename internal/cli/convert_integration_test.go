package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cruffinoni/ftl2gotpl/internal/config"
	"github.com/cruffinoni/ftl2gotpl/internal/report"
	"github.com/stretchr/testify/require"
)

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.NoError(t, err)
}

func TestRunConvertEndToEndAndReports(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "in")
	out := filepath.Join(root, "out")
	require.NoError(t, os.MkdirAll(filepath.Join(in, "nested"), 0o755))
	mustWrite(t, filepath.Join(in, "a.ftl"), `Hello ${name}`)
	mustWrite(t, filepath.Join(in, "nested", "b.ftl"), `<#if user??>${user.name}<#else>x</#if>`)

	cfg := config.Default()
	cfg.In = in
	cfg.Out = out
	cfg.Strict = false
	cfg.ReportJSON = filepath.Join(root, "report", "report.json")
	cfg.ReportCSV = filepath.Join(root, "report", "report.csv")

	require.NoError(t, runConvert(context.Background(), cfg))

	assertExists(t, filepath.Join(out, "a.gotmpl"))
	assertExists(t, filepath.Join(out, "nested", "b.gotmpl"))
	assertExists(t, cfg.ReportJSON)
	assertExists(t, cfg.ReportCSV)
}

func TestRunConvertRenderCheckValidSample(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "in")
	out := filepath.Join(root, "out")
	samples := filepath.Join(root, "samples")
	require.NoError(t, os.MkdirAll(in, 0o755))
	require.NoError(t, os.MkdirAll(samples, 0o755))

	mustWrite(t, filepath.Join(in, "mail.ftl"), `Hello ${name}`)
	mustWrite(t, filepath.Join(samples, "mail.ftl.json"), `{"name":"Ada"}`)

	cfg := config.Default()
	cfg.In = in
	cfg.Out = out
	cfg.RenderCheck = true
	cfg.SamplesRoot = samples

	require.NoError(t, runConvert(context.Background(), cfg))
}

func TestRunConvertRenderCheckMissingSampleIsNonFatal(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "in")
	out := filepath.Join(root, "out")
	samples := filepath.Join(root, "samples")
	require.NoError(t, os.MkdirAll(in, 0o755))
	require.NoError(t, os.MkdirAll(samples, 0o755))

	mustWrite(t, filepath.Join(in, "mail.ftl"), `Hello ${name}`)

	jsonReport := filepath.Join(root, "report.json")

	cfg := config.Default()
	cfg.In = in
	cfg.Out = out
	cfg.RenderCheck = true
	cfg.SamplesRoot = samples
	cfg.ReportJSON = jsonReport

	require.NoError(t, runConvert(context.Background(), cfg))

	raw, err := os.ReadFile(jsonReport)
	require.NoError(t, err)
	var rep report.JSONReport
	require.NoError(t, json.Unmarshal(raw, &rep))
	require.Equal(t, 1, rep.Summary.NoSample)
}

func TestRunConvertRenderCheckInvalidSampleReturnsExitCode3(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "in")
	out := filepath.Join(root, "out")
	samples := filepath.Join(root, "samples")
	require.NoError(t, os.MkdirAll(in, 0o755))
	require.NoError(t, os.MkdirAll(samples, 0o755))

	mustWrite(t, filepath.Join(in, "mail.ftl"), `Hello ${name}`)
	mustWrite(t, filepath.Join(samples, "mail.ftl.json"), `{invalid-json}`)

	cfg := config.Default()
	cfg.In = in
	cfg.Out = out
	cfg.Strict = false
	cfg.RenderCheck = true
	cfg.SamplesRoot = samples

	err := runConvert(context.Background(), cfg)
	require.Error(t, err)
	var exitErr *ExitError
	require.True(t, errors.As(err, &exitErr))
	require.Equal(t, ExitCodeValidationFailed, exitErr.Code)
}

func TestRunConvertUnsupportedFunctionReturnsExitCode2(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "in")
	out := filepath.Join(root, "out")
	require.NoError(t, os.MkdirAll(in, 0o755))

	mustWrite(t, filepath.Join(in, "mail.ftl"), `<#function f x><#return x></#function>`)

	cfg := config.Default()
	cfg.In = in
	cfg.Out = out
	cfg.Strict = true

	err := runConvert(context.Background(), cfg)
	require.Error(t, err)
	var exitErr *ExitError
	require.True(t, errors.As(err, &exitErr))
	require.Equal(t, ExitCodeConversionFailed, exitErr.Code)
}
