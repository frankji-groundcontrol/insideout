package surfaceparity

import (
	"go/ast"
	"testing"
)

// The CLI and MCP half: both reach the API only through apiclient, so a
// surface reaches a key when it calls a client method that sends it and fills
// the argument the key hangs on (a non-zero argument, or the struct field set).

type surface struct {
	name  string
	calls map[string][][]ast.Expr    // client method name to each call's arguments
	sets  map[string]map[string]bool // apiclient struct type to the fields this surface sets
}

func loadSurface(t *testing.T, name, rel string) surface {
	s := surface{name: name, calls: map[string][][]ast.Expr{}, sets: map[string]map[string]bool{}}
	for _, f := range parseDir(t, rel) {
		mentioned := map[string]bool{} // apiclient types this file names
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "apiclient" {
					mentioned[sel.Sel.Name] = true
				}
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
					s.calls[sel.Sel.Name] = append(s.calls[sel.Sel.Name], n.Args)
				}
			case *ast.CompositeLit:
				if sel, ok := n.Type.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && id.Name == "apiclient" {
						for _, el := range n.Elts {
							if kv, ok := el.(*ast.KeyValueExpr); ok {
								s.set(sel.Sel.Name, kv.Key.(*ast.Ident).Name)
							}
						}
					}
				}
			case *ast.AssignStmt:
				// x.Field = v, in a file that names the type: a field set after the literal.
				for _, l := range n.Lhs {
					if sel, ok := l.(*ast.SelectorExpr); ok {
						for typ := range mentioned {
							s.set(typ, sel.Sel.Name)
						}
					}
				}
			}
			return true
		})
	}
	if len(s.calls) < 20 {
		t.Fatalf("%s: found only %d call names in %s; the scan is broken", name, len(s.calls), rel)
	}
	return s
}

func (s surface) set(typ, field string) {
	if s.sets[typ] == nil {
		s.sets[typ] = map[string]bool{}
	}
	s.sets[typ][field] = true
}

// reaches reports whether the surface fills d: it calls d.method, and the key
// needs nothing more, or the surface sets the struct field, or some call passes
// a value other than a zero literal in the argument.
func (s surface) reaches(d dep) bool {
	calls := s.calls[d.method]
	switch {
	case len(calls) == 0:
		return false
	case d.field != "":
		return s.sets[d.typ][d.field]
	case d.arg < 0:
		return true
	}
	for _, args := range calls {
		end := min(d.arg+1, len(args))
		if d.tail {
			end = len(args) // a variadic tail is one argument
		}
		for _, a := range args[min(d.arg, len(args)):end] {
			if !zeroLiteral(a) {
				return true
			}
		}
	}
	return false
}

func zeroLiteral(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		return e.Value == `""` || e.Value == "``" || e.Value == "0"
	case *ast.Ident:
		return e.Name == "nil" || e.Name == "false"
	}
	return false
}
