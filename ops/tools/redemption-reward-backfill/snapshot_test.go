// Copyright (C) 2026 QuantumNous
package main

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// Only a private minimal SELECT export is imported; no production DSN is accepted.
func TestPrivateSnapshotPreview(t *testing.T) {
	file, dsn := os.Getenv("BACKFILL_SNAPSHOT_FILE"), os.Getenv("BACKFILL_SNAPSHOT_DSN")
	if file == "" || dsn == "" {
		t.Skip("private snapshot preview not requested")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"127.0.0.1", "localhost"}, u.Hostname())
	require.Equal(t, "/invitation_snapshot_backfill", u.Path)
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	var snapshot struct {
		Codes         []fixtureCode
		Users         []fixtureUser
		Subscriptions []fixtureSub
		Options       []fixtureOption
		Logs          []fixtureLog
	}
	require.NoError(t, common.Unmarshal(data, &snapshot))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	for _, table := range []any{&backfillRecord{}, &fixtureLog{}, &fixtureCode{}, &fixtureSub{}, &fixtureUser{}, &fixtureOption{}} {
		require.NoError(t, db.Migrator().DropTable(table))
	}
	require.NoError(t, db.AutoMigrate(&fixtureUser{}, &fixtureCode{}, &fixtureSub{}, &fixtureOption{}, &fixtureLog{}))
	for _, rows := range []any{&snapshot.Codes, &snapshot.Users, &snapshot.Subscriptions, &snapshot.Options} {
		require.NoError(t, db.Create(rows).Error)
	}
	if len(snapshot.Logs) > 0 {
		require.NoError(t, db.Create(&snapshot.Logs).Error)
	}
	p, err := Preview(db, latestSafeCutoff)
	require.NoError(t, err)
	require.Len(t, p.Entries, 69)
	require.Equal(t, "165625000", p.TotalQuota)
	codes := 0
	owners := map[int]bool{}
	for _, e := range p.Entries {
		if e.Reason == "" && !e.Completed {
			codes++
			owners[e.InviterID] = true
		}
	}
	require.Equal(t, 34, codes)
	require.Len(t, owners, 2)
	require.False(t, db.Migrator().HasTable(ledgerTable))
	encoded, err := common.Marshal(p)
	require.NoError(t, err)
	out := filepath.Join(filepath.Dir(file), "production-snapshot-preview.json")
	require.NoError(t, os.WriteFile(out, encoded, 0600))
	total, err := decimal.NewFromString(p.TotalQuota)
	require.NoError(t, err)
	display, err := displayQuota(total, p.Settings)
	require.NoError(t, err)
	require.Equal(t, "$331.250000", display)
	report := map[string]any{"historical_codes": len(p.Entries), "pending_codes": codes, "beneficiary_count": len(owners), "pending_reward_quota": p.TotalQuota, "pending_reward_display": display, "source": "minimal read-only production SELECT export in isolated local copy", "production_apply_performed": false, "local_preview_database_writes": false, "plan_bound_to_local_copy_not_production": true}
	encoded, err = common.Marshal(report)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(file), "snapshot-preview-summary.json"), encoded, 0600))
}
