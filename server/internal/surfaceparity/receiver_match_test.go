package surfaceparity

import (
	"go/ast"
	"go/parser"
	"path/filepath"
	"slices"
	"testing"
)

// A same-named method on another type must not count as a client call: the
// surface half matches calls by receiver type (rv5-PARC-03 limits).

const decoyFixture = `package decoy

import "github.com/frankji-groundcontrol/insideout/server/internal/apiclient"

type notClient struct{}

func (notClient) AgentPropose(projectID, nodeID, kind, summary, detail string, items []apiclient.ProposeItem) {}

func decoy() { notClient{}.AgentPropose("p", "n", "k", "s", "d", nil) }
`

const realCallFixture = `package decoy

import "github.com/frankji-groundcontrol/insideout/server/internal/apiclient"

func real(c *apiclient.Client) { _, _ = c.AgentPropose("p", "n", "k", "s", "d", nil) }
`

func TestSameNamedMethodOnAnotherTypeIsNotAClientCall(t *testing.T) {
	projectID := clientSends(t, loadAPI(t).routes(t))["POST /api/v1/agent/propose"]["projectId"]
	if len(projectID) == 0 {
		t.Fatal("the probe found no projectId on POST /agent/propose")
	}
	for _, c := range []struct {
		src  string
		want bool
	}{{decoyFixture, false}, {realCallFixture, true}} {
		// Placed in this directory so the type checker resolves the import.
		name := filepath.Join(serverRoot(), "internal", "surfaceparity", "decoy_fixture.go")
		f, err := parser.ParseFile(fset, name, c.src, 0)
		if err != nil {
			t.Fatal(err)
		}
		s := scanSurface(t, "fixture", []*ast.File{f})
		if got := slices.ContainsFunc(projectID, s.reaches); got != c.want {
			t.Errorf("reaches projectId = %v, want %v, for:\n%s", got, c.want, c.src)
		}
	}
}
