// Copyright (C) 2026 QuantumNous
package main

import (
	"errors"
	"github.com/glebarez/sqlite"
	sqlmysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"path/filepath"
	"strconv"
	"strings"
)

// Bind reviewed plans to a connection destination, without hashing or saving secrets.
func databaseTarget(db *gorm.DB) (string, error) {
	switch d := db.Dialector.(type) {
	case *postgres.Dialector:
		c, err := pgconn.ParseConfig(d.Config.DSN)
		if err != nil {
			return "", errors.New("invalid PostgreSQL target configuration")
		}
		var database, role, path, currentSchema string
		if err := db.Raw("SELECT pg_catalog.current_database(), current_user, pg_catalog.current_schemas(false)::text, current_schema").Row().Scan(&database, &role, &path, &currentSchema); err != nil {
			return "", err
		}
		// Unqualified source tables and ledger resolve through this effective path.
		identity := []string{"postgres", c.Host, strconv.Itoa(int(c.Port)), database, role, path}
		for _, table := range []string{"users", "redemptions", "user_subscriptions", "options", "logs"} {
			var resolved string
			if err := db.Raw("SELECT coalesce(pg_catalog.to_regclass(?)::oid::text,'')", table).Scan(&resolved).Error; err != nil {
				return "", err
			}
			if resolved == "" {
				return "", errors.New("missing required source relation")
			}
			identity = append(identity, resolved)
		}
		// GORM's HasTable uses current_schema, whereas SQL name lookup may fall
		// back to later schemas. Reject that split so an existing ledger cannot
		// be overlooked and replaced with an empty ledger in an earlier schema.
		for _, table := range []string{"users", "redemptions", "user_subscriptions", "options", "logs", ledgerTable} {
			var schema string
			if err := db.Raw("SELECT n.nspname FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE c.oid=pg_catalog.to_regclass(?)", table).Scan(&schema).Error; err != nil {
				return "", err
			}
			if schema != "" && schema != currentSchema {
				return "", errors.New("source tables and existing ledger must reside in current_schema; fallback schemas are not supported")
			}
		}
		return Digest(identity)
	case *mysql.Dialector:
		c, err := sqlmysql.ParseDSN(d.Config.DSN)
		if err != nil {
			return "", errors.New("invalid MySQL target configuration")
		}
		var database string
		if err := db.Raw("SELECT DATABASE()").Scan(&database).Error; err != nil {
			return "", err
		}
		return Digest([]string{"mysql", c.Net, c.Addr, database})
	case *sqlite.Dialector:
		name := strings.TrimPrefix(strings.Split(d.DSN, "?")[0], "file:")
		if name == "" || name == ":memory:" {
			return "", errors.New("backfill requires a named persistent SQLite database")
		}
		path, err := filepath.Abs(name)
		if err != nil {
			return "", errors.New("invalid SQLite target path")
		}
		return Digest([]string{"sqlite", path})
	default:
		return "", errors.New("unsupported database target")
	}
}
