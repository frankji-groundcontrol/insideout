package surfaceparity

import (
	"go/ast"
	"go/parser"
	"slices"
	"sort"
	"testing"
)

// A nested request struct: the server half must walk into it, and the client
// probe must follow it to the field that fills each nested key (rv5-PARC-03
// limits).

const nestedFixture = `package fixture

type req struct {
	Kind  string ` + "`json:\"kind\"`" + `
	Items []item ` + "`json:\"items\"`" + `
	One   *item  ` + "`json:\"one\"`" + `
	Tree  node   ` + "`json:\"tree\"`" + `
}

type item struct {
	Title string ` + "`json:\"title\"`" + `
	Hint  string ` + "`json:\"parentHint\"`" + `
}

type node struct {
	Title    string ` + "`json:\"title\"`" + `
	Children []node ` + "`json:\"children\"`" + `
}
`

func TestStructKeysWalkNestedFields(t *testing.T) {
	f, err := parser.ParseFile(fset, "nested_fixture.go", nestedFixture, 0)
	if err != nil {
		t.Fatal(err)
	}
	p := apiPackage{types: map[string]ast.Expr{}}
	for _, d := range f.Decls {
		for _, s := range d.(*ast.GenDecl).Specs {
			p.types[s.(*ast.TypeSpec).Name.Name] = s.(*ast.TypeSpec).Type
		}
	}
	keys := map[string]bool{}
	p.structKeys(ast.NewIdent("req"), keys, 0)
	got := sortedKeys(keys)
	// A type met again inside itself is walked once: tree.children stops there.
	want := []string{"items", "items[].parentHint", "items[].title", "kind",
		"one", "one.parentHint", "one.title", "tree", "tree.children", "tree.title"}
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Errorf("structKeys(req) = %v, want %v", got, want)
	}
}

func TestClientProbeSeesNestedFields(t *testing.T) {
	sends := clientSends(t, loadAPI(t).routes(t))["POST /api/v1/agent/propose"]
	if !slices.ContainsFunc(sends["items[].title"], func(d dep) bool { return d.typ == "ProposeItem" && d.field == "Title" }) {
		t.Errorf("items[].title: deps %+v, want ProposeItem.Title", sends["items[].title"])
	}
	// action is the client's constant: it hangs on the items argument alone.
	if a := sends["items[].action"]; len(a) != 1 || a[0].field != "" || a[0].arg != 5 {
		t.Errorf("items[].action: deps %+v, want AgentPropose's items argument", a)
	}
}
