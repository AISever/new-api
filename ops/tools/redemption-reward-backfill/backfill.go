// Copyright (C) 2026 QuantumNous
package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math/big"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type Entry struct {
	ID           int    `json:"redemption_id"`
	RedeemedTime int64  `json:"redeemed_time"`
	InviteeID    int    `json:"invitee_id"`
	InviteeName  string `json:"invitee_username"`
	InviterID    int    `json:"inviter_id"`
	InviterName  string `json:"inviter_username"`
	Basis        int64  `json:"basis_quota"`
	Ratio        string `json:"ratio"`
	Reward       int64  `json:"reward_quota"`
	Reason       string `json:"skip_reason,omitempty"`
	FactHash     string `json:"fact_sha256"`
	Completed    bool   `json:"already_backfilled"`
}
type Plan struct {
	Target     string            `json:"database_target_sha256"`
	Version    int               `json:"version"`
	Before     int64             `json:"before_exclusive"`
	Settings   map[string]string `json:"settings"`
	Entries    []Entry           `json:"entries"`
	TotalQuota string            `json:"pending_reward_quota"`
}

const ledgerTable = "redemption_reward_backfills"

// First start of the verified production image with transactional redemption rewards.
// A later cutoff could repay a successful reward whose post-commit log was lost.
const latestSafeCutoff int64 = 1793589890

var settingKeys = []string{"InviteRewardMode", "InviteRewardRatio", "QuotaPerUnit", "general_setting.quota_display_type", "USDExchangeRate", "general_setting.custom_currency_symbol", "general_setting.custom_currency_exchange_rate"}

type sourceCode struct {
	ID                     int
	Status                 int
	Quota                  int64
	RedeemType             string
	UsedUserID             int
	RedeemedTime           int64
	RedeemedSubscriptionID int
	DeletedAt              gorm.DeletedAt
}
type sourceUser struct {
	ID                int
	Username          string
	InviterID         int
	Status            int
	InviteRewardRatio *float64
	DeletedAt         gorm.DeletedAt
}
type sourceSub struct {
	ID          int
	UserID      int
	AmountTotal int64
}
type sourceOption struct {
	Key   string
	Value string
}
type facts struct {
	Code         sourceCode
	Invitee      *sourceUser
	Inviter      *sourceUser
	Subscription *sourceSub
	NativeReward bool
}
type backfillRecord struct {
	RedemptionID int `gorm:"primaryKey;autoIncrement:false"`
	InviterID    int
	InviteeID    int
	RewardQuota  int64
	Ratio        string `gorm:"size:64"`
	FactHash     string `gorm:"size:64"`
	PlanHash     string `gorm:"size:64"`
	LogID        int
	CreatedAt    int64
}

func (backfillRecord) TableName() string { return ledgerTable }

type rewardLog struct {
	ID        int
	UserID    int
	Username  string
	CreatedAt int64
	Type      int
	Content   string
	Quota     int64
	RequestID string
	Other     string
}

func (rewardLog) TableName() string { return "logs" }

func Digest(v any) (string, error) {
	b, err := common.Marshal(v)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
func loadSettings(db *gorm.DB, lock bool) (map[string]string, error) {
	var rows []sourceOption
	q := db.Table("options").Select("key", "value").Where(map[string]any{"key": settingKeys}).Order(clause.OrderByColumn{Column: clause.Column{Name: "key"}})
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, r := range rows {
		result[r.Key] = r.Value
	}
	if mode := value(result, "InviteRewardMode", "fixed"); mode != "ratio" && mode != "fixed" {
		return nil, errors.New("invalid reward mode")
	}
	if _, err := ratioDecimal(value(result, "InviteRewardRatio", "0")); err != nil {
		return nil, err
	}
	return result, nil
}
func value(m map[string]string, key, fallback string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return fallback
}
func ratioDecimal(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil || d.IsNegative() || d.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.Zero, errors.New("invalid invitation ratio")
	}
	return d, nil
}

