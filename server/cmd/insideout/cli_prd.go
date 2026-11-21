package main

// PRD reads and edits on the CLI beyond `insideout prd <id>`: `prd update`
// (title, sections, status), `view` (an audience rendering, or its markdown
// with --export) and `readiness`. Committed versions run in
// cli_prd_versions.go.

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// runPrdUpdate is `insideout prd update [--title T] [--section key=value]…
// [--status S] <id>`: the section edits land first, then the status change.
func runPrdUpdate(args []string) (bool, int) {
	f, ok := parseWriteFlags("prd update", args)
	if !ok || !f.need(1, "insideout prd update [--title T] [--section key=value]… [--status S] <id>") {
		return true, 2
	}
	var sec map[string]string
	if len(f.sections) > 0 {
		sec = map[string]string{}
		for _, s := range f.sections {
			k, v, ok := strings.Cut(s, "=")
			if !ok {
				fmt.Fprintf(os.Stderr, "prd update: --section wants key=value, got %q\n", s)
				return true, 2
			}
			sec[k] = v
		}
	}
	var t *string
	if *f.title != "" {
		t = f.title
	}
	c := clientFromFlag(*f.api)
	out, err := c.UpdatePrd(f.positional[0], t, sec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if *f.status != "" {
		return printJSON(c.SetPrdStatus(f.positional[0], *f.status))
	}
	return printJSON(out, nil)
}

// runPrdView is `insideout view [--audience A] [--export FILE] <prd-id>`.
func runPrdView(args []string) (bool, int) {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	api := apiFlag(fs)
	audience := fs.String("audience", "decision", "decision|management|delivery|validation")
	out := fs.String("export", "", "write the audience markdown view to this file")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout view [--audience A] [--export FILE] <prd-id>")
		return true, 2
	}
	c := clientFromFlag(*api)
	if *out == "" {
		return printJSON(c.PrdView(fs.Args()[0], *audience))
	}
	md, err := c.PrdExportAudience(fs.Args()[0], *audience)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if err := os.WriteFile(*out, md, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		return true, 1
	}
	fmt.Fprintf(os.Stderr, "wrote %s"+"\n", *out)
	return true, 0
}

// runPrdReadiness is `insideout readiness <prd-id>`.
func runPrdReadiness(args []string) (bool, int) {
	fs := flag.NewFlagSet("readiness", flag.ContinueOnError)
	api := apiFlag(fs)
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout readiness <prd-id>")
		return true, 2
	}
	return printJSON(clientFromFlag(*api).PrdReadiness(fs.Args()[0]))
}
