package surfaceparity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The server half of the walk: every route the api package registers, and the
// JSON fields its write handlers decode, found by AST so a new route or field
// joins the check without anyone listing it.

func serverRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// fset holds every file the walk parses, so a type checker can place them.
var fset = token.NewFileSet()

// parseDir parses the non-test Go files of one server directory.
func parseDir(t *testing.T, rel string) []*ast.File {
	t.Helper()
	dir := filepath.Join(serverRoot(), rel)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	var files []*ast.File
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		files = append(files, f)
	}
	return files
}

// apiPackage is the parsed api package: its functions, types and constants.
type apiPackage struct {
	files  []*ast.File
	funcs  map[string]*ast.FuncDecl
	types  map[string]ast.Expr
	consts map[string]string
}

func loadAPI(t *testing.T) apiPackage {
	p := apiPackage{files: parseDir(t, "internal/api"), funcs: map[string]*ast.FuncDecl{},
		types: map[string]ast.Expr{}, consts: map[string]string{}}
	for _, f := range p.files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				p.funcs[d.Name.Name] = d
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						p.types[s.Name.Name] = s.Type
					case *ast.ValueSpec:
						for i, n := range s.Names {
							if i < len(s.Values) {
								if v, ok := p.str(s.Values[i]); ok {
									p.consts[n.Name] = v
								}
							}
						}
					}
				}
			}
		}
	}
	return p
}

// str evaluates a string literal, a constant, or a sum of them.
func (p apiPackage) str(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			s, err := strconv.Unquote(e.Value)
			return s, err == nil
		}
	case *ast.Ident:
		s, ok := p.consts[e.Name]
		return s, ok
	case *ast.BinaryExpr:
		a, ok1 := p.str(e.X)
		b, ok2 := p.str(e.Y)
		return a + b, ok1 && ok2
	}
	return "", false
}

// routes maps every registered "METHOD /path" pattern to its handler's name:
// the innermost function the HandleFunc argument wraps.
func (p apiPackage) routes(t *testing.T) map[string]string {
	out := map[string]string{}
	for _, f := range p.files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); !ok || (sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle") {
				return true
			}
			pattern, ok := p.str(call.Args[0])
			if !ok {
				t.Errorf("a route pattern is not a constant string: %#v", call.Args[0])
				return true
			}
			h := call.Args[1]
			for {
				if c, ok := h.(*ast.CallExpr); ok && len(c.Args) > 0 {
					h = c.Args[len(c.Args)-1]
					continue
				}
				break
			}
			switch h := h.(type) {
			case *ast.SelectorExpr:
				out[pattern] = h.Sel.Name
			case *ast.Ident:
				out[pattern] = h.Name
			}
			return true
		})
	}
	if len(out) < 50 {
		t.Fatalf("found only %d routes in internal/api; the scan is broken", len(out))
	}
	return out
}

// requestFields maps each route to the JSON keys its handler decodes (the
// POST, PATCH and PUT bodies, and DELETE /me's password) (through httpx.DecodeJSON, in the handler or any package
// function it calls). A route that decodes no body is left out.
func (p apiPackage) requestFields(t *testing.T) map[string][]string {
	out := map[string][]string{}
	for pattern, handler := range p.routes(t) {
		keys := map[string]bool{}
		p.decodedKeys(handler, keys, map[string]bool{})
		if len(keys) == 0 {
			continue
		}
		for k := range keys {
			out[pattern] = append(out[pattern], k)
		}
		sort.Strings(out[pattern])
	}
	return out
}

