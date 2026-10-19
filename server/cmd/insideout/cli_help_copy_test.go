package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// rv4-USER-10 / rv4-PM-08: the CLI still used one word, "invite code", for
// the account invitation and the workspace code, and `setup --hosted` did
// not teach whoami or the invitation the way scripts/setup.sh does.

func rvCopyPinHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("INSIDEOUT_API", "")
	t.Setenv("INSIDEOUT_TOKEN", "")
}

// rvCopyCapture runs fn while stdout and stderr are pipes, and returns what
// the operator would have read. The pipes are closed before the read so a
// short write cannot block the test.
func rvCopyCapture(t *testing.T, fn func()) (string, string) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	closed := false
	defer func() {
		os.Stdout, os.Stderr = stdout, stderr
		if !closed {
			outW.Close()
			errW.Close()
		}
	}()
	fn()
	outW.Close()
	errW.Close()
	closed = true
	var out, errB bytes.Buffer
	if _, err := io.Copy(&out, outR); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(&errB, errR); err != nil {
		t.Fatal(err)
	}
	return out.String(), errB.String()
}

func TestRvCopyJoinHelpNamesTheWorkspaceCode(t *testing.T) {
	var summary string
	for _, v := range cliVerbs {
		if v.name == "join" {
			summary = v.summary
		}
	}
	low := strings.ToLower(summary)
	if !strings.Contains(low, "workspace code") || strings.Contains(low, "invite code") {
		t.Fatalf("join summary %q, want the workspace code and not \"invite code\"", summary)
	}
	var help bytes.Buffer
	printHelp(&help)
	if !strings.Contains(strings.ToLower(help.String()), "workspace code") {
		t.Fatal("insideout help does not name the workspace code")
	}
}

func TestRvCopyRegisterUsageNamesTheInvitation(t *testing.T) {
	rvCopyPinHome(t)
	var handled bool
	var code int
	_, stderr := rvCopyCapture(t, func() {
		handled, code = runAuthCommand("register", nil)
	})
	if !handled || code != 2 {
		t.Fatalf("register with no email: handled %v exit %d, want handled and 2", handled, code)
	}
	low := strings.ToLower(stderr)
	if !strings.Contains(low, "invitation code") || strings.Contains(low, "invite code") {
		t.Fatalf("register usage %q, want \"invitation code\" (the flag stays --invite)", stderr)
	}
}

func TestRvCopyHostedBlurbTeachesWhoamiAndTheInvitation(t *testing.T) {
	rvCopyPinHome(t)
	var handled bool
	var code int
	stdout, _ := rvCopyCapture(t, func() {
		handled, code = runSetupCommand([]string{"--hosted"})
	})
	if !handled || code != 0 {
		t.Fatalf("setup --hosted: handled %v exit %d, want handled and 0", handled, code)
	}
	for _, want := range []string{
		"insideout whoami",
		"insideout register --invite <invitation-code>",
		"invitation code",
		"administrator",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("setup --hosted does not teach %q", want)
		}
	}
	if strings.Contains(strings.ToLower(stdout), "invite code") {
		t.Error("setup --hosted still says \"invite code\"")
	}
	if strings.Contains(stdout, "insideout setup --local") {
		t.Error("setup --hosted tells a reader with no checkout to run insideout setup --local")
	}
}
