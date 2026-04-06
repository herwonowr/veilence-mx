// Package migrations provides embedded SQL migration files for golang-migrate.
package migrations

import "embed"

// FS contains the embedded migration SQL files.
//
//go:embed *.sql
var FS embed.FS
