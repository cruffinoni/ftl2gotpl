package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertIfAndInterpolation(t *testing.T) {
	c := NewConverter()
	input := `<#if client_id="mim">Hi ${user.name}<#else>Bye</#if>`
	got, err := c.Convert("sample.ftl", input)
	require.NoError(t, err)

	want := `{{if eq .client_id "mim"}}Hi {{.user.name}}{{else}}Bye{{end}}`
	require.Equal(t, want, got.Output)
}

func TestConvertListLocalVar(t *testing.T) {
	c := NewConverter()
	input := `<#list users as user>${user.name}</#list>`
	got, err := c.Convert("sample.ftl", input)
	require.NoError(t, err)

	want := `{{range $user_index, $user := .users}}{{$user.name}}{{end}}`
	require.Equal(t, want, got.Output)
}

func TestConvertFunctionIsUnsupported(t *testing.T) {
	c := NewConverter()
	input := `<#function f x><#return x></#function>${f("a")}`
	_, err := c.Convert("sample.ftl", input)
	require.Error(t, err)
}

func TestConvertFormatPriceFunctionUsesStubHelper(t *testing.T) {
	c := NewConverter()
	input := `<#function formatPrice p><#return p></#function>${formatPrice(ad.price!'')}`
	got, err := c.Convert("sample.ftl", input)
	require.NoError(t, err)

	want := `{{/* ftl function formatPrice ignored: using helper stub */}}{{formatPrice (default "" .ad.price)}}`
	require.Equal(t, want, got.Output)
	require.Equal(t, []string{"default", "formatPrice"}, got.Helpers)
}
