package ui

import (
	"context"
	"errors"
	"fmt"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/flow"
)

// The account questions: log in or register, then the sign-in method.

func runAccount(ctx context.Context, p *prompts, deps flow.Deps, base string) (Result, error) {
	for {
		account, mv, err := p.at("1.1.2").choose("Do you already have an account?", []string{
			"Log in",
			"Register",
		}, 0)
		if err != nil {
			return Result{}, err
		}
		if mv == moveBack {
			return Result{}, ErrBack
		}
		if mv == moveExit {
			return Result{}, ErrExit
		}
		if account == 1 {
			res, err := runRegister(ctx, p, deps, base)
			if errors.Is(err, ErrBack) {
				continue
			}
			return res, err
		}
		res, err := runLogin(ctx, p, deps, base)
		if errors.Is(err, ErrBack) {
			continue
		}
		return res, err
	}
}

func runLogin(ctx context.Context, p *prompts, deps flow.Deps, base string) (Result, error) {
	methods := []string{flow.AuthPassword, flow.AuthCode, flow.AuthMagic}
	for {
		methodChoice, mv, err := p.at("1.1.3").choose("How do you want to sign in?", []string{
			"Password",
			"Emailed login code",
			"Magic link",
			"Forgot password",
		}, 0)
		if err != nil || mv != moveOK {
			return Result{}, firstErr(err, moveErr(mv))
		}
		if methodChoice == 3 {
			res, err := runForgot(ctx, p, deps, base)
			if errors.Is(err, ErrBack) {
				continue
			}
			return res, err
		}
		for {
			email, mv, err := p.at("1.1.4").line("Email", "")
			if err != nil || mv == moveExit {
				return Result{}, firstErr(err, moveErr(mv))
			}
			if mv == moveBack {
				break
			}
			if methods[methodChoice] != flow.AuthPassword {
				yes, err := confirmSend(p, "1.1.4", email)
				if err != nil {
					return Result{}, err
				}
				if !yes {
					fmt.Fprintln(p.out, "Email not sent.")
					continue
				}
			}
			deps.OnMail = func(status string) { showMail(p, status) }
			deps.PromptSecret = func(method string) (string, error) {
				secret, mv, err := p.at("1.1.5").password(secretTitle(method))
				if err != nil {
					return "", err
				}
				if mv != moveOK {
					return "", moveErr(mv)
				}
				return secret, nil
			}
			res, err := flow.Run(ctx, flow.Inputs{
				Mode:       flow.ModeRemote,
				BaseURL:    base,
				AuthMethod: methods[methodChoice],
				Email:      email,
			}, deps)
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

func secretTitle(method string) string {
	switch method {
	case flow.AuthCode:
		return "Login code (check your email)"
	case flow.AuthMagic:
		return "Magic link (paste the link from your email)"
	default:
		return "Password"
	}
}
