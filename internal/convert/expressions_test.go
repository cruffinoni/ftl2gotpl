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
	want := `default '' .ad.price`
	require.Equal(t, want, got)

	helpers := m.helperList()
	require.True(t, slices.Equal(helpers, []string{"default"}))
}
