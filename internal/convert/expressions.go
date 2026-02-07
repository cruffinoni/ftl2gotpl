package convert

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var numberLiteralRe = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// expressionMapper rewrites FreeMarker expressions to Go template expressions.
type expressionMapper struct {
	locals  map[string]struct{}
	helpers map[string]struct{}
}

func newExpressionMapper(locals map[string]struct{}) *expressionMapper {
	cp := make(map[string]struct{}, len(locals))
	for k := range locals {
		cp[k] = struct{}{}
	}
	return &expressionMapper{
		locals:  cp,
		helpers: map[string]struct{}{},
	}
}

// setLocal records a variable for future local identifier resolution.
func (m *expressionMapper) setLocal(name string) {
	m.locals[name] = struct{}{}
}

func splitPath(expr string) (string, string, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", "", false
	}
	i := 0
	for i < len(expr) {
		ch := expr[i]
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' {
			i++
			continue
		}
		break
	}
	if i == 0 {
		return "", "", false
	}
	first := expr[:i]
	rest := expr[i:]
	return first, rest, true
}

func findMatchingBracket(s string, open int) int {
	depth := 0
	quote := byte(0)
	escaped := false
	for i := open; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == '[' {
			depth++
			continue
		}
		if ch == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// resolveIdentifier maps an identifier path to dot or local variable syntax.
func (m *expressionMapper) resolveIdentifier(expr string) (string, error) {
	if expr == "." {
		return ".", nil
	}
	if strings.HasPrefix(expr, "$") {
		return expr, nil
	}
	first, rest, ok := splitPath(expr)
	if !ok {
		return "", fmt.Errorf("unsupported identifier expression %q", expr)
	}
	current := "." + first
	if _, exists := m.locals[first]; exists {
		current = "$" + first
	}
	for i := 0; i < len(rest); {
		switch rest[i] {
		case '.':
			j := i + 1
			for j < len(rest) {
				ch := rest[j]
				if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' {
					j++
					continue
				}
				break
			}
			if j == i+1 {
				return "", fmt.Errorf("unsupported identifier expression %q", expr)
			}
			current = wrap(current) + rest[i:j]
			i = j
		case '[':
			end := findMatchingBracket(rest, i)
			if end < 0 {
				return "", fmt.Errorf("unsupported identifier expression %q", expr)
			}
			keyExpr := strings.TrimSpace(rest[i+1 : end])
			if keyExpr == "" {
				return "", fmt.Errorf("unsupported identifier expression %q", expr)
			}
			mappedKey, err := m.mapExpr(keyExpr)
			if err != nil {
				return "", err
			}
			current = "index " + wrap(current) + " " + wrap(mappedKey)
			i = end + 1
		default:
			return "", fmt.Errorf("unsupported identifier expression %q", expr)
		}
	}
	return current, nil
}

