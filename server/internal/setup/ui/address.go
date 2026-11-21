package ui

import (
	"net/url"
	"strings"

	"github.com/frankji-groundcontrol/insideout/server/internal/apiclient"
)

// hideDefaultAddress replaces the compiled-in service address with the
// word "default". A detail that does not name that address is unchanged.
func hideDefaultAddress(detail, base string) string {
	if detail == "" {
		return "default"
	}
	u, err := url.Parse(base)
	host := ""
	if err == nil {
		host = u.Host
	}
	if strings.Contains(detail, base) || (host != "" && strings.Contains(detail, host)) {
		return "default"
	}
	return apiclient.RedactDefault(detail)
}
