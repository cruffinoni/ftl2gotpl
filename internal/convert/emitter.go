package convert

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cruffinoni/ftl2gotpl/internal/ast"
	"github.com/cruffinoni/ftl2gotpl/internal/diagnostics"
	"github.com/cruffinoni/ftl2gotpl/internal/lexer"
	"github.com/cruffinoni/ftl2gotpl/internal/parser"
)

// Result is the conversion output for one template file.
type Result struct {
	Output   string
	Helpers  []string
	Features []string
}

// Converter transforms FreeMarker source into Go html/template source.
type Converter struct{}

// NewConverter builds a stateless converter.
func NewConverter() *Converter {
	return &Converter{}
}

func newEmitter(file string) *emitter {
	return &emitter{
		file:    file,
		helpers: map[string]struct{}{},
		scopes:  []map[string]struct{}{{}},
	}
}

// Convert lexes, parses, and emits a single input template.
func (c *Converter) Convert(file string, input string) (Result, error) {
	tokens, err := lexer.Lex(file, input)
	if err != nil {
		return Result{}, err
	}
	doc, err := parser.Parse(file, tokens)
	if err != nil {
		return Result{}, err
	}

	e := newEmitter(file)
	if err := e.emitDocument(doc); err != nil {
		return Result{}, err
	}
	return Result{
		Output:   e.buf.String(),
		Helpers:  e.helperList(),
		Features: detectFeatures(doc, e.helperList()),
	}, nil
}

// emitter performs AST emission and tracks local variable scope.
type emitter struct {
	file    string
	buf     strings.Builder
	helpers map[string]struct{}
	scopes  []map[string]struct{}
}

// emitDocument emits the parsed document in original order.
func (e *emitter) emitDocument(doc ast.Document) error {
	return e.emitNodes(doc.Nodes)
}

// emitNodes emits each node from a sequence.
func (e *emitter) emitNodes(nodes []ast.Node) error {
	for _, node := range nodes {
		if err := e.emitNode(node); err != nil {
			return err
		}
	}
	return nil
}

// emitNode dispatches one AST node to its dedicated emitter.
func (e *emitter) emitNode(node ast.Node) error {
	switch n := node.(type) {
	case ast.TextNode:
		e.buf.WriteString(n.Text)
		return nil
	case ast.InterpolationNode:
		expr, err := e.mapExprAt(n.Expr, n.Position.Line, n.Position.Column)
		if err != nil {
			return err
		}
		e.writeAction(expr)
		return nil
	case ast.IfNode:
		return e.emitIfNode(n)
	case ast.ListNode:
		return e.emitListNode(n)
	case ast.AssignNode:
		return e.emitAssignNode(n)
	case ast.SettingNode:
		e.writeComment("ftl setting ignored: " + n.Raw)
		return nil
	case ast.FunctionNode:
		return diagnostics.New(
			"EMIT_UNSUPPORTED_FUNCTION",
			e.file,
			n.Position.Line,
			n.Position.Column,
			fmt.Sprintf("unsupported FreeMarker function definition %q", n.Name),
			"",
		)
	case ast.MacroCallNode:
		return diagnostics.New(
			"EMIT_UNSUPPORTED_MACRO_CALL",
			e.file,
			n.Position.Line,
			n.Position.Column,
			fmt.Sprintf("unsupported FreeMarker macro call <%s>", n.Name),
			"",
		)
	case ast.BareDirectiveNode:
		return e.emitBareDirectiveNode(n)
	default:
		return diagnostics.New(
			"EMIT_UNSUPPORTED_NODE",
			e.file,
			node.Pos().Line,
			node.Pos().Column,
			fmt.Sprintf("unsupported AST node type %T", node),
			"",
		)
	}
}

// mapExprAt maps a FreeMarker expression and keeps source location on errors.
func (e *emitter) mapExprAt(expr string, line int, col int) (string, error) {
	mapper := newExpressionMapper(e.currentLocals())
	mapped, err := mapper.mapExpr(expr)
	if err != nil {
		return "", diagnostics.New(
			"EMIT_EXPRESSION_MAP",
			e.file,
			line,
			col,
			err.Error(),
			expr,
		)
	}
	for _, h := range mapper.helperList() {
		e.helpers[h] = struct{}{}
	}
	return mapped, nil
}

// writeAction writes a raw Go template action.
func (e *emitter) writeAction(action string) {
	e.buf.WriteString("{{")
	e.buf.WriteString(action)
	e.buf.WriteString("}}")
}

// writeComment writes a Go template comment, escaping comment terminators.
func (e *emitter) writeComment(text string) {
	e.buf.WriteString("{{/* ")
	e.buf.WriteString(strings.ReplaceAll(text, "*/", "* /"))
	e.buf.WriteString(" */}}")
}

// helperList returns deterministic helper names used during expression mapping.
func (e *emitter) helperList() []string {
	helpers := make([]string, 0, len(e.helpers))
	for h := range e.helpers {
		helpers = append(helpers, h)
	}
	sort.Strings(helpers)
	return helpers
}

// pushScope creates a new variable scope.
func (e *emitter) pushScope() {
	e.scopes = append(e.scopes, map[string]struct{}{})
}

// popScope removes the current scope while keeping a root scope alive.
func (e *emitter) popScope() {
	if len(e.scopes) > 1 {
		e.scopes = e.scopes[:len(e.scopes)-1]
	}
}

// declareLocal marks a variable as local in the current scope.
func (e *emitter) declareLocal(name string) {
	e.scopes[len(e.scopes)-1][name] = struct{}{}
}

// isLocal reports whether a variable is visible from current scopes.
func (e *emitter) isLocal(name string) bool {
	for i := len(e.scopes) - 1; i >= 0; i-- {
		if _, ok := e.scopes[i][name]; ok {
			return true
		}
	}
	return false
}

// currentLocals flattens all active scopes to feed expression resolution.
func (e *emitter) currentLocals() map[string]struct{} {
	out := map[string]struct{}{}
	for _, scope := range e.scopes {
		for k := range scope {
			out[k] = struct{}{}
		}
	}
	return out
}
