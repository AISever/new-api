// Copyright (C) 2026 QuantumNous
package main

import (
	"bytes"
	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIReadOnlyPreviewAndExplicitApply(t *testing.T) {
	db := seedDB(t)
	var database string
	require.NoError(t, db.Raw("PRAGMA database_list").Row().Scan(new(int), new(string), &database))
	t.Setenv("BACKFILL_DSN", database)
	t.Setenv("LOG_SQL_DSN", "")
	output := filepath.Join(t.TempDir(), "plan.json")
	var stdout bytes.Buffer
	require.NoError(t, run([]string{"--driver", "sqlite", "--before", "200", "--out", output}, &stdout))
	require.Contains(t, stdout.String(), "preview")
	require.Contains(t, stdout.String(), `"pending_reward_display":"$0.050000"`)
	require.NotContains(t, stdout.String(), "SECRET")
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	require.NotContains(t, string(data), "SECRET")
	info, _ := os.Stat(output)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	require.False(t, db.Migrator().HasTable(ledgerTable))
	assertCredit(t, db, 0, 0)
	var p Plan
	require.NoError(t, common.Unmarshal(data, &p))
	h, _ := Digest(p)
	require.Error(t, run([]string{"--driver", "sqlite", "--before", "200", "--out", output}, &stdout))
	require.Error(t, run([]string{"--driver", "sqlite", "--apply", output}, &stdout))
	require.False(t, db.Migrator().HasTable(ledgerTable))
	require.NoError(t, run([]string{"--driver", "sqlite", "--apply", output, "--sha256", h}, &stdout))
	assertCredit(t, db, 25000, 1)
}
func TestCLIRejectsUnsafeInputs(t *testing.T) {
	for _, args := range [][]string{nil, {"--driver", "postgres", "--before", "200"}, {"--driver", "sqlite", "--before", "-1", "--out", "/not/written"}, {"--driver", "sqlite", "--before", "9999999999", "--out", "/not/written"}, {"--driver", "other", "--before", "200", "--out", "/not/written"}, {"--driver", "sqlite", "--apply", "missing", "--sha256", "x", "--before", "200"}} {
		t.Setenv("BACKFILL_DSN", "postgres://user:DO-NOT-PRINT@127.0.0.1:1/private")
		t.Setenv("LOG_SQL_DSN", "")
		var stdout bytes.Buffer
		err := run(args, &stdout)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "DO-NOT-PRINT")
		require.NotContains(t, stdout.String(), "DO-NOT-PRINT")
	}
	t.Setenv("LOG_SQL_DSN", "separate")
	require.Error(t, run([]string{"--driver", "sqlite", "--before", "200", "--out", "unused"}, &bytes.Buffer{}))
}
