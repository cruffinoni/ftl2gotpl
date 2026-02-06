package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cruffinoni/ftl2gotpl/internal/ast"
	"github.com/cruffinoni/ftl2gotpl/internal/diagnostics"
	"github.com/cruffinoni/ftl2gotpl/internal/lexer"
)

var (
	listDirectiveRe   = regexp.MustCompile(`(?is)^(.*?)\s+as\s+([A-Za-z_][A-Za-z0-9_]*)$`)
	assignDirectiveRe = regexp.MustCompile(`(?is)^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+)$`)
)

// state stores parser progress while consuming lexer tokens.
type state struct {
	file   string
	tokens []lexer.Token
	index  int
}

// directiveKey normalizes directive tokens for stopper lookups.
func directiveKey(tok lexer.Token) string {
	if tok.Closing {
		return "close:" + tok.Name
	}
	return "dir:" + tok.Name
}

// parseAssign parses assign/local directives into AssignNode.
func parseAssign(file string, tok lexer.Token, local bool) (ast.Node, error) {
	match := assignDirectiveRe.FindStringSubmatch(strings.TrimSpace(tok.Args))
	if len(match) != 3 {
		return nil, diagnostics.New("PARSE_INVALID_ASSIGN", file, tok.PosLine, tok.PosCol, "assign/local must be '<#assign x = expr>'", tok.Raw)
	}
	name := strings.TrimSpace(match[1])
	expr := strings.TrimSpace(match[2])
	return ast.AssignNode{
		Position: ast.Position{Line: tok.PosLine, Column: tok.PosCol},
		Name:     name,
		Expr:     expr,
		Local:    local,
	}, nil
}

// Parse converts lexer tokens into an AST document.
func Parse(file string, tokens []lexer.Token) (ast.Document, error) {
	s := &state{
		file:   file,
		tokens: tokens,
	}

	nodes, stop, err := s.parseNodes(map[string]struct{}{})
	if err != nil {
		return ast.Document{}, err
	}
	if stop != nil {
		return ast.Document{}, diagnostics.New(
			"PARSE_UNEXPECTED_DIRECTIVE",
			file,
			stop.PosLine,
			stop.PosCol,
			fmt.Sprintf("unexpected directive %q", stop.Name),
			stop.Raw,
		)
	}
	return ast.Document{Nodes: nodes}, nil
}

// parseNodes parses nodes until EOF or until one stopper directive is reached.
func (s *state) parseNodes(stoppers map[string]struct{}) ([]ast.Node, *lexer.Token, error) {
	var nodes []ast.Node
	for s.index < len(s.tokens) {
		tok := s.tokens[s.index]
		s.index++

		switch tok.Kind {
		case lexer.TokenText:
			nodes = append(nodes, ast.TextNode{
				Position: ast.Position{Line: tok.PosLine, Column: tok.PosCol},
				Text:     tok.Value,
			})

		case lexer.TokenInterpolation:
			nodes = append(nodes, ast.InterpolationNode{
				Position: ast.Position{Line: tok.PosLine, Column: tok.PosCol},
				Expr:     strings.TrimSpace(tok.Value),
				AltStyle: tok.AltStyle,
			})

		case lexer.TokenMacroCall:
			nodes = append(nodes, ast.MacroCallNode{
				Position: ast.Position{Line: tok.PosLine, Column: tok.PosCol},
				Name:     tok.Name,
				Args:     tok.Args,
			})

		case lexer.TokenDirective:
			if _, ok := stoppers[directiveKey(tok)]; ok {
				return nodes, &tok, nil
			}
			if tok.Closing {
				return nil, nil, diagnostics.New(
					"PARSE_UNEXPECTED_CLOSING",
					s.file,
					tok.PosLine,
					tok.PosCol,
					fmt.Sprintf("unexpected closing directive </#%s>", tok.Name),
					tok.Raw,
				)
			}

			node, err := s.parseDirective(tok)
			if err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, node)
		}
	}

	return nodes, nil, nil
}

// parseDirective parses one non-closing directive token.
func (s *state) parseDirective(tok lexer.Token) (ast.Node, error) {
	pos := ast.Position{Line: tok.PosLine, Column: tok.PosCol}
	switch tok.Name {
	case "if":
		return s.parseIf(tok)
	case "list":
		return s.parseList(tok)
	case "assign":
		return parseAssign(s.file, tok, false)
	case "local":
		return parseAssign(s.file, tok, true)
	case "setting":
		return ast.SettingNode{Position: pos, Raw: strings.TrimSpace(tok.Args)}, nil
	case "ftl":
		return ast.SettingNode{Position: pos, Raw: "ftl " + strings.TrimSpace(tok.Args)}, nil
	case "function":
		return s.parseFunction(tok)
	case "return", "break":
		return ast.BareDirectiveNode{Position: pos, Name: tok.Name, Args: strings.TrimSpace(tok.Args)}, nil
	default:
		return nil, diagnostics.New(
			"PARSE_UNSUPPORTED_DIRECTIVE",
			s.file,
			tok.PosLine,
			tok.PosCol,
			fmt.Sprintf("unsupported directive <%s>", tok.Name),
			tok.Raw,
		)
	}
}

