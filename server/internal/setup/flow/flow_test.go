package flow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
	"github.com/frankji-groundcontrol/insideout/server/internal/clientconfig"
)

// The flow's LoginClient is consumer-defined; the real client must keep
// satisfying it (compile-time proof).
var _ LoginClient = (*apiclient.Client)(nil)

type fakeProber struct {
	got string
	err error
}

func (f *fakeProber) ProbeAPI(ctx context.Context, baseURL string) error {
	f.got = baseURL
	return f.err
}

// fakeClient records every call, in order, so tests can assert the
// per-method call sequences.
type fakeClient struct {
	calls      []string
	events     *[]string
	token      string
	refresh    string
	loginErr   error
	requestErr error
	verifyErr  error
	consumeErr error
}

func (f *fakeClient) record(call string) {
	f.calls = append(f.calls, call)
	if f.events != nil {
		*f.events = append(*f.events, call)
	}
}

func (f *fakeClient) Login(email, password string) error {
	f.record("login:" + email + ":" + password)
	return f.loginErr
}

func (f *fakeClient) RequestLoginCode(email string) error {
	f.record("requestCode:" + email)
	return f.requestErr
}

func (f *fakeClient) VerifyLoginCode(email, code string) error {
	f.record("verifyCode:" + email + ":" + code)
	return f.verifyErr
}

func (f *fakeClient) RequestMagicLink(email string) error {
	f.record("requestMagic:" + email)
	return f.requestErr
}

func (f *fakeClient) ConsumeMagicLink(token string) error {
	f.record("consume:" + token)
	return f.consumeErr
}

func (f *fakeClient) AuthConfig() (string, error) { return "password", nil }

func (f *fakeClient) Token() string { return f.token }

func (f *fakeClient) RefreshToken() string { return f.refresh }

type fakePersister struct {
	saved []clientconfig.Config
	err   error
}

func (f *fakePersister) Save(c clientconfig.Config) error {
	f.saved = append(f.saved, c)
	return f.err
}

const testBase = "https://example.test/api/v1"

func runRemote(t *testing.T, mutate func(*Inputs, *Deps)) (Result, error) {
	t.Helper()
	inputs := Inputs{
		Mode:       ModeRemote,
		BaseURL:    testBase,
		AuthMethod: AuthPassword,
		Email:      "user@example.test",
		Secret:     "secret-value",
	}
	deps := Deps{
		Prober:         &fakeProber{},
		Client:         &fakeClient{token: "tok-123"},
		Persister:      &fakePersister{},
		DefaultBaseURL: "https://default.test/api/v1",
	}
	if mutate != nil {
		mutate(&inputs, &deps)
	}
	res, err := Run(context.Background(), inputs, deps)
	return res, err
}

