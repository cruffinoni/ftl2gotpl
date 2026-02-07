package convert

import (
	"fmt"

	"github.com/cruffinoni/ftl2gotpl/internal/ast"
	"github.com/cruffinoni/ftl2gotpl/internal/diagnostics"
)

// emitIfNode converts FreeMarker if/elseif/else blocks to Go template actions.
func (e *emitter) emitIfNode(n ast.IfNode) error {
	cond, err := e.mapExprAt(n.Cond, n.Position.Line, n.Position.Column)
	if err != nil {
		return err
	}
	e.writeAction("if " + cond)

	e.pushScope()
	if err := e.emitNodes(n.Then); err != nil {
		e.popScope()
		return err
	}
	e.popScope()

	for _, alt := range n.ElseIf {
		altCond, mapErr := e.mapExprAt(alt.Cond, alt.Position.Line, alt.Position.Column)
		if mapErr != nil {
			return mapErr
		}
		e.writeAction("else if " + altCond)
		e.pushScope()
		if emitErr := e.emitNodes(alt.Body); emitErr != nil {
			e.popScope()
			return emitErr
		}
		e.popScope()
	}

	if len(n.Else) > 0 {
		e.writeAction("else")
		e.pushScope()
		if err := e.emitNodes(n.Else); err != nil {
			e.popScope()
			return err
		}
		e.popScope()
	}

	e.writeAction("end")
	return nil
}

// emitListNode converts a FreeMarker list block to a Go range action.
func (e *emitter) emitListNode(n ast.ListNode) error {
	seq, err := e.mapExprAt(n.SeqExpr, n.Position.Line, n.Position.Column)
	if err != nil {
		return err
	}

	indexVar := n.ItemVar + "_index"
	e.writeAction("range $" + indexVar + ", $" + n.ItemVar + " := " + seq)
	e.pushScope()
	e.declareLocal(indexVar)
	e.declareLocal(n.ItemVar)
	if err := e.emitNodes(n.Body); err != nil {
		e.popScope()
		return err
	}
	e.popScope()
	e.writeAction("end")
	return nil
}

// emitAssignNode maps assign/local directives to Go template assignments.
func (e *emitter) emitAssignNode(n ast.AssignNode) error {
	expr, err := e.mapExprAt(n.Expr, n.Position.Line, n.Position.Column)
	if err != nil {
		return err
	}
	if e.isLocal(n.Name) {
		e.writeAction("$" + n.Name + " = " + expr)
		return nil
	}
	e.writeAction("$" + n.Name + " := " + expr)
	e.declareLocal(n.Name)
	return nil
}

// emitBareDirectiveNode handles directives represented without a full block.
func (e *emitter) emitBareDirectiveNode(n ast.BareDirectiveNode) error {
	switch n.Name {
	case "break":
		e.writeAction("break")
		return nil
	case "return":
		return diagnostics.New(
			"EMIT_UNSUPPORTED_RETURN",
			e.file,
			n.Position.Line,
			n.Position.Column,
			"unsupported <#return> outside converted function semantics",
			n.Args,
		)
	default:
		return diagnostics.New(
			"EMIT_UNSUPPORTED_DIRECTIVE_NODE",
			e.file,
			n.Position.Line,
			n.Position.Column,
			fmt.Sprintf("unsupported directive node %q", n.Name),
			n.Args,
		)
	}
}
