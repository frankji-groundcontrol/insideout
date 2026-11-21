package ui

import (
	"context"
	"errors"
	"fmt"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/flow"
)

func runRegister(ctx context.Context, p *prompts, deps flow.Deps, base string) (Result, error) {
	mode := ""
	if deps.Client != nil {
		mode, _ = deps.Client.AuthConfig()
	}
	if mode == "closed" {
		fmt.Fprintln(p.out, "This server is not accepting registration. Ask the server's administrator, or log in.")
		return Result{}, ErrBack
	}
	invite := ""
	email := ""
	username := ""
	step := 0
	if mode == "open" {
		step = 1
	}
	for {
		switch step {
		case 0:
			got, mv, err := p.at("1.1.6.1").line("Invitation code", "")
			if err != nil || mv == moveExit {
				return Result{}, firstErr(err, moveErr(mv))
			}
			if mv == moveBack {
				return Result{}, ErrBack
			}
			invite = got
			step = 1
		case 1:
			got, mv, err := p.at("1.1.6.2").line("Email", "")
			if err != nil || mv == moveExit {
				return Result{}, firstErr(err, moveErr(mv))
			}
			if mv == moveBack {
				if mode == "open" {
					return Result{}, ErrBack
				}
				step = 0
				continue
			}
			email = got
			step = 2
		default:
			got, mv, err := p.at("1.1.6.3").line("Display name", "")
			if err != nil || mv == moveExit {
				return Result{}, firstErr(err, moveErr(mv))
			}
			if mv == moveBack {
				step = 1
				continue
			}
			username = got
			res, err := submitRegistration(ctx, p, deps, flow.Inputs{
				Mode:       flow.ModeRemote,
				BaseURL:    base,
				AuthMethod: flow.AuthRegister,
				Email:      email,
				Username:   username,
				InviteCode: invite,
			})
			if errors.Is(err, ErrBack) {
				continue
			}
			if errors.Is(err, ErrExit) {
				return Result{Result: res}, err
			}
			if err != nil {
				fmt.Fprintf(p.out, "%v\nFix this before continuing.\n", err)
				continue
			}
			return finishMCP(p, res)
		}
	}
}

// submitRegistration asks for the password when the flow needs it and runs
// the registration against the server.
func submitRegistration(ctx context.Context, p *prompts, deps flow.Deps, in flow.Inputs) (flow.Result, error) {
	deps.PromptSecret = func(string) (string, error) {
		secret, mv, err := p.at("1.1.6.4").password("Password")
		if err != nil {
			return "", err
		}
		if mv != moveOK {
			return "", moveErr(mv)
		}
		return secret, nil
	}
	return flow.Run(ctx, in, deps)
}
