package catalog

import (
	"strings"
	"testing"
)

func TestParseSkeletonFormRules(t *testing.T) {
	skeleton := strings.Join([]string{
		"# help for the required variable",
		"REQ=",
		"",
		"# stale help wiped by the blank line",
		"",
		"#OPT help line one",
		"#OPT help line two",
		"#OPT=default-value",
		"",
		"# FOO=spaced-comment-form",
		"",
		"##NOPE=double hash is only comment text",
		"",
		"# real help survives free text",
		"free text",
		"KEPT=x",
		"#TOKENISH=",
	}, "\n")
	vars, err := ParseSkeleton([]byte(skeleton))
	if err != nil {
		t.Fatalf("ParseSkeleton: %v", err)
	}
	got := map[string]Var{}
	for _, v := range vars {
		got[v.Name] = v
	}

	req := got["REQ"]
	if req.Name == "" {
		t.Fatal("REQ missing from parse")
	}
	if !req.Required {
		t.Error("uncommented assignment must be Required")
	}
	if req.Comment != "help for the required variable" {
		t.Errorf("REQ.Comment = %q", req.Comment)
	}
	if req.Default != "" || req.Secret {
		t.Errorf("required var: Default = %q Secret = %v, want \"\"/false", req.Default, req.Secret)
	}

	opt := got["OPT"]
	if opt.Required {
		t.Error("commented assignment must be optional")
	}
	if opt.Example != "default-value" || opt.Default != "default-value" {
		t.Errorf("OPT Example/Default = %q/%q, want the shown value as the default in effect",
			opt.Example, opt.Default)
	}
	if opt.Comment != "OPT help line one OPT help line two" {
		t.Errorf("OPT.Comment = %q (comment lines must join and attach)", opt.Comment)
	}
	if opt.Secret {
		t.Error("OPT must carry Secret = false")
	}

	if foo := got["FOO"]; foo.Required || foo.Example != "spaced-comment-form" {
		t.Errorf("# FOO=spaced-comment-form must parse as optional FOO, got %+v", foo)
	}
	if _, ok := got["NOPE"]; ok {
		t.Error("##NOPE= must parse as comment text, not as a key")
	}
	// A blank line clears the comment buffer; a free-text line is ignored but
	// leaves the buffer intact — python parse_skeleton's exact rules.
	if kept := got["KEPT"]; kept.Comment != "real help survives free text" {
		t.Errorf("KEPT.Comment = %q, want only the help after the blank line", kept.Comment)
	}
	if !got["TOKENISH"].Secret {
		t.Error("ParseSkeleton must fill Secret from the name rule (TOKENISH)")
	}
}

func TestEvaluateFiveStatuses(t *testing.T) {
	skeleton, err := ParseSkeleton([]byte("REQ=from-skeleton\n#OPT=opt-default\n#EMPTY=\n"))
	if err != nil {
		t.Fatalf("ParseSkeleton: %v", err)
	}
	cases := []struct {
		name string
		env  string
		want map[string]Status
	}{
		{
			name: "required unset is missing, optional unset is unset",
			env:  "",
			want: map[string]Status{"REQ": StatusMissing, "OPT": StatusUnset},
		},
		{
			name: "empty value counts as not set",
			env:  "REQ=\nOPT=\nEMPTY=\n",
			want: map[string]Status{"REQ": StatusMissing, "OPT": StatusUnset},
		},
		{
			name: "byte-identical to the example is default, even when required",
			env:  "REQ=from-skeleton\nOPT=opt-default\n",
			want: map[string]Status{"REQ": StatusDefault, "OPT": StatusDefault},
		},
		{
			name: "a supplied value is set",
			env:  "REQ=mine\nOPT=also-mine\nEMPTY=full\n",
			want: map[string]Status{"REQ": StatusSet, "OPT": StatusSet},
		},
		{
			name: "an empty example never counts as a default",
			env:  "EMPTY=\n",
			want: map[string]Status{"EMPTY": StatusUnset},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := map[string]State{}
			for _, s := range Evaluate([]byte(tc.env), skeleton) {
				got[s.Var.Name] = s
			}
			for name, want := range tc.want {
				if got[name].Status != want {
					t.Errorf("%s status = %q, want %q", name, got[name].Status, want)
				}
			}
		})
	}
	if states := Evaluate([]byte("EMPTY=full\n"), skeleton); states[2].Status != StatusSet {
		t.Errorf("a non-empty value with an empty example must be set, got %q", states[2].Status)
	}
}

