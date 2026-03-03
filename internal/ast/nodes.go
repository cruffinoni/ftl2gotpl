// Package ast defines FreeMarker syntax tree node types.
package ast

// Position is a 1-based source position within a template.
type Position struct {
	Line   int
	Column int
}

// Node is the common interface implemented by every AST node.
type Node interface {
	node()
	Pos() Position
}

// Document is the parser output root node.
type Document struct {
	Nodes []Node
}

// TextNode stores literal template text.
type TextNode struct {
	Position Position
	Text     string
}

func (n TextNode) node() {}

// Pos returns the source position of the node.
func (n TextNode) Pos() Position { return n.Position }

// InterpolationNode stores a ${...} or #{...} expression.
type InterpolationNode struct {
	Position Position
	Expr     string
	AltStyle bool
}

func (n InterpolationNode) node() {}

// Pos returns the source position of the node.
func (n InterpolationNode) Pos() Position { return n.Position }

// IfElseIf represents one elseif branch in an if block.
type IfElseIf struct {
	Position Position
	Cond     string
	Body     []Node
}

// IfNode represents <#if ...> with optional elseif and else branches.
type IfNode struct {
	Position Position
	Cond     string
	Then     []Node
	ElseIf   []IfElseIf
	Else     []Node
}

func (n IfNode) node() {}

// Pos returns the source position of the node.
func (n IfNode) Pos() Position { return n.Position }

// ListNode represents a <#list seq as item>...</#list> block.
type ListNode struct {
	Position Position
	SeqExpr  string
	ItemVar  string
	Body     []Node
}

func (n ListNode) node() {}

// Pos returns the source position of the node.
func (n ListNode) Pos() Position { return n.Position }

// AssignNode represents an <#assign ...> or <#local ...> directive.
type AssignNode struct {
	Position Position
	Name     string
	Expr     string
	Local    bool
}

func (n AssignNode) node() {}

// Pos returns the source position of the node.
func (n AssignNode) Pos() Position { return n.Position }

// SettingNode represents ignored FreeMarker setting directives.
type SettingNode struct {
	Position Position
	Raw      string
}

func (n SettingNode) node() {}

// Pos returns the source position of the node.
func (n SettingNode) Pos() Position { return n.Position }

// FunctionNode represents a FreeMarker function block.
type FunctionNode struct {
	Position Position
	Name     string
	Params   []string
	Body     []Node
}

func (n FunctionNode) node() {}

// Pos returns the source position of the node.
func (n FunctionNode) Pos() Position { return n.Position }

// BareDirectiveNode represents directives such as <#return> and <#break>.
type BareDirectiveNode struct {
	Position Position
	Name     string
	Args     string
}

func (n BareDirectiveNode) node() {}

// Pos returns the source position of the node.
func (n BareDirectiveNode) Pos() Position { return n.Position }

// MacroCallNode represents FreeMarker <@macro ...> calls.
type MacroCallNode struct {
	Position Position
	Name     string
	Args     string
}

func (n MacroCallNode) node() {}

// Pos returns the source position of the node.
func (n MacroCallNode) Pos() Position { return n.Position }
