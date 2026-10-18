package apiclient

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Review-4 ARCH-05: the hosted URL is baked into the shipped binaries. It must
// be a package-level string variable initialised with a literal — the one form
// `go build -ldflags "-X <pkg>.DefaultBase=<url>"` can retarget, and a const is
// silently skipped — so a fork or a second deployment ships binaries that call
// its own server without a source edit.
func TestDefaultBaseIsLinkTimeSettable(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "default_base.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "DefaultBase" {
				continue
			}
			if gd.Tok != token.VAR {
				t.Fatalf("DefaultBase is declared with %q: -ldflags -X ignores everything but a package-level string var", gd.Tok)
			}
			if len(vs.Values) != 1 {
				t.Fatal("DefaultBase must be initialised in its declaration")
			}
			if lit, ok := vs.Values[0].(*ast.BasicLit); !ok || lit.Kind != token.STRING {
				t.Fatal("DefaultBase must be initialised with a string literal, the only initialiser -X can replace")
			}
			return
		}
	}
	t.Fatal("DefaultBase is not declared in default_base.go")
}

// scripts/default-api.sh is how the installer, setup.sh and the docs learn the
// hosted URL; it must print exactly what the binaries compile in.
func TestDefaultAPIScriptPrintsDefaultBase(t *testing.T) {
	script := filepath.Join("..", "..", "..", "scripts", "default-api.sh")
	out, err := exec.Command("bash", script).Output()
	if err != nil {
		t.Fatalf("default-api.sh: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != DefaultBase {
		t.Fatalf("default-api.sh prints %q, DefaultBase is %q", got, DefaultBase)
	}
}

// The committed default is some deployment's API root — not pinned to one
// host, which a release may retarget, but always an https URL ending /api/v1.
func TestDefaultBaseIsAnHTTPSAPIv1URL(t *testing.T) {
	u, err := url.Parse(DefaultBase)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "/api/v1" {
		t.Fatalf("DefaultBase = %q, want https://<host>/api/v1", DefaultBase)
	}
}