func TestEvaluatePlaceholder(t *testing.T) {
	skeleton, err := ParseSkeleton([]byte(
		"A=\nB=\nC=\nD=\nE=\nDATABASE_URL=postgres://insideout_app:change_me@your-remote-host:5432/x\n"))
	if err != nil {
		t.Fatalf("ParseSkeleton: %v", err)
	}
	cases := []struct {
		env  string
		want Status
	}{
		{"A=CHANGE_ME_upper\n", StatusPlaceholder},
		{"B=postgres://u:p@your-remote-host:5432/x\n", StatusPlaceholder},
		{"C=postgres://u:p@10.0.0.1:5432/x\n", StatusSet},
		{"D=a<b\n", StatusSet}, // '<' alone is deliberately not a heuristic
		{"E=x-change_me-y\n", StatusPlaceholder},
		// A skeleton credential can never pass as a default: placeholder wins.
		{"DATABASE_URL=postgres://insideout_app:change_me@your-remote-host:5432/x\n", StatusPlaceholder},
	}
	for _, tc := range cases {
		name := strings.SplitN(tc.env, "=", 2)[0]
		for _, s := range Evaluate([]byte(tc.env), skeleton) {
			if s.Var.Name == name && s.Status != tc.want {
				t.Errorf("%q status = %q, want %q", tc.env, s.Status, tc.want)
			}
		}
	}
}

func TestEvaluatePreservesVarOrderAndCarriesValues(t *testing.T) {
	skeleton, err := ParseSkeleton([]byte("#ZZZ=\nAAA=\n"))
	if err != nil {
		t.Fatalf("ParseSkeleton: %v", err)
	}
	states := Evaluate([]byte("ZZZ=1\nAAA=2\n"), skeleton)
	if len(states) != 2 || states[0].Var.Name != "ZZZ" || states[1].Var.Name != "AAA" {
		t.Fatalf("Evaluate must preserve input order, got %+v", states)
	}
	if states[0].Value != "1" || states[0].Status != StatusSet ||
		states[1].Value != "2" || states[1].Status != StatusSet {
		t.Errorf("State must carry the live value, got %+v", states)
	}
}

func TestEvaluateParsesLikeBash(t *testing.T) {
	skeleton, err := ParseSkeleton([]byte(
		"PORT=\nPW=\nQ1=\nQ2=\nEXPORTED=\nDUP=\nBAD-NAME=\n"))
	if err != nil {
		t.Fatalf("ParseSkeleton: %v", err)
	}
	env := []byte(strings.Join([]string{
		"PORT=5442   # host side",
		"PW=p@ss#word",
		`Q1="two words"`,
		"Q2='single quoted'",
		"export EXPORTED=yes",
		"DUP=first",
		"DUP=second",
		"BAD-NAME=x",
	}, "\n"))
	got := map[string]State{}
	for _, s := range Evaluate(env, skeleton) {
		got[s.Var.Name] = s
	}
	want := map[string]string{
		"PORT": "5442", "PW": "p@ss#word", "Q1": "two words",
		"Q2": "single quoted", "EXPORTED": "yes", "DUP": "second",
	}
	for name, value := range want {
		if got[name].Value != value {
			t.Errorf("%s value = %q, want %q", name, got[name].Value, value)
		}
	}
	if got["DUP"].Status != StatusSet {
		t.Errorf("last duplicate wins, status = %q", got["DUP"].Status)
	}
	if got["BAD-NAME"].Value != "" {
		t.Errorf("a name with a dash is not a dotenv assignment, got %q", got["BAD-NAME"].Value)
	}
}

func TestSecretNames(t *testing.T) {
	secrets := []string{
		"DATABASE_URL", "DATABASE_OWNER_URL", "INSIDEOUT_JWT_SECRET",
		"INSIDEOUT_LLM_API_KEY", "INSIDEOUT_MAIL_API_KEY", "GITHUB_TOKEN",
		"INSIDEOUT_LLM_BASE_URL", "INSIDEOUT_GH_WEBHOOK_SECRET",
		"INSIDEOUT_GH_PRIVATE_KEY", "POSTGRES_APP_PASSWORD",
	}
	for _, name := range secrets {
		if !Secret(name) {
			t.Errorf("%s must be a secret", name)
		}
	}
	plain := []string{
		"INSIDEOUT_ADDR", "INSIDEOUT_ACCESS_TTL", "INSIDEOUT_COOKIE_SECURE",
		"INSIDEOUT_LLM_MODEL", "INSIDEOUT_LLM_SCHEMA", "INSIDEOUT_DEV_CORS",
		"INSIDEOUT_CORS_ORIGINS", "INSIDEOUT_PUBLIC_URL", "SERVER_PORT",
		"CLIENT_PORT", "POSTGRES_PORT",
	}
	for _, name := range plain {
		if Secret(name) {
			t.Errorf("%s must NOT be a secret (a name rule that over-sweeps makes the TUI useless)", name)
		}
	}
}

