package main

// Auth verbs — the CLI projection of every way the web can sign in:
// password (default), emailed 6-digit code, or magic link, plus register
// and the password-reset pair. Prompts go to stderr; stdout stays
// token-only for login/register (the `insideout login` contract), so
// `INSIDEOUT_TOKEN="$(insideout login …)"` keeps working.

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

func runAuthCommand(cmd string, args []string, api string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	useCode := fs.Bool("code", false, "email a 6-digit code and read it from stdin")
	useMagic := fs.Bool("magic", false, "email a sign-in link and read the pasted link/token from stdin")
	invite := fs.String("invite", "", "invitation code (required when signup is invitation-only)")
	if err := fs.Parse(args); err != nil {
		return true, 2
	}
	positional := fs.Args()
	c := apiclient.New(api)

	switch cmd {
	case "login":
		return loginCommand(c, positional, *useCode, *useMagic)
	case "register":
		return registerCommand(c, positional, *invite)
	case "forgot-password":
		email := argAt(positional, 0)
		if email == "" {
			fmt.Fprintln(os.Stderr, "usage: insideout forgot-password <email>")
			return true, 2
		}
		if err := c.ForgotPassword(email); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		fmt.Fprintln(os.Stderr, "if that account exists, a reset link is on its way; finish with `insideout reset-password` (paste the link) or in the browser")
		return true, 0
	case "reset-password":
		link := prompt("reset link or token: ")
		password := prompt("new password: ")
		if link == "" || password == "" {
			fmt.Fprintln(os.Stderr, "reset-password: link and password are required")
			return true, 2
		}
		if err := c.ResetPassword(tokenFromLink(link), password); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		fmt.Fprintln(os.Stderr, "password reset; every session was revoked — sign in again")
		return true, 0
	}
	return false, 0
}

func loginCommand(c *apiclient.Client, positional []string, useCode, useMagic bool) (bool, int) {
	email := argAt(positional, 0)
	if email == "" {
		fmt.Fprintln(os.Stderr, "usage: insideout login [--code|--magic] <email>  (secrets are read from stdin)")
		return true, 2
	}
	var err error
	switch {
	case useCode && useMagic:
		fmt.Fprintln(os.Stderr, "login: pick one of --code or --magic")
		return true, 2
	case useCode:
		if err = c.RequestLoginCode(email); err != nil {
			break
		}
		fmt.Fprintf(os.Stderr, "code sent (if the account exists) — check %s\n", email)
		code := prompt("code: ")
		if code == "" {
			fmt.Fprintln(os.Stderr, "login: empty code")
			return true, 2
		}
		err = c.VerifyLoginCode(email, code)
	case useMagic:
		if err = c.RequestMagicLink(email); err != nil {
			break
		}
		fmt.Fprintln(os.Stderr, "link sent (if the account exists) — paste the link (or token) from the email")
		link := prompt("link or token: ")
		if link == "" {
			fmt.Fprintln(os.Stderr, "login: empty link")
			return true, 2
		}
		err = c.ConsumeMagicLink(tokenFromLink(link))
	default:
		_, password, cerr := credentials(positional)
		if cerr != nil {
			fmt.Fprintln(os.Stderr, "login:", cerr)
			return true, 2
		}
		err = c.Login(email, password)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	fmt.Println(c.Token())
	return true, 0
}

func registerCommand(c *apiclient.Client, positional []string, inviteCode string) (bool, int) {
	email := argAt(positional, 0)
	if email == "" {
		fmt.Fprintln(os.Stderr, "usage: insideout register [--invite code] <email> [display-name]  (password is read from stdin)")
		return true, 2
	}
	username := argAt(positional, 1)
	if username == "" {
		username = emailLocalPart(email)
	}
	password := prompt("password: ")
	if password == "" {
		fmt.Fprintln(os.Stderr, "register: empty password")
		return true, 2
	}
	if err := c.Register(email, password, username, inviteCode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	fmt.Println(c.Token())
	return true, 0
}

// stdin is shared by every prompt in one command run: a per-call
// bufio.Reader would buffer ahead and swallow the second answer.
var stdin = bufio.NewReader(os.Stdin)

// prompt reads one trimmed line from stdin with the label on stderr —
// secrets never touch stdout.
func prompt(label string) string {
	fmt.Fprint(os.Stderr, label)
	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSpace(line)
}

func argAt(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

// tokenFromLink accepts either a pasted URL (…?token=abc…) or a raw
// token; both emailed flows land in the user's clipboard as the URL.
func tokenFromLink(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "token="); idx >= 0 {
		v := s[idx+len("token="):]
		if amp := strings.IndexAny(v, "& \t"); amp >= 0 {
			v = v[:amp]
		}
		return v
	}
	return s
}

func emailLocalPart(email string) string {
	if at := strings.Index(email, "@"); at > 0 {
		return email[:at]
	}
	return email
}
