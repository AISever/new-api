// Copyright (C) 2026 QuantumNous
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"os"
	"strings"
)

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("redemption-reward-backfill", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	driver := fs.String("driver", "", "postgres, mysql or sqlite")
	before := fs.Int64("before", 0, "exclusive historical cutoff, at most 1793589890")
	output := fs.String("out", "", "new private preview plan file")
	apply := fs.String("apply", "", "apply an existing reviewed plan")
	hash := fs.String("sha256", "", "exact preview file SHA256")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 {
		return errors.New("invalid arguments; use --driver and either --before/--out or --apply/--sha256")
	}
	if os.Getenv("LOG_SQL_DSN") != "" {
		return errors.New("separate log database unsupported: atomic reward and log delivery require the same database")
	}
	if *apply == "" {
		if *before <= 0 || *before > latestSafeCutoff || *output == "" || *hash != "" {
			return errors.New("preview requires --before (<=1793589890) and --out; it never applies")
		}
	} else if *before != 0 || *output != "" || len(*hash) != 64 {
		return errors.New("apply requires only --apply and --sha256, without preview flags")
	}
	if *driver != "sqlite" && *driver != "postgres" && *driver != "mysql" {
		return errors.New("explicit supported --driver required")
	}
	dsn := os.Getenv("BACKFILL_DSN")
	if dsn == "" {
		return errors.New("BACKFILL_DSN environment variable required; DSNs are never accepted as arguments")
	}
	var reviewed Plan
	if *apply != "" {
		info, err := os.Stat(*apply)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 16<<20 {
			return errors.New("cannot read regular plan file (maximum 16 MiB)")
		}
		if info.Mode().Perm()&0077 != 0 {
			return errors.New("plan contains private account data; permissions must be 0600 or stricter")
		}
		data, err := os.ReadFile(*apply)
		if err != nil {
			return errors.New("cannot read plan")
		}
		if err = common.Unmarshal(data, &reviewed); err != nil {
			return errors.New("invalid plan JSON")
		}
		actual, err := Digest(reviewed)
		if err != nil || actual != *hash {
			return errors.New("plan SHA256 mismatch")
		}
	}
	var dialect gorm.Dialector
	switch *driver {
	case "postgres":
		dialect = postgres.Open(dsn)
	case "mysql":
		dialect = mysql.Open(dsn)
	case "sqlite":
		path := strings.Split(dsn, "?")[0]
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("SQLite database must already exist; preview will not create a database")
		}
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dialect = sqlite.Open(dsn + sep + "_pragma=busy_timeout(5000)")
	}
	db, err := gorm.Open(dialect, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return errors.New("database connection failed; check private environment configuration")
	}
	raw, err := db.DB()
	if err != nil {
		return errors.New("database connection unavailable")
	}
	defer raw.Close()
	if *driver == "sqlite" {
		raw.SetMaxOpenConns(1)
	} else {
		raw.SetMaxOpenConns(4)
	}
	if *apply != "" {
		result, err := Apply(db, reviewed, *hash)
		report, _ := common.Marshal(map[string]any{"mode": "apply", "result": result, "success": err == nil})
		if _, writeErr := fmt.Fprintln(out, string(report)); writeErr != nil {
			return writeErr
		}
		return err
	}
	p, err := Preview(db, *before)
	if err != nil {
		return err
	}
	data, err := common.Marshal(p)
	if err != nil {
		return err
	}
	digest, err := Digest(p)
	if err != nil {
		return err
	}
	total, err := decimal.NewFromString(p.TotalQuota)
	if err != nil {
		return err
	}
	display, err := displayQuota(total, p.Settings)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("cannot create a new plan; existing files are never overwritten")
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		os.Remove(*output)
		return errors.New("cannot persist complete plan")
	}
	pending := 0
	for _, e := range p.Entries {
		if e.Reason == "" && !e.Completed {
			pending++
		}
	}
	report, _ := common.Marshal(map[string]any{"mode": "preview", "sha256": digest, "historical_codes": len(p.Entries), "pending_codes": pending, "pending_reward_quota": p.TotalQuota, "pending_reward_display": display, "database_writes": false})
	_, err = fmt.Fprintln(out, string(report))
	return err
}
func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