// helperList returns sorted helper names required by mapped expressions.
func (m *expressionMapper) helperList() []string {
	out := make([]string, 0, len(m.helpers))
	for h := range m.helpers {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

type builtinCall struct {
	name string
	args string
}

// firstTopLevelQuestion returns the first top-level question-mark index.
func firstTopLevelQuestion(expr string) int {
	depth := 0
	quote := byte(0)
	escaped := false
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '?':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findMatchingParen(s string, open int) int {
	depth := 0
	quote := byte(0)
	escaped := false
	for i := open; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// parseBuiltinChain parses a top-level FreeMarker builtin chain expression.
func parseBuiltinChain(expr string) (string, []builtinCall, bool) {
	pos := firstTopLevelQuestion(expr)
	if pos < 0 {
		return "", nil, false
	}
	base := strings.TrimSpace(expr[:pos])
	if base == "" {
		return "", nil, false
	}

	var calls []builtinCall
	i := pos
	for i < len(expr) {
		if expr[i] != '?' {
			rest := strings.TrimSpace(expr[i:])
			if rest == "" {
				break
			}
			return "", nil, false
		}
		i++
		start := i
		for i < len(expr) && (unicode.IsLetter(rune(expr[i])) || unicode.IsDigit(rune(expr[i])) || expr[i] == '_') {
			i++
		}
		if i == start {
			return "", nil, false
		}
		name := strings.ToLower(expr[start:i])
		args := ""
		if i < len(expr) && expr[i] == '(' {
			end := findMatchingParen(expr, i)
			if end < 0 {
				return "", nil, false
			}
			args = strings.TrimSpace(expr[i+1 : end])
			i = end + 1
		}
		calls = append(calls, builtinCall{name: name, args: args})
	}

	return base, calls, true
}

func parseFunctionCall(expr string) (string, []string, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil, false
	}

	name, rest, ok := splitPath(expr)
	if !ok || strings.TrimSpace(name) == "" {
		return "", nil, false
	}
	if strings.TrimSpace(rest) == "" || rest[0] != '(' {
		return "", nil, false
	}

	end := findMatchingParen(rest, 0)
	if end < 0 || end != len(rest)-1 {
		return "", nil, false
	}
	rawArgs := strings.TrimSpace(rest[1:end])
	if rawArgs == "" {
		return name, nil, true
	}
	return name, splitArgs(rawArgs), true
}

func splitArgs(s string) []string {
	var out []string
	start := 0
	depth := 0
	quote := byte(0)
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(s[start:]))
	return out
}

func splitTopLevel(expr string, sep string) []string {
	var parts []string
	start := 0
	depth := 0
	quote := byte(0)
	escaped := false

	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && strings.HasPrefix(expr[i:], sep) {
			parts = append(parts, strings.TrimSpace(expr[start:i]))
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	if start == 0 {
		return nil
	}
	parts = append(parts, strings.TrimSpace(expr[start:]))
	return parts
}

func splitTopLevelCompare(expr string) (string, string, string, bool) {
	for _, op := range []string{"==", "!=", ">=", "<=", ">", "<", "="} {
		parts := splitTopLevel(expr, op)
		if len(parts) == 2 {
			if op == "=" && (strings.Contains(parts[0], "<") || strings.Contains(parts[0], ">")) {
				continue
			}
			return parts[0], parts[1], op, true
		}
	}
	return "", "", "", false
}

func hasTopLevelArithmetic(expr string) bool {
	depth := 0
	quote := byte(0)
	escaped := false
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '+', '*', '/':
			if depth == 0 {
				return true
			}
		case '-':
			if depth == 0 {
				prev := byte(0)
				next := byte(0)
				if i > 0 {
					prev = expr[i-1]
				}
				if i+1 < len(expr) {
					next = expr[i+1]
				}
				if (unicode.IsLetter(rune(prev)) || unicode.IsDigit(rune(prev)) || prev == ')' || prev == '"') &&
					(unicode.IsLetter(rune(next)) || unicode.IsDigit(rune(next)) || next == '(' || next == '"') {
					return true
				}
			}
		}
	}
	return false
}

func splitTopLevelDefault(expr string) (string, string, bool) {
	depth := 0
	quote := byte(0)
	escaped := false
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '!':
			if depth == 0 {
				next := byte(0)
				if i+1 < len(expr) {
					next = expr[i+1]
				}
				if next == '=' {
					continue
				}
				left := strings.TrimSpace(expr[:i])
				right := strings.TrimSpace(expr[i+1:])
				if left != "" && right != "" {
					return left, right, true
				}
			}
		}
	}
	return "", "", false
}

func stripOuterParen(expr string) (string, bool) {
	if !strings.HasPrefix(expr, "(") || !strings.HasSuffix(expr, ")") {
		return "", false
	}
	depth := 0
	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i < len(expr)-1 {
				return "", false
			}
		}
	}
	if depth != 0 {
		return "", false
	}
	return strings.TrimSpace(expr[1 : len(expr)-1]), true
}

func isLiteral(expr string) bool {
	expr = strings.TrimSpace(expr)
	if expr == "true" || expr == "false" || expr == "nil" || expr == "null" {
		return true
	}
	if numberLiteralRe.MatchString(expr) {
		return true
	}
	if len(expr) >= 2 {
		if (expr[0] == '"' && expr[len(expr)-1] == '"') || (expr[0] == '\'' && expr[len(expr)-1] == '\'') {
			return true
		}
	}
	return false
}

