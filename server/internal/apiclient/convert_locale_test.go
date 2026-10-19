package apiclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// rv4-CLOSE-02: the web app tells the server which language the coach speaks
// when it converts an idea, but the CLI and the MCP server converted with an
// empty body, so the server stored its default ("zh") whatever the idea's
// language. These tests pin the client half: the locale rides the request
// body, and ConvertLocale chooses it the way an operator expects.

func TestIdeaConvertWithLocaleCarriesTheLocaleInTheBody(t *testing.T) {
	for _, tc := range []struct{ name, locale, wantBody string }{
		{"en", "en", `{"locale":"en"}`},
		{"zh", "zh", `{"locale":"zh"}`},
		{"none keeps the empty body", "", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var method, path, body string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				method, path, body = r.Method, r.URL.Path, strings.TrimSpace(string(raw))
				w.Write([]byte(`{"prdId":"p1","conversationId":"c1"}`))
			}))
			t.Cleanup(srv.Close)

			if _, err := New(srv.URL).IdeaConvertWithLocale("i1", tc.locale); err != nil {
				t.Fatalf("IdeaConvertWithLocale: %v", err)
			}
			if method != http.MethodPost || path != "/ideas/i1/convert" || body != tc.wantBody {
				t.Fatalf("sent %s %s %s, want POST /ideas/i1/convert %s", method, path, body, tc.wantBody)
			}
		})
	}
}

// pinLocaleEnv sets the three variables ConvertLocale reads; a test that
// leaves one out gets it empty, so the machine's own language cannot leak in.
func pinLocaleEnv(t *testing.T, all, messages, lang string) {
	t.Helper()
	t.Setenv("LC_ALL", all)
	t.Setenv("LC_MESSAGES", messages)
	t.Setenv("LANG", lang)
}

func TestConvertLocale(t *testing.T) {
	for _, tc := range []struct {
		name                string
		explicit            string
		all, messages, lang string
		want                string
	}{
		{"an explicit en wins over a zh environment", "en", "", "", "zh_CN.UTF-8", "en"},
		{"an explicit zh wins over an en environment", "zh", "en_US.UTF-8", "", "", "zh"},
		{"an app locale tag is accepted: en-US", "en-US", "", "", "", "en"},
		{"an app locale tag is accepted: zh-CN", "zh-CN", "", "", "", "zh"},
		{"LC_ALL beats LC_MESSAGES and LANG", "", "zh_CN.UTF-8", "en_US.UTF-8", "en_US.UTF-8", "zh"},
		{"LC_MESSAGES beats LANG", "", "", "en_US.UTF-8", "zh_CN.UTF-8", "en"},
		{"LANG alone: zh", "", "", "", "zh_CN.UTF-8", "zh"},
		{"LANG alone: en", "", "", "", "en_US.UTF-8", "en"},
		{"LC_ALL=C overrides a zh LANG", "", "C", "", "zh_CN.UTF-8", "en"},
		{"a language the coach lacks falls back to en", "", "", "", "fr_FR.UTF-8", "en"},
		{"no signal at all is en", "", "", "", "", "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pinLocaleEnv(t, tc.all, tc.messages, tc.lang)
			got, err := ConvertLocale(tc.explicit)
			if err != nil || got != tc.want {
				t.Fatalf("ConvertLocale(%q) = %q, %v; want %q", tc.explicit, got, err, tc.want)
			}
		})
	}
}

// An explicit language the coach cannot speak is refused, not mapped: the
// server turns any tag that is not English into Chinese, so "fr" would
// silently start a Chinese conversation.
func TestConvertLocaleRefusesALanguageItCannotCoachIn(t *testing.T) {
	pinLocaleEnv(t, "", "", "en_US.UTF-8")
	got, err := ConvertLocale("fr")
	if err == nil || !strings.Contains(err.Error(), "en or zh") {
		t.Fatalf("ConvertLocale(\"fr\") = %q, %v; want an error naming en or zh", got, err)
	}
}