func TestRunPasswordHappyPath(t *testing.T) {
	client := &fakeClient{token: "tok-123"}
	pers := &fakePersister{}
	res, err := runRemote(t, func(i *Inputs, d *Deps) {
		d.Client = client
		d.Persister = pers
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.APIBase != testBase || !res.SignedIn {
		t.Fatalf("Result = %+v, want APIBase %q SignedIn true", res, testBase)
	}
	if want := []string{"login:user@example.test:secret-value"}; strings.Join(client.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("client calls = %v, want %v", client.calls, want)
	}
	if len(pers.saved) != 1 {
		t.Fatalf("Save calls = %d, want 1", len(pers.saved))
	}
	want := clientconfig.Config{APIBase: testBase, Token: "tok-123"}
	if pers.saved[0] != want {
		t.Fatalf("saved = %+v, want %+v", pers.saved[0], want)
	}
}

func TestRunCodeHappyPath(t *testing.T) {
	events := []string{}
	client := &fakeClient{token: "tok-code", events: &events}
	pers := &fakePersister{}
	res, err := runRemote(t, func(i *Inputs, d *Deps) {
		i.AuthMethod = AuthCode
		i.Secret = ""
		d.Client = client
		d.Persister = pers
		d.PromptSecret = func(method string) (string, error) {
			if method != AuthCode {
				t.Fatalf("prompt method = %q, want %q", method, AuthCode)
			}
			events = append(events, "prompt:"+method)
			return "654321", nil
		}
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.SignedIn {
		t.Fatalf("Result = %+v, want SignedIn true", res)
	}
	want := []string{"requestCode:user@example.test", "verifyCode:user@example.test:654321"}
	if strings.Join(client.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("client calls = %v, want %v", client.calls, want)
	}
	if got := pers.saved[0].Token; got != "tok-code" {
		t.Fatalf("saved token = %q, want %q", got, "tok-code")
	}
	wantCalls := []string{"requestCode:user@example.test", "verifyCode:user@example.test:654321"}
	if strings.Join(client.calls, ",") != strings.Join(wantCalls, ",") {
		t.Fatalf("client calls = %v, want %v", client.calls, wantCalls)
	}
	wantEvents := []string{"requestCode:user@example.test", "prompt:code", "verifyCode:user@example.test:654321"}
	if strings.Join(events, ",") != strings.Join(wantEvents, ",") {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestRunMagicHappyPath(t *testing.T) {
	events := []string{}
	client := &fakeClient{token: "tok-magic", events: &events}
	res, err := runRemote(t, func(i *Inputs, d *Deps) {
		i.AuthMethod = AuthMagic
		i.Secret = ""
		d.Client = client
		d.PromptSecret = func(method string) (string, error) {
			if method != AuthMagic {
				t.Fatalf("prompt method = %q, want %q", method, AuthMagic)
			}
			events = append(events, "prompt:"+method)
			return "https://example.test/magic?token=abc123&next=/", nil
		}
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.SignedIn {
		t.Fatalf("Result = %+v, want SignedIn true", res)
	}
	// The pasted link must be reduced to its token before ConsumeMagicLink.
	want := []string{"requestMagic:user@example.test", "consume:abc123"}
	if strings.Join(client.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("client calls = %v, want %v", client.calls, want)
	}
	wantEvents := []string{"requestMagic:user@example.test", "prompt:magic", "consume:abc123"}
	if strings.Join(events, ",") != strings.Join(wantEvents, ",") {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

func TestRunEmptyBaseURLFallsBackToDefault(t *testing.T) {
	prober := &fakeProber{}
	res, err := runRemote(t, func(i *Inputs, d *Deps) {
		i.BaseURL = ""
		d.Prober = prober
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.APIBase != "https://default.test/api/v1" {
		t.Fatalf("APIBase = %q, want the DefaultBaseURL fallback", res.APIBase)
	}
	if prober.got != "https://default.test/api/v1" {
		t.Fatalf("probed %q, want the resolved default", prober.got)
	}
}

func TestRunBadURL(t *testing.T) {
	for _, baseURL := range []string{"ftp://example.test/api/v1", "https://", "not a url"} {
		_, err := runRemote(t, func(i *Inputs, d *Deps) { i.BaseURL = baseURL })
		var bad ErrBadURL
		if !errors.As(err, &bad) {
			t.Fatalf("BaseURL %q: error = %v, want ErrBadURL", baseURL, err)
		}
	}
}

func TestRunMissingFields(t *testing.T) {
	_, err := runRemote(t, func(i *Inputs, d *Deps) { i.Email = "" })
	var needEmail ErrEmailRequired
	if !errors.As(err, &needEmail) {
		t.Fatalf("empty email: error = %v, want ErrEmailRequired", err)
	}
	_, err = runRemote(t, func(i *Inputs, d *Deps) { i.Secret = "" })
	var needSecret ErrSecretRequired
	if !errors.As(err, &needSecret) {
		t.Fatalf("empty secret: error = %v, want ErrSecretRequired", err)
	}
}

func TestRunProbeFailure(t *testing.T) {
	sentinel := errors.New("connection refused")
	_, err := runRemote(t, func(i *Inputs, d *Deps) {
		d.Prober = &fakeProber{err: sentinel}
	})
	var probe *ProbeError
	if !errors.As(err, &probe) {
		t.Fatalf("error = %v, want ProbeError", err)
	}
	if probe.BaseURL != testBase {
		t.Fatalf("ProbeError.BaseURL = %q, want %q", probe.BaseURL, testBase)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("ProbeError must wrap the probe failure, got %v", err)
	}
}

func TestRunLocalNotImplemented(t *testing.T) {
	client := &fakeClient{}
	res, err := runRemote(t, func(i *Inputs, d *Deps) {
		i.Mode = ModeLocal
		d.Client = client
	})
	var local ErrLocalNotImplemented
	if !errors.As(err, &local) {
		t.Fatalf("error = %v, want ErrLocalNotImplemented", err)
	}
	if res != (Result{}) {
		t.Fatalf("Result = %+v, want zero", res)
	}
	if len(client.calls) != 0 {
		t.Fatalf("local mode must not touch the client, got %v", client.calls)
	}
}

func TestTokenFromLink(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://example.test/magic?token=abc123", "abc123"},
		{"https://example.test/magic?token=abc123&next=/", "abc123"},
		{"https://example.test/magic?next=/&token=abc123", "abc123"},
		{"https://example.test/magic?token=abc%201", "abc%201"},
		{"  https://example.test/magic?token=abc123  ", "abc123"},
		{"rawtoken", "rawtoken"},
		{"  rawtoken  ", "rawtoken"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := tokenFromLink(tc.in); got != tc.want {
			t.Errorf("tokenFromLink(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
