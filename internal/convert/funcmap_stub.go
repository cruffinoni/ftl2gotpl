package convert

import (
	"fmt"
	"html/template"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func isNilLike(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func indirect(v any) any {
	for v != nil {
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Pointer && rv.Kind() != reflect.Interface {
			break
		}
		if rv.IsNil() {
			return nil
		}
		v = rv.Elem().Interface()
	}
	return v
}

func toNumber(v any) (float64, bool) {
	v = indirect(v)
	if v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	case string:
		raw := strings.TrimSpace(t)
		if raw == "" {
			return 0, false
		}
		if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return float64(i), true
		}
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func toInt(v any) int {
	n, ok := toNumber(v)
	if !ok {
		return 0
	}
	return int(n)
}

func clampIndex(i int, size int) int {
	if i < 0 {
		return 0
	}
	if i > size {
		return size
	}
	return i
}

func valuesEqual(a any, b any) bool {
	a = indirect(a)
	b = indirect(b)
	if isNilLike(a) && isNilLike(b) {
		return true
	}
	if isNilLike(a) || isNilLike(b) {
		return false
	}

	if an, ok := toNumber(a); ok {
		if bn, ok2 := toNumber(b); ok2 {
			return an == bn
		}
	}

	return reflect.DeepEqual(a, b)
}

func mapKeyFrom(v any, keyType reflect.Type) (reflect.Value, bool) {
	v = indirect(v)
	if v == nil {
		return reflect.Value{}, false
	}

	rv := reflect.ValueOf(v)
	if rv.Type().AssignableTo(keyType) {
		return rv, true
	}
	if rv.Type().ConvertibleTo(keyType) {
		converted := rv.Convert(keyType)
		return converted, true
	}

	if keyType.Kind() == reflect.String {
		return reflect.ValueOf(fmt.Sprint(v)), true
	}
	return reflect.Value{}, false
}

// StubFuncMap returns helpers used by converted templates.
//
// The helpers are hardened for mixed runtime data (JSON-decoded maps, slices,
// numbers and nil values) and try to stay close to expected FreeMarker behavior
// for the built-ins currently supported by this converter.
func StubFuncMap() template.FuncMap {
	appendEuro := func(raw string) string {
		part := strings.TrimSpace(raw)
		part = strings.TrimSuffix(part, "€")
		part = strings.TrimSpace(part)
		if part == "" {
			return ""
		}
		return part + " €"
	}

	return template.FuncMap{
		"hasContent": func(v any) bool {
			v = indirect(v)
			if v == nil {
				return false
			}
			switch t := v.(type) {
			case string:
				return strings.TrimSpace(t) != ""
			}

			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
				return rv.Len() > 0
			}
			return true
		},
		"contains": func(container any, needle any) bool {
			container = indirect(container)
			needle = indirect(needle)
			if container == nil {
				return false
			}

			if s, ok := container.(string); ok {
				return strings.Contains(s, fmt.Sprint(needle))
			}

			rv := reflect.ValueOf(container)
			switch rv.Kind() {
			case reflect.Array, reflect.Slice:
				for i := 0; i < rv.Len(); i++ {
					if valuesEqual(rv.Index(i).Interface(), needle) {
						return true
					}
				}
				return false
			case reflect.Map:
				key, ok := mapKeyFrom(needle, rv.Type().Key())
				if !ok {
					return false
				}
				return rv.MapIndex(key).IsValid()
			default:
				return false
			}
		},
		"substring": func(v any, a any, b ...any) string {
			runes := []rune(fmt.Sprint(indirect(v)))
			start := clampIndex(toInt(a), len(runes))
			end := len(runes)
			if len(b) > 0 {
				end = clampIndex(toInt(b[0]), len(runes))
			}
			if start > end {
				return ""
			}
			return string(runes[start:end])
		},
		"indexOf": func(v any, sub any, start ...any) int {
			haystack := []rune(fmt.Sprint(indirect(v)))
			needle := []rune(fmt.Sprint(indirect(sub)))
			offset := 0
			if len(start) > 0 {
				offset = toInt(start[0])
			}
			if offset < 0 {
				offset = 0
			}
			if offset > len(haystack) {
				return -1
			}
			if len(needle) == 0 {
				return offset
			}
			if len(needle) > len(haystack) {
				return -1
			}
			for i := offset; i <= len(haystack)-len(needle); i++ {
				if string(haystack[i:i+len(needle)]) == string(needle) {
					return i
				}
			}
			return -1
		},
		"trim": func(v any) string {
			return strings.TrimSpace(fmt.Sprint(indirect(v)))
		},
		"toNumber": func(v any) any {
			n, ok := toNumber(v)
			if !ok {
				return int64(0)
			}
			if math.Mod(n, 1) == 0 {
				return int64(n)
			}
			return n
		},
		"numberToDatetime": func(v any) string {
			n, ok := toNumber(v)
			if !ok {
				return ""
			}
			ms := int64(n)
			sec := ms / 1000
			nsec := (ms % 1000) * int64(time.Millisecond)
			return time.Unix(sec, nsec).UTC().Format(time.RFC3339)
		},
		"toString": func(v any, _ ...any) string {
			return fmt.Sprint(indirect(v))
		},
		"exists": func(v any) bool {
			return !isNilLike(v)
		},
		"default": func(def any, v any) any {
			if isNilLike(v) {
				return def
			}
			return v
		},
		"safeHTML": func(v any) template.HTML {
			return template.HTML(fmt.Sprint(indirect(v)))
		},
		"templateName": func(v ...any) string {
			if len(v) == 0 {
				return ""
			}
			first := indirect(v[0])
			if isNilLike(first) {
				return ""
			}
			return strings.TrimSpace(fmt.Sprint(first))
		},
		"formatPrice": func(v any) string {
			v = indirect(v)
			if isNilLike(v) {
				return ""
			}
			raw := strings.TrimSpace(fmt.Sprint(v))
			if raw == "" {
				return ""
			}

			if strings.Contains(raw, "-") {
				parts := strings.SplitN(raw, "-", 2)
				firstPrice := appendEuro(parts[0])
				secondPrice := appendEuro(parts[1])
				if firstPrice == secondPrice {
					return firstPrice
				}
				if secondPrice == "" {
					return firstPrice
				}
				return firstPrice + "-" + secondPrice
			}

			base := raw
			if idx := strings.Index(base, "€"); idx >= 0 {
				base = base[:idx]
			}
			return appendEuro(base)
		},
	}
}
