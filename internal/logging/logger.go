package logging

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

// Configure sets a default slog logger with colorized levels on interactive terminals.
func Configure() {
	out := io.Writer(os.Stderr)
	if colorEnabled(os.Stderr) {
		out = colorizingWriter{out: os.Stderr}
	}
	handler := slog.NewTextHandler(out, &slog.HandlerOptions{})
	slog.SetDefault(slog.New(handler))
}

type colorizingWriter struct {
	out io.Writer
}

func (w colorizingWriter) Write(p []byte) (int, error) {
	colored := p
	colored = bytes.ReplaceAll(colored, []byte("level=ERROR"), []byte("level="+ansiRed+"ERROR"+ansiReset))
	colored = bytes.ReplaceAll(colored, []byte("level=WARN"), []byte("level="+ansiYellow+"WARN"+ansiReset))
	colored = bytes.ReplaceAll(colored, []byte("level=INFO"), []byte("level="+ansiGreen+"INFO"+ansiReset))
	colored = bytes.ReplaceAll(colored, []byte("level=DEBUG"), []byte("level="+ansiCyan+"DEBUG"+ansiReset))

	if _, err := w.out.Write(colored); err != nil {
		return 0, err
	}
	return len(p), nil
}

func colorEnabled(f *os.File) bool {
	if os.Getenv("CLICOLOR_FORCE") == "1" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}

	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
