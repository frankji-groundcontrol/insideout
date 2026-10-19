package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// ErrLogoutUnreachable is what Logout's error wraps when the request got no
// answer at all (offline, DNS, a timeout): the server session was not revoked,
// and the caller decides whether the local copy goes anyway.
var ErrLogoutUnreachable = errors.New("could not reach the server")

// Logout revokes the server-side session behind this client's refresh token
// (POST /auth/logout: no bearer, and the server answers 200 for any token), so a
// copy of the saved login (a backup, a stolen laptop) stops renewing. The access
// token already issued is stateless and lives out its short TTL. A client with
// no refresh token has nothing to revoke and sends nothing. The client keeps its
// credentials; the caller discards them (rv4-FR-F70).
func (c *Client) Logout() error {
	if c.refresh == "" {
		return nil
	}
	_, err := c.authPost("/auth/logout", map[string]string{"refreshToken": c.refresh})
	var noAnswer *url.Error
	if errors.As(err, &noAnswer) && !errors.Is(err, context.Canceled) { // a Ctrl-C is not an outage
		return fmt.Errorf("%w: %w", ErrLogoutUnreachable, err)
	}
	return err
}
