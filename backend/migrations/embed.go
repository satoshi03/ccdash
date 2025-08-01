package migrations

import "embed"

// FS embeds all migration files
//
//go:embed *.sql
var FS embed.FS