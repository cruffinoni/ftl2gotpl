// Package convert transforms FreeMarker templates into Go templates.
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

func normalizeFloatNumber(v float64) (any, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, fmt.Errorf("number must be finite")
	}
	if math.Trunc(v) == v {
		const maxInt64Float = float64(^uint64(0) >> 1)
		const minInt64Float = -maxInt64Float - 1
		if v < minInt64Float || v > maxInt64Float {
			return nil, fmt.Errorf("integral number %v is out of int64 range", v)
		}
		return int64(v), nil
	}
	return v, nil
}

func toNumber(v any) (any, error) {
	v = indirect(v)
	if v == nil {
		return nil, fmt.Errorf("number value is nil")
	}

	switch t := v.(type) {
	case int:
		return int64(t), nil
	case int8:
		return int64(t), nil
	case int16:
		return int64(t), nil
	case int32:
		return int64(t), nil
	case int64:
		return t, nil
	case uint:
		if uint64(t) > uint64(^uint64(0)>>1) {
			return nil, fmt.Errorf("integral number %d is out of int64 range", t)
		}
		return int64(t), nil
	case uint8:
		return int64(t), nil
	case uint16:
		return int64(t), nil
	case uint32:
		return int64(t), nil
	case uint64:
		if t > uint64(^uint64(0)>>1) {
			return nil, fmt.Errorf("integral number %d is out of int64 range", t)
		}
		return int64(t), nil
	case float32:
		return normalizeFloatNumber(float64(t))
	case float64:
		return normalizeFloatNumber(t)
	case bool:
		return nil, fmt.Errorf("boolean value is not numeric")
	case string:
		raw := strings.TrimSpace(t)
		if raw == "" {
			return nil, fmt.Errorf("number string is empty")
		}
		if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return i, nil
		}
		if u, err := strconv.ParseUint(raw, 10, 64); err == nil {
			if u > uint64(^uint64(0)>>1) {
				return nil, fmt.Errorf("integral number %s is out of int64 range", raw)
			}
			return int64(u), nil
		}
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return normalizeFloatNumber(f)
		}
		return nil, fmt.Errorf("invalid numeric string %q", raw)
	default:
		return nil, fmt.Errorf("unsupported numeric type %T", v)
	}
}

func toInt(v any) (int, error) {
	v = indirect(v)
	if v == nil {
		return 0, fmt.Errorf("integer value is nil")
	}

	var i64 int64
	switch t := v.(type) {
	case int:
		i64 = int64(t)
	case int8:
		i64 = int64(t)
	case int16:
		i64 = int64(t)
	case int32:
		i64 = int64(t)
	case int64:
		i64 = t
	case uint:
		if uint64(t) > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("integer %d is out of int64 range", t)
		}
		i64 = int64(t)
	case uint8:
		i64 = int64(t)
	case uint16:
		i64 = int64(t)
	case uint32:
		i64 = int64(t)
	case uint64:
		if t > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("integer %d is out of int64 range", t)
		}
		i64 = int64(t)
	case float32:
		f := float64(t)
		if math.IsNaN(f) || math.IsInf(f, 0) || math.Trunc(f) != f {
			return 0, fmt.Errorf("integer value required")
		}
		i64 = int64(f)
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || math.Trunc(t) != t {
			return 0, fmt.Errorf("integer value required")
		}
		const maxInt64Float = float64(^uint64(0) >> 1)
		const minInt64Float = -maxInt64Float - 1
		if t < minInt64Float || t > maxInt64Float {
			return 0, fmt.Errorf("integer %v is out of int64 range", t)
		}
		i64 = int64(t)
	default:
		return 0, fmt.Errorf("integer type required, got %T", v)
	}

	const maxInt = int64(^uint(0) >> 1)
	const minInt = -maxInt - 1
	if i64 < minInt || i64 > maxInt {
		return 0, fmt.Errorf("integer %d overflows int", i64)
	}
	return int(i64), nil
}

