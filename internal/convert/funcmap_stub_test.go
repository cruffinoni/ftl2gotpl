package convert

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStubFuncMapExistsAndDefault(t *testing.T) {
	fm := StubFuncMap()

	exists := fm["exists"].(func(any) bool)
	defaultFn := fm["default"].(func(any, any) any)

	assert.False(t, exists(nil))
	assert.True(t, exists(""))
	assert.Equal(t, "fallback", defaultFn("fallback", nil))
	assert.Equal(t, "", defaultFn("fallback", ""))
}

func TestStubFuncMapHasContent(t *testing.T) {
	fm := StubFuncMap()
	hasContent := fm["hasContent"].(func(any) bool)

	cases := map[string]struct {
		in   any
		want bool
	}{
		"nil":             {in: nil, want: false},
		"blank string":    {in: "   ", want: false},
		"non blank":       {in: "x", want: true},
		"empty slice":     {in: []any{}, want: false},
		"non empty slice": {in: []any{1}, want: true},
		"empty map":       {in: map[string]any{}, want: false},
		"non empty map":   {in: map[string]any{"a": 1}, want: true},
		"number":          {in: 0, want: true},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasContent(tc.in))
		})
	}
}

func TestStubFuncMapContains(t *testing.T) {
	fm := StubFuncMap()
	contains := fm["contains"].(func(any, any) bool)

	assert.True(t, contains("abcdef", "cd"))
	assert.True(t, contains([]any{1.0, "x"}, 1))
	assert.True(t, contains(map[string]any{"foo": 1}, "foo"))
	assert.False(t, contains(map[string]any{"foo": 1}, "bar"))
}

func TestStubFuncMapSubstringAndIndexOf(t *testing.T) {
	fm := StubFuncMap()
	substring := fm["substring"].(func(any, any, ...any) string)
	indexOf := fm["indexOf"].(func(any, any, ...any) int)

	assert.Equal(t, "onj", substring("bonjour", 1, 4))
	assert.Equal(t, "é", substring("école", 0, 1))
	assert.Equal(t, "", substring("abc", 3, 1))
	assert.Equal(t, 1, indexOf("école", "co"))
	assert.Equal(t, -1, indexOf("abc", "z"))
	assert.Equal(t, 3, indexOf("ababa", "ba", 2))
	assert.Equal(t, 0, indexOf("aba", "a", -5))
	assert.Equal(t, -1, indexOf("abc", "a", 4))
	assert.Equal(t, 5, indexOf("écoleé", "é", 1))
	assert.Equal(t, 2, indexOf("abc", "", 2))
	assert.Equal(t, -1, indexOf("abc", "", 4))
	assert.Equal(t, 2, indexOf(12345, 34, 1))
}

func TestStubFuncMapNumberAndDatetime(t *testing.T) {
	fm := StubFuncMap()
	toNumber := fm["toNumber"].(func(any) any)
	numberToDatetime := fm["numberToDatetime"].(func(any) string)

	assert.Equal(t, int64(42), toNumber("42"))
	assert.Equal(t, float64(3.5), toNumber("3.5"))
	assert.Equal(t, int64(1), toNumber(true))
	assert.Equal(t, "1970-01-01T00:00:00Z", numberToDatetime(0))
	assert.Equal(t, "2024-01-01T00:00:00Z", numberToDatetime("1704067200000"))
}

func TestStubFuncMapStringTrimAndSafeHTML(t *testing.T) {
	fm := StubFuncMap()
	trim := fm["trim"].(func(any) string)
	toString := fm["toString"].(func(any, ...any) string)
	safeHTML := fm["safeHTML"].(func(any) template.HTML)

	assert.Equal(t, "hello", trim("  hello  "))
	assert.Equal(t, "12", toString(12))
	assert.Equal(t, template.HTML("<b>x</b>"), safeHTML("<b>x</b>"))
}

func TestStubFuncMapFormatPrice(t *testing.T) {
	fm := StubFuncMap()
	formatPrice := fm["formatPrice"].(func(any) string)

	assert.Equal(t, "120 €", formatPrice("120€"))
	assert.Equal(t, "120 €", formatPrice("120"))
	assert.Equal(t, "120 €-130 €", formatPrice("120-130"))
	assert.Equal(t, "120 €", formatPrice("120-120"))
	assert.Equal(t, "", formatPrice(nil))
}

func TestStubFuncMapTemplateName(t *testing.T) {
	fm := StubFuncMap()
	templateName := fm["templateName"].(func(...any) string)

	assert.Equal(t, "", templateName())
	assert.Equal(t, "", templateName(nil))
	assert.Equal(t, "welcome-email", templateName("  welcome-email  "))
}