func (p apiPackage) decodedKeys(fn string, keys, seen map[string]bool) {
	d := p.funcs[fn]
	if d == nil || d.Body == nil || seen[fn] {
		return
	}
	seen[fn] = true
	ast.Inspect(d.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var name string
		switch f := call.Fun.(type) {
		case *ast.SelectorExpr:
			name = f.Sel.Name
		case *ast.Ident:
			name = f.Name
		}
		if name == "DecodeJSON" && len(call.Args) == 2 {
			if u, ok := call.Args[1].(*ast.UnaryExpr); ok {
				if id, ok := u.X.(*ast.Ident); ok {
					p.structKeys(localType(d.Body, id.Name), keys)
				}
			}
			return true
		}
		p.decodedKeys(name, keys, seen)
		return true
	})
}

// localType finds the declared type of a local variable: var x T, or x := T{}.
func localType(body *ast.BlockStmt, name string) ast.Expr {
	var typ ast.Expr
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ValueSpec:
			for _, id := range n.Names {
				if id.Name == name && n.Type != nil {
					typ = n.Type
				}
			}
		case *ast.AssignStmt:
			for i, l := range n.Lhs {
				if id, ok := l.(*ast.Ident); ok && id.Name == name && i < len(n.Rhs) {
					rhs := n.Rhs[i]
					if u, ok := rhs.(*ast.UnaryExpr); ok {
						rhs = u.X
					}
					if cl, ok := rhs.(*ast.CompositeLit); ok {
						typ = cl.Type
					}
				}
			}
		}
		return typ == nil
	})
	return typ
}

// structKeys adds the JSON keys of a struct type expression. A field whose
// type is a struct, a pointer to one or a slice of them adds its own keys
// below its key: items[].title, tree.title.
func (p apiPackage) structKeys(e ast.Expr, keys map[string]bool) {
	p.keysUnder(e, "", keys, map[string]bool{})
}

// keysUnder walks e with every key prefixed; onPath stops a type met again
// inside itself (a tree node's children), so it is walked once.
func (p apiPackage) keysUnder(e ast.Expr, prefix string, keys, onPath map[string]bool) {
	switch e := e.(type) {
	case *ast.StarExpr:
		p.keysUnder(e.X, prefix, keys, onPath)
	case *ast.Ident:
		if onPath[e.Name] {
			return
		}
		onPath[e.Name] = true
		defer delete(onPath, e.Name)
		p.keysUnder(p.types[e.Name], prefix, keys, onPath)
	case *ast.StructType:
		for _, f := range e.Fields.List {
			name := ""
			if f.Tag != nil {
				tag, _ := strconv.Unquote(f.Tag.Value)
				name, _, _ = strings.Cut(reflect.StructTag(tag).Get("json"), ",")
			}
			names := []string{name}
			switch {
			case name == "-":
				continue
			case name == "" && len(f.Names) == 0:
				p.keysUnder(f.Type, prefix, keys, onPath) // embedded
				continue
			case name == "":
				names = names[:0]
				for _, n := range f.Names {
					names = append(names, n.Name)
				}
			}
			for _, n := range names {
				keys[prefix+n] = true
				typ, below := f.Type, prefix+n+"."
				if s, ok := typ.(*ast.StarExpr); ok {
					typ = s.X
				}
				if a, ok := typ.(*ast.ArrayType); ok {
					typ, below = a.Elt, prefix+n+"[]."
				}
				p.keysUnder(typ, below, keys, onPath)
			}
		}
	}
}

// agentPolicy reads agentRoutePolicy from agent_scope.go: pattern to access.
func (p apiPackage) agentPolicy(t *testing.T) map[string]string {
	out := map[string]string{}
	for _, f := range p.files {
		ast.Inspect(f, func(n ast.Node) bool {
			vs, ok := n.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "agentRoutePolicy" || len(vs.Values) != 1 {
				return true
			}
			for _, el := range vs.Values[0].(*ast.CompositeLit).Elts {
				kv := el.(*ast.KeyValueExpr)
				k, _ := p.str(kv.Key)
				out[k] = kv.Value.(*ast.Ident).Name
			}
			return false
		})
	}
	if len(out) < 50 {
		t.Fatalf("agentRoutePolicy has %d entries; the scan is broken", len(out))
	}
	return out
}
