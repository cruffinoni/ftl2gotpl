package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/cruffinoni/ftl2gotpl/internal/diagnostics"
)

// TokenKind describes the syntactic category emitted by the lexer.
type TokenKind string

const (
	TokenText          TokenKind = "text"
	TokenDirective     TokenKind = "directive"
	TokenInterpolation TokenKind = "interpolation"
	TokenMacroCall     TokenKind = "macro_call"
)

// Token represents one lexical unit with source coordinates and metadata.
type Token struct {
	Kind     TokenKind
	PosLine  int
	PosCol   int
	Raw      string
	Value    string
	Name     string
	Args     string
	Closing  bool
	AltStyle bool
}

// scanner performs streaming lexical analysis over one template source string.
type scanner struct {
	src    string
	file   string
	index  int
	line   int
	column int
}

// consumeText consumes literal text until the next FreeMarker construct.
func (s *scanner) consumeText() Token {
	startLine, startCol := s.line, s.column
	start := s.index
	for !s.eof() {
		if s.hasPrefix("<#--") || s.hasPrefix("${") || s.hasPrefix("#{") || s.hasPrefix("<#") || s.hasPrefix("</#") || s.hasPrefix("<@") {
			break
		}
		s.advanceByString(string(s.src[s.index]))
	}
	text := s.src[start:s.index]
	return Token{
		Kind:    TokenText,
		PosLine: startLine,
		PosCol:  startCol,
		Value:   text,
		Raw:     text,
	}
}

// consumeComment skips FreeMarker comments (<#-- ... -->).
func (s *scanner) consumeComment() error {
	start := s.index
	idx := strings.Index(s.src[start:], "-->")
	if idx < 0 {
		return diagnostics.New("LEX_UNCLOSED_COMMENT", s.file, s.line, s.column, "unclosed FreeMarker comment", "")
	}
	end := start + idx + len("-->")
	s.advanceByString(s.src[start:end])
	return nil
}

// consumeInterpolation consumes ${...} and #{...} blocks with nesting.
func (s *scanner) consumeInterpolation() (Token, error) {
	startLine, startCol := s.line, s.column
	start := s.index
	alt := s.hasPrefix("#{")

	// consume opener
	s.advanceByString(s.src[s.index : s.index+2])
	depth := 1
	inQuote := byte(0)
	escaped := false

	for !s.eof() {
		ch := s.src[s.index]
		s.advanceByString(string(ch))

		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			inQuote = ch
			continue
		}
		if ch == '{' {
			depth++
			continue
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				raw := s.src[start:s.index]
				expr := strings.TrimSpace(raw[2 : len(raw)-1])
				return Token{
					Kind:     TokenInterpolation,
					PosLine:  startLine,
					PosCol:   startCol,
					Raw:      raw,
					Value:    expr,
					AltStyle: alt,
				}, nil
			}
		}
	}

	return Token{}, diagnostics.New("LEX_UNCLOSED_INTERPOLATION", s.file, startLine, startCol, "unclosed interpolation", "")
}

// splitNameArgs splits a tag body into name and trailing arguments.
func splitNameArgs(body string) (string, string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ""
	}
	i := 0
	for i < len(body) {
		r := rune(body[i])
		if unicode.IsSpace(r) {
			break
		}
		i++
	}
	name := strings.TrimSpace(body[:i])
	args := strings.TrimSpace(body[i:])
	return name, args
}

// parseTagToken interprets a raw tag and extracts normalized token fields.
func parseTagToken(raw string, line int, col int, file string) (Token, error) {
	if strings.HasPrefix(raw, "<@") {
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw, "<@"), ">"))
		name, args := splitNameArgs(body)
		if name == "" {
			return Token{}, diagnostics.New("LEX_INVALID_MACRO_CALL", file, line, col, "invalid macro call", raw)
		}
		return Token{
			Kind:    TokenMacroCall,
			PosLine: line,
			PosCol:  col,
			Raw:     raw,
			Name:    name,
			Args:    args,
		}, nil
	}

	if strings.HasPrefix(raw, "<#") || strings.HasPrefix(raw, "</#") {
		body := strings.TrimSuffix(strings.TrimPrefix(raw, "<"), ">")
		body = strings.TrimSpace(body)

		closing := false
		if strings.HasPrefix(body, "/") {
			closing = true
			body = strings.TrimSpace(strings.TrimPrefix(body, "/"))
		}
		body = strings.TrimPrefix(body, "#")
		body = strings.TrimSpace(body)

		name, args := splitNameArgs(body)
		if name == "" {
			return Token{}, diagnostics.New("LEX_INVALID_DIRECTIVE", file, line, col, "invalid directive tag", raw)
		}

		return Token{
			Kind:    TokenDirective,
			PosLine: line,
			PosCol:  col,
			Raw:     raw,
			Name:    strings.ToLower(name),
			Args:    args,
			Closing: closing,
		}, nil
	}

	return Token{}, diagnostics.New("LEX_UNKNOWN_TAG", file, line, col, fmt.Sprintf("unknown tag kind %q", raw), raw)
}

// consumeTag consumes directive and macro-call tags until >.
func (s *scanner) consumeTag() (Token, error) {
	startLine, startCol := s.line, s.column
	start := s.index
	inQuote := byte(0)
	escaped := false

	for !s.eof() {
		ch := s.src[s.index]
		s.advanceByString(string(ch))

		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inQuote = ch
			continue
		}
		if ch == '>' {
			raw := s.src[start:s.index]
			return parseTagToken(raw, startLine, startCol, s.file)
		}
	}

	return Token{}, diagnostics.New("LEX_UNCLOSED_TAG", s.file, startLine, startCol, "unclosed tag", "")
}

// hasPrefix reports whether remaining source starts with prefix.
func (s *scanner) hasPrefix(prefix string) bool {
	return strings.HasPrefix(s.src[s.index:], prefix)
}

// eof reports whether the scanner has consumed all input.
func (s *scanner) eof() bool {
	return s.index >= len(s.src)
}

// advanceByString updates index and line/column counters for a fragment.
func (s *scanner) advanceByString(fragment string) {
	for _, r := range fragment {
		s.index += len(string(r))
		if r == '\n' {
			s.line++
			s.column = 1
		} else {
			s.column++
		}
	}
}

// Lex tokenizes FreeMarker source into a sequence consumed by the parser.
func Lex(file string, src string) ([]Token, error) {
	s := &scanner{
		src:    src,
		file:   file,
		line:   1,
		column: 1,
	}
	var tokens []Token

	for !s.eof() {
		switch {
		case s.hasPrefix("<#--"):
			if err := s.consumeComment(); err != nil {
				return nil, err
			}
		case s.hasPrefix("${") || s.hasPrefix("#{"):
			tok, err := s.consumeInterpolation()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		case s.hasPrefix("<#") || s.hasPrefix("</#") || s.hasPrefix("<@"):
			tok, err := s.consumeTag()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		default:
			tok := s.consumeText()
			if tok.Value != "" {
				tokens = append(tokens, tok)
			}
		}
	}

	return tokens, nil
}
