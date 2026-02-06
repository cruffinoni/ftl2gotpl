package lexer

import (
	"testing"

	"github.com/cruffinoni/ftl2gotpl/internal/diagnostics"
	"github.com/stretchr/testify/require"
)

func TestLexBasicTemplate(t *testing.T) {
	src := `Hello ${user.name}! <#if user.active>ON<#else>OFF</#if>`
	tokens, err := Lex("sample.ftl", src)
	require.NoError(t, err)
	require.Len(t, tokens, 8)
	require.Equal(t, TokenInterpolation, tokens[1].Kind)
	require.Equal(t, "user.name", tokens[1].Value)
	require.Equal(t, TokenDirective, tokens[3].Kind)
	require.Equal(t, "if", tokens[3].Name)
	require.False(t, tokens[3].Closing)
	require.Equal(t, TokenDirective, tokens[7].Kind)
	require.Equal(t, "if", tokens[7].Name)
	require.True(t, tokens[7].Closing)
}

func TestLexReportsLineAndColumn(t *testing.T) {
	_, err := Lex("broken.ftl", "abc ${missing")
	require.Error(t, err)
	diag, ok := err.(diagnostics.Diagnostic)
	require.True(t, ok)
	require.Equal(t, 1, diag.Line)
	require.Equal(t, 5, diag.Column)
}
