package model

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Explicit local-only opt-in; never accept a production address/database by mistake.
func TestInvitationAmountsExternalDatabase(t *testing.T) {
	driver, dsn := os.Getenv("INVITATION_TEST_DRIVER"), os.Getenv("INVITATION_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated external database not requested")
	}
	var dialector gorm.Dialector
	switch driver {
	case "postgres":
		parsed, err := url.Parse(dsn)
		require.NoError(t, err)
		require.Contains(t, []string{"127.0.0.1", "localhost"}, parsed.Hostname())
		require.True(t, strings.HasPrefix(strings.TrimPrefix(parsed.Path, "/"), "invitation_test"))
		dialector = postgres.Open(dsn)
	case "mysql":
		require.True(t, strings.Contains(dsn, "@tcp(127.0.0.1:") && strings.Contains(dsn, ")/invitation_test"), "only local invitation_test databases are allowed")
		dialector = mysql.Open(dsn)
	default:
		t.Fatal("unknown isolated test driver")
	}
	oldDB, oldLog := DB, LOG_DB
	oldSQLite, oldPostgres, oldMySQL := common.UsingSQLite, common.UsingPostgreSQL, common.UsingMySQL
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	DB = db
	LOG_DB = db
	common.UsingSQLite = false
	common.UsingPostgreSQL = driver == "postgres"
	common.UsingMySQL = driver == "mysql"
	t.Cleanup(func() {
		raw, _ := db.DB()
		_ = raw.Close()
		DB = oldDB
		LOG_DB = oldLog
		common.UsingSQLite = oldSQLite
		common.UsingPostgreSQL = oldPostgres
		common.UsingMySQL = oldMySQL
	})
	require.NoError(t, db.AutoMigrate(&User{}, &Option{}, &TopUp{}, &SubscriptionOrder{}, &Redemption{}, &UserSubscription{}, &InvitationAmountSnapshot{}))
	// Re-running the isolated fixture must not depend on previous generated users.
	for _, table := range []any{&InvitationAmountSnapshot{}, &UserSubscription{}, &SubscriptionOrder{}, &TopUp{}, &Redemption{}, &Option{}, &User{}} {
		require.NoError(t, db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(table).Error)
	}
	// A second migration must preserve the same schema/data and unique snapshot key.
	require.NoError(t, db.AutoMigrate(&InvitationAmountSnapshot{}))
	require.NoError(t, db.Create(&User{Id: 1, Username: "owner", AffCode: "db-owner", Quota: 9876}).Error)
	require.NoError(t, db.Create(&User{Id: 2, Username: "friend", AffCode: "db-friend", InviterId: 1}).Error)
	require.NoError(t, db.Create(&TopUp{Id: 1, UserId: 2, Money: 99, TradeNo: "external-paid", PaymentProvider: PaymentProviderStripe, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationPayment("external-paid", PaymentProviderStripe, "2.10", "USD"))
	require.NoError(t, RecordInvitationPayment("external-paid", PaymentProviderStripe, "2.10", "USD"))
	require.NoError(t, db.Create(&SubscriptionOrder{Id: 1, UserId: 2, Money: 7, TradeNo: "external-sub", PaymentProvider: PaymentProviderStripe, Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, db.Create(&TopUp{Id: 2, UserId: 2, Money: 7, TradeNo: "external-sub", Status: common.TopUpStatusSuccess}).Error)
	require.NoError(t, RecordInvitationPayment("external-sub", PaymentProviderStripe, "3.20", "USD"))
	require.NoError(t, UpdateOption(InvitationVisibilityOption, `{"mode":"selected","user_ids":[1]}`))
	require.True(t, CanViewInvitationAmounts(1))
	require.False(t, CanViewInvitationAmounts(2))
	page, err := GetInvitationPage(1, 1, 20, "", true)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "5.3", page.Items[0].Amounts.Currencies[0].Total)
	var user User
	require.NoError(t, db.First(&user, 1).Error)
	require.Equal(t, 9876, user.Quota)
	t.Run("redemption_reward_once_under_contention", testRedemptionInviteRewardExternal)
}

func testRedemptionInviteRewardExternal(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Log{}, &SubscriptionPlan{}))
	oldMode, oldRatio := common.InviteRewardMode, common.InviteRewardRatio
	common.InviteRewardMode, common.InviteRewardRatio = "ratio", 0.25
	t.Cleanup(func() { common.InviteRewardMode, common.InviteRewardRatio = oldMode, oldRatio })
	require.NoError(t, DB.Create(&User{Id: 101, Username: "reward-owner", AffCode: "reward-owner", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, DB.Create(&User{Id: 102, Username: "reward-friend", AffCode: "reward-friend", InviterId: 101, Status: common.UserStatusEnabled}).Error)
	r := &Redemption{Key: "external-reward-race", Quota: 500001, Status: common.RedemptionCodeStatusEnabled}
	require.NoError(t, DB.Create(r).Error)
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := Redeem(r.Key, 102); err == nil {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	require.EqualValues(t, 1, wins.Load())
	var owner, friend User
	require.NoError(t, DB.First(&owner, 101).Error)
	require.NoError(t, DB.First(&friend, 102).Error)
	require.Equal(t, 125000, owner.AffQuota)
	require.Equal(t, 125000, owner.AffHistoryQuota)
	require.Zero(t, owner.Quota)
	require.Equal(t, 500001, friend.Quota)
	var snapshotCount int64
	require.NoError(t, DB.Model(&InvitationAmountSnapshot{}).Where("source_kind = ? AND source_id = ?", "redemption", r.Id).Count(&snapshotCount).Error)
	require.EqualValues(t, 1, snapshotCount)
	t.Run("distinct_codes_same_inviter", testRedemptionDistinctCodesExternal)
}

func testRedemptionDistinctCodesExternal(t *testing.T) {
	var wg sync.WaitGroup
	var wins atomic.Int32
	jobs := make([]struct {
		uid int
		key string
	}, 8)
	for i := range jobs {
		uid := 110 + i
		key := fmt.Sprintf("external-distinct-%d", i)
		require.NoError(t, DB.Create(&User{Id: uid, Username: key, AffCode: key, InviterId: 101, Status: common.UserStatusEnabled}).Error)
		require.NoError(t, DB.Create(&Redemption{Key: key, Quota: 100, Status: common.RedemptionCodeStatusEnabled}).Error)
		jobs[i] = struct {
			uid int
			key string
		}{uid, key}
	}
	for _, job := range jobs {
		wg.Add(1)
		go func(uid int, key string) {
			defer wg.Done()
			if _, err := Redeem(key, uid); err == nil {
				wins.Add(1)
			}
		}(job.uid, job.key)
	}
	wg.Wait()
	require.EqualValues(t, 8, wins.Load())
	var owner User
	require.NoError(t, DB.First(&owner, 101).Error)
	require.Equal(t, 125200, owner.AffQuota)
	require.Equal(t, 125200, owner.AffHistoryQuota)
	for _, job := range jobs {
		var friend User
		require.NoError(t, DB.First(&friend, job.uid).Error)
		require.Equal(t, 100, friend.Quota)
	}
}
