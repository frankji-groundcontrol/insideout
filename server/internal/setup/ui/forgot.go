package ui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/flow"
)

func confirmSend(p *prompts, step, email string) (bool, error) {
	yes, mv, err := p.at(step).confirm("Send email to "+email+"?", true)
	if err != nil || mv != moveOK {
		return false, firstErr(err, moveErr(mv))
	}
	return yes, nil
}

func showMail(p *prompts, status string) {
	fmt.Fprintln(p.out, apiclient.MailStatusLine(status))
}

type resetMailer interface {
	ForgotPasswordStatus(email string) (watch, mail string, err error)
}

func sendReset(client flow.LoginClient, email string) (string, error) {
	if mailer, ok := client.(resetMailer); ok {
		_, mail, err := mailer.ForgotPasswordStatus(email)
		return mail, err
	}
	return "", client.ForgotPassword(email)
}

func runForgot(ctx context.Context, p *prompts, deps flow.Deps, base string) (Result, error) {
	if deps.Client == nil {
		return Result{}, fmt.Errorf("insideout setup: no API client")
	}
	email, mv, err := p.at("1.1.3.1").line("Email", "")
	if err != nil || mv != moveOK {
		return Result{}, firstErr(err, moveErr(mv))
	}
	yes, err := confirmSend(p, "1.1.3.1", email)
	if err != nil {
		return Result{}, err
	}
	if !yes {
		fmt.Fprintln(p.out, "Email not sent.")
		return Result{}, ErrBack
	}
	mail, err := sendReset(deps.Client, email)
	if err != nil {
		fmt.Fprintf(p.out, "%v\nFix this before continuing.\n", err)
		return Result{}, ErrBack
	}
	showMail(p, mail)
	fmt.Fprintln(p.out, "If that email is registered, a reset link is on its way. It works once, for 60 minutes.")
	pasted, mv, err := p.at("1.1.3.2").line("Reset code or link", "")
	if err != nil || mv != moveOK {
		return Result{}, firstErr(err, moveErr(mv))
	}
	token := resetToken(pasted)
	password, mv, err := p.at("1.1.3.3").password("New password")
	if err != nil || mv != moveOK {
		return Result{}, firstErr(err, moveErr(mv))
	}
	if err := deps.Client.ResetPassword(token, password); err != nil {
		fmt.Fprintf(p.out, "%v\nFix this before continuing.\n", err)
		return Result{}, ErrBack
	}
	fmt.Fprintln(p.out, "Password updated. Signing in.")
	res, err := flow.Run(ctx, flow.Inputs{
		Mode:       flow.ModeRemote,
		BaseURL:    base,
		AuthMethod: flow.AuthPassword,
		Email:      email,
		Secret:     password,
	}, deps)
	if errors.Is(err, ErrBack) || errors.Is(err, ErrExit) || err != nil {
		return Result{Result: res}, err
	}
	return finishMCP(p, res)
}

func resetToken(pasted string) string {
	pasted = strings.TrimSpace(pasted)
	if u, err := url.Parse(pasted); err == nil {
		if token := u.Query().Get("token"); token != "" {
			return token
		}
	}
	const key = "token="
	if i := strings.Index(pasted, key); i >= 0 {
		rest := pasted[i+len(key):]
		if cut := strings.IndexAny(rest, " &#"); cut >= 0 {
			rest = rest[:cut]
		}
		return rest
	}
	return pasted
}