// parseIf parses <#if ...> including chained elseif and trailing else.
func (s *state) parseIf(tok lexer.Token) (ast.Node, error) {
	cond := strings.TrimSpace(tok.Args)
	if cond == "" {
		return nil, diagnostics.New("PARSE_INVALID_IF", s.file, tok.PosLine, tok.PosCol, "if directive requires a condition", tok.Raw)
	}
	pos := ast.Position{Line: tok.PosLine, Column: tok.PosCol}

	thenNodes, stop, err := s.parseNodes(map[string]struct{}{
		"dir:elseif": {},
		"dir:else":   {},
		"close:if":   {},
	})
	if err != nil {
		return nil, err
	}
	if stop == nil {
		return nil, diagnostics.New("PARSE_UNCLOSED_IF", s.file, tok.PosLine, tok.PosCol, "if directive not closed", tok.Raw)
	}

	node := ast.IfNode{
		Position: pos,
		Cond:     cond,
		Then:     thenNodes,
	}

	current := stop
	for current != nil && !current.Closing && current.Name == "elseif" {
		elseifCond := strings.TrimSpace(current.Args)
		if elseifCond == "" {
			return nil, diagnostics.New("PARSE_INVALID_ELSEIF", s.file, current.PosLine, current.PosCol, "elseif requires a condition", current.Raw)
		}
		body, nextStop, parseErr := s.parseNodes(map[string]struct{}{
			"dir:elseif": {},
			"dir:else":   {},
			"close:if":   {},
		})
		if parseErr != nil {
			return nil, parseErr
		}
		node.ElseIf = append(node.ElseIf, ast.IfElseIf{
			Position: ast.Position{Line: current.PosLine, Column: current.PosCol},
			Cond:     elseifCond,
			Body:     body,
		})
		current = nextStop
	}

	if current != nil && !current.Closing && current.Name == "else" {
		elseBody, nextStop, parseErr := s.parseNodes(map[string]struct{}{
			"close:if": {},
		})
		if parseErr != nil {
			return nil, parseErr
		}
		node.Else = elseBody
		current = nextStop
	}

	if current == nil || !current.Closing || current.Name != "if" {
		return nil, diagnostics.New("PARSE_UNCLOSED_IF", s.file, tok.PosLine, tok.PosCol, "if directive not closed", tok.Raw)
	}

	return node, nil
}

// parseList parses <#list seq as item> blocks.
func (s *state) parseList(tok lexer.Token) (ast.Node, error) {
	match := listDirectiveRe.FindStringSubmatch(strings.TrimSpace(tok.Args))
	if len(match) != 3 {
		return nil, diagnostics.New("PARSE_INVALID_LIST", s.file, tok.PosLine, tok.PosCol, "list directive must be '<#list expr as item>'", tok.Raw)
	}
	seqExpr := strings.TrimSpace(match[1])
	itemVar := strings.TrimSpace(match[2])
	if seqExpr == "" || itemVar == "" {
		return nil, diagnostics.New("PARSE_INVALID_LIST", s.file, tok.PosLine, tok.PosCol, "invalid list directive", tok.Raw)
	}

	body, stop, err := s.parseNodes(map[string]struct{}{
		"close:list": {},
	})
	if err != nil {
		return nil, err
	}
	if stop == nil || !stop.Closing || stop.Name != "list" {
		return nil, diagnostics.New("PARSE_UNCLOSED_LIST", s.file, tok.PosLine, tok.PosCol, "list directive not closed", tok.Raw)
	}

	return ast.ListNode{
		Position: ast.Position{Line: tok.PosLine, Column: tok.PosCol},
		SeqExpr:  seqExpr,
		ItemVar:  itemVar,
		Body:     body,
	}, nil
}

// parseFunction parses <#function ...> blocks.
func (s *state) parseFunction(tok lexer.Token) (ast.Node, error) {
	parts := strings.Fields(tok.Args)
	if len(parts) == 0 {
		return nil, diagnostics.New("PARSE_INVALID_FUNCTION", s.file, tok.PosLine, tok.PosCol, "function directive requires a name", tok.Raw)
	}
	name := parts[0]
	params := parts[1:]

	body, stop, err := s.parseNodes(map[string]struct{}{
		"close:function": {},
	})
	if err != nil {
		return nil, err
	}
	if stop == nil || !stop.Closing || stop.Name != "function" {
		return nil, diagnostics.New("PARSE_UNCLOSED_FUNCTION", s.file, tok.PosLine, tok.PosCol, "function directive not closed", tok.Raw)
	}

	return ast.FunctionNode{
		Position: ast.Position{Line: tok.PosLine, Column: tok.PosCol},
		Name:     name,
		Params:   params,
		Body:     body,
	}, nil
}
