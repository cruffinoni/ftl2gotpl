package templatecheck

import (
	"fmt"
	"html/template"

	"github.com/cruffinoni/ftl2gotpl/internal/convert"
)

// ParseConvertedTemplate verifies html/template parsing for converted content.
func ParseConvertedTemplate(name string, content string) error {
	t := template.New(name).Funcs(convert.StubFuncMap())
	if _, err := t.Parse(content); err != nil {
		return fmt.Errorf("parse converted template %q: %w", name, err)
	}
	return nil
}
