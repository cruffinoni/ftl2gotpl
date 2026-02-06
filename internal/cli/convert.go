package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/cruffinoni/ftl2gotpl/internal/config"
	"github.com/cruffinoni/ftl2gotpl/internal/convert"
	"github.com/cruffinoni/ftl2gotpl/internal/fswalk"
	"github.com/cruffinoni/ftl2gotpl/internal/rendercheck"
	"github.com/cruffinoni/ftl2gotpl/internal/report"
	"github.com/cruffinoni/ftl2gotpl/internal/templatecheck"
)

func writeReports(cfg config.Config, summary report.Summary, files []report.FileItem) error {
	if cfg.ReportJSON != "" {
		if err := report.WriteJSON(cfg.ReportJSON, report.NewJSONReport(summary, files)); err != nil {
			return err
		}
	}
	if cfg.ReportCSV != "" {
		if err := report.WriteCSV(cfg.ReportCSV, files); err != nil {
			return err
		}
	}
	return nil
}

func runConvert(ctx context.Context, cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	files, err := fswalk.DiscoverTemplates(cfg.In, cfg.Glob)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no template files matched %q under %q", cfg.Glob, cfg.In)
	}

	converter := convert.NewConverter()
	var (
		converted        int
		conversionFailed int
		parseFailed      int
		renderFailed     int
		noSample         int

		helpers   = map[string]struct{}{}
		fileItems = make([]report.FileItem, 0, len(files))

		stopErr  error
		stopCode = ExitCodeSuccess
	)

	for _, f := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		raw, err := os.ReadFile(f.AbsPath)
		if err != nil {
			return fmt.Errorf("read %q: %w", f.AbsPath, err)
		}

		item := report.FileItem{
			File: f.RelPath,
		}

		result, err := converter.Convert(f.RelPath, string(raw))
		if err != nil {
			conversionFailed++
			item.Status = report.StatusConversionError
			item.Diagnostics = []report.DiagnosticItem{report.ToDiagnosticItem(f.RelPath, err)}
			fileItems = append(fileItems, item)
			slog.Warn("conversion failed", "file", f.RelPath, "error", err)
			if cfg.Strict {
				stopErr = fmt.Errorf("conversion failed on %s: %w", f.RelPath, err)
				stopCode = ExitCodeConversionFailed
				break
			}
			continue
		}
		item.FeaturesDetected = append(item.FeaturesDetected, result.Features...)
		item.HelpersRequired = append(item.HelpersRequired, result.Helpers...)

		if err := templatecheck.ParseConvertedTemplate(f.RelPath, result.Output); err != nil {
			parseFailed++
			item.Status = report.StatusParseError
			item.Diagnostics = []report.DiagnosticItem{report.ToDiagnosticItem(f.RelPath, err)}
			fileItems = append(fileItems, item)
			slog.Warn("parse-check failed", "file", f.RelPath, "error", err)
			if cfg.Strict {
				stopErr = fmt.Errorf("parse-check failed on %s: %w", f.RelPath, err)
				stopCode = ExitCodeValidationFailed
				break
			}
			continue
		}

		if cfg.RenderCheck {
			samplePath := rendercheck.SamplePath(cfg.SamplesRoot, f.RelPath)
			item.RenderChecked = true
			item.SamplePath = samplePath
			status, renderErr := rendercheck.RenderConvertedTemplate(f.RelPath, result.Output, samplePath)
			if renderErr != nil {
				renderFailed++
				item.Status = report.StatusRenderError
				item.Diagnostics = []report.DiagnosticItem{report.ToDiagnosticItem(f.RelPath, renderErr)}
				fileItems = append(fileItems, item)
				slog.Warn("render-check failed", "file", f.RelPath, "error", renderErr)
				if cfg.Strict {
					stopErr = fmt.Errorf("render-check failed on %s: %w", f.RelPath, renderErr)
					stopCode = ExitCodeValidationFailed
					break
				}
				continue
			}
			if status == rendercheck.StatusNoSample {
				noSample++
				item.Status = report.StatusConvertedNoData
			}
		}
		if item.Status == "" {
			item.Status = report.StatusConverted
		}

		outPath := fswalk.MirrorOutputPath(cfg.Out, f.RelPath, cfg.Ext)
		if err := fswalk.EnsureParentDir(outPath); err != nil {
			return fmt.Errorf("prepare output path %q: %w", outPath, err)
		}
		if err := os.WriteFile(outPath, []byte(result.Output), 0o644); err != nil {
			return fmt.Errorf("write converted template %q: %w", outPath, err)
		}
		for _, h := range result.Helpers {
			helpers[h] = struct{}{}
		}
		converted++
		fileItems = append(fileItems, item)
	}

	helperList := make([]string, 0, len(helpers))
	for h := range helpers {
		helperList = append(helperList, h)
	}
	sort.Strings(helperList)

	slog.Info(
		"conversion summary",
		"discovered",
		len(files),
		"converted",
		converted,
		"conversion_failed",
		conversionFailed,
		"parse_failed",
		parseFailed,
		"render_failed",
		renderFailed,
		"no_sample",
		noSample,
		"input",
		filepath.Clean(cfg.In),
		"output",
		filepath.Clean(cfg.Out),
	)

	summary := report.Summary{
		Discovered:       len(files),
		Converted:        converted,
		ConversionFailed: conversionFailed,
		ParseFailed:      parseFailed,
		RenderFailed:     renderFailed,
		NoSample:         noSample,
		HelpersNeeded:    helperList,
	}

	if err := writeReports(cfg, summary, fileItems); err != nil {
		return fmt.Errorf("write report artifacts: %w", err)
	}

	if len(helperList) > 0 {
		slog.Info("helpers needed", "helpers", helperList)
	}
	if cfg.ReportJSON != "" || cfg.ReportCSV != "" {
		slog.Info("reports written", "json", cfg.ReportJSON, "csv", cfg.ReportCSV)
	}

	if stopErr != nil {
		return newExitError(stopCode, stopErr)
	}

	if conversionFailed > 0 {
		return newExitError(ExitCodeConversionFailed, fmt.Errorf("conversion finished with %d failed files", conversionFailed))
	}
	if parseFailed > 0 || renderFailed > 0 {
		return newExitError(ExitCodeValidationFailed, fmt.Errorf("validation finished with parse_failed=%d render_failed=%d", parseFailed, renderFailed))
	}

	return nil
}