var nativeID = regexp.MustCompile(`兑换码ID\s*(\d+)`)

func nativeRewards(db *gorm.DB) (map[int]bool, error) {
	var logs []rewardLog
	if err := db.Select("user_id", "content").Where("type = ? AND content LIKE ?", 4, "邀请兑换返奖 %").Find(&logs).Error; err != nil {
		return nil, err
	}
	ids := map[int]bool{}
	for _, l := range logs {
		match := nativeID.FindStringSubmatch(l.Content)
		if len(match) != 2 {
			return nil, errors.New("native reward log lacks a redemption ID; manual reconciliation required")
		}
		id, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, nil
}
func getFacts(db *gorm.DB, c sourceCode, native bool, lock bool) (facts, error) {
	f := facts{Code: c, NativeReward: native}
	getUser := func(id int) (*sourceUser, error) {
		var rows []sourceUser
		q := db.Unscoped().Table("users").Select("id", "username", "inviter_id", "status", "invite_reward_ratio", "deleted_at").Where("id = ?", id)
		if lock {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, nil
		}
		return &rows[0], nil
	}
	var err error
	f.Invitee, err = getUser(c.UsedUserID)
	if err != nil {
		return f, err
	}
	if f.Invitee != nil && f.Invitee.InviterID > 0 {
		f.Inviter, err = getUser(f.Invitee.InviterID)
		if err != nil {
			return f, err
		}
	}
	if c.RedeemType == "subscription" {
		var rows []sourceSub
		q := db.Table("user_subscriptions").Select("id", "user_id", "amount_total").Where("id = ?", c.RedeemedSubscriptionID)
		if lock {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.Find(&rows).Error; err != nil {
			return f, err
		}
		if len(rows) > 0 {
			f.Subscription = &rows[0]
		}
	}
	return f, nil
}
func entryFor(f facts, settings map[string]string) (Entry, error) {
	e := Entry{ID: f.Code.ID, RedeemedTime: f.Code.RedeemedTime, InviteeID: f.Code.UsedUserID}
	var err error
	e.FactHash, err = Digest(f)
	if err != nil {
		return e, err
	}
	if f.Invitee != nil {
		e.InviteeName = f.Invitee.Username
		e.InviterID = f.Invitee.InviterID
	}
	if f.Inviter != nil {
		e.InviterName = f.Inviter.Username
	}
	switch {
	case f.Code.RedeemedTime <= 0:
		e.Reason = "invalid_redeemed_time"
	case f.NativeReward:
		e.Reason = "existing_reward"
	case value(settings, "InviteRewardMode", "fixed") != "ratio":
		e.Reason = "fixed_mode"
	case f.Invitee == nil || f.Invitee.DeletedAt.Valid:
		e.Reason = "invitee_unavailable"
	case e.InviterID <= 0:
		e.Reason = "no_inviter"
	case e.InviterID == e.InviteeID:
		e.Reason = "self_invitation"
	case f.Inviter == nil || f.Inviter.DeletedAt.Valid || f.Inviter.Status != 1:
		e.Reason = "inviter_unavailable"
	}
	if e.Reason != "" {
		return e, nil
	}
	switch f.Code.RedeemType {
	case "", "quota":
		e.Basis = f.Code.Quota
	case "subscription":
		if f.Subscription == nil || f.Subscription.UserID != e.InviteeID {
			e.Reason = "missing_subscription_basis"
			return e, nil
		}
		e.Basis = f.Subscription.AmountTotal
	default:
		e.Reason = "unknown_redemption_type"
		return e, nil
	}
	if e.Basis <= 0 {
		e.Reason = "nonpositive_basis"
		return e, nil
	}
	e.Ratio = value(settings, "InviteRewardRatio", "0")
	if f.Inviter.InviteRewardRatio != nil {
		e.Ratio = strconv.FormatFloat(*f.Inviter.InviteRewardRatio, 'f', -1, 64)
	}
	ratio, err := ratioDecimal(e.Ratio)
	if err != nil {
		return e, err
	}
	e.Ratio = ratio.String()
	if ratio.IsZero() {
		e.Reason = "zero_ratio"
		return e, nil
	}
	d := decimal.NewFromInt(e.Basis).Mul(ratio).Truncate(0)
	if d.GreaterThan(decimal.NewFromInt(1<<63 - 1)) {
		return e, errors.New("reward overflow")
	}
	e.Reward = d.IntPart()
	if e.Reward == 0 {
		e.Reason = "rounds_to_zero"
	}
	return e, nil
}
func Preview(db *gorm.DB, before int64) (Plan, error) {
	p := Plan{Version: 1, Before: before, Entries: []Entry{}, TotalQuota: "0"}
	if before <= 0 || before > latestSafeCutoff {
		return p, errors.New("cutoff must be positive and no later than the first reward-capable production start")
	}
	var err error
	p.Target, err = databaseTarget(db)
	if err != nil {
		return p, err
	}
	p.Settings, err = loadSettings(db, false)
	if err != nil {
		return p, err
	}
	native, err := nativeRewards(db)
	if err != nil {
		return p, err
	}
	var rows []sourceCode
	if err = db.Unscoped().Table("redemptions").Select("id", "status", "quota", "redeem_type", "used_user_id", "redeemed_time", "redeemed_subscription_id", "deleted_at").Where("status = ? AND redeemed_time < ?", 3, before).Order("id").Find(&rows).Error; err != nil {
		return p, err
	}
	completed := map[int]bool{}
	if db.Migrator().HasTable(ledgerTable) {
		var records []backfillRecord
		if err = db.Find(&records).Error; err != nil {
			return p, err
		}
		for _, r := range records {
			completed[r.RedemptionID] = true
		}
	}
	total := new(big.Int)
	for _, c := range rows {
		f, err := getFacts(db, c, native[c.ID], false)
		if err != nil {
			return p, err
		}
		e, err := entryFor(f, p.Settings)
		if err != nil {
			return p, err
		}
		e.Completed = completed[e.ID]
		p.Entries = append(p.Entries, e)
		if e.Reason == "" && !e.Completed {
			total.Add(total, big.NewInt(e.Reward))
		}
	}
	p.TotalQuota = total.String()
	return p, nil
}

// Used by the apply transaction and ordinary user log API; no raw code is logged.
func displayQuota(amount decimal.Decimal, settings map[string]string) (string, error) {
	unit, err := decimal.NewFromString(value(settings, "QuotaPerUnit", "500000"))
	if err != nil || !unit.IsPositive() {
		return "", errors.New("invalid quota display unit")
	}
	currency := value(settings, "general_setting.quota_display_type", "USD")
	symbol := "$"
	rate := decimal.NewFromInt(1)
	switch currency {
	case "USD":
	case "CNY":
		symbol = "¥"
		rate, err = decimal.NewFromString(value(settings, "USDExchangeRate", "7.3"))
	case "CUSTOM":
		symbol = value(settings, "general_setting.custom_currency_symbol", "¤")
		rate, err = decimal.NewFromString(value(settings, "general_setting.custom_currency_exchange_rate", "1"))
	case "TOKENS":
		symbol = ""
	default:
		return "", errors.New("invalid currency display type")
	}
	if err != nil || !rate.IsPositive() {
		return "", errors.New("invalid currency display rate")
	}
	display := fmt.Sprintf("%s%s", symbol, amount.Div(unit).Mul(rate).StringFixed(6))
	if currency == "TOKENS" {
		display = amount.String() + " 点额度"
	}
	return display, nil
}

func logContent(e Entry, settings map[string]string) (string, error) {
	display, err := displayQuota(decimal.NewFromInt(e.Reward), settings)
	if err != nil {
		return "", err
	}
	at := time.Unix(e.RedeemedTime, 0).In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02 15:04:05")
	return fmt.Sprintf("历史兑换返奖补发 %s，兑换码ID %d，受邀用户 %s，奖励额度 %d，原兑换时间 %s（北京时间）", display, e.ID, e.InviteeName, e.Reward, at), nil
}

type ApplyResult struct {
	Credited         int `json:"credited"`
	AlreadyCompleted int `json:"already_completed"`
	Skipped          int `json:"skipped"`
}

var schemaMu sync.Mutex

func ensureLedger(db *gorm.DB) error {
	schemaMu.Lock()
	defer schemaMu.Unlock()
	if !db.Migrator().HasTable(ledgerTable) {
		migrator := db
		if db.Dialector.Name() == "mysql" {
			migrator = db.Set("gorm:table_options", "ENGINE=InnoDB")
		}
		if err := migrator.Migrator().CreateTable(&backfillRecord{}); err != nil && !db.Migrator().HasTable(ledgerTable) {
			return err
		}
	}
	columns, err := db.Migrator().ColumnTypes(&backfillRecord{})
	if err != nil {
		return err
	}
	names := map[string]bool{}
	pk := false
	for _, c := range columns {
		names[c.Name()] = true
		if c.Name() == "redemption_id" {
			is, ok := c.PrimaryKey()
			pk = ok && is
		}
	}
	for _, key := range []string{"redemption_id", "inviter_id", "invitee_id", "reward_quota", "ratio", "fact_hash", "plan_hash", "log_id", "created_at"} {
		if !names[key] {
			return errors.New("incompatible existing backfill ledger")
		}
	}
	if !pk {
		return errors.New("backfill ledger requires a primary key on redemption_id")
	}
	return nil
}
func equal(a, b any) bool {
	x, err := Digest(a)
	if err != nil {
		return false
	}
	y, err := Digest(b)
	return err == nil && x == y
}
func sameEntry(a, b Entry) bool { a.Completed = false; b.Completed = false; return equal(a, b) }
func validatePlan(db *gorm.DB, p Plan, hash string) error {
	actual, err := Digest(p)
	if err != nil || hash != actual || len(hash) != 64 {
		return errors.New("plan SHA256 mismatch")
	}
	if p.Version != 1 || p.Before <= 0 {
		return errors.New("unsupported or invalid plan")
	}
	total := new(big.Int)
	last := 0
	for _, e := range p.Entries {
		if e.ID <= last {
			return errors.New("duplicate or unsorted redemption IDs")
		}
		last = e.ID
		if e.Reason == "" && !e.Completed {
			total.Add(total, big.NewInt(e.Reward))
		}
		if e.Reason == "" {
			if _, err := logContent(e, p.Settings); err != nil {
				return err
			}
		}
	}
	if total.String() != p.TotalQuota {
		return errors.New("plan total mismatch")
	}
	current, err := Preview(db, p.Before)
	if err != nil {
		return err
	}
	if current.Target != p.Target || !equal(current.Settings, p.Settings) || len(current.Entries) != len(p.Entries) {
		return errors.New("source/settings changed; create a fresh preview")
	}
	for i, e := range p.Entries {
		now := current.Entries[i]
		if !sameEntry(e, now) || (e.Completed && !now.Completed) {
			return fmt.Errorf("redemption %d changed; create a fresh preview", e.ID)
		}
	}
	return nil
}
func Apply(db *gorm.DB, p Plan, hash string) (ApplyResult, error) {
	result := ApplyResult{}
	if err := checkTransactionalTables(db); err != nil {
		return result, err
	}
	if err := validatePlan(db, p, hash); err != nil {
		return result, err
	}
	eligible := false
	for _, e := range p.Entries {
		if e.Reason == "" {
			eligible = true
		}
	}
	if !eligible {
		result.Skipped = len(p.Entries)
		return result, nil
	}
	if err := ensureLedger(db); err != nil {
		return result, err
	}
	if err := checkTransactionalTables(db); err != nil {
		return result, err
	}
	for _, expected := range p.Entries {
		if expected.Reason != "" {
			result.Skipped++
			continue
		}
		completed := false
		err := db.Transaction(func(tx *gorm.DB) error {
			settings, err := loadSettings(tx, true)
			if err != nil {
				return err
			}
			if !equal(settings, p.Settings) {
				return errors.New("settings changed during apply")
			}
			var codes []sourceCode
			if err := tx.Unscoped().Table("redemptions").Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "status", "quota", "redeem_type", "used_user_id", "redeemed_time", "redeemed_subscription_id", "deleted_at").Where("id = ? AND status = ? AND redeemed_time < ?", expected.ID, 3, p.Before).Find(&codes).Error; err != nil {
				return err
			}
			if len(codes) != 1 {
				return errors.New("redemption state changed")
			}
			var previous []backfillRecord
			if err := tx.Where("redemption_id = ?", expected.ID).Find(&previous).Error; err != nil {
				return err
			}
			if len(previous) > 0 {
				completed = true
				return nil
			}
			native, err := nativeRewards(tx)
			if err != nil {
				return err
			}
			f, err := getFacts(tx, codes[0], native[expected.ID], true)
			if err != nil {
				return err
			}
			entry, err := entryFor(f, settings)
			if err != nil {
				return err
			}
			if !sameEntry(entry, expected) {
				return errors.New("redemption eligibility changed during apply")
			}
			if entry.Reason != "" || entry.Reward <= 0 {
				return errors.New("refusing nonpositive or ineligible credit")
			}
			content, err := logContent(entry, settings)
			if err != nil {
				return err
			}
			record := backfillRecord{RedemptionID: entry.ID, InviterID: entry.InviterID, InviteeID: entry.InviteeID, RewardQuota: entry.Reward, Ratio: entry.Ratio, FactHash: entry.FactHash, PlanHash: hash, CreatedAt: time.Now().Unix()}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			const max int64 = 1<<63 - 1
			update := tx.Table("users").Where("id = ? AND status = ? AND deleted_at IS NULL AND aff_quota <= ? AND aff_history <= ?", entry.InviterID, 1, max-entry.Reward, max-entry.Reward).Updates(map[string]any{"aff_quota": gorm.Expr("aff_quota + ?", entry.Reward), "aff_history": gorm.Expr("aff_history + ?", entry.Reward)})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected != 1 {
				return errors.New("reward counters overflow, are NULL, or inviter is unavailable")
			}
			log := rewardLog{UserID: entry.InviterID, Username: entry.InviterName, CreatedAt: record.CreatedAt, Type: 4, Content: content, RequestID: fmt.Sprintf("redemption-backfill:%d", entry.ID), Other: "{}"}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
			return tx.Model(&backfillRecord{}).Where("redemption_id = ?", entry.ID).Update("log_id", log.ID).Error
		})
		if err != nil {
			return result, fmt.Errorf("redemption %d failed; committed earlier codes remain safely resumable: %w", expected.ID, err)
		}
		if completed {
			result.AlreadyCompleted++
		} else {
			result.Credited++
		}
	}
	return result, nil
}

func checkTransactionalTables(db *gorm.DB) error {
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	var rows []struct {
		TableName string
		Engine    string
	}
	tables := []string{"users", "redemptions", "user_subscriptions", "options", "logs", ledgerTable}
	if err := db.Table("information_schema.tables").Select("TABLE_NAME AS table_name, ENGINE AS engine").Where("table_schema = DATABASE()").Where("table_name IN ?", tables).Find(&rows).Error; err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.TableName] = true
		if r.Engine != "InnoDB" {
			return fmt.Errorf("table %s must use InnoDB for atomic rewards and logs", r.TableName)
		}
	}
	for _, table := range tables[:5] {
		if !seen[table] {
			return fmt.Errorf("missing required table %s", table)
		}
	}
	return nil
}
