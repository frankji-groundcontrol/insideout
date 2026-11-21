package local

import (
	"fmt"
	"os"

	"github.com/frankji-groundcontrol/insideout/server/internal/setup/catalog"
	"github.com/frankji-groundcontrol/insideout/server/internal/setup/envwrite"
)

// wireDatabaseURL writes the runtime URL into .env and says so, redacted.
func wireDatabaseURL(envPath, dsn string) bool {
	if err := envwrite.SetVar(envPath, "DATABASE_URL", dsn); err != nil {
		fmt.Fprintf(os.Stderr, "setup: write DATABASE_URL: %v\n", err)
		return false
	}
	fmt.Fprintf(os.Stderr, "setup: wrote DATABASE_URL → %s\n", catalog.RedactDSN(dsn))
	return true
}

// bundledDSN is the bundled database as the app role: loopback, sslmode
// off, port/password from .env with the documented dev defaults.
func bundledDSN(port, password string) string {
	return bundledRoleDSN(appRole, port, password, "insideout_app_dev_password")
}

// bundledOwnerDSN is the same bundled database as insideout_owner, the
// role migrations run as (POSTGRES_OWNER_PASSWORD in docker-compose.yml).
func bundledOwnerDSN(port, password string) string {
	return bundledRoleDSN(ownerRole, port, password, "insideout_owner_dev_password")
}

func bundledRoleDSN(role, port, password, devPassword string) string {
	if port == "" {
		port = "5442"
	}
	if password == "" {
		password = devPassword
	}
	return fmt.Sprintf("postgres://%s:%s@127.0.0.1:%s/insideout?sslmode=disable", role, password, port)
}
