// Package db embeds the SQL migration files so the server binary is
// self-contained — no separate migration tool or file deployment needed.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