func unescapeSingleQuotedString(expr string) (string, error) {
	if len(expr) < 2 || expr[0] != '\'' || expr[len(expr)-1] != '\'' {
		return "", fmt.Errorf("not a single-quoted literal: %q", expr)
	}
	inner := expr[1 : len(expr)-1]
	var b strings.Builder
	escaped := false
	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if escaped {
			switch ch {
			case '\\', '\'', '"':
				b.WriteByte(ch)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			default:
				return "", fmt.Errorf("unsupported escape sequence \\%c in literal %q", ch, expr)
			}
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		b.WriteByte(ch)
	}
	if escaped {
		return "", fmt.Errorf("unterminated escape in literal %q", expr)
	}
	return b.String(), nil
}

func normalizeStringLiteral(expr string) (string, bool, error) {
	if len(expr) < 2 {
		return expr, false, nil
	}
	if expr[0] == '"' && expr[len(expr)-1] == '"' {
		return expr, true, nil
	}
	if expr[0] == '\'' && expr[len(expr)-1] == '\'' {
		unescaped, err := unescapeSingleQuotedString(expr)
		if err != nil {
			return "", true, err
		}
		return strconv.Quote(unescaped), true, nil
	}
	return expr, false, nil
}

func wrap(expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return expr
	}
	if strings.ContainsAny(expr, " \t\n") {
		return "(" + expr + ")"
	}
	return expr
}

func joinWrapped(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, wrap(p))
	}
	return strings.Join(out, " ")
}

