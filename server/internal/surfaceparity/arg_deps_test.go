package surfaceparity

import (
	"reflect"
	"slices"
)

// Where a sent key comes from: the argument, or the field inside one, whose
// zeroing changes the key (rv5-PARC-03).

// dep is what a sent key hangs on: method's argument arg (-1: nothing, any
// call sends it), and for a field inside it that field and its struct's type.
type dep struct {
	method string
	arg    int
	field  string // Go field name inside the argument, or "" for the whole argument
	typ    string // the struct type that declares field
	tail   bool   // arg is a variadic tail
	at     []int  // the field's path inside the argument (leaf.path)
}

// leaf is one exported struct field inside an argument. path walks from the
// argument to it: a field index, or -1 for a pointer's target or a slice's
// first element.
type leaf struct {
	path       []int
	typ, field string
}

// leaves lists every exported field inside a value of type t, as deep as the
// type goes; a struct met again inside itself is walked once.
func leaves(t reflect.Type, path []int, onPath map[reflect.Type]bool) []leaf {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice:
		return leaves(t.Elem(), append(slices.Clone(path), -1), onPath)
	case reflect.Struct:
		if onPath[t] {
			return nil
		}
		onPath[t] = true
		defer delete(onPath, t)
		var out []leaf
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			p := append(slices.Clone(path), i)
			out = append(out, leaf{p, t.Name(), f.Name})
			out = append(out, leaves(f.Type, p, onPath)...)
		}
		return out
	}
	return nil
}

// at follows a leaf path from v.
func at(v reflect.Value, path []int) reflect.Value {
	for _, i := range path {
		switch {
		case i >= 0:
			v = v.Field(i)
		case v.Kind() == reflect.Pointer:
			v = v.Elem()
		default:
			v = v.Index(0)
		}
	}
	return v
}

// argDeps zeroes each argument of p in turn (a struct argument field by
// field), and each field inside one, and returns those whose zeroing changes
// the key. A field that changes it stands in for the argument or field that
// holds it, so items[].title hangs on ProposeItem.Title, not on items. The
// call-site index of ft.In(i) is i-1: In(0) is the receiver.
func argDeps(p probe, changed func([]reflect.Value) bool) []dep {
	var deps []dep
	last := p.ft.NumIn() - 1
	for a := range p.ft.NumIn() {
		if a == p.clientAt {
			continue
		}
		var spots []leaf
		if p.ft.In(a).Kind() != reflect.Struct {
			spots = append(spots, leaf{}) // the whole argument
		}
		for _, l := range append(spots, leaves(p.ft.In(a), nil, map[reflect.Type]bool{})...) {
			alt, _ := fullArgs(p.ft, p.clientAt) // fresh: a clone would share slices
			at(alt[a], l.path).SetZero()
			if changed(alt) {
				deps = append(deps, dep{method: p.name, arg: a - 1, field: l.field, typ: l.typ,
					tail: l.field == "" && p.ft.IsVariadic() && a == last, at: l.path})
			}
		}
	}
	all := slices.Clone(deps)
	deps = slices.DeleteFunc(deps, func(d dep) bool {
		return slices.ContainsFunc(all, func(e dep) bool {
			return e.arg == d.arg && len(e.at) > len(d.at) && slices.Equal(e.at[:len(d.at)], d.at)
		})
	})
	if len(deps) == 0 {
		deps = []dep{{method: p.name, arg: -1}}
	}
	return deps
}
