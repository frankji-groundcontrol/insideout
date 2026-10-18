package main

// version is what `insideout version` prints. An unstamped build (go run,
// scripts/install.sh) reports "dev"; release.yml stamps the tag with
//
//	go build -ldflags "-X main.version=$GITHUB_REF_NAME" ./cmd/insideout
//
// The linker silently ignores -X for a symbol that does not exist, so the
// name and the release.yml flag are pinned by scripts/tests/check_release_gate.py.
var version = "dev"