// mapExpr converts one FreeMarker expression into its Go template equivalent.
func (m *expressionMapper) mapExpr(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", fmt.Errorf("empty expression")
	}

	if strings.HasPrefix(expr, "!") && !strings.HasPrefix(expr, "!=") {
		inner, err := m.mapExpr(strings.TrimSpace(expr[1:]))
		if err != nil {
			return "", err
		}
		m.helpers["not"] = struct{}{}
		return "not " + wrap(inner), nil
	}

	if lhs, rhs, ok := splitTopLevelDefault(expr); ok {
		left, err := m.mapExpr(lhs)
		if err != nil {
			return "", err
		}
		right, err := m.mapExpr(rhs)
		if err != nil {
			return "", err
		}
		m.helpers["default"] = struct{}{}
		return "default " + wrap(right) + " " + wrap(left), nil
	}

	if strings.HasSuffix(expr, "??") {
		base := strings.TrimSpace(strings.TrimSuffix(expr, "??"))
		mapped, err := m.mapExpr(base)
		if err != nil {
			return "", err
		}
		m.helpers["exists"] = struct{}{}
		return "exists " + wrap(mapped), nil
	}

	if parts := splitTopLevel(expr, "||"); len(parts) > 1 {
		m.helpers["or"] = struct{}{}
		mapped := make([]string, 0, len(parts))
		for _, p := range parts {
			sub, err := m.mapExpr(p)
			if err != nil {
				return "", err
			}
			mapped = append(mapped, wrap(sub))
		}
		return "or " + strings.Join(mapped, " "), nil
	}
	if parts := splitTopLevel(expr, "&&"); len(parts) > 1 {
		m.helpers["and"] = struct{}{}
		mapped := make([]string, 0, len(parts))
		for _, p := range parts {
			sub, err := m.mapExpr(p)
			if err != nil {
				return "", err
			}
			mapped = append(mapped, wrap(sub))
		}
		return "and " + strings.Join(mapped, " "), nil
	}

	if lhs, rhs, op, ok := splitTopLevelCompare(expr); ok {
		left, err := m.mapExpr(lhs)
		if err != nil {
			return "", err
		}
		right, err := m.mapExpr(rhs)
		if err != nil {
			return "", err
		}
		switch op {
		case "==":
			return "eq " + wrap(left) + " " + wrap(right), nil
		case "!=":
			return "ne " + wrap(left) + " " + wrap(right), nil
		case "=":
			return "eq " + wrap(left) + " " + wrap(right), nil
		case ">":
			return "gt " + wrap(left) + " " + wrap(right), nil
		case "<":
			return "lt " + wrap(left) + " " + wrap(right), nil
		case ">=":
			return "ge " + wrap(left) + " " + wrap(right), nil
		case "<=":
			return "le " + wrap(left) + " " + wrap(right), nil
		}
	}

	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		inner, ok := stripOuterParen(expr)
		if ok {
			mapped, err := m.mapExpr(inner)
			if err != nil {
				return "", err
			}
			return "(" + mapped + ")", nil
		}
	}

	if base, calls, ok := parseBuiltinChain(expr); ok {
		mapped, err := m.mapExpr(base)
		if err != nil {
			return "", err
		}
		current := mapped
		for _, call := range calls {
			args := make([]string, 0)
			rawArgs := strings.TrimSpace(call.args)
			if rawArgs != "" {
				parts := splitArgs(rawArgs)
				args = make([]string, 0, len(parts))
				for _, p := range parts {
					sub, mapErr := m.mapExpr(p)
					if mapErr != nil {
						return "", mapErr
					}
					args = append(args, sub)
				}
			}
			switch call.name {
			case "size":
				current = "len " + wrap(current)
			case "has_content":
				m.helpers["hasContent"] = struct{}{}
				current = "hasContent " + wrap(current)
			case "contains":
				if len(args) != 1 {
					return "", fmt.Errorf("?contains expects one argument")
				}
				m.helpers["contains"] = struct{}{}
				current = "contains " + wrap(current) + " " + wrap(args[0])
			case "substring":
				if len(args) < 1 || len(args) > 2 {
					return "", fmt.Errorf("?substring expects one or two arguments")
				}
				m.helpers["substring"] = struct{}{}
				current = "substring " + wrap(current) + " " + wrap(args[0])
				if len(args) == 2 {
					current += " " + wrap(args[1])
				}
			case "index_of":
				if len(args) < 1 || len(args) > 2 {
					return "", fmt.Errorf("?index_of expects one or two arguments")
				}
				m.helpers["indexOf"] = struct{}{}
				current = "indexOf " + wrap(current) + " " + wrap(args[0])
				if len(args) == 2 {
					current += " " + wrap(args[1])
				}
			case "trim":
				m.helpers["trim"] = struct{}{}
				current = "trim " + wrap(current)
			case "index":
				if len(args) != 0 {
					return "", fmt.Errorf("?index expects no arguments")
				}
				if !strings.HasPrefix(current, "$") || strings.ContainsAny(current[1:], ".[") {
					return "", fmt.Errorf("?index is only supported on loop item variables")
				}
				indexVar := strings.TrimPrefix(current, "$") + "_index"
				if _, ok := m.locals[indexVar]; !ok {
					return "", fmt.Errorf("?index is only supported on loop item variables")
				}
				current = "$" + indexVar
			case "number":
				m.helpers["toNumber"] = struct{}{}
				current = "toNumber " + wrap(current)
			case "number_to_datetime":
				m.helpers["numberToDatetime"] = struct{}{}
				current = "numberToDatetime " + wrap(current)
			case "string":
				m.helpers["toString"] = struct{}{}
				if len(args) == 0 {
					current = "toString " + wrap(current)
				} else {
					current = "toString " + wrap(current) + " " + joinWrapped(args)
				}
			case "no_esc":
				m.helpers["safeHTML"] = struct{}{}
				current = "safeHTML " + wrap(current)
			default:
				return "", fmt.Errorf("unsupported builtin ?%s", call.name)
			}
		}
		return current, nil
	}

	if name, rawArgs, ok := parseFunctionCall(expr); ok {
		mappedArgs := make([]string, 0, len(rawArgs))
		for _, rawArg := range rawArgs {
			sub, err := m.mapExpr(rawArg)
			if err != nil {
				return "", err
			}
			mappedArgs = append(mappedArgs, sub)
		}

		switch name {
		case "formatPrice":
			if len(mappedArgs) != 1 {
				return "", fmt.Errorf("formatPrice expects one argument")
			}
			m.helpers["formatPrice"] = struct{}{}
			return "formatPrice " + wrap(mappedArgs[0]), nil
		default:
			return "", fmt.Errorf("unsupported function call %q", name)
		}
	}

	if isLiteral(expr) {
		if normalized, isString, err := normalizeStringLiteral(expr); err != nil {
			return "", err
		} else if isString {
			return normalized, nil
		}
		return expr, nil
	}

	if hasTopLevelArithmetic(expr) {
		return "", fmt.Errorf("unsupported arithmetic expression %q", expr)
	}

	return m.resolveIdentifier(expr)
}
