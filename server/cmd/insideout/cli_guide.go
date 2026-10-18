package main

// `insideout guide [--out file] <project-id>` — fetch the scaffolded
// insideout.yaml matching guide and print (or write) it, so users can
// commit the guidance system to their repo themselves.

import (
	"flag"
	"fmt"
	"os"
)

func runGuideCommand(args []string) (bool, int) {
	fs := flag.NewFlagSet("guide", flag.ContinueOnError)
	api := apiFlag(fs)
	out := fs.String("out", "", "write the guide to this file instead of stdout")
	if err := fs.Parse(args); err != nil || len(fs.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "usage: insideout guide [--out insideout.yaml] <project-id>")
		return true, 2
	}
	g, err := clientFromFlag(*api).Guide(fs.Args()[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if *out == "" {
		os.Stdout.Write(g)
		return true, 0
	}
	if err := os.WriteFile(*out, g, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		return true, 1
	}
	fmt.Fprintf(os.Stderr, "wrote %s — commit it at the repo root\n", *out)
	return true, 0
}
