package parser

import (
	"testing"

	"github.com/cruffinoni/ftl2gotpl/internal/ast"
	"github.com/cruffinoni/ftl2gotpl/internal/lexer"
	"github.com/stretchr/testify/require"
)

func TestParseIfElseAndList(t *testing.T) {
	src := `<#assign x = user.name><#if x??><#list items as item>${item}</#list><#else>EMPTY</#if>`
	tokens, err := lexer.Lex("sample.ftl", src)
	require.NoError(t, err)

	doc, err := Parse("sample.ftl", tokens)
	require.NoError(t, err)

	require.Len(t, doc.Nodes, 2)

	_, ok := doc.Nodes[0].(ast.AssignNode)
	require.True(t, ok)

	ifNode, ok := doc.Nodes[1].(ast.IfNode)
	require.True(t, ok)
	require.Len(t, ifNode.Then, 1)
	_, ok = ifNode.Then[0].(ast.ListNode)
	require.True(t, ok)
}
