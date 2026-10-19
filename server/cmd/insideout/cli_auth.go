package main

// Auth verbs — the CLI projection of every way the web can sign in:
// password (default), emailed 6-digit code, or magic link, plus register
// and the password-reset pair. Prompts go to stderr; stdout stays
// token-only for login/register (the `insideout login` contract), so
// `INSIDEOUT_TOKEN="$(insideout login …)"` keeps working. A successful
// login/register also saves the session (saveLogin).

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
	"golang.org/x/term"
)

func runAuthCommand(cmd string, args []string) (bool, int) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	api := apiFlag(fs)
	useCode := fs.Bool("code", false, "email a 6-digit code and read it from stdin")
	useMagic := fs.Bool("magic", false, "email a sign-in link and read the pasted link/token from stdin")
	invite := fs.String("invite", "", "invitation code (required when signup is invitation-only)")
	if err := fs.Parse(args); err != nil {
		return true, 2
	}
	positional := fs.Args()
	// Sign-in verbs carry no bearer: build the client from the resolved base only.
	creds, err := clientconfig.Resolve(*api)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	c := apiclient.New(creds.Base)
	c.SetContext(cliCtx)

	switch cmd {
	case "login":
		return loginCommand(c, creds.Base, positional, *useCode, *useMagic)
	case "register":
		return registerCommand(c, creds.Base, positional, *invite)
	case "logout":
		return logoutCommand(creds.Base)
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
		link, lerr := readSecret("reset link or token: ")
		if lerr != nil {
			fmt.Fprintln(os.Stderr, lerr)
			return true, 1
		}
		password, perr := readSecret("new password: ")
		if perr != nil {
			fmt.Fprintln(os.Stderr, perr)
			return true, 1
		}
		if link == "" || password == "" {
			fmt.Fprintln(os.Stderr, "reset-password: link and password are required")
			return true, 2
		}
		if err := c.ResetPassword(tokenFromLink(link), password); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		if err := forgetSavedLogin(creds.Base); err != nil {
			fmt.Fprintf(os.Stderr, "insideout: password reset, but the saved login could not be removed (%v)\n", err)
		}
		fmt.Fprintln(os.Stderr, "password reset; every session was revoked — sign in again")
		return true, 0
	}
	return false, 0
}

// logoutCommand revokes the saved session for base, then deletes that saved
// login. No saved login prints one line and exits 0, so a second logout is
// the same as the first. An unreachable server still loses the local copy;
// the session then expires by itself. Any other refusal keeps the file, so
// a retry presents the same refresh token.
func logoutCommand(base string) (bool, int) {
	cfg, err := savedLoginFor(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if cfg.Token == "" && cfg.RefreshToken == "" {
		fmt.Fprintln(os.Stderr, "not signed in")
		return true, 0
	}
	c := apiclient.New(base)
	c.SetContext(cliCtx)
	c.SetToken(cfg.Token)
	if cfg.RefreshToken != "" {
		c.SetRenewal(cfg.RefreshToken, apiclient.Renewal{})
	}
	err = c.Logout()
	unreachable := errors.Is(err, apiclient.ErrLogoutUnreachable)
	if err != nil && !unreachable {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	if ferr := forgetSavedLogin(base); ferr != nil {
		fmt.Fprintln(os.Stderr, ferr)
		return true, 1
	}
	if unreachable {
		fmt.Fprintln(os.Stderr, "insideout: the server session could not be revoked and expires by itself")
		return true, 0
	}
	fmt.Fprintln(os.Stderr, "insideout: signed out")
	return true, 0
}

func loginCommand(c *apiclient.Client, base string, positional []string, useCode, useMagic bool) (bool, int) {
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
		code, serr := readSecret("code: ")
		if serr != nil {
			err = serr
			break
		}
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
		link, serr := readSecret("link or token: ")
		if serr != nil {
			err = serr
			break
		}
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
	saveLogin(base, c)
	fmt.Println(c.Token())
	return true, 0
}

func registerCommand(c *apiclient.Client, base string, positional []string, inviteCode string) (bool, int) {
	email := argAt(positional, 0)
	if email == "" {
		fmt.Fprintln(os.Stderr, "usage: insideout register [--invite <invitation-code>] <email> [display-name]  (password is read from stdin; the invitation code is required when signup is invitation-only)")
		return true, 2
	}
	username := argAt(positional, 1)
	if username == "" {
		username = emailLocalPart(email)
	}
	password, serr := readSecret("password: ")
	if serr != nil {
		fmt.Fprintln(os.Stderr, serr)
		return true, 1
	}
	if password == "" {
		fmt.Fprintln(os.Stderr, "register: empty password")
		return true, 2
	}
	if err := c.Register(email, password, username, inviteCode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	saveLogin(base, c)
	fmt.Println(c.Token())
	return true, 0
}

// saveLogin keeps the session a sign-in just returned as the one saved login
// (replacing any other: a token belongs to the server that issued it), so the
// next command is signed in without an export line; `insideout setup` saves
// the same file. stdout stays the bare token for scripts, the path goes to
// stderr. A save that fails is a warning, not a failed sign-in: the token is
// still printed and good.
func saveLogin(base string, c *apiclient.Client) {
	path, err := clientconfig.DefaultPath()
	if err == nil {
		err = clientconfig.Save(path, clientconfig.Config{APIBase: base, Token: c.Token(), RefreshToken: c.RefreshToken()})
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "insideout: signed in, but could not save the login (%v); set INSIDEOUT_TOKEN to the printed token to use it\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "insideout: signed in to %s; login saved to %s\n", base, path)
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

// readSecret reads one secret answer without echoing it. On a terminal
// it must first discard whatever the shared buffered reader already
// holds (a pasted multi-line input would otherwise leave stale bytes
// that shadow the term.ReadPassword answer), then read raw with echo
// off. term.ReadPassword consumes the trailing Enter itself (x/term's
// readPasswordLine returns without the newline), so nothing may be
// read afterwards: an extra read would block for another Enter and
// echo it, echo being already restored. On a pipe it stays on the
// shared buffered reader so agents, scripts and tests keep working
// unchanged.
func readSecret(label string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return prompt(label), nil
	}
	fmt.Fprint(os.Stderr, label)
	stdin.Reset(os.Stdin)
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr)
	return strings.TrimSpace(string(secret)), nil
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
