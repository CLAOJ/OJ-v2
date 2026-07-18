package db

import (
	"regexp"

	"gorm.io/gorm"
)

var ddlRe = regexp.MustCompile(`(?is)^\s*(CREATE|ALTER|DROP|TRUNCATE|RENAME)\b`)

// RegisterDDLGuard panics on any DDL statement. The schema of the shared
// database is owned by Django migrations; OJ-v2 must only touch rows.
func RegisterDDLGuard(g *gorm.DB) {
	check := func(tx *gorm.DB) {
		if ddlRe.MatchString(tx.Statement.SQL.String()) {
			panic("DDL blocked (schema is Django-owned): " + tx.Statement.SQL.String())
		}
	}
	_ = g.Callback().Raw().Before("gorm:raw").Register("claoj:ddl_guard", check)
}
