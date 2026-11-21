package surfaceparity

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// The Go client half: every exported apiclient.Client method is called by
// reflection against a server that only records, with each argument set, and
// then once more per argument (and per field of a struct argument) left zero.
// A key whose value changes or vanishes depends on that argument, which is
// what the CLI and MCP call sites are then checked against.

// dep is what a sent key hangs on: method's argument arg (-1: nothing, any
// call sends it), and for a struct argument its field.
type dep struct {
	method string
	arg    int
	field  string // Go field name of a struct argument, or ""
	typ    string // that struct's type name
	tail   bool   // arg is a variadic tail
}

// sent is one recorded request: route pattern and its body's keys and values.
type sent map[string]map[string]string

type recorder struct {
	mu  sync.Mutex
	got sent
}

func (rc *recorder) handler(patterns map[string]string) http.Handler {
	mux := http.NewServeMux()
	for p := range patterns {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			var body map[string]json.RawMessage
			_ = json.Unmarshal(raw, &body)
			rc.mu.Lock()
			if rc.got[p] == nil {
				rc.got[p] = map[string]string{}
			}
			for k, v := range body {
				rc.got[p][k] = string(v)
			}
			rc.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"accessToken":"a","refreshToken":"r","id":"x"}`)
		})
	}
	return mux
}

// probe is one exported method that sends requests: a Client method, or a
// method of a struct a Client method takes that runs on a *Client (the
// read-then-write Apply of a patch). clientAt is the *Client's place in ft.In.
type probe struct {
	name     string
	fn       reflect.Value
	ft       reflect.Type
	clientAt int
}

var clientType = reflect.TypeOf(&apiclient.Client{})

func probes() []probe {
	var out []probe
	structs := map[reflect.Type]bool{}
	for i := range clientType.NumMethod() {
		m := clientType.Method(i)
		out = append(out, probe{m.Name, m.Func, m.Type, 0})
		for a := 1; a < m.Type.NumIn(); a++ {
			if m.Type.In(a).Kind() == reflect.Struct {
				structs[m.Type.In(a)] = true
			}
		}
	}
	for st := range structs {
		for i := range st.NumMethod() {
			m := st.Method(i)
			for a := 1; a < m.Type.NumIn(); a++ {
				if m.Type.In(a) == clientType {
					out = append(out, probe{m.Name, m.Func, m.Type, a})
				}
			}
		}
	}
	return out
}

// clientSends maps route pattern and key to every way the Go client sends it.
func clientSends(t *testing.T, patterns map[string]string) map[string]map[string][]dep {
	rc := &recorder{got: sent{}}
	srv := httptest.NewServer(rc.handler(patterns))
	defer srv.Close()

	run := func(p probe, in []reflect.Value) sent {
		c := apiclient.New(srv.URL + "/api/v1")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		c.SetContext(ctx)
		_ = c.Login(probeID, probeID) // a signed-in client, as every verb runs on
		rc.got = sent{}
		in = cloneArgs(in)
		in[p.clientAt] = reflect.ValueOf(c)
		if p.ft.IsVariadic() {
			p.fn.CallSlice(in)
		} else {
			p.fn.Call(in)
		}
		return rc.got
	}

	out := map[string]map[string][]dep{}
	for _, p := range probes() {
		if p.name == "SetContext" || p.name == "SetRenewal" {
			continue
		}
		in, ok := fullArgs(p.ft, p.clientAt)
		if !ok {
			continue
		}
		for pattern, keys := range run(p, in) {
			if out[pattern] == nil {
				out[pattern] = map[string][]dep{}
			}
			for key, val := range keys {
				deps := argDeps(p, in, func(alt []reflect.Value) bool { return changes(run(p, alt), pattern, key, val) })
				out[pattern][key] = append(out[pattern][key], deps...)
			}
		}
	}
	if len(out) < 30 {
		t.Fatalf("the Go client reached only %d routes; the probe is broken", len(out))
	}
	return out
}

// argDeps zeroes each argument of p in turn (each exported field of a struct
// argument) and returns those whose zeroing changes the key. The call-site
// index of ft.In(i) is i-1: In(0) is the receiver.
func argDeps(p probe, in []reflect.Value, changed func([]reflect.Value) bool) []dep {
	var deps []dep
	for a := range in {
		if a == p.clientAt {
			continue
		}
		if in[a].Kind() == reflect.Struct {
			for f := range in[a].NumField() {
				if !in[a].Type().Field(f).IsExported() {
					continue
				}
				alt := cloneArgs(in)
				alt[a].Field(f).SetZero()
				if changed(alt) {
					deps = append(deps, dep{method: p.name, arg: a - 1, field: in[a].Type().Field(f).Name, typ: in[a].Type().Name()})
				}
			}
			continue
		}
		alt := cloneArgs(in)
		alt[a] = reflect.Zero(alt[a].Type())
		if changed(alt) {
			deps = append(deps, dep{method: p.name, arg: a - 1, tail: p.ft.IsVariadic() && a == len(in)-1})
		}
	}
	if len(deps) == 0 {
		deps = []dep{{method: p.name, arg: -1}}
	}
	return deps
}

// changes reports whether a run with one argument zeroed still reached pattern
// but sent key with another value or not at all. A run that no longer reached
// the route (the argument was a path id) or sent nothing (a client-side check
// refused it) says nothing about where the key comes from.
func changes(alt sent, pattern, key, val string) bool {
	body, hit := alt[pattern]
	return hit && body[key] != val
}

func cloneArgs(args []reflect.Value) []reflect.Value {
	out := make([]reflect.Value, len(args))
	for i, a := range args {
		out[i] = reflect.New(a.Type()).Elem()
		out[i].Set(a)
	}
	return out
}

// fullArgs builds a non-zero value for every parameter, or reports that one
// has a type the probe does not fill (a func, a channel, a context).
func fullArgs(mt reflect.Type, clientAt int) ([]reflect.Value, bool) {
	var args []reflect.Value
	for i := range mt.NumIn() {
		if i == clientAt {
			args = append(args, reflect.Zero(mt.In(i))) // run puts the client here
			continue
		}
		v, ok := nonZero(mt.In(i))
		if !ok {
			return nil, false
		}
		args = append(args, v)
	}
	return args, true
}

// probeID is a value every client-side check accepts where an id, a title or
// any other free string is wanted.
const probeID = "00000000-0000-4000-8000-0000000000a1"

func nonZero(t reflect.Type) (reflect.Value, bool) {
	v := reflect.New(t).Elem()
	if t == reflect.TypeOf(json.RawMessage(nil)) {
		v.SetBytes([]byte(`{"probe":1}`))
		return v, true
	}
	switch t.Kind() {
	case reflect.String:
		v.SetString(probeID)
	case reflect.Int, reflect.Int64:
		v.SetInt(7)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Float64:
		v.SetFloat(1.5)
	case reflect.Pointer:
		e, ok := nonZero(t.Elem())
		if !ok {
			return v, false
		}
		p := reflect.New(t.Elem())
		p.Elem().Set(e)
		v.Set(p)
	case reflect.Slice:
		e, ok := nonZero(t.Elem())
		if !ok {
			return v, false
		}
		v.Set(reflect.Append(v, e))
	case reflect.Map:
		k, ok1 := nonZero(t.Key())
		e, ok2 := nonZero(t.Elem())
		if !ok1 || !ok2 {
			return v, false
		}
		v.Set(reflect.MakeMap(t))
		v.SetMapIndex(k, e)
	case reflect.Struct:
		for i := range t.NumField() {
			if !t.Field(i).IsExported() {
				continue
			}
			f, ok := nonZero(t.Field(i).Type)
			if !ok {
				return v, false
			}
			v.Field(i).Set(f)
		}
	default:
		return v, false
	}
	return v, true
}
