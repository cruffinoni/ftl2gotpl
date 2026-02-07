package convert

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapExprBuiltins(t *testing.T) {
	m := newExpressionMapper(map[string]struct{}{})
	got, err := m.mapExpr(`ad.price!''`)
	require.NoError(t, err)
	want := `default "" .ad.price`
	require.Equal(t, want, got)

	helpers := m.helperList()
	require.True(t, slices.Equal(helpers, []string{"default"}))
}

func TestMapExprFormatPriceCall(t *testing.T) {
	m := newExpressionMapper(map[string]struct{}{})
	got, err := m.mapExpr(`formatPrice(ad.price!'')`)
	require.NoError(t, err)
	require.Equal(t, `formatPrice (default "" .ad.price)`, got)

	helpers := m.helperList()
	require.True(t, slices.Equal(helpers, []string{"default", "formatPrice"}))
}

func TestMapExprIndexOfBuiltin(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "one argument",
			expr: `p?index_of("-")`,
			want: `indexOf .p "-"`,
		},
		{
			name: "two arguments",
			expr: `p?index_of("-", 2)`,
			want: `indexOf .p "-" 2`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			m := newExpressionMapper(map[string]struct{}{})
			got, err := m.mapExpr(tc.expr)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.True(t, slices.Equal(m.helperList(), []string{"indexOf"}))
		})
	}
}

func TestMapExprIndexOfBuiltinArity(t *testing.T) {
	tests := []string{
		`p?index_of()`,
		`p?index_of("a", 1, 2)`,
	}

	for _, expr := range tests {
		m := newExpressionMapper(map[string]struct{}{})
		_, err := m.mapExpr(expr)
		require.ErrorContains(t, err, "?index_of expects one or two arguments")
	}
}

func TestMapExprIndexBuiltin(t *testing.T) {
	m := newExpressionMapper(map[string]struct{}{
		"user":       {},
		"user_index": {},
	})
	got, err := m.mapExpr(`user?index`)
	require.NoError(t, err)
	require.Equal(t, `$user_index`, got)
}

func TestMapExprIndexBuiltinValidation(t *testing.T) {
	tests := []struct {
		name   string
		locals map[string]struct{}
		expr   string
		err    string
	}{
		{
			name:   "no args allowed",
			locals: map[string]struct{}{"user": {}, "user_index": {}},
			expr:   `user?index(1)`,
			err:    "?index expects no arguments",
		},
		{
			name:   "requires loop local",
			locals: map[string]struct{}{},
			expr:   `user?index`,
			err:    "?index is only supported on loop item variables",
		},
		{
			name:   "disallow nested path",
			locals: map[string]struct{}{"user": {}, "user_index": {}},
			expr:   `user.name?index`,
			err:    "?index is only supported on loop item variables",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			m := newExpressionMapper(tc.locals)
			_, err := m.mapExpr(tc.expr)
			require.ErrorContains(t, err, tc.err)
		})
	}
}

func TestMapExprSingleQuotedLiteralNormalization(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "plain",
			expr: `'abc'`,
			want: `"abc"`,
		},
		{
			name: "empty",
			expr: `''`,
			want: `""`,
		},
		{
			name: "escaped apostrophe",
			expr: `'l\'abc'`,
			want: `"l'abc"`,
		},
		{
			name: "number unchanged",
			expr: `42`,
			want: `42`,
		},
		{
			name: "bool unchanged",
			expr: `true`,
			want: `true`,
		},
		{
			name: "null unchanged",
			expr: `null`,
			want: `null`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newExpressionMapper(map[string]struct{}{})
			got, err := m.mapExpr(tc.expr)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestMapExprBracketAccess(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "string key",
			expr: `user.metadata.attributes["userType"]`,
			want: `index .user.metadata.attributes "userType"`,
		},
		{
			name: "local index key",
			expr: `users[user_index].name`,
			want: `(index .users $user_index).name`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			m := newExpressionMapper(map[string]struct{}{"user_index": {}})
			got, err := m.mapExpr(tc.expr)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