func strictString(v any, name string) (string, error) {
	v = indirect(v)
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string, got %T", name, v)
	}
	return s, nil
}

type numericFormatPattern struct {
	prefix       string
	suffix       string
	minIntDigits int
	minFrac      int
	maxFrac      int
	useGrouping  bool
}

func parseNumericFormatPattern(pattern string) (numericFormatPattern, error) {
	first := strings.IndexAny(pattern, "0#")
	last := strings.LastIndexAny(pattern, "0#")
	if first < 0 || last < 0 || first > last {
		return numericFormatPattern{}, fmt.Errorf("unsupported format pattern %q", pattern)
	}

	core := pattern[first : last+1]
	for _, r := range core {
		switch r {
		case '0', '#', ',', '.':
		default:
			return numericFormatPattern{}, fmt.Errorf("unsupported numeric format pattern %q", pattern)
		}
	}

	if strings.Count(core, ".") > 1 {
		return numericFormatPattern{}, fmt.Errorf("unsupported numeric format pattern %q", pattern)
	}

	parts := strings.Split(core, ".")
	intPart := parts[0]
	if intPart == "" {
		return numericFormatPattern{}, fmt.Errorf("unsupported numeric format pattern %q", pattern)
	}

	if strings.Contains(intPart, ",") {
		groups := strings.Split(intPart, ",")
		if len(groups) < 2 {
			return numericFormatPattern{}, fmt.Errorf("unsupported numeric format pattern %q", pattern)
		}
		for i, group := range groups {
			if group == "" {
				return numericFormatPattern{}, fmt.Errorf("unsupported numeric format pattern %q", pattern)
			}
			if i > 0 && len(group) != 3 {
				return numericFormatPattern{}, fmt.Errorf("unsupported numeric format pattern %q", pattern)
			}
		}
	}

	fractionPart := ""
	if len(parts) == 2 {
		fractionPart = parts[1]
		if fractionPart == "" || strings.Contains(fractionPart, ",") {
			return numericFormatPattern{}, fmt.Errorf("unsupported numeric format pattern %q", pattern)
		}
	}

	return numericFormatPattern{
		prefix:       pattern[:first],
		suffix:       pattern[last+1:],
		minIntDigits: strings.Count(intPart, "0"),
		minFrac:      strings.Count(fractionPart, "0"),
		maxFrac:      len(fractionPart),
		useGrouping:  strings.Contains(intPart, ","),
	}, nil
}

