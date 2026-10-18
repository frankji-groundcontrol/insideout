package apiclient

import "net/http"

// Failing returns a client whose every request fails with err and never
// touches the network. A caller that could not resolve its credentials uses
// it so each verb reports the real problem instead of falling back to a
// default server.
func Failing(err error) *Client {
	return &Client{hc: &http.Client{Transport: failTransport{err}}}
}

type failTransport struct{ err error }

func (f failTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		r.Body.Close()
	}
	return nil, f.err
}
