package diagnostics

import "fmt"

// Diagnostic is a structured parser or conversion error with source metadata.
type Diagnostic struct {
	Code    string
	Message string
	File    string
	Line    int
	Column  int
	Snippet string
}

// Error implements the error interface with location and error code formatting.
func (d Diagnostic) Error() string {
	location := d.File
	if d.Line > 0 {
		location = fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Column)
	}
	if d.Code == "" {
		return fmt.Sprintf("%s: %s", location, d.Message)
	}
	return fmt.Sprintf("%s [%s]: %s", location, d.Code, d.Message)
}

// New constructs a Diagnostic value.
func New(code string, file string, line int, column int, msg string, snippet string) Diagnostic {
	return Diagnostic{
		Code:    code,
		Message: msg,
		File:    file,
		Line:    line,
		Column:  column,
		Snippet: snippet,
	}
}
