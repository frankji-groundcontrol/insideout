// Package ui is the interactive face of `insideout setup`: numbered
// questions on a terminal. Probe, sign-in, and saving the config stay in
// the flow package.
package ui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/flow"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/probe"
)

// Result carries the flow outcome plus the answers the command prints
// after a successful sign-in.
type Result struct {
	flow.Result
	// RegisterMCP is the person's answer to the MCP-registration question.
	RegisterMCP bool
}

// ErrNoTTY is returned when setup cannot ask questions.
type ErrNoTTY struct{}

func (ErrNoTTY) Error() string {
	return "insideout setup: interactive setup needs a terminal; " +
		"run instead: insideout setup --hosted | --local --postgres bundled|own"
}

// Run asks the numbered questions and drives flow.Run. It refuses to
// start without a terminal. Back on the first question stays there.
// Exit stops the wizard.
func Run(ctx context.Context, deps flow.Deps) (Result, error) {
	if !IsTerminal(os.Stdin) || !IsTerminal(os.Stdout) {
		return Result{}, ErrNoTTY{}
	}
	p := &prompts{in: bufio.NewReader(os.Stdin), out: os.Stdout, tty: os.Stdin}
	fmt.Fprintln(p.out, "InsideOut — interactive setup")
	for {
		mode, mv, err := p.at("1").choose("What are you setting up?", []string{
			"Remote default: the managed service",
			"Connect to a service you host",
			"Run a server on this machine",
		}, 0)
		if err != nil {
			return Result{}, err
		}
		if mv == moveBack {
			fmt.Fprintln(p.out, "This is the first step.")
			continue
		}
		if mv == moveExit {
			return Result{}, ErrExit
		}
		var res Result
		switch mode {
		case 0:
			res, err = runConnect(ctx, p, deps, false)
		case 1:
			res, err = runConnect(ctx, p, deps, true)
		default:
			res, err = runLocal(ctx, p, deps)
		}
		if errors.Is(err, ErrBack) {
			continue
		}
		return res, err
	}
}

func runLocal(ctx context.Context, p *prompts, deps flow.Deps) (Result, error) {
	for {
		choice, mv, err := p.at("1.3.1").choose("PostgreSQL for this server", []string{
			"Bundled Postgres in Docker (127.0.0.1:5442)",
			"A Postgres URL I already have",
		}, 0)
		if err != nil {
			return Result{}, err
		}
		if err = moveErr(mv); err != nil {
			return Result{}, err
		}
		postgres := "bundled"
		ownDSN := ""
		if choice == 1 {
			postgres = "own"
			secret, mv, err := p.at("1.3.2").password("Database URL")
			if err != nil {
				return Result{}, err
			}
			if err = moveErr(mv); err != nil {
				if errors.Is(err, ErrBack) {
					continue
				}
				return Result{}, err
			}
			ownDSN = secret
		}
		if deps.LocalSetup == nil {
			res, err := flow.Run(ctx, flow.Inputs{Mode: flow.ModeLocal}, deps)
			return Result{Result: res}, err
		}
		if err := deps.LocalSetup(ctx, postgres, ownDSN); err != nil {
			if errors.Is(err, ErrBack) {
				continue
			}
			if errors.Is(err, ErrExit) {
				return Result{}, err
			}
			fmt.Fprintf(p.out, "%v\nFix the settings before continuing.\n", err)
			continue
		}
		if deps.StartServer == nil {
			return Result{}, nil
		}
		start, mv, err := p.at("1.3.4").confirm("Start the API now?", false)
		if err != nil {
			return Result{}, err
		}
		if mv == moveBack {
			continue
		}
		if mv == moveExit {
			return Result{}, ErrExit
		}
		if err := deps.StartServer(start); err != nil {
			return Result{}, err
		}
		return Result{}, nil
	}
}

func runConnect(ctx context.Context, p *prompts, deps flow.Deps, askURL bool) (Result, error) {
	base := deps.DefaultBaseURL
	for {
		if askURL {
			got, mv, err := p.at("1.2.1").line("API base URL", "default")
			if err != nil {
				return Result{}, err
			}
			if err = moveErr(mv); err != nil {
				return Result{}, err
			}
			if got == "" || got == "default" {
				base = deps.DefaultBaseURL
			} else {
				base = got
			}
		}
		hideAddr := !askURL || base == deps.DefaultBaseURL
		if askURL {
			p.at("1.2.2")
		} else {
			p.at("1.1.1")
		}
		if !hostReachable(ctx, p, deps, base, hideAddr) {
			if askURL {
				continue
			}
			_, mv, err := p.at("1.1.1").choose("The service is not healthy.", []string{"Try again"}, 0)
			if err != nil {
				return Result{}, err
			}
			if err = moveErr(mv); err != nil {
				return Result{}, err
			}
			continue
		}
		res, err := runAccount(ctx, p, deps, base)
		if errors.Is(err, ErrBack) && askURL {
			continue
		}
		return res, err
	}
}

type serviceChecker interface {
	CheckService(ctx context.Context, baseURL string) ([]probe.Check, error)
}

func hostReachable(ctx context.Context, p *prompts, deps flow.Deps, base string, hideAddr bool) bool {
	target := base
	if hideAddr {
		target = "the default service"
	}
	fmt.Fprintf(p.out, "\n%sChecking %s\n", stepPrefix(p.step), target)
	checker, ok := deps.Prober.(serviceChecker)
	if !ok {
		if deps.Prober == nil {
			fmt.Fprintln(p.out, "Healthy.")
			return true
		}
		if err := deps.Prober.ProbeAPI(ctx, base); err != nil {
			fmt.Fprintf(p.out, "Not healthy: %s\nFix this before the next step.\n", apiclient.RedactDefault(err.Error()))
			return false
		}
		fmt.Fprintln(p.out, "Healthy.")
		return true
	}
	checks, err := checker.CheckService(ctx, base)
	for _, c := range checks {
		mark := "✓"
		if c.Status != "ok" {
			mark = "✗"
		}
		detail := c.Detail
		if hideAddr {
			detail = hideDefaultAddress(detail, base)
		}
		fmt.Fprintf(p.out, "  %s %-8s %s\n", mark, c.Name, detail)
	}
	if err != nil {
		fmt.Fprintln(p.out, "Not healthy. Fix this before the next step.")
		return false
	}
	fmt.Fprintln(p.out, "Healthy.")
	return true
}

func finishMCP(p *prompts, res flow.Result) (Result, error) {
	show, mv, err := p.at("1.1.7").confirm("Show the MCP registration snippet?", true)
	if err != nil || mv != moveOK {
		return Result{Result: res}, firstErr(err, moveErr(mv))
	}
	return Result{Result: res, RegisterMCP: show}, nil
}

func firstErr(err, nav error) error {
	if err != nil {
		return err
	}
	return nav
}

// IsTerminal reports whether f is a character device (a terminal).
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
