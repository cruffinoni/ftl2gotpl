package cli

import (
	"github.com/cruffinoni/ftl2gotpl/internal/config"
	"github.com/spf13/cobra"
)

// NewRootCmd wires CLI flags to configuration and executes conversion.
func NewRootCmd() *cobra.Command {
	cfg := config.Default()

	cmd := &cobra.Command{
		Use:           "ftl2gotpl",
		Short:         "Convert FreeMarker templates to Go html/template syntax",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConvert(cmd.Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.In, "in", "", "Input root directory containing .ftl templates")
	cmd.Flags().StringVar(&cfg.Out, "out", "", "Output root directory for converted templates")
	cmd.Flags().StringVar(&cfg.Glob, "glob", cfg.Glob, "Glob pattern relative to --in (supports **)")
	cmd.Flags().StringVar(&cfg.Ext, "ext", cfg.Ext, "Output file extension (example: .gotmpl)")
	cmd.Flags().BoolVar(&cfg.RenderCheck, "render-check", cfg.RenderCheck, "Enable render checks (M3)")
	cmd.Flags().StringVar(&cfg.SamplesRoot, "samples-root", cfg.SamplesRoot, "Path to sample JSON root")
	cmd.Flags().BoolVar(&cfg.Strict, "strict", cfg.Strict, "Enable strict conversion behavior")
	cmd.Flags().StringVar(&cfg.ReportJSON, "report-json", "", "Optional JSON report output path")
	cmd.Flags().StringVar(&cfg.ReportCSV, "report-csv", "", "Optional CSV report output path")

	_ = cmd.MarkFlagRequired("in")
	_ = cmd.MarkFlagRequired("out")

	return cmd
}
