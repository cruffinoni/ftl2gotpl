package convert

import (
	"html/template"
	"testing"
	"time"

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

	emptyString := ""
	spaceString := "   "
	zeroNumber := 0
	falseBool := false
	var nilStringPointer *string

	cases := map[string]struct {
		in   any
		want bool
	}{
		"nil":                     {in: nil, want: false},
		"empty string":            {in: "", want: false},
		"whitespace string":       {in: "   ", want: true},
		"non empty string":        {in: "x", want: true},
		"empty slice":             {in: []any{}, want: false},
		"non empty slice":         {in: []any{1}, want: true},
		"empty map":               {in: map[string]any{}, want: false},
		"non empty map":           {in: map[string]any{"a": 1}, want: true},
		"zero number":             {in: 0, want: true},
		"false bool":              {in: false, want: true},
		"zero time":               {in: time.Time{}, want: true},
		"non-zero time":           {in: time.Unix(1704067200, 0).UTC(), want: true},
		"unknown struct is empty": {in: struct{ Value string }{Value: "x"}, want: false},
		"nil pointer":             {in: nilStringPointer, want: false},
		"pointer empty string":    {in: &emptyString, want: false},
		"pointer whitespace":      {in: &spaceString, want: true},
		"pointer zero number":     {in: &zeroNumber, want: true},
		"pointer false bool":      {in: &falseBool, want: true},
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
	contains := fm["contains"].(func(any, any) (bool, error))

	got, err := contains("abcdef", "cd")
	assert.NoError(t, err)
	assert.True(t, got)

	_, err = contains([]any{1.0, "x"}, "x")
	assert.Error(t, err)

	_, err = contains("abcdef", 123)
	assert.Error(t, err)
}

func TestStubFuncMapSubstringAndIndexOf(t *testing.T) {
	fm := StubFuncMap()
	substring := fm["substring"].(func(any, any, ...any) (string, error))
	indexOf := fm["indexOf"].(func(any, any, ...any) (int, error))

	gotSub, err := substring("bonjour", 1, 4)
	assert.NoError(t, err)
	assert.Equal(t, "onj", gotSub)

	gotSub, err = substring("école", 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, "é", gotSub)

	_, err = substring("abc", 3, 1)
	assert.Error(t, err)

	_, err = substring("abc", "1", 2)
	assert.Error(t, err)

	_, err = substring(123, 0, 1)
	assert.Error(t, err)

	gotIdx, err := indexOf("école", "co")
	assert.NoError(t, err)
	assert.Equal(t, 1, gotIdx)

	gotIdx, err = indexOf("abc", "z")
	assert.NoError(t, err)
	assert.Equal(t, -1, gotIdx)

	gotIdx, err = indexOf("ababa", "ba", 2)
	assert.NoError(t, err)
	assert.Equal(t, 3, gotIdx)

	gotIdx, err = indexOf("aba", "a", -5)
	assert.NoError(t, err)
	assert.Equal(t, 0, gotIdx)

	gotIdx, err = indexOf("abc", "a", 4)
	assert.NoError(t, err)
	assert.Equal(t, -1, gotIdx)

	gotIdx, err = indexOf("écoleé", "é", 1)
	assert.NoError(t, err)
	assert.Equal(t, 5, gotIdx)

	gotIdx, err = indexOf("abc", "", 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, gotIdx)

	gotIdx, err = indexOf("abc", "", 4)
	assert.NoError(t, err)
	assert.Equal(t, -1, gotIdx)

	_, err = indexOf(12345, "34", 1)
	assert.Error(t, err)

	_, err = indexOf("12345", 34, 1)
	assert.Error(t, err)

	_, err = indexOf("12345", "34", "1")
	assert.Error(t, err)
}

func TestStubFuncMapNumberAndDatetime(t *testing.T) {
	fm := StubFuncMap()
	toNumber := fm["toNumber"].(func(any) (any, error))
	numberToDatetime := fm["numberToDatetime"].(func(any) (time.Time, error))

	gotNum, err := toNumber("42")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), gotNum)

	gotNum, err = toNumber("3.5")
	assert.NoError(t, err)
	assert.Equal(t, float64(3.5), gotNum)

	gotNum, err = toNumber(12.0)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), gotNum)

	for _, invalid := range []any{nil, "", "   ", "oops", true, struct{}{}, []any{1}} {
		_, err = toNumber(invalid)
		assert.Error(t, err)
	}

	gotTime, err := numberToDatetime(0)
	assert.NoError(t, err)
	assert.Equal(t, time.Unix(0, 0).UTC(), gotTime)

	gotTime, err = numberToDatetime("1704067200000")
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), gotTime)

	_, err = numberToDatetime("1704067200000.5")
	assert.Error(t, err)

	_, err = numberToDatetime(false)
	assert.Error(t, err)
}

