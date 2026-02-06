package convert

import (
	"sort"

	"github.com/cruffinoni/ftl2gotpl/internal/ast"
)

func detectFeatures(doc ast.Document, helpers []string) []string {
	set := map[string]struct{}{}

	var walk func(nodes []ast.Node)
	walk = func(nodes []ast.Node) {
		for _, n := range nodes {
			switch t := n.(type) {
			case ast.TextNode:
				set["node:text"] = struct{}{}
			case ast.InterpolationNode:
				set["node:interpolation"] = struct{}{}
			case ast.IfNode:
				set["directive:if"] = struct{}{}
				if len(t.ElseIf) > 0 {
					set["directive:elseif"] = struct{}{}
				}
				if len(t.Else) > 0 {
					set["directive:else"] = struct{}{}
				}
				walk(t.Then)
				for _, alt := range t.ElseIf {
					walk(alt.Body)
				}
				walk(t.Else)
			case ast.ListNode:
				set["directive:list"] = struct{}{}
				walk(t.Body)
			case ast.AssignNode:
				if t.Local {
					set["directive:local"] = struct{}{}
				} else {
					set["directive:assign"] = struct{}{}
				}
			case ast.SettingNode:
				set["directive:setting"] = struct{}{}
			case ast.FunctionNode:
				set["directive:function"] = struct{}{}
				walk(t.Body)
			case ast.BareDirectiveNode:
				set["directive:"+t.Name] = struct{}{}
			case ast.MacroCallNode:
				set["call:macro"] = struct{}{}
			}
		}
	}

	walk(doc.Nodes)
	for _, h := range helpers {
		set["helper:"+h] = struct{}{}
	}

	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
