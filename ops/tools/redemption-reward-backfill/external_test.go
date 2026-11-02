// Copyright (C) 2026 QuantumNous
package main

import (
	"errors"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestExternalTransactions(t *testing.T) {
	driver, dsn := os.Getenv("BACKFILL_TEST_DRIVER"), os.Getenv("BACKFILL_TEST_DSN")
	if dsn == "" {
		t.Skip("explicit isolated database not supplied")
	}
	var dialect gorm.Dialector
	switch driver {
	case "postgres":
		u, err := url.Parse(dsn)
		require.NoError(t, err)
		require.Contains(t, []string{"127.0.0.1", "localhost"}, u.Hostname())
		require.Equal(t, "/invitation_test_backfill", u.Path)
		dialect = postgres.Open(dsn)
	case "mysql":
		require.Contains(t, dsn, "@tcp(127.0.0.1:")
		require.True(t, strings.Contains(dsn, ")/invitation_test_backfill?"))
		dialect = mysql.Open(dsn)
	default:
		t.Fatal("unknown isolated driver")
	}
	db, err := gorm.Open(dialect, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	raw, _ := db.DB()
	raw.SetMaxOpenConns(12)
	t.Cleanup(func() { raw.Close() })
	reset := func() {
		t.Helper()
		for _, table := range []any{&backfillRecord{}, &fixtureLog{}, &fixtureCode{}, &fixtureSub{}, &fixtureUser{}, &fixtureOption{}} {
			require.NoError(t, db.Migrator().DropTable(table))
		}
		seed(t, db)
	}
	t.Run("same_code_8_workers", func(t *testing.T) {
		reset()
		p, h := plan(t, db)
		var wg sync.WaitGroup
		errs := make(chan error, 8)
		start := make(chan struct{})
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); <-start; _, err := Apply(db, p, h); errs <- err }()
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		assertCredit(t, db, 25000, 1)
	})
	t.Run("8_codes_shared_inviter", func(t *testing.T) {
		reset()
		for i := 2; i <= 8; i++ {
			require.NoError(t, db.Create(&fixtureCode{ID: i, Key: "external-code", Status: 3, Quota: 500001, RedeemType: "quota", UsedUserID: 2, RedeemedTime: 100}).Error)
		}
		p, h := plan(t, db)
		var wg sync.WaitGroup
		errs := make(chan error, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); _, err := Apply(db, p, h); errs <- err }()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		assertCredit(t, db, 200000, 8)
		var n int64
		require.NoError(t, db.Model(&backfillRecord{}).Count(&n).Error)
		require.EqualValues(t, 8, n)
	})
	t.Run("log_failure_is_atomic", func(t *testing.T) {
		reset()
		p, h := plan(t, db)
		require.NoError(t, db.Callback().Create().Before("gorm:create").Register("fail-external-log", func(tx *gorm.DB) {
			if tx.Statement.Table == "logs" {
				tx.AddError(errors.New("injected log failure"))
			}
		}))
		_, err := Apply(db, p, h)
		require.Error(t, err)
		assertCredit(t, db, 0, 0)
		var n int64
		require.NoError(t, db.Model(&backfillRecord{}).Count(&n).Error)
		require.Zero(t, n)
		require.NoError(t, db.Callback().Create().Remove("fail-external-log"))
		_, err = Apply(db, p, h)
		require.NoError(t, err)
		assertCredit(t, db, 25000, 1)
	})
	if driver == "postgres" {
		t.Run("fallback_schema_cannot_hide_existing_ledger", func(t *testing.T) {
			reset()
			p, h := plan(t, db)
			_, err := Apply(db, p, h)
			require.NoError(t, err)
			require.NoError(t, db.Exec("DROP SCHEMA IF EXISTS backfill_empty CASCADE").Error)
			require.NoError(t, db.Exec("CREATE SCHEMA backfill_empty").Error)
			other, err := gorm.Open(postgres.Open(dsn+"&search_path=backfill_empty,public"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			require.NoError(t, err)
			raw, _ := other.DB()
			defer raw.Close()
			_, err = Preview(other, 200)
			require.Error(t, err)
			assertCredit(t, db, 25000, 1)
		})
		t.Run("same_database_different_schema_rejected", func(t *testing.T) {
			for _, schema := range []string{"backfill_source", "backfill_other"} {
				require.NoError(t, db.Exec("DROP SCHEMA IF EXISTS "+schema+" CASCADE").Error)
				require.NoError(t, db.Exec("CREATE SCHEMA "+schema).Error)
			}
			openSchema := func(schema string) *gorm.DB {
				d, err := gorm.Open(postgres.Open(dsn+"&search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
				require.NoError(t, err)
				raw, _ := d.DB()
				t.Cleanup(func() { raw.Close() })
				seed(t, d)
				return d
			}
			source, other := openSchema("backfill_source"), openSchema("backfill_other")
			p, h := plan(t, source)
			_, err := Apply(other, p, h)
			require.Error(t, err)
			assertCredit(t, other, 0, 0)
			require.False(t, other.Migrator().HasTable(ledgerTable))
		})
	}
	if driver == "mysql" {
		t.Run("nontransactional_source_rejected", func(t *testing.T) {
			reset()
			require.NoError(t, db.Exec("ALTER TABLE logs ENGINE=MyISAM").Error)
			p, h := plan(t, db)
			_, err := Apply(db, p, h)
			require.ErrorContains(t, err, "table logs must use InnoDB")
			assertCredit(t, db, 0, 0)
			require.False(t, db.Migrator().HasTable(ledgerTable))
		})
	}
}
