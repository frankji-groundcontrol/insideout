package httpx

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// rv5-PARC-08: one error body shape {error, code}. Every code the server emits
// is lowercase snake_case and every extra key is camelCase, like the rest of
// the API's bodies.

var (
	snakeCode = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)
	camelKey  = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
)

func TestAnErrorWithoutACodeGetsOneFromItsStatus(t *testing.T) {
	for status, want := range map[int]string{
		http.StatusBadRequest:          "bad_request",
		http.StatusNotFound:            "not_found",
		http.StatusForbidden:           "forbidden",
		http.StatusInternalServerError: "internal_server_error",
	} {
		rec := httptest.NewRecorder()
		WriteError(rec, status, "nope", "", nil)
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["code"] != want {
			t.Errorf("status %d: code %v, want %q", status, body["code"], want)
		}
	}
}

// TestEveryEmittedCodeIsSnakeCaseAndEveryExtraKeyCamelCase reads every
// WriteError and SSE errorEvent call under server/internal and checks the
// literal code and the literal keys of the extra map.
func TestEveryEmittedCodeIsSnakeCaseAndEveryExtraKeyCamelCase(t *testing.T) {
	fset := token.NewFileSet()
	checked := 0
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			codeArg, extraArg := -1, -1
			switch sel.Sel.Name {
			case "WriteError":
				codeArg, extraArg = 3, 4
			case "errorEvent":
				codeArg = 1
			default:
				return true
			}
			if len(call.Args) <= codeArg {
				return true
			}
			checked++
			if lit, ok := call.Args[codeArg].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				code, _ := strconv.Unquote(lit.Value)
				if code != "" && !snakeCode.MatchString(code) {
					t.Errorf("%s: code %q is not lowercase snake_case", fset.Position(lit.Pos()), code)
				}
			}
			if extraArg >= 0 && len(call.Args) > extraArg {
				if cl, ok := call.Args[extraArg].(*ast.CompositeLit); ok {
					for _, el := range cl.Elts {
						kv, ok := el.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						if lit, ok := kv.Key.(*ast.BasicLit); ok && lit.Kind == token.STRING {
							key, _ := strconv.Unquote(lit.Value)
							if !camelKey.MatchString(key) {
								t.Errorf("%s: extra key %q is not camelCase", fset.Position(lit.Pos()), key)
							}
						}
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 100 {
		t.Fatalf("checked %d error calls; the walk missed the handlers", checked)
	}
}
