package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// ErrLogoutUnreachable is what Logout's error wraps when the request got no
// answer at all (offline, DNS, a timeout): the server session was not revoked,
// and the caller decides whether the local copy goes anyway.
var ErrLogoutUnreachable = errors.New("could not reach the server")

// Logout revokes the server-side session behind this client's refresh token
// (POST /auth/logout: no bearer; the handler answers 200 for any token, known or
// not, so any other answer is a failure to report) and then empties the client:
// a nil return means it holds no credentials. A copy of the saved login (a
// backup, a stolen laptop) stops renewing; the access token already issued is
// stateless and lives out its short TTL. A client with no refresh token has
// nothing to revoke and sends nothing. The handler never answers 401, so one
// can only come from something in front of the server: it is taken as already
// signed out. On any error the client keeps its tokens, so a retry presents the
// same refresh token instead of reporting a revocation that never happened
// (rv4-FR-F70).
func (c *Client) Logout() error {
	if c.refresh != "" {
		out, err := c.authPost("/auth/logout", map[string]string{"refreshToken": c.refresh})
		var noAnswer *url.Error
		switch {
		case errors.As(err, &noAnswer) && !errors.Is(err, context.Canceled): // a Ctrl-C is not an outage
			return fmt.Errorf("%w: %w", ErrLogoutUnreachable, err)
		case err != nil && out.status != http.StatusUnauthorized:
			return err
		}
	}
	c.token, c.refresh = "", ""
	return nil
}
