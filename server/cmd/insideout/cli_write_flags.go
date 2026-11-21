package main

// The flag types the write verbs share. writeFlags is the one flag set the
// idea, PRD-edit, evidence and workspace verbs ran under when a single
// function (cli_crud.go) held them all; each verb still accepts every flag
// in it, so no command line that parsed before stops parsing.

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// stringList is a repeatable string flag (--section k=v --section k2=v2).
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ", ") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// multiFlag collects repeated string flags.
type multiFlag []string

func (m *multiFlag) String() string     { return "" }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

// writeFlags is a parsed write-verb command line.
type writeFlags struct {
	api, title, content, desc, status *string
	sections                          stringList
	positional                        []string
}

// parseWriteFlags parses args under the flag set named verb; false means
// the flags did not parse and the verb exits 2.
func parseWriteFlags(verb string, args []string) (*writeFlags, bool) {
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	f := &writeFlags{api: apiFlag(fs)}
	f.title = fs.String("title", "", "title")
	f.content = fs.String("content", "", "content (idea body / section text)")
	f.desc = fs.String("desc", "", "description")
	f.status = fs.String("status", "", "PRD status transition (applies after section edits)")
	fs.Var(&f.sections, "section", "section key=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return nil, false
	}
	f.positional = fs.Args()
	return f, true
}

// need reports whether the verb got n positional arguments, printing its
// usage line when it did not.
func (f *writeFlags) need(n int, usage string) bool {
	if len(f.positional) != n {
		fmt.Fprintln(os.Stderr, "usage: "+usage)
		return false
	}
	return true
}
