package main

// version is what `insideout version` prints. It is stamped at link time, and
// each build recipe supplies its own value:
//
//	release.yml          the tag:           -X main.version=$GITHUB_REF_NAME
//	scripts/install.sh   git describe:      -X main.version=$(git describe --tags --always --dirty)
//	server/Dockerfile    the VERSION arg:   -X main.version=${VERSION}
//
// A plain `go run` or `go build` is unstamped and reports "dev", as does an
// image built without --build-arg VERSION. The linker silently ignores -X for
// a symbol that does not exist, so the name and every recipe's flag are pinned
// by scripts/tests/check_release_gate.py.
var version = "dev"