func groupThousands(s string) string {
	if len(s) <= 3 {
		return s
	}
	head := len(s) % 3
	if head == 0 {
		head = 3
	}
	var b strings.Builder
	b.WriteString(s[:head])
	for i := head; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func formatNumericWithPattern(v float64, pattern numericFormatPattern) string {
	negative := v < 0
	if negative {
		v = -v
	}

	var rounded float64
	if pattern.maxFrac == 0 {
		rounded = math.Round(v)
	} else {
		pow := math.Pow10(pattern.maxFrac)
		rounded = math.Round(v*pow) / pow
	}
	if rounded == 0 {
		negative = false
	}

	numberText := strconv.FormatFloat(rounded, 'f', pattern.maxFrac, 64)
	intPart := numberText
	fracPart := ""
	if dot := strings.IndexByte(numberText, '.'); dot >= 0 {
		intPart = numberText[:dot]
		fracPart = numberText[dot+1:]
	}

	for len(intPart) < pattern.minIntDigits {
		intPart = "0" + intPart
	}
	if pattern.useGrouping {
		intPart = groupThousands(intPart)
	}
	for len(fracPart) > pattern.minFrac && strings.HasSuffix(fracPart, "0") {
		fracPart = fracPart[:len(fracPart)-1]
	}

	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	b.WriteString(pattern.prefix)
	b.WriteString(intPart)
	if fracPart != "" {
		b.WriteByte('.')
		b.WriteString(fracPart)
	}
	b.WriteString(pattern.suffix)
	return b.String()
}

func parseDatetimeLayout(pattern string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("unsupported format pattern %q", pattern)
	}

	var b strings.Builder
	hasToken := false

	for i := 0; i < len(pattern); {
		if pattern[i] == '\'' {
			i++
			closed := false
			for i < len(pattern) {
				if pattern[i] == '\'' {
					if i+1 < len(pattern) && pattern[i+1] == '\'' {
						b.WriteByte('\'')
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				b.WriteByte(pattern[i])
				i++
			}
			if !closed {
				return "", fmt.Errorf("unterminated quoted literal in pattern %q", pattern)
			}
			continue
		}

		switch {
		case strings.HasPrefix(pattern[i:], "yyyy"):
			b.WriteString("2006")
			i += 4
			hasToken = true
		case strings.HasPrefix(pattern[i:], "MM"):
			b.WriteString("01")
			i += 2
			hasToken = true
		case strings.HasPrefix(pattern[i:], "dd"):
			b.WriteString("02")
			i += 2
			hasToken = true
		case strings.HasPrefix(pattern[i:], "HH"):
			b.WriteString("15")
			i += 2
			hasToken = true
		case strings.HasPrefix(pattern[i:], "mm"):
			b.WriteString("04")
			i += 2
			hasToken = true
		case strings.HasPrefix(pattern[i:], "ss"):
			b.WriteString("05")
			i += 2
			hasToken = true
		default:
			ch := pattern[i]
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				return "", fmt.Errorf("unsupported datetime token in pattern %q", pattern)
			}
			b.WriteByte(ch)
			i++
		}
	}

	if !hasToken {
		return "", fmt.Errorf("unsupported format pattern %q", pattern)
	}
	return b.String(), nil
}

func formatValueWithPattern(value any, pattern string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("format pattern cannot be empty")
	}

	if strings.ContainsAny(pattern, "0#") {
		numericPattern, err := parseNumericFormatPattern(pattern)
		if err != nil {
			return "", err
		}
		n, err := toNumber(value)
		if err != nil {
			return "", fmt.Errorf("numeric format requires a numeric value: %w", err)
		}

		switch t := n.(type) {
		case int64:
			return formatNumericWithPattern(float64(t), numericPattern), nil
		case float64:
			return formatNumericWithPattern(t, numericPattern), nil
		default:
			return "", fmt.Errorf("numeric format requires a numeric value")
		}
	}

	layout, err := parseDatetimeLayout(pattern)
	if err != nil {
		return "", err
	}

	value = indirect(value)
	t, ok := value.(time.Time)
	if !ok {
		return "", fmt.Errorf("datetime format requires time.Time value, got %T", value)
	}
	return t.UTC().Format(layout), nil
}

