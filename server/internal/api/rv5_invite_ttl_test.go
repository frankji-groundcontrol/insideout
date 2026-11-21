package api

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/google/uuid"
)

// rv5-PARA-12: an invitation lifetime outside 1 to 720 hours is refused, not
// silently replaced by the default. An absent ttlHours still takes the
// default. Needs DATABASE_URL and DATABASE_OWNER_URL, like the admin suite.
func TestAnInvitationLifetimeOutOfRangeIsRefused(t *testing.T) {
	w, admin := newAdminWorld(t)
	invite := func(ttl string) (int, string) {
		body := `{"email":"ttl-` + uuid.NewString() + `@test.local"`
		if ttl != "" {
			body += `,"ttlHours":` + ttl
		}
		return w.call(admin, http.MethodPost, "/api/v1/admin/invitations", body+"}")
	}
	for _, ttl := range []int{0, -1, 721, 100000} {
		status, body := invite(strconv.Itoa(ttl))
		expect(t, "ttlHours "+strconv.Itoa(ttl), status, body, http.StatusBadRequest)
	}
	for _, ttl := range []string{"", "1", "720"} {
		status, body := invite(ttl)
		expect(t, "ttlHours "+ttl, status, body, http.StatusCreated)
	}
}
