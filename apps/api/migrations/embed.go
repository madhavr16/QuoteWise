// Package migrations embeds the SQL schema migrations (goose format,
// sequential numbering, up-only by policy — ADR-016). They are applied
// by the store layer at startup; rollbacks are deliberate, manual
// operations requiring explicit approval.
package migrations

import "embed"

// FS holds every SQL migration in this directory. Files are embedded
// at the FS root (no directory prefix), so the goose directory
// argument is ".".
//
//go:embed *.sql
var FS embed.FS