func TestMask(t *testing.T) {
	if got := Mask(""); got != "" {
		t.Errorf(`Mask("") = %q, want ""`, got)
	}
	if got := Mask("hunter2"); got != "set (7 chars)" {
		t.Errorf("Mask = %q, want %q", got, "set (7 chars)")
	}
}

func TestRedactDSN(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "userinfo blanked",
			in:   "postgres://u:hunter2@10.0.0.1/db",
			want: "postgres://***@10.0.0.1/db",
		},
		{
			name: "last @ bounds the password",
			in:   "postgres://insideout_app:p@ss/w:rd&x@10.0.0.1:5432/insideout?sslmode=require",
			want: "postgres://***@10.0.0.1:5432/insideout?sslmode=require",
		},
		{
			name: "postgresql scheme preserved",
			in:   "postgresql://u:p@h/db",
			want: "postgresql://***@h/db",
		},
		{
			name: "a credential query parameter masks the whole value",
			in:   "postgres://u@h/db?password=x",
			want: "set (28 chars)",
		},
		{
			name: "libpq keyword form masks the whole value",
			in:   "host=db password=xyz",
			want: "set (20 chars)",
		},
		{
			name: "no userinfo leaves the DSN as-is",
			in:   "postgres://localhost:5432/db",
			want: "postgres://localhost:5432/db",
		},
		{name: "empty stays empty", in: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RedactDSN(tc.in); got != tc.want {
				t.Errorf("RedactDSN(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDisplayNeverReturnsSecretBytes(t *testing.T) {
	vars, err := ParseSkeleton([]byte(
		"DATABASE_URL=\nINSIDEOUT_JWT_SECRET=\n#GITHUB_TOKEN=\n#INSIDEOUT_LLM_MODEL=x\n"))
	if err != nil {
		t.Fatalf("ParseSkeleton: %v", err)
	}
	env := []byte(strings.Join([]string{
		"DATABASE_URL=postgres://u:hunter2@10.0.0.1/db",
		"INSIDEOUT_JWT_SECRET=deadbeef" + strings.Repeat("0", 56),
		"GITHUB_TOKEN=ghp_change_me_yet_REAL",
		"INSIDEOUT_LLM_MODEL=claude-opus-5",
	}, "\n"))
	got := map[string]State{}
	for _, s := range Evaluate(env, vars) {
		got[s.Var.Name] = s
		display := s.Display()
		for _, leak := range []string{"hunter2", "deadbeef", "ghp_", "REAL"} {
			if strings.Contains(display, leak) {
				t.Errorf("%s display leaks %q: %q", s.Var.Name, leak, display)
			}
		}
	}
	if d := got["DATABASE_URL"].Display(); d != "postgres://***@10.0.0.1/db" {
		t.Errorf("DATABASE_URL display = %q, want the redacted DSN", d)
	}
	if d := got["INSIDEOUT_JWT_SECRET"].Display(); d != "set (64 chars)" {
		t.Errorf("INSIDEOUT_JWT_SECRET display = %q, want %q", d, "set (64 chars)")
	}
	if d := got["GITHUB_TOKEN"].Display(); d != "set (22 chars)" {
		t.Errorf("a secret whose value contains a placeholder stays masked, display = %q", d)
	}
	if d := got["INSIDEOUT_LLM_MODEL"].Display(); d != "claude-opus-5" {
		t.Errorf("a non-secret is shown plainly, display = %q", d)
	}
}

func TestDisplayEmptyAndNonSecretDSN(t *testing.T) {
	// An unset value renders as nothing; a DSN-shaped value redacts by shape,
	// whatever its name is.
	vars, err := ParseSkeleton([]byte("#EMPTY=\nSOME_URL=\n"))
	if err != nil {
		t.Fatalf("ParseSkeleton: %v", err)
	}
	states := Evaluate([]byte("SOME_URL=postgres://a:b@c/d\n"), vars)
	if d := states[0].Display(); d != "" {
		t.Errorf("empty display = %q, want \"\"", d)
	}
	if d := states[1].Display(); d != "postgres://***@c/d" {
		t.Errorf("DSN shape must redact regardless of name, display = %q", d)
	}
}
