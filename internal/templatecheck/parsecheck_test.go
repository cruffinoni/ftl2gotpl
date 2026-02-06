package templatecheck

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConvertedTemplate(t *testing.T) {
	require.NoError(t, ParseConvertedTemplate("ok", `{{if eq .client_id "mim"}}ok{{end}}`))
	require.Error(t, ParseConvertedTemplate("bad", `{{if .a}}`))
}