func numberToDatetime(v any) (time.Time, error) {
	n, err := toNumber(v)
	if err != nil {
		return time.Time{}, err
	}
	ms, ok := n.(int64)
	if !ok {
		return time.Time{}, fmt.Errorf("epoch milliseconds must be an integer")
	}
	return time.Unix(0, ms*int64(time.Millisecond)).UTC(), nil
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
				return len(t) > 0
			case bool:
				return true
			case time.Time:
				return true
			}

			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
				return rv.Len() > 0
			case reflect.Bool,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
				reflect.Float32, reflect.Float64:
				return true
			}
			return false
		},
		"contains": func(container any, needle any) (bool, error) {
			s, err := strictString(container, "contains container")
			if err != nil {
				return false, err
			}
			needleText, err := strictString(needle, "contains needle")
			if err != nil {
				return false, err
			}
			return strings.Contains(s, needleText), nil
		},
		"substring": func(v any, a any, b ...any) (string, error) {
			base, err := strictString(v, "substring value")
			if err != nil {
				return "", err
			}
			runes := []rune(base)
			start, err := toInt(a)
			if err != nil {
				return "", fmt.Errorf("substring start must be an integer: %w", err)
			}
			end := len(runes)
			if len(b) > 0 {
				end, err = toInt(b[0])
				if err != nil {
					return "", fmt.Errorf("substring end must be an integer: %w", err)
				}
			}

			if start < 0 || end < 0 || start > len(runes) || end > len(runes) {
				return "", fmt.Errorf("substring indices are out of bounds")
			}
			if start > end {
				return "", fmt.Errorf("substring start cannot be greater than end")
			}
			return string(runes[start:end]), nil
		},
		"indexOf": func(v any, sub any, start ...any) (int, error) {
			base, err := strictString(v, "indexOf value")
			if err != nil {
				return -1, err
			}
			needleText, err := strictString(sub, "indexOf needle")
			if err != nil {
				return -1, err
			}
			haystack := []rune(base)
			needle := []rune(needleText)

			offset := 0
			if len(start) > 0 {
				offset, err = toInt(start[0])
				if err != nil {
					return -1, fmt.Errorf("indexOf start must be an integer: %w", err)
				}
			}
			if offset < 0 {
				offset = 0
			}
			if offset > len(haystack) {
				return -1, nil
			}
			if len(needle) == 0 {
				return offset, nil
			}
			if len(needle) > len(haystack) {
				return -1, nil
			}
			for i := offset; i <= len(haystack)-len(needle); i++ {
				if string(haystack[i:i+len(needle)]) == string(needle) {
					return i, nil
				}
			}
			return -1, nil
		},
		"trim": func(v any) (string, error) {
			s, err := strictString(v, "trim value")
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(s), nil
		},
		"toNumber":         toNumber,
		"numberToDatetime": numberToDatetime,
		"toString": func(args ...any) (string, error) {
			if len(args) == 0 {
				return "", nil
			}

			value := args[0]
			formatArgs := args[1:]
			if len(formatArgs) == 0 {
				value = indirect(value)
				if value == nil {
					return "", nil
				}
				return fmt.Sprint(value), nil
			}

			if len(formatArgs) > 2 {
				return "", fmt.Errorf("toString expects at most two format arguments")
			}

			pattern, ok := indirect(formatArgs[0]).(string)
			if !ok {
				return "", fmt.Errorf("toString format argument 1 must be a string")
			}
			if pattern == "" {
				return "", fmt.Errorf("toString format argument 1 cannot be empty")
			}

			if len(formatArgs) == 2 {
				locale, ok := indirect(formatArgs[1]).(string)
				if !ok {
					return "", fmt.Errorf("toString format argument 2 must be a string")
				}
				if locale == "" {
					return "", fmt.Errorf("toString format argument 2 cannot be empty")
				}
				if locale != "UTC" && locale != "en_US" {
					return "", fmt.Errorf("unsupported toString locale/timezone %q", locale)
				}
			}

			return formatValueWithPattern(value, pattern)
		},
		"safeAccess": func(root any, path ...any) any {
			current := root
			for _, segment := range path {
				current = indirect(current)
				if current == nil {
					return nil
				}

				rv := reflect.ValueOf(current)
				switch rv.Kind() {
				case reflect.Map:
					key, ok := mapKeyFrom(segment, rv.Type().Key())
					if !ok {
						return nil
					}
					next := rv.MapIndex(key)
					if !next.IsValid() {
						return nil
					}
					current = next.Interface()
				case reflect.Struct:
					fieldName, ok := indirect(segment).(string)
					if !ok || fieldName == "" {
						return nil
					}
					field := rv.FieldByName(fieldName)
					if !field.IsValid() || !field.CanInterface() {
						return nil
					}
					current = field.Interface()
				case reflect.Slice, reflect.Array:
					idx, err := toInt(segment)
					if err != nil || idx < 0 || idx >= rv.Len() {
						return nil
					}
					current = rv.Index(idx).Interface()
				default:
					return nil
				}
			}
			return current
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