func TestStubFuncMapToStringStrict(t *testing.T) {
	fm := StubFuncMap()
	toString := fm["toString"].(func(...any) (string, error))

	got, err := toString()
	assert.NoError(t, err)
	assert.Equal(t, "", got)

	got, err = toString(12)
	assert.NoError(t, err)
	assert.Equal(t, "12", got)

	got, err = toString(12345, "#,###")
	assert.NoError(t, err)
	assert.Equal(t, "12,345", got)

	got, err = toString(1234.5, "$ #,##0.00")
	assert.NoError(t, err)
	assert.Equal(t, "$ 1,234.50", got)

	got, err = toString(1234.5, "#,##0.##")
	assert.NoError(t, err)
	assert.Equal(t, "1,234.5", got)

	ts := time.Date(2024, 5, 20, 10, 30, 45, 0, time.UTC)
	got, err = toString(ts, "yyyy-MM-dd HH:mm:ss")
	assert.NoError(t, err)
	assert.Equal(t, "2024-05-20 10:30:45", got)

	got, err = toString(ts, "yyyy-MM-dd", "UTC")
	assert.NoError(t, err)
	assert.Equal(t, "2024-05-20", got)

	_, err = toString(12, "")
	assert.Error(t, err)

	_, err = toString(12, 42)
	assert.Error(t, err)

	_, err = toString(12, "unknown")
	assert.Error(t, err)

	_, err = toString(12, "yyyy-MM-dd")
	assert.Error(t, err)

	_, err = toString(ts, "#,###")
	assert.Error(t, err)

	_, err = toString(ts, "yyyy-MM-dd", "fr_FR")
	assert.Error(t, err)

	_, err = toString(12, "#,##0", "#,##0", "#,##0")
	assert.Error(t, err)
}

func TestStubFuncMapStringTrimAndSafeHTML(t *testing.T) {
	fm := StubFuncMap()
	trim := fm["trim"].(func(any) (string, error))
	safeHTML := fm["safeHTML"].(func(any) template.HTML)

	gotTrimmed, err := trim("  hello  ")
	assert.NoError(t, err)
	assert.Equal(t, "hello", gotTrimmed)

	_, err = trim(12)
	assert.Error(t, err)

	assert.Equal(t, template.HTML("<b>x</b>"), safeHTML("<b>x</b>"))
}

func TestStubFuncMapSafeAccess(t *testing.T) {
	type profile struct {
		Name string
	}
	type root struct {
		Profile profile
		Items   []map[string]any
	}

	fm := StubFuncMap()
	safeAccess := fm["safeAccess"].(func(any, ...any) any)

	data := map[string]any{
		"user": map[string]any{
			"name": "alice",
			"tags": []any{"a", map[string]any{"value": "b"}},
		},
	}
	assert.Equal(t, "alice", safeAccess(data, "user", "name"))
	assert.Equal(t, "b", safeAccess(data, "user", "tags", 1, "value"))
	assert.Nil(t, safeAccess(data, "user", "missing"))
	assert.Nil(t, safeAccess(data, "user", "tags", 10))
	assert.Nil(t, safeAccess(data, "user", "tags", "bad-index"))

	structData := root{
		Profile: profile{Name: "bob"},
		Items:   []map[string]any{{"label": "first"}},
	}
	assert.Equal(t, "bob", safeAccess(structData, "Profile", "Name"))
	assert.Equal(t, "first", safeAccess(structData, "Items", 0, "label"))
	assert.Nil(t, safeAccess(structData, "Items", 2, "label"))
	assert.Nil(t, safeAccess(structData, "Unknown"))
	assert.Nil(t, safeAccess(nil, "anything"))
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
