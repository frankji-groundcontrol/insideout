package main

import (
	"flag"
	"fmt"
	"os"
)

// runExportCommand is `insideout export` (rv5-PARB-06): the whole PRD as
// markdown, one audience's markdown, or the print HTML, the same export the
// web and the MCP prd_export tool give, on stdout or in --out.
func runExportCommand(args []string) (bool, int) {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	api := apiFlag(fs)
	format := fs.String("format", "markdown", "markdown|print (print is HTML)")
	audience := fs.String("audience", "", "decision|management|delivery|validation (markdown only)")
	out := fs.String("out", "", "write the export to this file instead of stdout")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 || (*format != "markdown" && *format != "print") {
		fmt.Fprintln(os.Stderr, "usage: insideout export [--format markdown|print] [--audience A] [--out FILE] <prd-id>")
		return true, 2
	}
	body, err := clientFromFlag(*api).PrdExport(fs.Args()[0], *format, *audience)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if *out == "" {
		os.Stdout.Write(body)
		return true, 0
	}
	if err := os.WriteFile(*out, body, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		return true, 1
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
	return true, 0
}
